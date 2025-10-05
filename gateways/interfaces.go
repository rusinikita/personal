package gateways

import (
	"context"
	"time"

	"personal/domain"
)

type DB interface {
	// Existing methods
	CreateFood(ctx context.Context, food *domain.Food) (int64, error)
	GetFood(ctx context.Context, id int64) (*domain.Food, error)

	// New methods for consumption logging
	AddConsumptionLog(ctx context.Context, log *domain.ConsumptionLog) error
	SearchFood(ctx context.Context, filter domain.FoodFilter) ([]*domain.Food, error)

	// Methods for testing verification
	GetConsumptionLog(ctx context.Context, userID int64, consumedAt time.Time) (*domain.ConsumptionLog, error)
	GetConsumptionLogsByUser(ctx context.Context, userID int64) ([]*domain.ConsumptionLog, error)

	// Nutrition stats methods
	GetLastConsumptionTime(ctx context.Context, userID int64) (*time.Time, error)
	GetNutritionStats(ctx context.Context, filter domain.NutritionStatsFilter) ([]domain.NutritionStats, error)

	// Top products methods
	GetTopProducts(ctx context.Context, userID int64, from time.Time, to time.Time, limit int) ([]domain.FoodStats, error)

	// Exercise methods
	CreateExercise(ctx context.Context, exercise *domain.Exercise) (int64, error)
	ListWithLastUsed(ctx context.Context, userID int64) ([]domain.Exercise, error)

	// Workout methods
	CreateWorkout(ctx context.Context, workout *domain.Workout) (int64, error)
	CloseWorkout(ctx context.Context, workoutID int64, completedAt time.Time) error
	ListWorkouts(ctx context.Context, userID int64) ([]domain.Workout, error)

	// Set methods
	CreateSet(ctx context.Context, set *domain.Set) (int64, error)
	GetLastSet(ctx context.Context, userID int64) (*domain.WorkoutSet, error)
}

type DBMaintainer interface {
	ApplyMigrations(ctx context.Context) error
	TruncateUserData(ctx context.Context, userID int64) error
}
