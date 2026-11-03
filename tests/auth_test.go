package tests

// Covers the OAuth 2.1 authorization server from docs/functions/auth-spec.md.

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"personal/action/auth"
	"personal/domain"
	"personal/gateways"
	"personal/transport/web"
)

const (
	oauthBaseURL     = "https://personal.test"
	oauthMCPURL      = oauthBaseURL + "/app/mcp"
	oauthJWTSecret   = "test-oauth-secret"
	oauthClientHost  = "example.com"
	oauthRedirectURI = "https://example.com/callback"
	oauthVerifier    = "verifier-0123456789-0123456789-0123456789-abcdef"
	oauthClientName  = "Totally Trusted Client"
)

// oauthFixture is a router with the OAuth routes and a protected /app/mcp,
// plus a TLS server that serves client metadata documents. Every client_id
// host is dialed to that server.
type oauthFixture struct {
	s      *IntegrationTestSuite
	userID int64
	router *gin.Engine

	mu       sync.Mutex
	handlers map[string]http.HandlerFunc
	fetches  map[string]int
}

func (s *IntegrationTestSuite) newOAuthFixture(userID int64) *oauthFixture {
	t := s.T()
	t.Setenv("USERS", fmt.Sprintf("%d:alice:s3cret", userID))
	t.Setenv("JWT_SECRET", oauthJWTSecret)
	t.Setenv("BASE_URL", oauthBaseURL)
	auth.InitializeAuth()

	f := &oauthFixture{s: s, userID: userID, handlers: map[string]http.HandlerFunc{}, fetches: map[string]int{}}

	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		f.fetches[r.URL.Path]++
		h := f.handlers[r.URL.Path]
		f.mu.Unlock()
		if h == nil {
			http.NotFound(w, r)
			return
		}
		h(w, r)
	}))
	t.Cleanup(srv.Close)

	transport := srv.Client().Transport.(*http.Transport).Clone()
	addr := srv.Listener.Addr().String()
	transport.DialContext = func(ctx context.Context, network, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, network, addr)
	}

	prevHosts, prevClient := auth.AllowedClientHosts, auth.ClientMetadataHTTPClient
	auth.AllowedClientHosts = []string{oauthClientHost}
	auth.ClientMetadataHTTPClient = &http.Client{Transport: transport}
	t.Cleanup(func() {
		auth.AllowedClientHosts, auth.ClientMetadataHTTPClient = prevHosts, prevClient
	})

	gin.SetMode(gin.TestMode)
	f.router = gin.New()
	web.RegisterOAuth(f.router, s.Repo())
	f.router.Group("/app", auth.Middleware()).Any("/mcp", func(c *gin.Context) {
		c.String(http.StatusOK, "%d", gateways.UserIDFromContext(c.Request.Context()))
	})

	return f
}

func randomHex() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// path reserves a fresh path, so cached documents never leak between tests.
func (f *oauthFixture) path(name string) string {
	return "/clients/" + randomHex() + "/" + name
}

func (f *oauthFixture) handle(path string, h http.HandlerFunc) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.handlers[path] = h
}

func (f *oauthFixture) fetchCount(path string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.fetches[path]
}

func clientDocument(clientID string, redirectURIs ...string) map[string]any {
	if len(redirectURIs) == 0 {
		redirectURIs = []string{oauthRedirectURI, "http://localhost/callback"}
	}
	return map[string]any{
		"client_id":                  clientID,
		"client_name":                oauthClientName,
		"redirect_uris":              redirectURIs,
		"token_endpoint_auth_method": "none",
	}
}

func jsonHandler(doc any) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(doc)
	}
}

// addClient serves a valid document and returns its client_id and path.
func (f *oauthFixture) addClient(redirectURIs ...string) (string, string) {
	path := f.path("client.json")
	clientID := "https://" + oauthClientHost + path
	f.handle(path, jsonHandler(clientDocument(clientID, redirectURIs...)))
	return clientID, path
}

