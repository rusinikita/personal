package gateways

import (
	"context"

	"personal/domain"
)

type DB interface {
	AddFood(ctx context.Context, food *domain.Food) (int64, error)
	GetFood(ctx context.Context, id int64) (*domain.Food, error)
}

type DBMaintainer interface {
	ApplyMigrations(ctx context.Context) error
}
