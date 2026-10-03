# Auth (OAuth 2.1 + Web Session) - Complete Specification

## Overview

The `action/auth` package covers two independent login flows that share one user store and one JWT secret:

1. **OAuth 2.1 Authorization Server** — protects the MCP endpoint (`/app/mcp`) for machine clients (Claude Desktop, other MCP clients). The server acts as its own provider: it implements the Authorization Code flow, issues bearer JWT access tokens, and supports dynamic client registration. No third-party identity provider is involved.
2. **Web Session Login** — a shared, cookie-based login for human browser access to the *new* `/web/*` dashboards (Money, Progress browse, Workouts, Navigation home page — each its own backlog item, built on top of this) plus `/money/import`. It replaces the inconsistent per-page auth that existed before: `/money/import` used to run its own HTTP Basic Auth with `IMPORT_USERNAME`/`IMPORT_PASSWORD` (now retired).

**Out of scope for the web session flow:** `action/progress/dashboard_web.go` (served at `/web/progress`) is purpose-built for black-and-white screenshot/e-ink display and stays exactly as it is today, unauthenticated included. It is not routed through `WebMiddleware`.

`GET /web/design-system` (the webui component demo page, `action/webui`) **is** routed through `WebMiddleware`, same as every other `/web/*` dashboard — it's a `/web/*` route rendered through the shared shell, and the shell now shows the logged-in username (see `webui-spec.md`), so it needs a real session like any other page.

## Go Code Structure

### Domain Models

```go
package auth

import (
    "time"
    "github.com/golang-jwt/jwt/v5"
)

// Claims are the custom JWT claims issued after a successful token exchange
// or web session login. UserName is only consumed by the web session flow
// (rendered into the shell's user menu, see webui-spec.md) but is set on
// every token, OAuth included, since both flows share this one struct.
type Claims struct {
    UserID   int64  `json:"user_id"`
    UserName string `json:"username"`
    jwt.RegisteredClaims
}

// Client represents a registered OAuth client (hardcoded, no persistence)
type Client struct {
    ID     string `json:"id"`
    Secret string `json:"secret"`
}

// User represents a login credential, loaded from the USERS env var
type User struct {
    ID       int64  `json:"id"`
    UserName string `json:"username"`
    Password string `json:"password"`
}

// AuthorizationCode is a short-lived code exchanged for a JWT, stored in-memory
type AuthorizationCode struct {
    Code        string    `json:"code"`
    ClientID    string    `json:"client_id"`
    UserID      string    `json:"user_id"`
    RedirectURI string    `json:"redirect_uri"`
    State       string    `json:"state"`
    ExpiresAt   time.Time `json:"expires_at"`
}
```

There is no repository/DB layer for auth — users and clients are held in package-level variables (`users`, `clients`), authorization codes in the `AuthCodes` map.

## HTTP Handlers

### WellKnownHandler
`GET /.well-known/oauth-authorization-server` (and the `*path` / `oauth-protected-resource` variants) — returns OAuth server metadata (endpoints, supported grant/response types, PKCE methods).

### RegisterClientHandler
`POST /oauth/register` — dynamic client registration; always returns the single hardcoded Claude client's ID/secret regardless of request body.

### AuthorizeHandler
`GET/POST /oauth/authorize` — GET renders the login page; POST validates client_id and username/password against the `USERS` env var, generates a short-lived authorization code, and redirects to `redirect_uri?code=...`.

### LoginPageHandler
Renders the plain HTML login form used by `AuthorizeHandler` (GET).

### TokenHandler
`POST /oauth/token` — exchanges a valid, unexpired, single-use authorization code (+ matching client_id/secret) for a signed JWT access token.

### Middleware
Applied to the `/app` route group (and skippable via `AUTH_DISABLED`). Validates the `Authorization: Bearer <jwt>` header, returns `401` with a `WWW-Authenticate` challenge header on missing/invalid/expired tokens, otherwise injects `user_id` into the request context via `gateways.WithUserID`.