func pkceChallenge(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func authorizeParams(clientID string, overrides map[string]string) url.Values {
	v := url.Values{
		"client_id":             {clientID},
		"redirect_uri":          {oauthRedirectURI},
		"response_type":         {"code"},
		"state":                 {"state-123"},
		"code_challenge":        {pkceChallenge(oauthVerifier)},
		"code_challenge_method": {"S256"},
	}
	for k, val := range overrides {
		if val == "" {
			v.Del(k)
		} else {
			v.Set(k, val)
		}
	}
	return v
}

func (f *oauthFixture) get(target string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	f.router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, target, nil))
	return w
}

func (f *oauthFixture) postForm(target string, form url.Values) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, target, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	f.router.ServeHTTP(w, req)
	return w
}

func (f *oauthFixture) getAuthorize(params url.Values) *httptest.ResponseRecorder {
	return f.get("/oauth/authorize?" + params.Encode())
}

func (f *oauthFixture) postAuthorize(params url.Values, password string) *httptest.ResponseRecorder {
	form := url.Values{}
	for k, v := range params {
		form[k] = v
	}
	form.Set("username", "alice")
	form.Set("password", password)
	return f.postForm("/oauth/authorize", form)
}

// obtainCode runs a successful POST /oauth/authorize and returns the code.
func (f *oauthFixture) obtainCode(params url.Values) string {
	w := f.postAuthorize(params, "s3cret")
	require.Equal(f.s.T(), http.StatusFound, w.Code, w.Body.String())
	loc, err := url.Parse(w.Header().Get("Location"))
	require.NoError(f.s.T(), err)
	code := loc.Query().Get("code")
	require.NotEmpty(f.s.T(), code)
	return code
}

func codeExchangeForm(clientID, code string) url.Values {
	return url.Values{
		"grant_type":    {"authorization_code"},
		"client_id":     {clientID},
		"code":          {code},
		"redirect_uri":  {oauthRedirectURI},
		"code_verifier": {oauthVerifier},
	}
}

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Error        string `json:"error"`
}

func (f *oauthFixture) token(form url.Values) (int, tokenResponse) {
	w := f.postForm("/oauth/token", form)
	var res tokenResponse
	require.NoError(f.s.T(), json.Unmarshal(w.Body.Bytes(), &res), w.Body.String())
	return w.Code, res
}

// obtainTokens runs the full authorization code flow for clientID.
func (f *oauthFixture) obtainTokens(clientID string) tokenResponse {
	code := f.obtainCode(authorizeParams(clientID, nil))
	status, res := f.token(codeExchangeForm(clientID, code))
	require.Equal(f.s.T(), http.StatusOK, status, res.Error)
	return res
}

func refreshForm(clientID, refreshToken string) url.Values {
	return url.Values{
		"grant_type":    {"refresh_token"},
		"client_id":     {clientID},
		"refresh_token": {refreshToken},
	}
}

func (f *oauthFixture) callMCP(accessToken string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/app/mcp", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	w := httptest.NewRecorder()
	f.router.ServeHTTP(w, req)
	return w
}

func signTestJWT(userID int64, audience ...string) string {
	claims := &auth.Claims{
		UserID:   userID,
		UserName: "alice",
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    oauthBaseURL,
			Audience:  audience,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}
	token, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(oauthJWTSecret))
	return token
}

// --- Discovery --------------------------------------------------------------

func (s *IntegrationTestSuite) TestAuth_WellKnown() {
	f := s.newOAuthFixture(s.UserID())

	w := f.get("/.well-known/oauth-authorization-server")
	s.Require().Equal(http.StatusOK, w.Code)
	var meta map[string]any
	s.Require().NoError(json.Unmarshal(w.Body.Bytes(), &meta))

	pw := f.get("/.well-known/oauth-protected-resource")
	var resource map[string]any
	s.Require().NoError(json.Unmarshal(pw.Body.Bytes(), &resource))

	t := s.T()
	assert.Equal(t, oauthBaseURL, meta["issuer"])
	assert.Equal(t, oauthBaseURL+"/oauth/authorize", meta["authorization_endpoint"])
	assert.Equal(t, oauthBaseURL+"/oauth/token", meta["token_endpoint"])
	assert.Equal(t, true, meta["client_id_metadata_document_supported"])
	assert.Equal(t, []any{"none"}, meta["token_endpoint_auth_methods_supported"])
	assert.NotContains(t, meta, "registration_endpoint")
	assert.Equal(t, []any{"S256"}, meta["code_challenge_methods_supported"])
	assert.Equal(t, []any{"authorization_code", "refresh_token"}, meta["grant_types_supported"])
	assert.Equal(t, []any{"offline_access"}, meta["scopes_supported"])
	assert.Equal(t, true, meta["authorization_response_iss_parameter_supported"])
	assert.NotContains(t, meta, "subject_types_supported")
	assert.NotContains(t, meta, "id_token_signing_alg_values_supported")
	assert.Equal(t, []any{meta["issuer"]}, resource["authorization_servers"])
}

