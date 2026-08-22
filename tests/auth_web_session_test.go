package tests

// Covers the "Web Session Login" flow from docs/functions/auth-spec.md
// (backlog: "Authorization for web dashboards", 20-08-26).

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"personal/action/auth"
	"personal/gateways"
)

// setupWebSessionAuth seeds action/auth's package-level user store and JWT
// secret for the duration of one test (t.Setenv auto-restores after it).
func setupWebSessionAuth(t *testing.T) {
	t.Setenv("USERS", "42:alice:s3cret")
	t.Setenv("JWT_SECRET", "test-jwt-secret")
	t.Setenv("BASE_URL", "http://localhost:8081")
	auth.InitializeAuth()
}

// webSessionRouter wires the routes the same way main.go does: public
// login/logout routes, plus one protected route behind WebMiddleware whose
// handler echoes back what the middleware put in context, so tests can
// assert on it directly.
func webSessionRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/web/login", auth.WebLoginPageHandler)
	r.POST("/web/login", auth.WebLoginHandler)
	r.GET("/web/logout", auth.WebLogoutHandler)

	protected := r.Group("/web/protected")
	protected.Use(auth.WebMiddleware())
	protected.GET("/dashboard", func(c *gin.Context) {
		userID := gateways.UserIDFromContext(c.Request.Context())
		userName := c.GetString("user_name")
		c.String(http.StatusOK, "%d:%s", userID, userName)
	})

	return r
}

var csrfFieldRe = regexp.MustCompile(`name="csrf_token" value="([^"]+)"`)

// csrfCookie returns the csrf_token cookie set by a GET /web/login response.
func csrfCookie(t *testing.T, res *http.Response) *http.Cookie {
	for _, c := range res.Cookies() {
		if c.Name == "csrf_token" {
			return c
		}
	}
	t.Fatal("csrf_token cookie not set by GET /web/login")
	return nil
}

// csrfFormValue extracts the hidden csrf_token form value from the login
// page body (must match the cookie for a legitimate request).
func csrfFormValue(t *testing.T, body string) string {
	m := csrfFieldRe.FindStringSubmatch(body)
	require.Len(t, m, 2, "login page must render a hidden csrf_token field")
	return m[1]
}

func sessionCookie(res *http.Response) *http.Cookie {
	for _, c := range res.Cookies() {
		if c.Name == "session" {
			return c
		}
	}
	return nil
}

// --- GET /web/login -----------------------------------------------------

