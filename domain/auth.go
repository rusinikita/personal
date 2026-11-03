package domain

import (
	"time"

	"github.com/google/uuid"
)

// OAuthCode is a single-use authorization code, exchanged for tokens.
type OAuthCode struct {
	CodeHash      string
	ClientID      string
	UserID        int64
	RedirectURI   string
	CodeChallenge string
	Resource      string
	ExpiresAt     time.Time
}

// OAuthRefreshToken is one link of a rotating refresh token family.
type OAuthRefreshToken struct {
	TokenHash string
	FamilyID  uuid.UUID
	ClientID  string
	UserID    int64
	Resource  string
	ExpiresAt time.Time
	UsedAt    *time.Time
}
