package gateways

import (
	"context"
	"time"

	"personal/domain"
)

type DB interface {
	// Existing methods
	AddFood(ctx context.Context, food *domain.Food) (int64, error)
	GetFood(ctx context.Context, id int64) (*domain.Food, error)

	// New methods for consumption logging
	AddConsumptionLog(ctx context.Context, log *domain.ConsumptionLog) error
	SearchFood(ctx context.Context, filter domain.FoodFilter) ([]*domain.Food, error)

	// Methods for testing verification
	GetConsumptionLog(ctx context.Context, userID int64, consumedAt time.Time) (*domain.ConsumptionLog, error)
	GetConsumptionLogsByUser(ctx context.Context, userID int64) ([]*domain.ConsumptionLog, error)
	DeleteConsumptionLog(ctx context.Context, userID int64, consumedAt time.Time) error
}

type DBMaintainer interface {
	ApplyMigrations(ctx context.Context) error
}
