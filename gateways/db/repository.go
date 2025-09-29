package db

import (
	"context"
	"embed"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Masterminds/squirrel"
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
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
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

func (r *repository) AddConsumptionLog(ctx context.Context, log *domain.ConsumptionLog) error {
	query := `
		INSERT INTO consumption_log (user_id, consumed_at, food_id, food_name, amount_g, meal_type, note, nutrients)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	_, err := r.db.Exec(ctx, query,
		log.UserID,
		log.ConsumedAt,
		log.FoodID,
		log.FoodName,
		log.AmountG,
		log.MealType,
		log.Note,
		log.Nutrients,
	)

	return err
}

func (r *repository) SearchFood(ctx context.Context, filter domain.FoodFilter) ([]*domain.Food, error) {
	// Create PostgreSQL-compatible query builder
	psql := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)

	// Build the base SELECT query
	query := psql.Select(
		"id", "name", "description", "barcode", "food_type", "is_archived",
		"serving_size_g", "serving_name", "nutrients", "food_composition",
		"created_at", "updated_at",
	).From("food")

	// Add WHERE conditions based on filter
	if len(filter.IDs) > 0 {
		query = query.Where(squirrel.Eq{"id": filter.IDs})
	}

	if filter.Name != nil && *filter.Name != "" {
		query = query.Where("LOWER(name) LIKE LOWER('%' || ? || '%')", *filter.Name)
	}

	if filter.Barcode != nil && *filter.Barcode != "" {
		query = query.Where(squirrel.Eq{"barcode": *filter.Barcode})
	}

	// Add ORDER BY for consistent results
	query = query.OrderBy("name ASC")

	// Generate SQL and args
	sql, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build SQL query: %w", err)
	}

	// Execute the query
	rows, err := r.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Scan results
	var foods []*domain.Food
	for rows.Next() {
		food := &domain.Food{}
		err := rows.Scan(
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
		foods = append(foods, food)
	}

	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return foods, nil
}

func (r *repository) GetConsumptionLog(ctx context.Context, userID int64, consumedAt time.Time) (*domain.ConsumptionLog, error) {
	query := `
		SELECT user_id, consumed_at, food_id, food_name, amount_g, meal_type, note, nutrients
		FROM consumption_log
		WHERE user_id = $1 AND consumed_at = $2`

	log := &domain.ConsumptionLog{}
	err := r.db.QueryRow(ctx, query, userID, consumedAt).Scan(
		&log.UserID,
		&log.ConsumedAt,
		&log.FoodID,
		&log.FoodName,
		&log.AmountG,
		&log.MealType,
		&log.Note,
		&log.Nutrients,
	)

	if err != nil {
		return nil, err
	}

	return log, nil
}

func (r *repository) GetConsumptionLogsByUser(ctx context.Context, userID int64) ([]*domain.ConsumptionLog, error) {
	query := `
		SELECT user_id, consumed_at, food_id, food_name, amount_g, meal_type, note, nutrients
		FROM consumption_log
		WHERE user_id = $1
		ORDER BY consumed_at DESC`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []*domain.ConsumptionLog
	for rows.Next() {
		log := &domain.ConsumptionLog{}
		err := rows.Scan(
			&log.UserID,
			&log.ConsumedAt,
			&log.FoodID,
			&log.FoodName,
			&log.AmountG,
			&log.MealType,
			&log.Note,
			&log.Nutrients,
		)
		if err != nil {
			return nil, err
		}
		logs = append(logs, log)
	}

	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return logs, nil
}

func (r *repository) DeleteConsumptionLog(ctx context.Context, userID int64, consumedAt time.Time) error {
	query := `DELETE FROM consumption_log WHERE user_id = $1 AND consumed_at = $2`

	_, err := r.db.Exec(ctx, query, userID, consumedAt)
	return err
}

func (r *repository) GetLastConsumptionTime(ctx context.Context, userID int64) (*time.Time, error) {
	query := `
		SELECT consumed_at
		FROM consumption_log
		WHERE user_id = $1
		ORDER BY consumed_at DESC
		LIMIT 1`

	var lastTime time.Time
	err := r.db.QueryRow(ctx, query, userID).Scan(&lastTime)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, nil
		}
		return nil, err
	}

	return &lastTime, nil
}

func (r *repository) GetNutritionStats(ctx context.Context, filter domain.NutritionStatsFilter) ([]domain.NutritionStats, error) {
	switch filter.Aggregation {
	case domain.AggregationTypeTotal:
		return r.getNutritionStatsTotal(ctx, filter)
	case domain.AggregationTypeByDay:
		return r.getNutritionStatsByDay(ctx, filter)
	default:
		return nil, fmt.Errorf("unknown aggregation type: %s", filter.Aggregation)
	}
}

func (r *repository) getNutritionStatsTotal(ctx context.Context, filter domain.NutritionStatsFilter) ([]domain.NutritionStats, error) {
	query := `
		SELECT
			$2::timestamptz as period_start,
			$3::timestamptz as period_end,
			COALESCE(SUM((nutrients->>'calories')::numeric), 0) as total_calories,
			COALESCE(SUM((nutrients->>'protein_g')::numeric), 0) as total_protein,
			COALESCE(SUM((nutrients->>'total_fat_g')::numeric), 0) as total_fat,
			COALESCE(SUM((nutrients->>'carbohydrates_g')::numeric), 0) as total_carbs,
			COALESCE(SUM(amount_g), 0) as total_weight
		FROM consumption_log
		WHERE user_id = $1
		  AND consumed_at >= $2
		  AND consumed_at <= $3`

	var stats domain.NutritionStats
	err := r.db.QueryRow(ctx, query, filter.UserID, filter.From, filter.To).Scan(
		&stats.PeriodStart,
		&stats.PeriodEnd,
		&stats.TotalCalories,
		&stats.TotalProtein,
		&stats.TotalFat,
		&stats.TotalCarbs,
		&stats.TotalWeight,
	)
	if err != nil {
		return nil, err
	}

	// Return empty array if no data
	if stats.TotalCalories == 0 && stats.TotalProtein == 0 && stats.TotalFat == 0 &&
		stats.TotalCarbs == 0 && stats.TotalWeight == 0 {
		return []domain.NutritionStats{}, nil
	}

	return []domain.NutritionStats{stats}, nil
}

func (r *repository) getNutritionStatsByDay(ctx context.Context, filter domain.NutritionStatsFilter) ([]domain.NutritionStats, error) {
	if filter.Timezone == nil {
		return nil, fmt.Errorf("timezone is required for by_day aggregation")
	}

	query := `
		SELECT
			date_trunc('day', consumed_at AT TIME ZONE $4)::date as day,
			COALESCE(SUM((nutrients->>'calories')::numeric), 0) as total_calories,
			COALESCE(SUM((nutrients->>'protein_g')::numeric), 0) as total_protein,
			COALESCE(SUM((nutrients->>'total_fat_g')::numeric), 0) as total_fat,
			COALESCE(SUM((nutrients->>'carbohydrates_g')::numeric), 0) as total_carbs,
			COALESCE(SUM(amount_g), 0) as total_weight
		FROM consumption_log
		WHERE user_id = $1
		  AND consumed_at >= $2
		  AND consumed_at <= $3
		GROUP BY day
		ORDER BY day ASC`

	rows, err := r.db.Query(ctx, query, filter.UserID, filter.From, filter.To, filter.Timezone.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []domain.NutritionStats
	for rows.Next() {
		var dayDate time.Time
		var stats domain.NutritionStats

		err := rows.Scan(
			&dayDate,
			&stats.TotalCalories,
			&stats.TotalProtein,
			&stats.TotalFat,
			&stats.TotalCarbs,
			&stats.TotalWeight,
		)
		if err != nil {
			return nil, err
		}

		// Set period start and end for the day in the specified timezone
		stats.PeriodStart = time.Date(dayDate.Year(), dayDate.Month(), dayDate.Day(), 0, 0, 0, 0, filter.Timezone)
		stats.PeriodEnd = time.Date(dayDate.Year(), dayDate.Month(), dayDate.Day(), 23, 59, 59, 0, filter.Timezone)

		results = append(results, stats)
	}

	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return results, nil
}
