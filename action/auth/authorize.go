package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"personal/action/webui"
	"personal/domain"
	"personal/gateways"
)

// HashToken is how authorization codes and refresh tokens are stored.
func HashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

// authorizeRequest holds the /oauth/authorize parameters, read from the
// query on GET and from the form on POST.
type authorizeRequest struct {
	ClientID            string
	RedirectURI         string
	ResponseType        string
	State               string
	CodeChallenge       string
	CodeChallengeMethod string
	Resource            string
	Scope               string
}

func readAuthorizeRequest(c *gin.Context) authorizeRequest {
	get := c.Query
	if c.Request.Method == http.MethodPost {
		get = c.PostForm
	}
	return authorizeRequest{
		ClientID:            get("client_id"),
		RedirectURI:         get("redirect_uri"),
		ResponseType:        get("response_type"),
		State:               get("state"),
		CodeChallenge:       get("code_challenge"),
		CodeChallengeMethod: get("code_challenge_method"),
		Resource:            get("resource"),
		Scope:               get("scope"),
	}
}

// AuthorizeHandler serves `/oauth/authorize`: GET renders the login page,
// POST checks the password and redirects back with an authorization code.
func AuthorizeHandler(c *gin.Context) {
	req := readAuthorizeRequest(c)

	metadata, err := fetchClientMetadata(c.Request.Context(), req.ClientID)
	if err != nil {
		renderAuthorizeError(c, "Invalid client: "+err.Error())
		return
	}
	if req.RedirectURI == "" || !redirectURIAllowed(metadata, req.RedirectURI) {
		renderAuthorizeError(c, "Invalid redirect_uri for this client.")
		return
	}

	if req.ResponseType != "code" {
		redirectWithParams(c, req.RedirectURI, req.State, map[string]string{
			"error": "unsupported_response_type", "error_description": "response_type must be code",
		})
		return
	}
	if req.CodeChallenge == "" || req.CodeChallengeMethod != "S256" {
		redirectWithParams(c, req.RedirectURI, req.State, map[string]string{
			"error": "invalid_request", "error_description": "S256 code_challenge is required",
		})
		return
	}
	if req.Resource != "" && req.Resource != MCPResourceURL() {
		redirectWithParams(c, req.RedirectURI, req.State, map[string]string{
			"error": "invalid_target", "error_description": "unknown resource",
		})
		return
	}

	if c.Request.Method != http.MethodPost {
		renderOAuthLoginPage(c, http.StatusOK, req, "")
		return
	}

	user := authenticateUser(c.PostForm("username"), c.PostForm("password"))
	if user == nil {
		renderOAuthLoginPage(c, http.StatusUnauthorized, req, "Invalid username or password")
		return
	}

	db := gateways.DBFromContext(c.Request.Context())
	if err := db.DeleteExpiredOAuthState(c.Request.Context()); err != nil {
		log.Printf("delete expired oauth state: %v", err)
	}

	code, err := generateRandomToken()
	if err != nil {
		renderAuthorizeError(c, "Failed to generate authorization code.")
		return
	}

	resource := req.Resource
	if resource == "" {
		resource = MCPResourceURL()
	}

	err = db.CreateOAuthCode(c.Request.Context(), domain.OAuthCode{
		CodeHash:      HashToken(code),
		ClientID:      req.ClientID,
		UserID:        user.ID,
		RedirectURI:   req.RedirectURI,
		CodeChallenge: req.CodeChallenge,
		Resource:      resource,
		ExpiresAt:     time.Now().Add(authCodeExpiry),
	})
	if err != nil {
		renderAuthorizeError(c, "Failed to store authorization code.")
		return
	}

	redirectWithParams(c, req.RedirectURI, req.State, map[string]string{"code": code})
}