### WebLoginPageHandler
`GET /web/login` — renders the cookie-login form (username/password + hidden `csrf_token` + hidden `redirect`) through `webui.RenderPage`. Sets the `csrf_token` cookie (short-lived, non-httpOnly, `SameSite=Lax`) if not already present, and embeds the same value in the form.

### WebLoginHandler
`POST /web/login` — rejects the request (400) unless the posted `csrf_token` matches the `csrf_token` cookie. Validates `username`/`password` against the `USERS` env var (same store as OAuth). On success, signs a JWT with the same `Claims` shape, secret, and 14-day expiry as `TokenHandler`, sets it as the `session` cookie (`httpOnly`, `Secure`, `SameSite=Lax`), and redirects to the posted `redirect` value if it's a safe same-site relative path, else to `/web/design-system`. On failure, re-renders the login page with an error message.

### WebLogoutHandler
`GET /web/logout` — clears the `session` cookie and redirects to `/web/login`. A plain `<a href>` in the shell's user dropdown (see `webui-spec.md`) is enough to trigger it — no CSRF check needed, so it doesn't need to be a POST/form.

### WebMiddleware
Applied to `/money/import` (replacing the retired `money.BasicAuthMiddleware`), `GET /web/design-system`, and future `/web/*` dashboard route groups as they're added (Money, Progress browse, Workouts, home — each a separate backlog item). Reads the `session` cookie and validates the JWT the same way `Middleware()` does for the bearer token; on a missing/invalid/expired token it redirects (302) to `/web/login?redirect=<original request path>` instead of returning a JSON 401. On success it injects `user_id` into the request context via `gateways.WithUserID`, and sets the username (from the JWT's `username` claim) on the gin context (`c.Set("user_name", ...)`) for handlers to pass into `webui.PageData.UserName`. Skippable via `AUTH_DISABLED`, same as `Middleware()`.

## Configuration

- **USERS**: `ID:USERNAME:PASSWORD;ID:USERNAME:PASSWORD` — required unless `AUTH_DISABLED` is set; shared by both flows
- **JWT_SECRET**: HS256 signing secret — required unless `AUTH_DISABLED` is set; shared by both flows
- **BASE_URL**: public base URL used to build issuer/endpoint URLs in OAuth metadata and JWT `iss` claim
- **AUTH_DISABLED**: when set (any value), `main.go` skips mounting OAuth routes, `Middleware()`, and `WebMiddleware` entirely
- **Token Expiry**: 14 days (both the OAuth bearer token and the web `session` cookie)
- **Authorization Code Expiry**: 10 minutes, single-use
- **CSRF Token Expiry**: 10 minutes, matches the `csrf_token` cookie lifetime

## E2E Tests

In `tests/auth_test.go`:

- `TestAuth_*`: well known; unauthorized; authorize, GET; authorize, POST and token

In `tests/auth_web_session_test.go`:

- `TestWebLogin_*`: GET, renders form with CSRF cookie and hidden field; GET, preserves redirect param; POST, valid credentials, sets session cookie and redirects; POST, valid credentials, then protected route sees user; POST, wrong password, does not set session cookie; POST, CSRF mismatch, rejected; POST, missing CSRF cookie, rejected; POST, open redirect guard, rejects absolute URL; POST, open redirect guard, rejects protocol relative URL; POST, no redirect param, falls back to design system
- `TestWebMiddleware_*`: no cookie, redirects to login with redirect param; tampered cookie, redirects to login
- `TestWebLogout_*`: clears session cookie and redirects to login; is plain GET link, no CSRF required

## Changelog

- **03-10-26** — migrated to the new spec template: removed Best Practices Applied and the sequence diagrams together with every reference to them; Configuration narrowed (retired variables note dropped); added E2E Tests; added Changelog (Go Code Structure, Configuration, E2E Tests, Changelog)
- **22-08-26** — added cookie-based web session login for `/web/*`: login and logout handlers, `WebMiddleware`, CSRF token (Overview, Go Code Structure, HTTP Handlers, Configuration)
- **19-08-26** — initial version, assembled from `docs/actions/oauth_action.md`
