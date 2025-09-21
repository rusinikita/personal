package auth

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
)

var BaseAuthURL = "http://localhost:8081"

// var BaseAuthURL = "splendid-chicken-42.telebit.io"

// Claims represents the JWT claims.
type Claims struct {
	Email string `json:"email"`
	Name  string `json:"name"`
	jwt.StandardClaims
}

// OAuthAuthorizationServer represents the response for the `/.well-known/oauth-authorization-server` endpoint.
type OAuthAuthorizationServer struct {
	Issuer                            string   `json:"issuer"`
	AuthorizationEndpoint             string   `json:"authorization_endpoint"`
	TokenEndpoint                     string   `json:"token_endpoint"`
	RegistrationEndpoint              string   `json:"registration_endpoint,omitempty"`
	JWKSURI                           string   `json:"jwks_uri,omitempty"`
	ResponseTypesSupported            []string `json:"response_types_supported"`
	ScopesSupported                   []string `json:"scopes_supported"`
	TokenEndpointAuthMethodsSupported []string `json:"token_endpoint_auth_methods_supported"`
	GrantTypesSupported               []string `json:"grant_types_supported"`
	CodeChallengeMethodsSupported     []string `json:"code_challenge_methods_supported"`
	RevocationEndpoint                string   `json:"revocation_endpoint,omitempty"`
	IntrospectionEndpoint             string   `json:"introspection_endpoint,omitempty"`
	SubjectTypesSupported             []string `json:"subject_types_supported"`
	IDTokenSigningAlgValuesSupported  []string `json:"id_token_signing_alg_values_supported"`
	ResponseModesSupported            []string `json:"response_modes_supported"`
}

// Client represents an OAuth 2.1 client.
type Client struct {
	ID     string `json:"id"`
	Secret string `json:"secret"`
}

// User represents a user.
type User struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// AuthorizationCode represents an authorization code.
type AuthorizationCode struct {
	Code        string    `json:"code"`
	ClientID    string    `json:"client_id"`
	UserID      string    `json:"user_id"`
	RedirectURI string    `json:"redirect_uri"`
	State       string    `json:"state"`
	ExpiresAt   time.Time `json:"expires_at"`
}

var (
	claudeClientID = "bgqalwyS2sUsAfJa"
	// Client storage - starts with hardcoded client and allows dynamic registration.
	clients = map[string]*Client{
		// Claude client
		claudeClientID: {
			ID:     claudeClientID,
			Secret: "qWuZ-JuSv-CtLqgxk88xMQMTMgSitb9k_J-lKRE9ck0%3D",
		},
	}
	users = map[string]*User{
		"my-user": {
			ID:       "my-user",
			Email:    "user@example.com",
			Password: "password",
		},
	}

	// In-memory storage for authorization codes.
	AuthCodes = make(map[string]*AuthorizationCode)

	// JWT secret key.
	jwtSecret = []byte("my-secret-key")
)

// WellKnownHandler handles the `/.well-known/oauth-authorization-server` endpoint.
func WellKnownHandler(c *gin.Context) {
	serverMeta := &OAuthAuthorizationServer{
		Issuer:                            BaseAuthURL,
		AuthorizationEndpoint:             BaseAuthURL + "/oauth/authorize",
		TokenEndpoint:                     BaseAuthURL + "/oauth/token",
		RegistrationEndpoint:              BaseAuthURL + "/oauth/register",
		ResponseTypesSupported:            []string{"code"},
		ScopesSupported:                   []string{"openid", "profile", "email"},
		TokenEndpointAuthMethodsSupported: []string{"client_secret_post"},
		GrantTypesSupported:               []string{"authorization_code"},
		CodeChallengeMethodsSupported:     []string{"S256", "plain"},
		SubjectTypesSupported:             []string{"public"},
		IDTokenSigningAlgValuesSupported:  []string{"HS256"},
		ResponseModesSupported:            []string{"query", "fragment"},
	}

	c.JSON(http.StatusOK, serverMeta)
}

// ClientRegistrationRequest represents a client registration request.
type ClientRegistrationRequest struct {
	RedirectURIs []string `json:"redirect_uris"`
	ClientName   string   `json:"client_name,omitempty"`
	GrantTypes   []string `json:"grant_types,omitempty"`
}

// ClientRegistrationResponse represents a client registration response.
type ClientRegistrationResponse struct {
	ClientID     string   `json:"client_id"`
	ClientSecret string   `json:"client_secret"`
	RedirectURIs []string `json:"redirect_uris"`
	GrantTypes   []string `json:"grant_types"`
}

