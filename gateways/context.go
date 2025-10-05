package gateways

import (
	"context"
)

// contextKey is a private type for context keys to avoid collisions
type contextKey string

const (
	dbContextKey     contextKey = "database"
	userIDContextKey contextKey = "user_id"
)

// WithDB adds a database interface to the context
func WithDB(ctx context.Context, db DB) context.Context {
	return context.WithValue(ctx, dbContextKey, db)
}

// WithUserID adds a user ID to the context
func WithUserID(ctx context.Context, id int64) context.Context {
	return context.WithValue(ctx, userIDContextKey, id)
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