func (s *IntegrationTestSuite) TestAuth_ProtectedResource() {
	f := s.newOAuthFixture(s.UserID())

	for _, path := range []string{"/.well-known/oauth-protected-resource", "/.well-known/oauth-protected-resource/app/mcp"} {
		s.Run(path, func() {
			w := f.get(path)
			s.Require().Equal(http.StatusOK, w.Code)
			var doc map[string]any
			s.Require().NoError(json.Unmarshal(w.Body.Bytes(), &doc))
			s.Equal(oauthMCPURL, doc["resource"])
			s.Equal([]any{oauthBaseURL}, doc["authorization_servers"])
		})
	}
}

func (s *IntegrationTestSuite) TestAuth_Unauthorized() {
	f := s.newOAuthFixture(s.UserID())

	w := f.get("/app/mcp")

	s.Equal(http.StatusUnauthorized, w.Code)
	s.Equal(`Bearer resource_metadata="`+oauthBaseURL+`/.well-known/oauth-protected-resource/app/mcp"`, w.Header().Get("WWW-Authenticate"))
}

// --- Client metadata --------------------------------------------------------

func (s *IntegrationTestSuite) TestAuth_ClientMetadata_Rejected() {
	f := s.newOAuthFixture(s.UserID())

	validPath := f.path("valid.json")
	validID := "https://" + oauthClientHost + validPath
	f.handle(validPath, jsonHandler(clientDocument(validID)))

	mismatchPath := f.path("mismatch.json")
	f.handle(mismatchPath, jsonHandler(clientDocument("https://"+oauthClientHost+"/someone-else.json")))

	redirectPath := f.path("redirect.json")
	redirectTarget := f.path("target.json")
	redirectID := "https://" + oauthClientHost + redirectPath
	f.handle(redirectPath, func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, redirectTarget, http.StatusFound)
	})
	f.handle(redirectTarget, jsonHandler(clientDocument(redirectID)))

	bigPath := f.path("big.json")
	bigID := "https://" + oauthClientHost + bigPath
	bigDoc := clientDocument(bigID)
	bigDoc["client_name"] = strings.Repeat("x", 6000)
	f.handle(bigPath, jsonHandler(bigDoc))

	secretPath := f.path("secret.json")
	secretID := "https://" + oauthClientHost + secretPath
	secretDoc := clientDocument(secretID)
	secretDoc["token_endpoint_auth_method"] = "client_secret_basic"
	f.handle(secretPath, jsonHandler(secretDoc))

	tests := []struct {
		name        string
		clientID    string
		noFetchPath string
	}{
		{name: "host not allowed, no fetch", clientID: "https://evil.test" + validPath, noFetchPath: validPath},
		{name: "non-https URL", clientID: "http://" + oauthClientHost + validPath, noFetchPath: validPath},
		{name: "non-443 port", clientID: "https://" + oauthClientHost + ":8443" + validPath, noFetchPath: validPath},
		{name: "document client_id mismatch", clientID: "https://" + oauthClientHost + mismatchPath},
		{name: "redirect response not followed", clientID: redirectID, noFetchPath: redirectTarget},
		{name: "oversized document", clientID: bigID},
		{name: "confidential client", clientID: secretID},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			w := f.getAuthorize(authorizeParams(tt.clientID, nil))

			s.Equal(http.StatusBadRequest, w.Code)
			s.Empty(w.Header().Get("Location"))
			s.Contains(w.Body.String(), "Invalid client")
			if tt.noFetchPath != "" {
				s.Zero(f.fetchCount(tt.noFetchPath))
			}
		})
	}
}

