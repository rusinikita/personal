package auth

import (
	"crypto/rand"
	"encoding/base64"
	"html/template"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"personal/action/webui"
	"personal/gateways"
)

const (
	sessionCookieName = "session"
	csrfCookieName    = "csrf_token"
	sessionExpiry     = 14 * 24 * time.Hour
	csrfExpiry        = 10 * time.Minute
)

// loginPageData is what loginFormTemplate renders.
type loginPageData struct {
	CSRFToken string
	Redirect  string
	Error     string
}

const loginFormContentSrc = `<h1>Login</h1>
{{if .Error}}<p style="color: var(--pico-del-color)">{{.Error}}</p>{{end}}
<form method="POST" action="/web/login">
    <input type="hidden" name="csrf_token" value="{{.CSRFToken}}">
    <input type="hidden" name="redirect" value="{{.Redirect}}">
    <label for="username">Username</label>
    <input type="text" id="username" name="username" required>
    <label for="password">Password</label>
    <input type="password" id="password" name="password" required>
    <button type="submit">Log in</button>
</form>`

var loginFormTemplate = template.Must(template.New("webLoginForm").Parse(loginFormContentSrc))

// WebLoginPageHandler renders the cookie-login form (username/password +
// CSRF token + redirect target) through the shared webui shell. See
// docs/functions/auth-spec.md ("Web Session Login").
func WebLoginPageHandler(c *gin.Context) {
	renderLoginPage(c, http.StatusOK, c.Query("redirect"), "")
}

func renderLoginPage(c *gin.Context, status int, redirect, errMsg string) {
	token, err := csrfTokenForRequest(c)
	if err != nil {
		c.String(http.StatusInternalServerError, "failed to generate csrf token")
		return
	}

	var content strings.Builder
	if err := loginFormTemplate.Execute(&content, loginPageData{
		CSRFToken: token,
		Redirect:  redirect,
		Error:     errMsg,
	}); err != nil {
		c.String(http.StatusInternalServerError, "template error: %v", err)
		return
	}

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.Status(status)
	if err := webui.RenderPage(c.Writer, webui.PageData{
		Title:   "Login",
		Content: template.HTML(content.String()),
	}); err != nil {
		c.String(http.StatusInternalServerError, "render error: %v", err)
	}
}

// csrfTokenForRequest reuses the existing csrf_token cookie if the browser
// already carries one, otherwise mints a new one and sets it — implements
// the double-submit CSRF pattern (see docs/functions/auth-spec.md).
func csrfTokenForRequest(c *gin.Context) (string, error) {
	if existing, err := c.Cookie(csrfCookieName); err == nil && existing != "" {
		return existing, nil
	}

	token, err := generateRandomToken()
	if err != nil {
		return "", err
	}

	c.SetCookieData(&http.Cookie{
		Name:     csrfCookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   int(csrfExpiry.Seconds()),
		HttpOnly: false,
		SameSite: http.SameSiteLaxMode,
	})

	return token, nil
}

func generateRandomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// WebLoginHandler validates the CSRF token (double-submit cookie vs. form
// field) and the username/password against the USERS env var, then issues
// a session cookie carrying the same Claims shape/secret/expiry as the
// OAuth bearer token. See docs/functions/auth-spec.md ("Web Session Login").
func WebLoginHandler(c *gin.Context) {
	redirect := c.PostForm("redirect")

	csrfCookieValue, err := c.Cookie(csrfCookieName)
	if err != nil || csrfCookieValue == "" || csrfCookieValue != c.PostForm("csrf_token") {
		c.String(http.StatusBadRequest, "invalid or missing csrf token")
		return
	}

	username := c.PostForm("username")
	password := c.PostForm("password")

	user := authenticateUser(username, password)
	if user == nil {
		renderLoginPage(c, http.StatusUnauthorized, redirect, "Invalid username or password")
		return
	}

	claims := &Claims{
		UserID:   user.ID,
		UserName: user.UserName,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: &jwt.NumericDate{Time: time.Now().Add(sessionExpiry)},
			Issuer:    BaseAuthURL,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		c.String(http.StatusInternalServerError, "failed to generate session token")
		return
	}

	c.SetCookieData(&http.Cookie{
		Name:     sessionCookieName,
		Value:    tokenString,
		Path:     "/",
		MaxAge:   int(sessionExpiry.Seconds()),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})

	c.Redirect(http.StatusFound, safeRedirectTarget(redirect))
}

// safeRedirectTarget only allows same-site relative paths, guarding against
// open redirects. Anything else falls back to /web/design-system — the only
// WebMiddleware-protected page that exists as of this backlog item; revisit
// this fallback to the nav home page once that backlog item ships.
func safeRedirectTarget(target string) string {
	if strings.HasPrefix(target, "/") && !strings.HasPrefix(target, "//") {
		return target
	}
	return "/web/design-system"
}

// WebLogoutHandler clears the session cookie and redirects to the login
// page. A plain GET works — logout carries no CSRF requirement since a
// forged logout only logs the victim out.
func WebLogoutHandler(c *gin.Context) {
	c.SetCookieData(&http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})
	c.Redirect(http.StatusFound, "/web/login")
}

// WebMiddleware validates the session cookie the same way Middleware()
// validates the bearer token, but redirects to the login page (instead of
// returning a JSON 401) on a missing/invalid/expired token — this flow is
// for human browser access, not machine clients. On success it injects
// user_id into the request context via gateways.WithUserID, and sets the
// username on the gin context for handlers to pass into
// webui.PageData.UserName. See docs/functions/auth-spec.md.
func WebMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString, err := c.Cookie(sessionCookieName)
		if err != nil || tokenString == "" {
			redirectToLogin(c)
			return
		}

		claims := &Claims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return jwtSecret, nil
		})
		if err != nil || !token.Valid {
			redirectToLogin(c)
			return
		}

		c.Request = c.Request.WithContext(gateways.WithUserID(c.Request.Context(), claims.UserID))
		c.Set("user_name", claims.UserName)

		c.Next()
	}
}

func redirectToLogin(c *gin.Context) {
	target := "/web/login?redirect=" + url.QueryEscape(c.Request.URL.RequestURI())
	c.Abort()
	c.Redirect(http.StatusFound, target)
}