func TestWebLogin_GET_RendersFormWithCSRFCookieAndHiddenField(t *testing.T) {
	setupWebSessionAuth(t)
	r := webSessionRouter()

	req := httptest.NewRequest(http.MethodGet, "/web/login", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	res := w.Result()

	assert.Equal(t, http.StatusOK, res.StatusCode)
	body := w.Body.String()
	assert.Contains(t, body, `name="username"`)
	assert.Contains(t, body, `name="password"`)

	cookie := csrfCookie(t, res)
	formValue := csrfFormValue(t, body)
	assert.Equal(t, cookie.Value, formValue, "csrf cookie and hidden field must match (double-submit)")
	assert.False(t, cookie.HttpOnly, "csrf cookie is non-httpOnly by design (double-submit pattern needs no server-side lookup)")
}

func TestWebLogin_GET_PreservesRedirectParam(t *testing.T) {
	setupWebSessionAuth(t)
	r := webSessionRouter()

	req := httptest.NewRequest(http.MethodGet, "/web/login?redirect=/web/money", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Contains(t, w.Body.String(), `name="redirect" value="/web/money"`)
}

// --- POST /web/login ------------------------------------------------------

func doLoginGET(t *testing.T, r *gin.Engine, redirect string) (*http.Cookie, string) {
	target := "/web/login"
	if redirect != "" {
		target += "?redirect=" + url.QueryEscape(redirect)
	}
	req := httptest.NewRequest(http.MethodGet, target, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	res := w.Result()
	return csrfCookie(t, res), csrfFormValue(t, w.Body.String())
}

func TestWebLogin_POST_ValidCredentials_SetsSessionCookieAndRedirects(t *testing.T) {
	setupWebSessionAuth(t)
	r := webSessionRouter()

	cookie, csrf := doLoginGET(t, r, "/web/protected/dashboard")

	form := url.Values{
		"username":   {"alice"},
		"password":   {"s3cret"},
		"csrf_token": {csrf},
		"redirect":   {"/web/protected/dashboard"},
	}
	req := httptest.NewRequest(http.MethodPost, "/web/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	res := w.Result()

	assert.Equal(t, http.StatusFound, res.StatusCode)
	assert.Equal(t, "/web/protected/dashboard", res.Header.Get("Location"))

	session := sessionCookie(res)
	require.NotNil(t, session, "successful login must set the session cookie")
	assert.True(t, session.HttpOnly)
	assert.True(t, session.Secure)
	assert.Equal(t, http.SameSiteLaxMode, session.SameSite)
}

func TestWebLogin_POST_ValidCredentials_ThenProtectedRouteSeesUser(t *testing.T) {
	setupWebSessionAuth(t)
	r := webSessionRouter()

	cookie, csrf := doLoginGET(t, r, "/web/protected/dashboard")
	form := url.Values{
		"username": {"alice"}, "password": {"s3cret"},
		"csrf_token": {csrf}, "redirect": {"/web/protected/dashboard"},
	}
	req := httptest.NewRequest(http.MethodPost, "/web/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	session := sessionCookie(w.Result())
	require.NotNil(t, session)

	req2 := httptest.NewRequest(http.MethodGet, "/web/protected/dashboard", nil)
	req2.AddCookie(session)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusOK, w2.Code)
	assert.Equal(t, "42:alice", w2.Body.String(), "WebMiddleware must inject both user_id and username")
}

func TestWebLogin_POST_WrongPassword_DoesNotSetSessionCookie(t *testing.T) {
	setupWebSessionAuth(t)
	r := webSessionRouter()

	cookie, csrf := doLoginGET(t, r, "")
	form := url.Values{
		"username": {"alice"}, "password": {"wrong"},
		"csrf_token": {csrf},
	}
	req := httptest.NewRequest(http.MethodPost, "/web/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.NotEqual(t, http.StatusFound, w.Code)
	assert.Nil(t, sessionCookie(w.Result()))
}

func TestWebLogin_POST_CSRFMismatch_Rejected(t *testing.T) {
	setupWebSessionAuth(t)
	r := webSessionRouter()

	cookie, _ := doLoginGET(t, r, "")
	form := url.Values{
		"username": {"alice"}, "password": {"s3cret"},
		"csrf_token": {"not-the-real-token"},
	}
	req := httptest.NewRequest(http.MethodPost, "/web/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Nil(t, sessionCookie(w.Result()))
}

func TestWebLogin_POST_MissingCSRFCookie_Rejected(t *testing.T) {
	setupWebSessionAuth(t)
	r := webSessionRouter()

	_, csrf := doLoginGET(t, r, "")
	form := url.Values{
		"username": {"alice"}, "password": {"s3cret"},
		"csrf_token": {csrf}, // no cookie attached to the request at all
	}
	req := httptest.NewRequest(http.MethodPost, "/web/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestWebLogin_POST_OpenRedirectGuard_RejectsAbsoluteURL(t *testing.T) {
	setupWebSessionAuth(t)
	r := webSessionRouter()

	cookie, csrf := doLoginGET(t, r, "")
	form := url.Values{
		"username": {"alice"}, "password": {"s3cret"},
		"csrf_token": {csrf}, "redirect": {"https://evil.example.com/phish"},
	}
	req := httptest.NewRequest(http.MethodPost, "/web/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusFound, w.Code)
	assert.Equal(t, "/web/design-system", w.Header().Get("Location"),
		"an unsafe redirect target must fall back to /web/design-system, not be honored")
}

func TestWebLogin_POST_OpenRedirectGuard_RejectsProtocolRelativeURL(t *testing.T) {
	setupWebSessionAuth(t)
	r := webSessionRouter()

	cookie, csrf := doLoginGET(t, r, "")
	form := url.Values{
		"username": {"alice"}, "password": {"s3cret"},
		"csrf_token": {csrf}, "redirect": {"//evil.example.com/phish"},
	}
	req := httptest.NewRequest(http.MethodPost, "/web/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, "/web/design-system", w.Header().Get("Location"))
}

func TestWebLogin_POST_NoRedirectParam_FallsBackToDesignSystem(t *testing.T) {
	setupWebSessionAuth(t)
	r := webSessionRouter()

	cookie, csrf := doLoginGET(t, r, "")
	form := url.Values{
		"username": {"alice"}, "password": {"s3cret"},
		"csrf_token": {csrf},
	}
	req := httptest.NewRequest(http.MethodPost, "/web/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, "/web/design-system", w.Header().Get("Location"))
}

// --- WebMiddleware ----------------------------------------------------------

func TestWebMiddleware_NoCookie_RedirectsToLoginWithRedirectParam(t *testing.T) {
	setupWebSessionAuth(t)
	r := webSessionRouter()

	req := httptest.NewRequest(http.MethodGet, "/web/protected/dashboard", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusFound, w.Code)
	loc, err := url.Parse(w.Header().Get("Location"))
	require.NoError(t, err)
	assert.Equal(t, "/web/login", loc.Path)
	assert.Equal(t, "/web/protected/dashboard", loc.Query().Get("redirect"))
}

func TestWebMiddleware_TamperedCookie_RedirectsToLogin(t *testing.T) {
	setupWebSessionAuth(t)
	r := webSessionRouter()

	req := httptest.NewRequest(http.MethodGet, "/web/protected/dashboard", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: "not-a-real-jwt"})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusFound, w.Code)
	assert.Contains(t, w.Header().Get("Location"), "/web/login")
}

// --- GET /web/logout ---------------------------------------------------------

func TestWebLogout_ClearsSessionCookieAndRedirectsToLogin(t *testing.T) {
	setupWebSessionAuth(t)
	r := webSessionRouter()

	req := httptest.NewRequest(http.MethodGet, "/web/logout", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	res := w.Result()

	assert.Equal(t, http.StatusFound, res.StatusCode)
	assert.Equal(t, "/web/login", res.Header.Get("Location"))

	cleared := sessionCookie(res)
	require.NotNil(t, cleared, "logout must send a Set-Cookie that clears the session")
	assert.True(t, cleared.MaxAge < 0, "logout must clear the session cookie via MaxAge < 0")
}

func TestWebLogout_IsPlainGETLink_NoCSRFRequired(t *testing.T) {
	// Logout must work as a plain <a href> click (GET, no csrf_token cookie or
	// form field at all) since docs/functions/auth-spec.md scopes CSRF
	// protection to the login form only.
	setupWebSessionAuth(t)
	r := webSessionRouter()

	req := httptest.NewRequest(http.MethodGet, "/web/logout", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusFound, w.Code)
}