func (s *IntegrationTestSuite) TestAuth_ClientMetadata_FailureNotCached() {
	f := s.newOAuthFixture(s.UserID())

	path := f.path("flaky.json")
	clientID := "https://" + oauthClientHost + path
	var calls int
	f.handle(path, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			http.Error(w, "boom", http.StatusInternalServerError)
			return
		}
		jsonHandler(clientDocument(clientID))(w, r)
	})

	s.Equal(http.StatusBadRequest, f.getAuthorize(authorizeParams(clientID, nil)).Code)
	s.Equal(http.StatusOK, f.getAuthorize(authorizeParams(clientID, nil)).Code)
	s.Equal(2, f.fetchCount(path))
}

func (s *IntegrationTestSuite) TestAuth_ClientMetadata_Cached() {
	f := s.newOAuthFixture(s.UserID())
	clientID, path := f.addClient()

	s.Equal(http.StatusOK, f.getAuthorize(authorizeParams(clientID, nil)).Code)
	s.Equal(http.StatusOK, f.getAuthorize(authorizeParams(clientID, nil)).Code)
	s.Equal(1, f.fetchCount(path))
}

// --- Authorize --------------------------------------------------------------

func (s *IntegrationTestSuite) TestAuth_Authorize_Page() {
	f := s.newOAuthFixture(s.UserID())
	clientID, _ := f.addClient()

	s.Run("shows client_id host and redirect host, not client_name", func() {
		w := f.getAuthorize(authorizeParams(clientID, nil))

		s.Equal(http.StatusOK, w.Code)
		body := w.Body.String()
		s.Contains(body, "Authorize "+oauthClientHost)
		s.Contains(body, "sent to <strong>"+oauthClientHost+"</strong>")
		s.NotContains(body, oauthClientName)
		s.Contains(body, `id="username"`)
	})

	s.Run("escapes query values", func() {
		w := f.getAuthorize(authorizeParams(clientID, map[string]string{"state": `"><script>alert(1)</script>`}))

		s.Equal(http.StatusOK, w.Code)
		s.NotContains(w.Body.String(), "<script>alert(1)</script>")
		s.Contains(w.Body.String(), "&lt;script&gt;")
	})

	s.Run("warns about loopback redirect", func() {
		w := f.getAuthorize(authorizeParams(clientID, map[string]string{"redirect_uri": "http://localhost:53682/callback"}))

		s.Equal(http.StatusOK, w.Code)
		s.Contains(w.Body.String(), "on this computer")
	})
}

func (s *IntegrationTestSuite) TestAuth_Authorize_Rejected() {
	f := s.newOAuthFixture(s.UserID())
	clientID, _ := f.addClient()
	foreignID, _ := f.addClient(oauthRedirectURI, "https://attacker.test/callback")

	tests := []struct {
		name      string
		clientID  string
		overrides map[string]string
		wantError string // empty: error page, no redirect
	}{
		{name: "unlisted redirect URI, no redirect", clientID: clientID, overrides: map[string]string{"redirect_uri": "https://example.com/other"}},
		{name: "redirect on another host", clientID: foreignID, overrides: map[string]string{"redirect_uri": "https://attacker.test/callback"}},
		{name: "missing redirect URI", clientID: clientID, overrides: map[string]string{"redirect_uri": ""}},
		{name: "missing challenge", clientID: clientID, overrides: map[string]string{"code_challenge": ""}, wantError: "invalid_request"},
		{name: "plain challenge method", clientID: clientID, overrides: map[string]string{"code_challenge_method": "plain"}, wantError: "invalid_request"},
		{name: "foreign resource", clientID: clientID, overrides: map[string]string{"resource": "https://other.test/mcp"}, wantError: "invalid_target"},
		{name: "unsupported response type", clientID: clientID, overrides: map[string]string{"response_type": "token"}, wantError: "unsupported_response_type"},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			w := f.getAuthorize(authorizeParams(tt.clientID, tt.overrides))

			if tt.wantError == "" {
				s.Equal(http.StatusBadRequest, w.Code)
				s.Empty(w.Header().Get("Location"))
				return
			}

			s.Require().Equal(http.StatusFound, w.Code)
			loc, err := url.Parse(w.Header().Get("Location"))
			s.Require().NoError(err)
			s.Equal(oauthClientHost, loc.Host)
			s.Equal(tt.wantError, loc.Query().Get("error"))
			s.Equal("state-123", loc.Query().Get("state"))
			s.Equal(oauthBaseURL, loc.Query().Get("iss"))
			s.Empty(loc.Query().Get("code"))
		})
	}
}

