package auth

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"personal/gateways"
)

var BaseAuthURL = "http://localhost:8081"

// InitializeAuth initializes users and JWT secret from environment variables.
// This function must be called before using any OAuth functionality.
func InitializeAuth() {
	BaseAuthURL = os.Getenv("BASE_URL")
	if BaseAuthURL == "" {
		log.Fatal("BASE_URL environment variable not set")
	}

	// Load users from environment variable
	usersEnv := os.Getenv("USERS")
	if usersEnv == "" {
		log.Fatal("USERS environment variable not set")
	}

	var err error
	users, err = parseUsers(usersEnv)
	if err != nil {
		log.Fatalf("Failed to parse USERS: %v", err)
	}

	// Load JWT secret from environment variable
	jwtSecretEnv := os.Getenv("JWT_SECRET")
	if jwtSecretEnv == "" {
		log.Fatal("JWT_SECRET environment variable not set")
	}
	jwtSecret = []byte(jwtSecretEnv)
}

// parseUsers parses the USERS environment variable into a slice of User structs.
// Expected format: "ID:USERNAME:PASSWORD;ID:USERNAME:PASSWORD"
func parseUsers(usersEnv string) ([]User, error) {
	if usersEnv == "" {
		return nil, fmt.Errorf("USERS environment variable is empty")
	}

	userStrings := strings.Split(usersEnv, ";")
	users := make([]User, 0, len(userStrings))

	for _, userString := range userStrings {
		parts := strings.Split(userString, ":")
		if len(parts) != 3 {
			return nil, fmt.Errorf("invalid user format: %s (expected ID:USERNAME:PASSWORD)", userString)
		}

		id, err := strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid user ID in %s: %w", userString, err)
		}

		users = append(users, User{
			ID:       id,
			UserName: parts[1],
			Password: parts[2],
		})
	}

	return users, nil
}

// Claims represents the JWT claims.
type Claims struct {
	UserID int64 `json:"user_id"`
	jwt.RegisteredClaims
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
	ID       int64  `json:"id"`
	UserName string `json:"username"`
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
			Secret: "qWuZ-JuSv-CtLqgxk88xMQMTMgSitb9k_J-lKRE9ck0=",
		},
	}
	users     []User
	jwtSecret []byte

	// In-memory storage for authorization codes.
	AuthCodes = make(map[string]*AuthorizationCode)
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

	username := c.PostForm("username")
	password := c.PostForm("password")

	var userID int64
	for _, user := range users {
		if user.UserName == username && user.Password == password {
			userID = user.ID
		}
	}

	if userID == 0 {
		c.String(http.StatusUnauthorized, "Invalid username or password")
		return
	}

	code := generateAuthorizationCode()
	AuthCodes[code] = &AuthorizationCode{
		Code:        code,
		ClientID:    clientID,
		UserID:      strconv.Itoa(int(userID)),
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

	userID, err := strconv.Atoi(authCode.UserID)
	if err != nil {
		c.String(http.StatusInternalServerError, "User not found: %s", err.Error())
		return
	}

	expiresIn := 14 * 24 * time.Hour

	claims := &Claims{
		UserID: int64(userID),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: &jwt.NumericDate{
				Time: time.Now().Add(expiresIn),
			},
			Issuer: BaseAuthURL,
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
		"expires_in":   expiresIn.Seconds(),
	})
}

// LoginPageHandler handles the login page.
func LoginPageHandler(c *gin.Context) {
	c.Writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, `
<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <meta name="color-scheme" content="light dark">
	<link
	  rel="stylesheet"
	  href="https://cdn.jsdelivr.net/npm/@picocss/pico@2/css/pico.classless.violet.min.css"
	>
	<title>Authorization</title>
  </head>
  <body>
    <main>
      	<h1>Authorization</h1>
		<form action="/oauth/authorize" method="post">
			<fieldset>
				<input type="hidden" name="client_id" value="%s"/>
				<input type="hidden" name="redirect_uri" value="%s"/>
				<input type="hidden" name="response_type" value="%s"/>
				<input type="hidden" name="state" value="%s"/>

				<label for="username">Username</label>
				<input type="text" id="username" name="username">
				<label for="password">Password</label>
				<input type="password" id="password" name="password">
			<fieldset>
			<input type="submit" value="Submit">
		</form>
    </main>
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

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		claims := &Claims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
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

		c.Request = c.Request.WithContext(gateways.WithUserID(c.Request.Context(), claims.UserID))

		c.Next()
	}
}
