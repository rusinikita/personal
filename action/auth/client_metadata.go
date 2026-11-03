package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"sync"
	"time"
)

const (
	cimdFetchTimeout = 5 * time.Second
	cimdMaxSize      = 5 << 10
	cimdCacheTTL     = time.Hour
)

var (
	// AllowedClientHosts are the only hosts a client_id URL may point at.
	AllowedClientHosts = []string{"claude.ai", "chatgpt.com"}

	// ClientMetadataHTTPClient fetches client documents. Its redirect policy
	// and timeout are always overridden by fetchClientMetadata.
	ClientMetadataHTTPClient = http.DefaultClient

	clientMetadataCache = struct {
		sync.Mutex
		entries map[string]cachedClientMetadata
	}{entries: map[string]cachedClientMetadata{}}
)

type cachedClientMetadata struct {
	metadata  *ClientMetadata
	fetchedAt time.Time
}

// ClientMetadata is the part of a fetched Client ID Metadata Document the
// server uses. ClientName is self-asserted and never shown as the client's
// identity; the login page shows the host of ClientID instead.
type ClientMetadata struct {
	ClientID                string   `json:"client_id"`
	ClientName              string   `json:"client_name"`
	RedirectURIs            []string `json:"redirect_uris"`
	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method"`
}

// validateClientIDURL accepts only an https URL on port 443 whose host is on
// the allowed list, so the server never fetches an arbitrary URL.
func validateClientIDURL(clientID string) (*url.URL, error) {
	u, err := url.Parse(clientID)
	if err != nil {
		return nil, errors.New("client_id is not a URL")
	}
	if u.Scheme != "https" {
		return nil, errors.New("client_id must be an https URL")
	}
	if u.User != nil || u.Fragment != "" || u.Path == "" || u.Path == "/" {
		return nil, errors.New("client_id must be an https URL with a path and no credentials or fragment")
	}
	if port := u.Port(); port != "" && port != "443" {
		return nil, errors.New("client_id must use port 443")
	}
	if !slices.Contains(AllowedClientHosts, u.Hostname()) {
		return nil, fmt.Errorf("client host %q is not allowed", u.Hostname())
	}
	return u, nil
}

// fetchClientMetadata returns the client's metadata document, from the
// in-memory cache when fresh. Failed fetches are not cached.
func fetchClientMetadata(ctx context.Context, clientID string) (*ClientMetadata, error) {
	if _, err := validateClientIDURL(clientID); err != nil {
		return nil, err
	}

	clientMetadataCache.Lock()
	entry, ok := clientMetadataCache.entries[clientID]
	clientMetadataCache.Unlock()
	if ok && time.Since(entry.fetchedAt) < cimdCacheTTL {
		return entry.metadata, nil
	}

	metadata, err := downloadClientMetadata(ctx, clientID)
	if err != nil {
		return nil, err
	}

	clientMetadataCache.Lock()
	clientMetadataCache.entries[clientID] = cachedClientMetadata{metadata: metadata, fetchedAt: time.Now()}
	clientMetadataCache.Unlock()

	return metadata, nil
}

func downloadClientMetadata(ctx context.Context, clientID string) (*ClientMetadata, error) {
	client := *ClientMetadataHTTPClient
	client.Timeout = cimdFetchTimeout
	client.CheckRedirect = func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, clientID, nil)
	if err != nil {
		return nil, fmt.Errorf("build client metadata request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	res, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch client metadata: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("client metadata returned status %d", res.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(res.Body, cimdMaxSize+1))
	if err != nil {
		return nil, fmt.Errorf("read client metadata: %w", err)
	}
	if len(body) > cimdMaxSize {
		return nil, errors.New("client metadata document is too large")
	}

	var metadata ClientMetadata
	if err := json.Unmarshal(body, &metadata); err != nil {
		return nil, errors.New("client metadata is not valid JSON")
	}
	if metadata.ClientID != clientID {
		return nil, errors.New("client metadata client_id does not match its URL")
	}
	if metadata.TokenEndpointAuthMethod != "" && metadata.TokenEndpointAuthMethod != "none" {
		return nil, errors.New("only public clients (token_endpoint_auth_method none) are supported")
	}
	if len(metadata.RedirectURIs) == 0 {
		return nil, errors.New("client metadata has no redirect_uris")
	}

	return &metadata, nil
}

func isLoopbackHost(host string) bool {
	return host == "localhost" || host == "127.0.0.1"
}

// redirectURIAllowed checks redirectURI against the document's redirect_uris.
// Loopback URIs match with the port ignored; any other URI must match exactly
// and share the client_id host.
func redirectURIAllowed(metadata *ClientMetadata, redirectURI string) bool {
	target, err := url.Parse(redirectURI)
	if err != nil || target.Fragment != "" || target.Host == "" {
		return false
	}

	if isLoopbackHost(target.Hostname()) {
		for _, registered := range metadata.RedirectURIs {
			r, err := url.Parse(registered)
			if err != nil {
				continue
			}
			if r.Scheme == target.Scheme && r.Hostname() == target.Hostname() &&
				r.Path == target.Path && r.RawQuery == target.RawQuery {
				return true
			}
		}
		return false
	}

	clientURL, err := url.Parse(metadata.ClientID)
	if err != nil || target.Hostname() != clientURL.Hostname() {
		return false
	}
	return slices.Contains(metadata.RedirectURIs, redirectURI)
}