func (s *IntegrationTestSuite) TestAuth_Authorize_WrongPassword() {
	f := s.newOAuthFixture(s.UserID())
	clientID, _ := f.addClient()

	w := f.postAuthorize(authorizeParams(clientID, nil), "wrong")

	s.Equal(http.StatusUnauthorized, w.Code)
	s.Empty(w.Header().Get("Location"))
	s.Contains(w.Body.String(), "Invalid username or password")
}

func (s *IntegrationTestSuite) TestAuth_Authorize_Success() {
	f := s.newOAuthFixture(s.UserID())
	clientID, _ := f.addClient()

	tests := []struct {
		name        string
		redirectURI string
	}{
		{name: "registered redirect carries code, state and iss", redirectURI: oauthRedirectURI},
		{name: "loopback redirect on any port", redirectURI: "http://localhost:53682/callback"},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			w := f.postAuthorize(authorizeParams(clientID, map[string]string{"redirect_uri": tt.redirectURI}), "s3cret")

			s.Require().Equal(http.StatusFound, w.Code, w.Body.String())
			loc, err := url.Parse(w.Header().Get("Location"))
			s.Require().NoError(err)
			s.True(strings.HasPrefix(loc.String(), tt.redirectURI+"?"), loc.String())
			s.NotEmpty(loc.Query().Get("code"))
			s.Equal("state-123", loc.Query().Get("state"))
			s.Equal(oauthBaseURL, loc.Query().Get("iss"))
		})
	}
}

// --- Token ------------------------------------------------------------------

func (s *IntegrationTestSuite) TestAuth_Token_FullFlow() {
	userID := s.UserID()
	f := s.newOAuthFixture(userID)
	clientID, _ := f.addClient()

	code := f.obtainCode(authorizeParams(clientID, map[string]string{"resource": oauthMCPURL}))
	status, res := f.token(codeExchangeForm(clientID, code))

	s.Require().Equal(http.StatusOK, status, res.Error)
	s.Equal("Bearer", res.TokenType)
	s.Equal(3600, res.ExpiresIn)
	s.NotEmpty(res.RefreshToken, "refresh token is issued without offline_access")

	claims := &auth.Claims{}
	_, err := jwt.ParseWithClaims(res.AccessToken, claims, func(*jwt.Token) (any, error) { return []byte(oauthJWTSecret), nil })
	s.Require().NoError(err)
	s.Equal(jwt.ClaimStrings{oauthMCPURL}, claims.Audience)
	s.Equal(userID, claims.UserID)

	stored, err := s.Repo().GetOAuthRefreshToken(context.Background(), auth.HashToken(res.RefreshToken))
	s.Require().NoError(err)
	s.Require().NotNil(stored)
	s.Equal(clientID, stored.ClientID)
	s.Equal(oauthMCPURL, stored.Resource)

	w := f.callMCP(res.AccessToken)
	s.Equal(http.StatusOK, w.Code)
	s.Equal(fmt.Sprint(userID), w.Body.String())
}

func (s *IntegrationTestSuite) TestAuth_Token_Rejected() {
	f := s.newOAuthFixture(s.UserID())
	clientID, _ := f.addClient()
	otherID, _ := f.addClient()

	tests := []struct {
		name   string
		mutate func(form url.Values)
	}{
		{name: "wrong verifier", mutate: func(form url.Values) { form.Set("code_verifier", strings.Repeat("w", 43)) }},
		{name: "code of another client", mutate: func(form url.Values) { form.Set("client_id", otherID) }},
		{name: "mismatched redirect URI", mutate: func(form url.Values) { form.Set("redirect_uri", "http://localhost/callback") }},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			form := codeExchangeForm(clientID, f.obtainCode(authorizeParams(clientID, nil)))
			tt.mutate(form)

			status, res := f.token(form)

			s.Equal(http.StatusBadRequest, status)
			s.Equal("invalid_grant", res.Error)
		})
	}
}

