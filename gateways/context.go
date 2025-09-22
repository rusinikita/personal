package gateways

import (
	"context"
)

// contextKey is a private type for context keys to avoid collisions
type contextKey string

const (
	dbContextKey contextKey = "database"
)

// WithDB adds a database interface to the context
func WithDB(ctx context.Context, db DB) context.Context {
	return context.WithValue(ctx, dbContextKey, db)
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
