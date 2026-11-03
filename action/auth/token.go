package auth

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"personal/domain"
	"personal/gateways"
)

// TokenHandler serves `POST /oauth/token` for public clients: the
// authorization_code and refresh_token grants.
func TokenHandler(c *gin.Context) {
	c.Header("Cache-Control", "no-store")

	if c.PostForm("client_secret") != "" {
		tokenError(c, http.StatusUnauthorized, "invalid_client", "only public clients are supported")
		return
	}

	switch c.PostForm("grant_type") {
	case "authorization_code":
		exchangeAuthorizationCode(c)
	case "refresh_token":
		exchangeRefreshToken(c)
	default:
		tokenError(c, http.StatusBadRequest, "unsupported_grant_type", "grant_type must be authorization_code or refresh_token")
	}
}

func exchangeAuthorizationCode(c *gin.Context) {
	ctx := c.Request.Context()
	db := gateways.DBFromContext(ctx)

	code, err := db.ConsumeOAuthCode(ctx, HashToken(c.PostForm("code")))
	if err != nil {
		tokenError(c, http.StatusInternalServerError, "server_error", "failed to read authorization code")
		return
	}
	if code == nil {
		tokenError(c, http.StatusBadRequest, "invalid_grant", "invalid or expired authorization code")
		return
	}
	if c.PostForm("client_id") != code.ClientID {
		tokenError(c, http.StatusBadRequest, "invalid_grant", "authorization code was issued to another client")
		return
	}
	if c.PostForm("redirect_uri") != code.RedirectURI {
		tokenError(c, http.StatusBadRequest, "invalid_grant", "redirect_uri does not match")
		return
	}
	if !verifyPKCE(c.PostForm("code_verifier"), code.CodeChallenge) {
		tokenError(c, http.StatusBadRequest, "invalid_grant", "invalid code_verifier")
		return
	}
	if resource := c.PostForm("resource"); resource != "" && resource != code.Resource {
		tokenError(c, http.StatusBadRequest, "invalid_target", "resource does not match")
		return
	}

	issueTokens(c, code.ClientID, code.UserID, code.Resource, uuid.New(), "")
}

func exchangeRefreshToken(c *gin.Context) {
	ctx := c.Request.Context()
	db := gateways.DBFromContext(ctx)

	token, err := db.GetOAuthRefreshToken(ctx, HashToken(c.PostForm("refresh_token")))
	if err != nil {
		tokenError(c, http.StatusInternalServerError, "server_error", "failed to read refresh token")
		return
	}
	if token == nil || !token.ExpiresAt.After(time.Now()) || c.PostForm("client_id") != token.ClientID {
		tokenError(c, http.StatusBadRequest, "invalid_grant", "invalid, expired or revoked refresh token")
		return
	}
	if resource := c.PostForm("resource"); resource != "" && resource != token.Resource {
		tokenError(c, http.StatusBadRequest, "invalid_target", "resource does not match")
		return
	}

	if token.UsedAt != nil {
		if time.Since(*token.UsedAt) > refreshReuseWindow {
			if err := db.DeleteOAuthRefreshTokenFamily(ctx, token.FamilyID); err != nil {
				tokenError(c, http.StatusInternalServerError, "server_error", "failed to revoke refresh token")
				return
			}
			tokenError(c, http.StatusBadRequest, "invalid_grant", "invalid, expired or revoked refresh token")
			return
		}
		issueTokens(c, token.ClientID, token.UserID, token.Resource, token.FamilyID, "")
		return
	}

	issueTokens(c, token.ClientID, token.UserID, token.Resource, token.FamilyID, token.TokenHash)
}

// issueTokens signs an access token and stores a new refresh token in the
// family; with rotateHash set, that token is marked used in the same transaction.
func issueTokens(c *gin.Context, clientID string, userID int64, resource string, familyID uuid.UUID, rotateHash string) {
	ctx := c.Request.Context()
	db := gateways.DBFromContext(ctx)

	user := findUserByID(userID)
	if user == nil {
		tokenError(c, http.StatusBadRequest, "invalid_grant", "user no longer exists")
		return
	}

	now := time.Now()
	claims := &Claims{
		UserID:   user.ID,
		UserName: user.UserName,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    BaseAuthURL,
			Subject:   strconv.FormatInt(user.ID, 10),
			Audience:  jwt.ClaimStrings{resource},
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(accessTokenExpiry)),
		},
	}
	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(jwtSecret)
	if err != nil {
		tokenError(c, http.StatusInternalServerError, "server_error", "failed to sign access token")
		return
	}

	refreshToken, err := generateRandomToken()
	if err != nil {
		tokenError(c, http.StatusInternalServerError, "server_error", "failed to generate refresh token")
		return
	}
	next := domain.OAuthRefreshToken{
		TokenHash: HashToken(refreshToken),
		FamilyID:  familyID,
		ClientID:  clientID,
		UserID:    user.ID,
		Resource:  resource,
		ExpiresAt: now.Add(refreshTokenExpiry),
	}
	if rotateHash != "" {
		err = db.RotateOAuthRefreshToken(ctx, rotateHash, next)
	} else {
		err = db.CreateOAuthRefreshToken(ctx, next)
	}
	if err != nil {
		tokenError(c, http.StatusInternalServerError, "server_error", "failed to store refresh token")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token":  accessToken,
		"token_type":    "Bearer",
		"expires_in":    int(accessTokenExpiry.Seconds()),
		"refresh_token": refreshToken,
	})
}

func verifyPKCE(verifier, challenge string) bool {
	if len(verifier) < 43 || len(verifier) > 128 {
		return false
	}
	sum := sha256.Sum256([]byte(verifier))
	expected := base64.RawURLEncoding.EncodeToString(sum[:])
	return subtle.ConstantTimeCompare([]byte(expected), []byte(challenge)) == 1
}

func tokenError(c *gin.Context, status int, code, description string) {
	c.JSON(status, gin.H{"error": code, "error_description": description})
}