func (s *IntegrationTestSuite) TestAuth_Token_CodeReuse() {
	f := s.newOAuthFixture(s.UserID())
	clientID, _ := f.addClient()
	code := f.obtainCode(authorizeParams(clientID, nil))

	status, _ := f.token(codeExchangeForm(clientID, code))
	s.Require().Equal(http.StatusOK, status)

	status, res := f.token(codeExchangeForm(clientID, code))
	s.Equal(http.StatusBadRequest, status)
	s.Equal("invalid_grant", res.Error)
}

func (s *IntegrationTestSuite) TestAuth_Token_ExpiredCode() {
	userID := s.UserID()
	f := s.newOAuthFixture(userID)
	clientID, _ := f.addClient()

	s.Require().NoError(s.Repo().CreateOAuthCode(context.Background(), domain.OAuthCode{
		CodeHash:      auth.HashToken("expired-code"),
		ClientID:      clientID,
		UserID:        userID,
		RedirectURI:   oauthRedirectURI,
		CodeChallenge: pkceChallenge(oauthVerifier),
		Resource:      oauthMCPURL,
		ExpiresAt:     time.Now().Add(-time.Minute),
	}))

	status, res := f.token(codeExchangeForm(clientID, "expired-code"))

	s.Equal(http.StatusBadRequest, status)
	s.Equal("invalid_grant", res.Error)
}

// --- Refresh ----------------------------------------------------------------

func (s *IntegrationTestSuite) TestAuth_Refresh_Rotation() {
	f := s.newOAuthFixture(s.UserID())
	clientID, _ := f.addClient()
	first := f.obtainTokens(clientID)

	status, second := f.token(refreshForm(clientID, first.RefreshToken))

	s.Require().Equal(http.StatusOK, status, second.Error)
	s.NotEqual(first.RefreshToken, second.RefreshToken)
	s.Equal(http.StatusOK, f.callMCP(second.AccessToken).Code)

	old, err := s.Repo().GetOAuthRefreshToken(context.Background(), auth.HashToken(first.RefreshToken))
	s.Require().NoError(err)
	s.Require().NotNil(old)
	s.NotNil(old.UsedAt)

	next, err := s.Repo().GetOAuthRefreshToken(context.Background(), auth.HashToken(second.RefreshToken))
	s.Require().NoError(err)
	s.Require().NotNil(next)
	s.Nil(next.UsedAt)
	s.Equal(old.FamilyID, next.FamilyID)
}

func (s *IntegrationTestSuite) TestAuth_Refresh_ReuseInsideWindow() {
	f := s.newOAuthFixture(s.UserID())
	clientID, _ := f.addClient()
	first := f.obtainTokens(clientID)

	status, a := f.token(refreshForm(clientID, first.RefreshToken))
	s.Require().Equal(http.StatusOK, status)
	status, b := f.token(refreshForm(clientID, first.RefreshToken))
	s.Require().Equal(http.StatusOK, status, b.Error)

	tokenA, err := s.Repo().GetOAuthRefreshToken(context.Background(), auth.HashToken(a.RefreshToken))
	s.Require().NoError(err)
	tokenB, err := s.Repo().GetOAuthRefreshToken(context.Background(), auth.HashToken(b.RefreshToken))
	s.Require().NoError(err)
	s.Require().NotNil(tokenA)
	s.Require().NotNil(tokenB)
	s.Equal(tokenA.FamilyID, tokenB.FamilyID)
}

func (s *IntegrationTestSuite) seedRefreshToken(userID int64, raw, clientID string, familyID uuid.UUID, expiresAt time.Time, usedAt *time.Time) {
	s.Require().NoError(s.Repo().CreateOAuthRefreshToken(context.Background(), domain.OAuthRefreshToken{
		TokenHash: auth.HashToken(raw),
		FamilyID:  familyID,
		ClientID:  clientID,
		UserID:    userID,
		Resource:  oauthMCPURL,
		ExpiresAt: expiresAt,
		UsedAt:    usedAt,
	}))
}

