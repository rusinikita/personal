# OAuth 2.1 Authorization

## Requirements

This document describes the implementation of an OAuth 2.1 Authorization Server.
The goal is to protect the MCP endpoint with OAuth 2.1 and have the server itself act as the provider.

- The server will act as an OAuth 2.1 Authorization Server.
- The server will protect the MCP endpoint with OAuth 2.1.
- Clients accessing the MCP endpoint without a valid token will receive an error and must proceed with the authorization flow.
- The server will implement the Authorization Code Flow.
- The server will issue JWT tokens to authenticated users.
- The server will provide a `/.well-known/oauth-authorization-server` endpoint.
- The server will provide a `/oauth/authorize` endpoint to initiate the authorization flow.
- The server will provide a `/oauth/token` endpoint to exchange authorization codes for access tokens.
- The server will use a hardcoded client ID and client secret.
- The server will use a hardcoded username and password for user authentication.

## Implementation

### Domain structure

```go
// Claims represents the JWT claims.
type Claims struct {
	Email string `json:"email"`
	Name  string `json:"name"`
	jwt.StandardClaims
}

// OAuthAuthorizationServer represents the response for the `/.well-known/oauth-authorization-server` endpoint.
type OAuthAuthorizationServer struct {
	Issuer                string   `json:"issuer"`
	AuthorizationEndpoint string   `json:"authorization_endpoint"`
	TokenEndpoint         string   `json:"token_endpoint"`
	JWKSURI               string   `json:"jwks_uri"`
	ResponseTypesSupported []string `json:"response_types_supported"`
	ScopesSupported       []string `json:"scopes_supported"`
	TokenEndpointAuthMethodsSupported []string `json:"token_endpoint_auth_methods_supported"`
}

// Client represents an OAuth 2.1 client.
type Client struct {
	ID           string   `json:"id"`
	Secret       string   `json:"secret"`
	RedirectURIs []string `json:"redirect_uris"`
}

// User represents a user.
type User struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// AuthorizationCode represents an authorization code.
type AuthorizationCode struct {
	Code      string `json:"code"`
	ClientID  string `json:"client_id"`
	UserID    string `json:"user_id"`
	ExpiresAt time.Time `json:"expires_at"`
}
```

### Database

No database will be used. Client and user information will be hardcoded. Authorization codes will be stored in memory.


### External API

There are no external APIs for this feature.

### handler/command/tool/worker

#### OAuth Middleware

- The middleware will be applied to the MCP endpoint.
- It will extract the JWT token from the `Authorization` header.
- It will validate the token's signature and claims.
- If the token is valid, the request will be allowed to proceed.
- If the token is invalid or missing, the middleware will return a `401 Unauthorized` error with a `WWW-Authenticate` header to indicate that the client should initiate the OAuth flow.

#### `/.well-known/oauth-authorization-server` Handler

- This handler will return a JSON object with the authorization server's metadata.
- The metadata will include the issuer, authorization endpoint, token endpoint, and other relevant information.

#### `/oauth/authorize` Handler

- This handler will be responsible for initiating the authorization flow.
- It will receive the client ID, redirect URI, response type, and scope as query parameters.
- It will validate the client ID and redirect URI against the hardcoded values.
- It will authenticate the user by showing a login page where the user can enter the hardcoded username and password.
- After the user is authenticated, it will generate an authorization code, store it in memory, and redirect the user back to the client's redirect URI with the code.

#### `/oauth/token` Handler

- This handler will be responsible for exchanging an authorization code for an access token.
- It will receive the grant type, code, redirect URI, client ID, and client secret as form parameters.
- It will validate the grant type, code, redirect URI, client ID, and client secret against the hardcoded values.
- It will look up the authorization code in the in-memory storage and verify that it is valid and has not expired.
- It will generate a JWT access token and return it to the client in a JSON response.
- It will delete the authorization code from the in-memory storage after it has been used.

## E2E Tests

I will create a new test file `tests/auth_test.go` to test the OAuth flow.

The test will cover the following scenarios:
1. Accessing a protected resource without a token should return a `401 Unauthorized` error.
2. Accessing a protected resource with an invalid token should return a `401 Unauthorized` error.
3. The `/.well-known/oauth-authorization-server` endpoint should return the correct JSON response.
4. The `/oauth/authorize` handler should redirect to the login page if the user is not authenticated.
5. The `/oauth/authorize` handler should generate an authorization code and redirect to the client's redirect URI after the user is authenticated.
6. The `/oauth/token` handler should return an access token when given a valid authorization code.
7. The `/oauth/token` handler should return an error when given an invalid authorization code.
8. Accessing a protected resource with a valid token should grant access.