// redirectWithParams redirects to redirectURI with params, state and iss
// (RFC 9207) added to its query.
func redirectWithParams(c *gin.Context, redirectURI, state string, params map[string]string) {
	target, err := url.Parse(redirectURI)
	if err != nil {
		renderAuthorizeError(c, "Invalid redirect_uri.")
		return
	}

	query := target.Query()
	for k, v := range params {
		query.Set(k, v)
	}
	if state != "" {
		query.Set("state", state)
	}
	query.Set("iss", BaseAuthURL)
	target.RawQuery = query.Encode()

	c.Redirect(http.StatusFound, target.String())
}

type oauthLoginPageData struct {
	Request      authorizeRequest
	ClientHost   string
	RedirectHost string
	Loopback     bool
	Error        string
}

const oauthLoginContentSrc = `<h1>Authorize {{.ClientHost}}</h1>
<p><strong>{{.ClientHost}}</strong> asks for access to your account. After login you will be sent to <strong>{{.RedirectHost}}</strong>.</p>
{{if .Loopback}}<p style="color: var(--pico-del-color)">The redirect goes to an application on this computer ({{.RedirectHost}}). Continue only if you started this connection yourself.</p>{{end}}
{{if .Error}}<p style="color: var(--pico-del-color)">{{.Error}}</p>{{end}}
<form method="POST" action="/oauth/authorize">
    <input type="hidden" name="client_id" value="{{.Request.ClientID}}">
    <input type="hidden" name="redirect_uri" value="{{.Request.RedirectURI}}">
    <input type="hidden" name="response_type" value="{{.Request.ResponseType}}">
    <input type="hidden" name="state" value="{{.Request.State}}">
    <input type="hidden" name="code_challenge" value="{{.Request.CodeChallenge}}">
    <input type="hidden" name="code_challenge_method" value="{{.Request.CodeChallengeMethod}}">
    <input type="hidden" name="resource" value="{{.Request.Resource}}">
    <input type="hidden" name="scope" value="{{.Request.Scope}}">
    <label for="username">Username</label>
    <input type="text" id="username" name="username" required>
    <label for="password">Password</label>
    <input type="password" id="password" name="password" required>
    <button type="submit">Log in and allow</button>
</form>`

var oauthLoginTemplate = template.Must(template.New("oauthLogin").Parse(oauthLoginContentSrc))

// renderOAuthLoginPage renders the login form that doubles as the consent
// screen: it names the client by its client_id host and shows the redirect host.
func renderOAuthLoginPage(c *gin.Context, status int, req authorizeRequest, errMsg string) {
	clientURL, _ := url.Parse(req.ClientID)
	redirectURL, _ := url.Parse(req.RedirectURI)

	var content strings.Builder
	if err := oauthLoginTemplate.Execute(&content, oauthLoginPageData{
		Request:      req,
		ClientHost:   clientURL.Hostname(),
		RedirectHost: redirectURL.Host,
		Loopback:     isLoopbackHost(redirectURL.Hostname()),
		Error:        errMsg,
	}); err != nil {
		c.String(http.StatusInternalServerError, "template error: %v", err)
		return
	}

	renderOAuthPage(c, status, "Authorization", template.HTML(content.String()))
}

var authorizeErrorTemplate = template.Must(template.New("authorizeError").Parse(
	`<h1>Authorization failed</h1><p>{{.}}</p>`))

// renderAuthorizeError shows an error page instead of redirecting, for
// errors where the redirect target cannot be trusted.
func renderAuthorizeError(c *gin.Context, message string) {
	var content strings.Builder
	if err := authorizeErrorTemplate.Execute(&content, message); err != nil {
		c.String(http.StatusInternalServerError, "template error: %v", err)
		return
	}
	renderOAuthPage(c, http.StatusBadRequest, "Authorization failed", template.HTML(content.String()))
}

func renderOAuthPage(c *gin.Context, status int, title string, content template.HTML) {
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.Status(status)
	if err := webui.RenderPage(c.Writer, webui.PageData{Title: title, Content: content}); err != nil {
		c.String(http.StatusInternalServerError, "render error: %v", err)
	}
}
