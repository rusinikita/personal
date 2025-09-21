package db

import (
	"context"
	"embed"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"personal/domain"
	"personal/gateways"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// postgres is pgx.Conn methods abstraction
type postgres interface {
	Ping(ctx context.Context) error
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

type repository struct {
	db postgres
}

var _ gateways.DB = (*repository)(nil)
var _ gateways.DBMaintainer = (*repository)(nil)

func NewRepository(db postgres) (gateways.DB, gateways.DBMaintainer) {
	r := &repository{db: db}
	return r, r
}

func (r *repository) AddFood(ctx context.Context, food *domain.Food) (int64, error) {
	query := `
		INSERT INTO food (name, description, barcode, food_type, is_archived,
		                 serving_size_g, serving_name, nutrients, food_composition,
		                 created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id`

	now := time.Now()
	food.CreatedAt = now
	food.UpdatedAt = now

	var id int64
	err := r.db.QueryRow(ctx, query,
		food.Name,
		food.Description,
		food.Barcode,
		food.FoodType,
		food.IsArchived,
		food.ServingSizeG,
		food.ServingName,
		food.Nutrients,
		food.FoodComposition,
		food.CreatedAt,
		food.UpdatedAt,
	).Scan(&id)

	return id, err
}

func (r *repository) GetFood(ctx context.Context, id int64) (*domain.Food, error) {
	query := `
		SELECT id, name, description, barcode, food_type, is_archived,
		       serving_size_g, serving_name, nutrients, food_composition,
		       created_at, updated_at
		FROM food
		WHERE id = $1`

	food := &domain.Food{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&food.ID,
		&food.Name,
		&food.Description,
		&food.Barcode,
		&food.FoodType,
		&food.IsArchived,
		&food.ServingSizeG,
		&food.ServingName,
		&food.Nutrients,
		&food.FoodComposition,
		&food.CreatedAt,
		&food.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return food, nil
}

func (r *repository) ApplyMigrations(ctx context.Context) error {
	// Read all .sql files from migrations directory
	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("failed to read migrations directory: %w", err)
	}

	// Filter and sort SQL files
	var sqlFiles []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sql") {
			sqlFiles = append(sqlFiles, entry.Name())
		}
	}
	sort.Strings(sqlFiles) // Apply migrations in alphabetical order

	// Apply each migration file
	for _, filename := range sqlFiles {
		migrationPath := filepath.Join("migrations", filename)
		migrationSQL, err := migrationsFS.ReadFile(migrationPath)
		if err != nil {
			return fmt.Errorf("failed to read migration file %s: %w", filename, err)
		}

		_, err = r.db.Exec(ctx, string(migrationSQL))
		if err != nil {
			return fmt.Errorf("failed to apply migration %s: %w", filename, err)
		}
	}

	return nil
}