func (s *IntegrationTestSuite) TestAuth_Refresh_ReuseAfterWindowRevokesFamily() {
	userID := s.UserID()
	f := s.newOAuthFixture(userID)
	clientID, _ := f.addClient()

	family := uuid.New()
	usedAt := time.Now().Add(-time.Minute)
	s.seedRefreshToken(userID, "stolen-old", clientID, family, time.Now().Add(time.Hour), &usedAt)
	s.seedRefreshToken(userID, "current", clientID, family, time.Now().Add(time.Hour), nil)

	status, res := f.token(refreshForm(clientID, "stolen-old"))

	s.Equal(http.StatusBadRequest, status)
	s.Equal("invalid_grant", res.Error)
	current, err := s.Repo().GetOAuthRefreshToken(context.Background(), auth.HashToken("current"))
	s.Require().NoError(err)
	s.Nil(current, "the whole family is revoked")

	status, res = f.token(refreshForm(clientID, "current"))
	s.Equal(http.StatusBadRequest, status)
	s.Equal("invalid_grant", res.Error)
}

func (s *IntegrationTestSuite) TestAuth_Refresh_Rejected() {
	userID := s.UserID()
	f := s.newOAuthFixture(userID)
	clientID, _ := f.addClient()
	otherID, _ := f.addClient()

	s.seedRefreshToken(userID, "live", clientID, uuid.New(), time.Now().Add(time.Hour), nil)
	s.seedRefreshToken(userID, "expired", clientID, uuid.New(), time.Now().Add(-time.Minute), nil)

	tests := []struct {
		name     string
		clientID string
		token    string
	}{
		{name: "another client_id", clientID: otherID, token: "live"},
		{name: "expired token", clientID: clientID, token: "expired"},
		{name: "unknown token", clientID: clientID, token: "unknown"},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			status, res := f.token(refreshForm(tt.clientID, tt.token))

			s.Equal(http.StatusBadRequest, status)
			s.Equal("invalid_grant", res.Error)
		})
	}
}

// --- Middleware -------------------------------------------------------------

func (s *IntegrationTestSuite) TestAuth_Middleware() {
	userID := s.UserID()
	f := s.newOAuthFixture(userID)

	tests := []struct {
		name       string
		token      string
		wantStatus int
	}{
		{name: "valid token", token: signTestJWT(userID, oauthMCPURL), wantStatus: http.StatusOK},
		{name: "wrong aud", token: signTestJWT(userID, "https://other.test/app/mcp"), wantStatus: http.StatusUnauthorized},
		{name: "web session cookie used as bearer", token: signTestJWT(userID), wantStatus: http.StatusUnauthorized},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			w := f.callMCP(tt.token)

			s.Equal(tt.wantStatus, w.Code)
			if tt.wantStatus == http.StatusOK {
				s.Equal(fmt.Sprint(userID), w.Body.String())
			} else {
				s.Contains(w.Header().Get("WWW-Authenticate"), `error="invalid_token"`)
			}
		})
	}
}

// --- Cleanup ----------------------------------------------------------------

func (s *IntegrationTestSuite) TestAuth_Cleanup() {
	userID := s.UserID()
	f := s.newOAuthFixture(userID)
	clientID, _ := f.addClient()
	ctx := context.Background()

	s.seedRefreshToken(userID, "cleanup-expired", clientID, uuid.New(), time.Now().Add(-time.Minute), nil)
	s.seedRefreshToken(userID, "cleanup-live", clientID, uuid.New(), time.Now().Add(time.Hour), nil)
	s.Require().NoError(s.Repo().CreateOAuthCode(ctx, domain.OAuthCode{
		CodeHash: auth.HashToken("cleanup-code"), ClientID: clientID, UserID: userID,
		RedirectURI: oauthRedirectURI, CodeChallenge: "x", Resource: oauthMCPURL,
		ExpiresAt: time.Now().Add(-time.Minute),
	}))

	f.obtainCode(authorizeParams(clientID, nil))

	expired, err := s.Repo().GetOAuthRefreshToken(ctx, auth.HashToken("cleanup-expired"))
	s.Require().NoError(err)
	s.Nil(expired)
	live, err := s.Repo().GetOAuthRefreshToken(ctx, auth.HashToken("cleanup-live"))
	s.Require().NoError(err)
	s.NotNil(live)
	code, err := s.Repo().ConsumeOAuthCode(ctx, auth.HashToken("cleanup-code"))
	s.Require().NoError(err)
	s.Nil(code)
}
