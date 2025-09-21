package tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"personal/action/auth"
)

func setupRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	return router
}

func TestAuth_WellKnown(t *testing.T) {
	router := setupRouter()
	router.GET("/.well-known/oauth-authorization-server", auth.WellKnownHandler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/.well-known/oauth-authorization-server", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	assert.Equal(t, "http://localhost:8081", response["issuer"])
	assert.Equal(t, "http://localhost:8081/oauth/authorize", response["authorization_endpoint"])
	assert.Equal(t, "http://localhost:8081/oauth/token", response["token_endpoint"])
	assert.Equal(t, "http://localhost:8081/oauth/register", response["registration_endpoint"])
	assert.Equal(t, []interface{}{"code"}, response["response_types_supported"])
	assert.Equal(t, []interface{}{"openid", "profile", "email"}, response["scopes_supported"])
	assert.Equal(t, []interface{}{"client_secret_post"}, response["token_endpoint_auth_methods_supported"])
}

func TestAuth_Unauthorized(t *testing.T) {
	router := setupRouter()
	authRequired := router.Group("/")
	authRequired.Use(auth.Middleware())
	{
		authRequired.GET("/mcp", func(c *gin.Context) {
			c.String(http.StatusOK, "OK")
		})
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/mcp", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Equal(t, "Bearer realm=\"mcp\", authorization_uri=\"http://localhost:8081/oauth/authorize\"", w.Header().Get("WWW-Authenticate"))
	assert.JSONEq(t, `{"error": "invalid_request", "error_description": "Authorization header is missing"}`, w.Body.String())
}

func TestAuth_Authorize_GET(t *testing.T) {
	router := setupRouter()
	router.GET("/oauth/authorize", auth.AuthorizeHandler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/oauth/authorize?client_id=my-client&redirect_uri=http://localhost:8080/callback&response_type=code", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "<h1>Login</h1>")
}

func TestAuth_Authorize_POST_and_Token(t *testing.T) {
	router := setupRouter()
	router.Any("/oauth/authorize", auth.AuthorizeHandler)
	router.POST("/oauth/token", auth.TokenHandler)
	authRequired := router.Group("/")
	authRequired.Use(auth.Middleware())
	{
		authRequired.GET("/mcp", func(c *gin.Context) {
			c.String(http.StatusOK, "OK")
		})
	}

	// 1. Authorize
	form := url.Values{}
	form.Add("client_id", "my-client")
	form.Add("redirect_uri", "http://localhost:8080/callback")
	form.Add("response_type", "code")
	form.Add("state", "test-state-123")
	form.Add("username", "my-user")
	form.Add("password", "password")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/oauth/authorize", strings.NewReader(form.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusFound, w.Code)
	location := w.Header().Get("Location")
	assert.Contains(t, location, "http://localhost:8080/callback?code=")
	assert.Contains(t, location, "state=test-state-123")

	u, _ := url.Parse(location)
	q := u.Query()
	code := q.Get("code")
	state := q.Get("state")
	assert.Equal(t, "test-state-123", state)

	// 2. Token
	form = url.Values{}
	form.Add("grant_type", "authorization_code")
	form.Add("code", code)
	form.Add("redirect_uri", "http://localhost:8080/callback")
	form.Add("client_id", "my-client")
	form.Add("client_secret", "my-client-secret")

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/oauth/token", strings.NewReader(form.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var tokenResponse map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &tokenResponse)

	accessToken := tokenResponse["access_token"].(string)

	// 3. Access protected resource
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/mcp", nil)
	req.Header.Add("Authorization", "Bearer "+accessToken)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
