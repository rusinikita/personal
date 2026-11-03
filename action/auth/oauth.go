package auth

import (
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

const (
	accessTokenExpiry  = time.Hour
	refreshTokenExpiry = 30 * 24 * time.Hour
	refreshReuseWindow = 30 * time.Second
	authCodeExpiry     = 10 * time.Minute

	mcpPath = "/app/mcp"
)

var BaseAuthURL = "http://localhost:8081"

var (
	users     []User
	jwtSecret []byte
)

// InitializeAuth initializes users and JWT secret from environment variables.
// This function must be called before using any OAuth functionality.
func InitializeAuth() {
	BaseAuthURL = os.Getenv("BASE_URL")
	if BaseAuthURL == "" {
		log.Fatal("BASE_URL environment variable not set")
	}

	usersEnv := os.Getenv("USERS")
	if usersEnv == "" {
		log.Fatal("USERS environment variable not set")
	}

	var err error
	users, err = parseUsers(usersEnv)
	if err != nil {
		log.Fatalf("Failed to parse USERS: %v", err)
	}

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

// MCPResourceURL is the protected resource: the audience of every access token.
func MCPResourceURL() string {
	return BaseAuthURL + mcpPath
}

func protectedResourceMetadataURL() string {
	return BaseAuthURL + "/.well-known/oauth-protected-resource" + mcpPath
}

// Claims are the custom JWT claims of both the OAuth access token and the
// web session cookie. UserName is only consumed by the web session flow
// (rendered into the shell's user menu, see webui-spec.md). Access tokens
// additionally set Audience to the MCP URL; session cookies leave it empty.
type Claims struct {
	UserID   int64  `json:"user_id"`
	UserName string `json:"username"`
	jwt.RegisteredClaims
}

// User represents a login credential, loaded from the USERS env var.
type User struct {
	ID       int64  `json:"id"`
	UserName string `json:"username"`
	Password string `json:"password"`
}

// authenticateUser returns the matching User for a username/password pair
// from the USERS-env-loaded store, or nil if none match. Shared by the
// OAuth authorization-code flow and the web session login flow.
func authenticateUser(username, password string) *User {
	for i := range users {
		if users[i].UserName == username && users[i].Password == password {
			return &users[i]
		}
	}
	return nil
}

// findUserByID looks up a user by ID, or nil if none match.
func findUserByID(id int64) *User {
	for i := range users {
		if users[i].ID == id {
			return &users[i]
		}
	}
	return nil
}

// authorizationServerMetadata is the RFC 8414 document.
type authorizationServerMetadata struct {
	Issuer                                     string   `json:"issuer"`
	AuthorizationEndpoint                      string   `json:"authorization_endpoint"`
	TokenEndpoint                              string   `json:"token_endpoint"`
	ResponseTypesSupported                     []string `json:"response_types_supported"`
	ResponseModesSupported                     []string `json:"response_modes_supported"`
	GrantTypesSupported                        []string `json:"grant_types_supported"`
	TokenEndpointAuthMethodsSupported          []string `json:"token_endpoint_auth_methods_supported"`
	CodeChallengeMethodsSupported              []string `json:"code_challenge_methods_supported"`
	ScopesSupported                            []string `json:"scopes_supported"`
	ClientIDMetadataDocumentSupported          bool     `json:"client_id_metadata_document_supported"`
	AuthorizationResponseISSParameterSupported bool     `json:"authorization_response_iss_parameter_supported"`
}

// WellKnownHandler serves `/.well-known/oauth-authorization-server`.
func WellKnownHandler(c *gin.Context) {
	c.JSON(http.StatusOK, authorizationServerMetadata{
		Issuer:                                     BaseAuthURL,
		AuthorizationEndpoint:                      BaseAuthURL + "/oauth/authorize",
		TokenEndpoint:                              BaseAuthURL + "/oauth/token",
		ResponseTypesSupported:                     []string{"code"},
		ResponseModesSupported:                     []string{"query"},
		GrantTypesSupported:                        []string{"authorization_code", "refresh_token"},
		TokenEndpointAuthMethodsSupported:          []string{"none"},
		CodeChallengeMethodsSupported:              []string{"S256"},
		ScopesSupported:                            []string{"offline_access"},
		ClientIDMetadataDocumentSupported:          true,
		AuthorizationResponseISSParameterSupported: true,
	})
}

// ProtectedResourceHandler serves the RFC 9728 document for the MCP endpoint.
func ProtectedResourceHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"resource":                 MCPResourceURL(),
		"authorization_servers":    []string{BaseAuthURL},
		"bearer_methods_supported": []string{"header"},
		"scopes_supported":         []string{"offline_access"},
	})
}

// Middleware protects the MCP endpoint with a bearer access token.
func Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		challenge := `Bearer resource_metadata="` + protectedResourceMetadataURL() + `"`

		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Header("WWW-Authenticate", challenge)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error":             "invalid_request",
				"error_description": "Authorization header is missing",
			})
			return
		}

		tokenString, ok := strings.CutPrefix(authHeader, "Bearer ")
		if !ok {
			c.Header("WWW-Authenticate", challenge+`, error="invalid_request"`)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error":             "invalid_request",
				"error_description": "Invalid authorization header format",
			})
			return
		}

		claims := &Claims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return jwtSecret, nil
		},
			jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
			jwt.WithIssuer(BaseAuthURL),
			jwt.WithAudience(MCPResourceURL()),
			jwt.WithExpirationRequired(),
		)
		if err != nil || !token.Valid {
			c.Header("WWW-Authenticate", challenge+`, error="invalid_token"`)
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
