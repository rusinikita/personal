package gateways

import (
	"context"
)

// contextKey is a private type for context keys to avoid collisions
type contextKey string

const (
	dbContextKey       contextKey = "database"
	userIDContextKey   contextKey = "user_id"
	telegramContextKey contextKey = "telegram"
)

// WithDB adds a database interface to the context
func WithDB(ctx context.Context, db DB) context.Context {
	return context.WithValue(ctx, dbContextKey, db)
}

// WithUserID adds a user ID to the context
func WithUserID(ctx context.Context, id int64) context.Context {
	return context.WithValue(ctx, userIDContextKey, id)
}

// WithTelegram adds a Telegram gateway to the context
func WithTelegram(ctx context.Context, tg Telegram) context.Context {
	return context.WithValue(ctx, telegramContextKey, tg)
}

// DBFromContext extracts the database interface from the context
// Returns nil if no database is found in the context
func DBFromContext(ctx context.Context) DB {
	db, ok := ctx.Value(dbContextKey).(DB)
	if !ok {
		return nil
	}
	return db
}

// UserIDFromContext extracts the user_id interface from the context
// Returns 0 if no user_id is found in the context
func UserIDFromContext(ctx context.Context) int64 {
	userID, ok := ctx.Value(userIDContextKey).(int64)
	if !ok {
		return 0
	}

	return userID
}

// TelegramFromContext extracts the Telegram gateway from the context
// Returns nil if no Telegram gateway is found in the context
func TelegramFromContext(ctx context.Context) Telegram {
	tg, ok := ctx.Value(telegramContextKey).(Telegram)
	if !ok {
		return nil
	}
	return tg
}
