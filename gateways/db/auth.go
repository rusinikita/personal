package db

// OAuth authorization server state (see docs/functions/auth-spec.md)

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"personal/domain"
)

func (r *repository) CreateOAuthCode(ctx context.Context, code domain.OAuthCode) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO oauth_code (code_hash, client_id, user_id, redirect_uri, code_challenge, resource, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		code.CodeHash, code.ClientID, code.UserID, code.RedirectURI, code.CodeChallenge, code.Resource, code.ExpiresAt.UTC())
	return err
}

func (r *repository) ConsumeOAuthCode(ctx context.Context, codeHash string) (*domain.OAuthCode, error) {
	var code domain.OAuthCode
	err := r.db.QueryRow(ctx, `
		DELETE FROM oauth_code
		WHERE code_hash = $1 AND expires_at > $2
		RETURNING code_hash, client_id, user_id, redirect_uri, code_challenge, resource, expires_at`,
		codeHash, time.Now().UTC()).
		Scan(&code.CodeHash, &code.ClientID, &code.UserID, &code.RedirectURI, &code.CodeChallenge, &code.Resource, &code.ExpiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &code, nil
}

func (r *repository) CreateOAuthRefreshToken(ctx context.Context, token domain.OAuthRefreshToken) error {
	return insertOAuthRefreshToken(ctx, r.db, token)
}

type execer interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

func insertOAuthRefreshToken(ctx context.Context, db execer, token domain.OAuthRefreshToken) error {
	_, err := db.Exec(ctx, `
		INSERT INTO oauth_refresh_token (token_hash, family_id, client_id, user_id, resource, expires_at, used_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		token.TokenHash, token.FamilyID, token.ClientID, token.UserID, token.Resource, token.ExpiresAt.UTC(), token.UsedAt)
	return err
}

func (r *repository) GetOAuthRefreshToken(ctx context.Context, tokenHash string) (*domain.OAuthRefreshToken, error) {
	var token domain.OAuthRefreshToken
	err := r.db.QueryRow(ctx, `
		SELECT token_hash, family_id, client_id, user_id, resource, expires_at, used_at
		FROM oauth_refresh_token
		WHERE token_hash = $1`, tokenHash).
		Scan(&token.TokenHash, &token.FamilyID, &token.ClientID, &token.UserID, &token.Resource, &token.ExpiresAt, &token.UsedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &token, nil
}

func (r *repository) RotateOAuthRefreshToken(ctx context.Context, oldHash string, next domain.OAuthRefreshToken) error {
	return r.inTx(ctx, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `
			UPDATE oauth_refresh_token SET used_at = $2
			WHERE token_hash = $1 AND used_at IS NULL`, oldHash, time.Now().UTC()); err != nil {
			return err
		}
		return insertOAuthRefreshToken(ctx, tx, next)
	})
}

func (r *repository) DeleteOAuthRefreshTokenFamily(ctx context.Context, familyID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM oauth_refresh_token WHERE family_id = $1`, familyID)
	return err
}

func (r *repository) DeleteExpiredOAuthState(ctx context.Context) error {
	now := time.Now().UTC()
	if _, err := r.db.Exec(ctx, `DELETE FROM oauth_code WHERE expires_at <= $1`, now); err != nil {
		return err
	}
	_, err := r.db.Exec(ctx, `DELETE FROM oauth_refresh_token WHERE expires_at <= $1`, now)
	return err
}