// RegisterClientHandler handles dynamic client registration.
func RegisterClientHandler(c *gin.Context) {
	var req ClientRegistrationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "error_description": "Invalid JSON"})
		return
	}

	// Generate client ID and secret
	client := clients[claudeClientID]

	// Default grant types if not specified
	grantTypes := req.GrantTypes
	if len(grantTypes) == 0 {
		grantTypes = []string{"authorization_code"}
	}

	response := ClientRegistrationResponse{
		ClientID:     client.ID,
		ClientSecret: client.Secret,
		RedirectURIs: req.RedirectURIs,
		GrantTypes:   grantTypes,
	}

	c.JSON(http.StatusCreated, response)
}

// AuthorizeHandler handles the `/oauth/authorize` endpoint.
func AuthorizeHandler(c *gin.Context) {
	if c.Request.Method == http.MethodGet {
		LoginPageHandler(c)
		return
	}

	clientID := c.PostForm("client_id")
	redirectURI := c.PostForm("redirect_uri")
	responseType := c.PostForm("response_type")
	state := c.PostForm("state")

	_, ok := clients[clientID]
	if !ok {
		c.String(http.StatusBadRequest, "Invalid client ID")
		return
	}

	if responseType != "code" {
		c.String(http.StatusBadRequest, "Invalid response type")
		return
	}

	password := c.PostForm("password")

	// support only one user for now
	user, ok := users["my-user"]
	if !ok || user.Password != password {
		c.String(http.StatusUnauthorized, "Invalid username or password")
		return
	}

	code := generateAuthorizationCode()
	AuthCodes[code] = &AuthorizationCode{
		Code:        code,
		ClientID:    clientID,
		UserID:      user.ID,
		RedirectURI: redirectURI,
		State:       state,
		ExpiresAt:   time.Now().Add(10 * time.Minute),
	}

	redirectURL := redirectURI + "?code=" + code
	if state != "" {
		redirectURL += "&state=" + state
	}

	c.Redirect(http.StatusFound, redirectURL)
}

func generateAuthorizationCode() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

// TokenHandler handles the `/oauth/token` endpoint.
func TokenHandler(c *gin.Context) {
	grantType := c.PostForm("grant_type")
	if grantType != "authorization_code" {
		c.String(http.StatusBadRequest, "Invalid grant type")
		return
	}

	code := c.PostForm("code")
	authCode, ok := AuthCodes[code]
	if !ok || authCode.ExpiresAt.Before(time.Now()) {
		c.String(http.StatusBadRequest, "Invalid or expired authorization code")
		return
	}

	delete(AuthCodes, code)

	clientID := c.PostForm("client_id")
	clientSecret := c.PostForm("client_secret")

	client, ok := clients[clientID]
	if !ok || client.Secret != clientSecret {
		c.String(http.StatusBadRequest, "Invalid client ID or secret")
		return
	}

	user, ok := users[authCode.UserID]
	if !ok {
		c.String(http.StatusInternalServerError, "User not found")
		return
	}

	claims := &Claims{
		Email: user.Email,
		Name:  user.ID,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(1 * time.Hour).Unix(),
			Issuer:    BaseAuthURL,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to generate token")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token": tokenString,
		"token_type":   "Bearer",
		"expires_in":   3600,
	})
}

// LoginPageHandler handles the login page.
func LoginPageHandler(c *gin.Context) {
	c.Writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, `
		<!DOCTYPE html>
		<html>
		<head>
			<title>Login</title>
		</head>
		<body>
			<h1>Login</h1>
			<form action="/oauth/authorize" method="post">
				<input type="hidden" name="client_id" value="%s"/>
				<input type="hidden" name="redirect_uri" value="%s"/>
				<input type="hidden" name="response_type" value="%s"/>
				<input type="hidden" name="state" value="%s"/>
				<label for="password">Password:</label><br>
				<input type="password" id="password" name="password"><br><br>
				<input type="submit" value="Submit">
			</form>
		</body>
		</html>
	`, c.Query("client_id"), c.Query("redirect_uri"), c.Query("response_type"), c.Query("state"))
}

// Middleware creates a new OAuth middleware.
func Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Header("WWW-Authenticate", `Bearer realm="mcp", authorization_uri="`+BaseAuthURL+`/oauth/authorize"`)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error":             "invalid_request",
				"error_description": "Authorization header is missing",
			})
			return
		}

		if !strings.HasPrefix(authHeader, "Bearer ") {
			c.Header("WWW-Authenticate", `Bearer realm="mcp", authorization_uri="`+BaseAuthURL+`/oauth/authorize", error="invalid_request", error_description="Invalid authorization header format"`)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error":             "invalid_request",
				"error_description": "Invalid authorization header format",
			})
			return
		}

		tokenString := authHeader[len("Bearer "):]
		token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
			return jwtSecret, nil
		})

		if err != nil || !token.Valid {
			c.Header("WWW-Authenticate", `Bearer realm="mcp", authorization_uri="`+BaseAuthURL+`/oauth/authorize", error="invalid_token", error_description="Invalid or expired token"`)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error":             "invalid_token",
				"error_description": "Invalid or expired token",
			})
			return
		}

		c.Next()
	}
}
