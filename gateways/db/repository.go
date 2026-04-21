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
	"personal/util"
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

func (r *repository) CreateFood(ctx context.Context, food *domain.Food) (int64, error) {
	query := `
		INSERT INTO food (name, user_id, description, barcode, food_type, is_archived,
		                 serving_size_g, serving_name, nutrients, food_composition,
		                 created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id`

	now := time.Now()
	food.CreatedAt = now
	food.UpdatedAt = now

	var id int64
	err := r.db.QueryRow(ctx, query,
		food.Name,
		food.UserID,
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
		SELECT id, name, user_id, description, barcode, food_type, is_archived,
		       serving_size_g, serving_name, nutrients, food_composition,
		       created_at, updated_at
		FROM food
		WHERE id = $1`

	food := &domain.Food{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&food.ID,
		&food.Name,
		&food.UserID,
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

func (r *repository) TruncateUserData(ctx context.Context, userID int64) error {
	_, err := r.db.Exec(ctx, `DELETE FROM consumption_log WHERE user_id = $1`, userID)
	if err != nil {
		return err
	}

	_, err = r.db.Exec(ctx, `DELETE FROM food WHERE user_id = $1`, userID)
	if err != nil {
		return err
	}

	_, err = r.db.Exec(ctx, `DELETE FROM sets WHERE user_id = $1`, userID)
	if err != nil {
		return err
	}

	_, err = r.db.Exec(ctx, `DELETE FROM workouts WHERE user_id = $1`, userID)
	if err != nil {
		return err
	}

	_, err = r.db.Exec(ctx, `DELETE FROM exercises WHERE user_id = $1`, userID)
	if err != nil {
		return err
	}

	_, err = r.db.Exec(ctx, `DELETE FROM activity_progress WHERE user_id = $1`, userID)
	if err != nil {
		return err
	}

	_, err = r.db.Exec(ctx, `DELETE FROM activities WHERE user_id = $1`, userID)
	if err != nil {
		return err
	}

	_, err = r.db.Exec(ctx, `DELETE FROM life_parts WHERE user_id = $1`, userID)
	if err != nil {
		return err
	}

	_, err = r.db.Exec(ctx, `DELETE FROM transactions WHERE user_id = $1`, userID)
	if err != nil {
		return err
	}

	_, err = r.db.Exec(ctx, `DELETE FROM budgets WHERE user_id = $1`, userID)
	if err != nil {
		return err
	}

	return nil
}

// ---------------------------------------------------------------------------
// Money tracking
// ---------------------------------------------------------------------------

func (r *repository) AddTransactions(ctx context.Context, txs []*domain.Transaction) ([]*domain.Transaction, error) {
	now := time.Now().UTC()
	for _, tx := range txs {
		tx.CreatedAt = now
		var id int64
		err := r.db.QueryRow(ctx, `
			INSERT INTO transactions
				(user_id, type, amount_original, currency, amount_eur, account, category,
				 merchant, note, original_description, transacted_at, created_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
			RETURNING id`,
			tx.UserID, tx.Type, tx.AmountOriginal, tx.Currency, tx.AmountEUR,
			tx.Account, tx.Category, tx.Merchant, tx.Note, tx.OriginalDescription,
			tx.TransactedAt, tx.CreatedAt,
		).Scan(&id)
		if err != nil {
			return nil, err
		}
		tx.ID = id
	}
	return txs, nil
}

func (r *repository) EditTransactions(ctx context.Context, userID int64, updates []domain.TransactionUpdate) (int, error) {
	// Verify all IDs belong to userID first — atomicity guarantee.
	ids := make([]int64, len(updates))
	for i, u := range updates {
		ids[i] = u.ID
	}

	psql := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)
	checkSQL, checkArgs, err := psql.
		Select("id").
		From("transactions").
		Where(squirrel.Eq{"id": ids, "user_id": userID}).
		ToSql()
	if err != nil {
		return 0, err
	}

	rows, err := r.db.Query(ctx, checkSQL, checkArgs...)
	if err != nil {
		return 0, err
	}
	found := 0
	for rows.Next() {
		found++
	}
	rows.Close()
	if rows.Err() != nil {
		return 0, rows.Err()
	}
	if found != len(ids) {
		return 0, fmt.Errorf("one or more transaction IDs not found or not owned by user")
	}

	// Apply each update individually.
	for _, u := range updates {
		q := psql.Update("transactions").Where(squirrel.Eq{"id": u.ID, "user_id": userID})
		if u.Type != nil {
			q = q.Set("type", *u.Type)
		}
		if u.AmountOriginal != nil {
			q = q.Set("amount_original", *u.AmountOriginal)
		}
		if u.Currency != nil {
			q = q.Set("currency", *u.Currency)
		}
		if u.AmountEUR != nil {
			q = q.Set("amount_eur", *u.AmountEUR)
		}
		if u.Account != nil {
			q = q.Set("account", *u.Account)
		}
		if u.Category != nil {
			q = q.Set("category", *u.Category)
		}
		if u.Merchant != nil {
			q = q.Set("merchant", *u.Merchant)
		}
		if u.Note != nil {
			q = q.Set("note", *u.Note)
		}
		if u.OriginalDescription != nil {
			q = q.Set("original_description", *u.OriginalDescription)
		}
		if u.TransactedAt != nil {
			q = q.Set("transacted_at", *u.TransactedAt)
		}
		sql, args, err := q.ToSql()
		if err != nil {
			return 0, err
		}
		if _, err = r.db.Exec(ctx, sql, args...); err != nil {
			return 0, err
		}
	}
	return len(updates), nil
}

func (r *repository) DeleteTransaction(ctx context.Context, id int64, userID int64) error {
	tag, err := r.db.Exec(ctx,
		`DELETE FROM transactions WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("transaction not found")
	}
	return nil
}

func (r *repository) SetBudget(ctx context.Context, b *domain.Budget) (int64, error) {
	b.CreatedAt = time.Now().UTC()
	var id int64
	err := r.db.QueryRow(ctx, `
		INSERT INTO budgets (user_id, name, category, amount_eur, starts_at, ends_at, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
		ON CONFLICT (user_id, name) DO UPDATE
			SET category   = EXCLUDED.category,
			    amount_eur = EXCLUDED.amount_eur,
			    starts_at  = EXCLUDED.starts_at,
			    ends_at    = EXCLUDED.ends_at
		RETURNING id`,
		b.UserID, b.Name, b.Category, b.AmountEUR, b.StartsAt, b.EndsAt, b.CreatedAt,
	).Scan(&id)
	return id, err
}

func (r *repository) GetTransactions(ctx context.Context, filter domain.TransactionFilter) ([]*domain.Transaction, int, error) {
	psql := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)

	base := psql.Select(
		"id", "user_id", "type", "amount_original", "currency", "amount_eur",
		"account", "category", "merchant", "note", "original_description",
		"transacted_at", "created_at",
	).From("transactions").Where(squirrel.Eq{"user_id": filter.UserID})

	if filter.From != nil {
		base = base.Where(squirrel.GtOrEq{"transacted_at": *filter.From})
	}
	if filter.To != nil {
		base = base.Where(squirrel.LtOrEq{"transacted_at": *filter.To})
	}
	if filter.Account != nil {
		base = base.Where(squirrel.Eq{"account": *filter.Account})
	}
	if filter.Category != nil {
		base = base.Where("category LIKE ?", *filter.Category+"%")
	}
	if filter.Type != nil {
		base = base.Where(squirrel.Eq{"type": *filter.Type})
	}
	if filter.Merchant != nil {
		base = base.Where(squirrel.Eq{"merchant": *filter.Merchant})
	}

	// Count query
	countQ := base.RemoveColumns().Column("COUNT(*)")
	countSQL, countArgs, err := countQ.ToSql()
	if err != nil {
		return nil, 0, err
	}
	var total int
	if err = r.db.QueryRow(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Data query
	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	dataQ := base.OrderBy("transacted_at DESC").Limit(uint64(limit)).Offset(uint64(filter.Offset))
	dataSQL, dataArgs, err := dataQ.ToSql()
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.db.Query(ctx, dataSQL, dataArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var result []*domain.Transaction
	for rows.Next() {
		tx := &domain.Transaction{}
		if err = rows.Scan(
			&tx.ID, &tx.UserID, &tx.Type, &tx.AmountOriginal, &tx.Currency, &tx.AmountEUR,
			&tx.Account, &tx.Category, &tx.Merchant, &tx.Note, &tx.OriginalDescription,
			&tx.TransactedAt, &tx.CreatedAt,
		); err != nil {
			return nil, 0, err
		}
		result = append(result, tx)
	}
	if rows.Err() != nil {
		return nil, 0, rows.Err()
	}
	return result, total, nil
}

func (r *repository) GetSpendingByCategory(ctx context.Context, userID int64, from, to time.Time, depth int) ([]domain.SpendingByCategory, error) {
	if depth < 1 {
		depth = 1
	}
	// Build category prefix expression: array_to_string(ARRAY[...split_parts...], '/')
	parts := make([]string, depth)
	for i := range parts {
		parts[i] = fmt.Sprintf("split_part(category, '/', %d)", i+1)
	}
	catExpr := fmt.Sprintf("array_to_string(ARRAY[%s], '/', '')", join(parts, ", "))
	// Trim trailing slashes that appear when there are fewer depth levels than requested
	catExpr = fmt.Sprintf("TRIM(TRAILING '/' FROM %s)", catExpr)

	sql := fmt.Sprintf(`
		SELECT %s AS cat, SUM(amount_eur) AS total_eur, COUNT(*) AS cnt
		FROM transactions
		WHERE user_id = $1
		  AND type = 'expense'
		  AND transacted_at >= $2
		  AND transacted_at <= $3
		GROUP BY cat
		ORDER BY total_eur DESC`, catExpr)

	rows, err := r.db.Query(ctx, sql, userID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.SpendingByCategory
	for rows.Next() {
		var s domain.SpendingByCategory
		if err = rows.Scan(&s.Category, &s.TotalEUR, &s.Count); err != nil {
			return nil, err
		}
		result = append(result, s)
	}
	return result, rows.Err()
}

func (r *repository) GetTopMerchants(ctx context.Context, userID int64, from, to time.Time, limit int) ([]domain.MerchantSummary, error) {
	if limit <= 0 {
		limit = 10
	}
	rows, err := r.db.Query(ctx, `
		SELECT merchant, SUM(amount_eur) AS total_eur, COUNT(*) AS cnt
		FROM transactions
		WHERE user_id = $1
		  AND type = 'expense'
		  AND transacted_at >= $2
		  AND transacted_at <= $3
		GROUP BY merchant
		ORDER BY total_eur DESC
		LIMIT $4`, userID, from, to, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.MerchantSummary
	for rows.Next() {
		var m domain.MerchantSummary
		if err = rows.Scan(&m.Merchant, &m.TotalEUR, &m.Count); err != nil {
			return nil, err
		}
		result = append(result, m)
	}
	return result, rows.Err()
}

func (r *repository) GetSpendingForPeriod(ctx context.Context, userID int64, from, to time.Time) ([]domain.SpendingByCategory, error) {
	return r.GetSpendingByCategory(ctx, userID, from, to, 1)
}

func (r *repository) GetBudgetProgress(ctx context.Context, userID int64, at time.Time) ([]domain.BudgetProgress, error) {
	rows, err := r.db.Query(ctx, `
		SELECT b.id, b.user_id, b.name, b.category, b.amount_eur, b.starts_at, b.ends_at, b.created_at,
		       COALESCE(SUM(t.amount_eur), 0) AS spent_eur
		FROM budgets b
		LEFT JOIN transactions t
		       ON t.user_id = b.user_id
		      AND t.type = 'expense'
		      AND t.category LIKE b.category || '%'
		      AND t.transacted_at >= b.starts_at
		      AND t.transacted_at <= b.ends_at
		WHERE b.user_id = $1
		  AND b.starts_at <= $2
		  AND b.ends_at >= $2
		GROUP BY b.id, b.user_id, b.name, b.category, b.amount_eur, b.starts_at, b.ends_at, b.created_at
		ORDER BY b.starts_at`, userID, at)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.BudgetProgress
	for rows.Next() {
		var bp domain.BudgetProgress
		if err = rows.Scan(
			&bp.ID, &bp.UserID, &bp.Name, &bp.Category, &bp.AmountEUR,
			&bp.StartsAt, &bp.EndsAt, &bp.CreatedAt, &bp.SpentEUR,
		); err != nil {
			return nil, err
		}
		bp.RemainingEUR = bp.AmountEUR - bp.SpentEUR
		result = append(result, bp)
	}
	return result, rows.Err()
}

func (r *repository) GetBalance(ctx context.Context, userID int64, from, to time.Time) (domain.BalanceResult, error) {
	var income, expense float64
	err := r.db.QueryRow(ctx, `
		SELECT
			COALESCE(SUM(CASE WHEN type = 'income'  THEN amount_eur ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN type = 'expense' THEN amount_eur ELSE 0 END), 0)
		FROM transactions
		WHERE user_id = $1
		  AND type IN ('income', 'expense')
		  AND transacted_at >= $2
		  AND transacted_at <= $3`, userID, from, to,
	).Scan(&income, &expense)
	if err != nil {
		return domain.BalanceResult{}, err
	}
	return domain.BalanceResult{
		From:       from,
		To:         to,
		IncomeEUR:  income,
		ExpenseEUR: expense,
		BalanceEUR: income - expense,
	}, nil
}

// join is a local helper because strings.Join is not in scope here.
func join(parts []string, sep string) string {
	result := ""
	for i, p := range parts {
		if i > 0 {
			result += sep
		}
		result += p
	}
	return result
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
		"id", "name", "user_id", "description", "barcode", "food_type", "is_archived",
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
			&food.UserID,
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
	// Build query using squirrel
	psql := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)

	// Base SELECT with aggregations
	selectBuilder := psql.Select(
		"COALESCE(SUM((nutrients->>'calories')::numeric), 0) as total_calories",
		"COALESCE(SUM((nutrients->>'protein_g')::numeric), 0) as total_protein",
		"COALESCE(SUM((nutrients->>'total_fat_g')::numeric), 0) as total_fat",
		"COALESCE(SUM((nutrients->>'carbohydrates_g')::numeric), 0) as total_carbs",
		"COALESCE(SUM(amount_g), 0) as total_weight",
	).From("consumption_log").
		Where(squirrel.Eq{"user_id": filter.UserID}).
		Where(squirrel.GtOrEq{"consumed_at": filter.From}).
		Where(squirrel.LtOrEq{"consumed_at": filter.To})

	// Add aggregation-specific columns
	switch filter.Aggregation {
	case domain.AggregationTypeTotal:
		// For total: return filter time range as period_start
		selectBuilder = selectBuilder.
			Column(squirrel.Expr("min(consumed_at) as period_start")).
			Column(squirrel.Expr("max(consumed_at) as period_end"))

	case domain.AggregationTypeByDay:
		// For by_day: return start of each day as period_start, then group and sort
		selectBuilder = selectBuilder.
			Column(squirrel.Expr("date_trunc('day', consumed_at) as period_start")).
			Column(squirrel.Expr("(date_trunc('day', consumed_at) + (INTERVAL '1 day' - INTERVAL '1 second')) as period_end")).
			GroupBy("period_start", "period_end").
			OrderBy("period_start ASC")

	default:
		return nil, fmt.Errorf("unknown aggregation type: %s", filter.Aggregation)
	}

	// Generate SQL
	query, args, err := selectBuilder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build SQL query: %w", err)
	}

	// Execute query
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute SQL query: %w", err)
	}
	defer rows.Close()

	// Parse results
	var results []domain.NutritionStats
	for rows.Next() {
		var stats domain.NutritionStats

		err := rows.Scan(
			&stats.TotalCalories,
			&stats.TotalProtein,
			&stats.TotalFat,
			&stats.TotalCarbs,
			&stats.TotalWeight,
			&stats.PeriodStart,
			&stats.PeriodEnd,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		// Skip empty results (all zeros)
		if stats.TotalCalories == 0 && stats.TotalProtein == 0 && stats.TotalFat == 0 &&
			stats.TotalCarbs == 0 && stats.TotalWeight == 0 {
			continue
		}

		results = append(results, stats)
	}

	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return results, nil
}

func (r *repository) GetTopProducts(ctx context.Context, userID int64, from time.Time, to time.Time, limit int) ([]domain.FoodStats, error) {
	query := `
		SELECT cl.food_id,
		       f.name as food_name,
		       COALESCE(f.serving_name, '') as serving_name,
		       COUNT(*) as log_count
		FROM consumption_log cl
		JOIN food f ON cl.food_id = f.id
		WHERE cl.user_id = $1
		  AND cl.consumed_at >= $2
		  AND cl.consumed_at <= $3
		  AND cl.food_id IS NOT NULL
		GROUP BY cl.food_id, f.name, f.serving_name
		ORDER BY log_count DESC, cl.food_id ASC
		LIMIT $4`

	rows, err := r.db.Query(ctx, query, userID, from, to, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	var results []domain.FoodStats
	for rows.Next() {
		var stats domain.FoodStats
		err := rows.Scan(
			&stats.FoodID,
			&stats.FoodName,
			&stats.ServingName,
			&stats.LogCount,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		results = append(results, stats)
	}

	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return results, nil
}

func (r *repository) CreateExercise(ctx context.Context, exercise *domain.Exercise) (int64, error) {
	query := `
		INSERT INTO exercises (user_id, name, equipment_type, created_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id`

	now := time.Now()
	exercise.CreatedAt = now

	var id int64
	err := r.db.QueryRow(ctx, query,
		exercise.UserID,
		exercise.Name,
		exercise.EquipmentType,
		exercise.CreatedAt,
	).Scan(&id)

	return id, err
}

func (r *repository) ListWithLastUsed(ctx context.Context, userID int64) ([]domain.Exercise, error) {
	query := `
		SELECT e.id, e.user_id, e.name, e.equipment_type, e.created_at,
		       MAX(s.created_at) as last_used_at
		FROM exercises e
		LEFT JOIN sets s ON e.id = s.exercise_id
		WHERE e.user_id = $1
		GROUP BY e.id, e.user_id, e.name, e.equipment_type, e.created_at
		ORDER BY e.created_at DESC`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query exercises: %w", err)
	}
	defer rows.Close()

	var exercises []domain.Exercise
	for rows.Next() {
		var ex domain.Exercise
		err := rows.Scan(
			&ex.ID,
			&ex.UserID,
			&ex.Name,
			&ex.EquipmentType,
			&ex.CreatedAt,
			&ex.LastUsedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan exercise: %w", err)
		}
		exercises = append(exercises, ex)
	}

	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return exercises, nil
}

func (r *repository) ListExercises(ctx context.Context, userID int64, limit int64) ([]domain.Exercise, error) {
	query := `
		SELECT e.id, e.user_id, e.name, e.equipment_type, e.created_at,
		       MAX(s.created_at) as last_used_at
		FROM exercises e
		LEFT JOIN sets s ON e.id = s.exercise_id AND s.user_id = $1
		WHERE e.user_id = $1
		GROUP BY e.id, e.user_id, e.name, e.equipment_type, e.created_at
		ORDER BY last_used_at DESC NULLS LAST, e.name ASC
		LIMIT $2`

	rows, err := r.db.Query(ctx, query, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query exercises: %w", err)
	}
	defer rows.Close()

	var exercises []domain.Exercise
	for rows.Next() {
		var ex domain.Exercise
		err := rows.Scan(
			&ex.ID,
			&ex.UserID,
			&ex.Name,
			&ex.EquipmentType,
			&ex.CreatedAt,
			&ex.LastUsedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan exercise: %w", err)
		}
		exercises = append(exercises, ex)
	}

	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return exercises, nil
}

func (r *repository) GetExercise(ctx context.Context, exerciseID int64, userID int64) (*domain.Exercise, error) {
	query := `
		SELECT e.id, e.user_id, e.name, e.equipment_type, e.created_at,
		       MAX(s.created_at) as last_used_at
		FROM exercises e
		LEFT JOIN sets s ON e.id = s.exercise_id AND s.user_id = $2
		WHERE e.id = $1 AND e.user_id = $2
		GROUP BY e.id, e.user_id, e.name, e.equipment_type, e.created_at`

	var ex domain.Exercise
	err := r.db.QueryRow(ctx, query, exerciseID, userID).Scan(
		&ex.ID, &ex.UserID, &ex.Name, &ex.EquipmentType, &ex.CreatedAt, &ex.LastUsedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get exercise: %w", err)
	}
	return &ex, nil
}

func (r *repository) UpdateExercise(ctx context.Context, exercise *domain.Exercise) error {
	query := `UPDATE exercises SET name = $1, equipment_type = $2 WHERE id = $3 AND user_id = $4`
	tag, err := r.db.Exec(ctx, query, exercise.Name, exercise.EquipmentType, exercise.ID, exercise.UserID)
	if err != nil {
		return fmt.Errorf("failed to update exercise: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("exercise not found")
	}
	return nil
}

func (r *repository) MoveSetsBetweenExercises(ctx context.Context, sourceID, targetID, userID int64) (int64, error) {
	tag, err := r.db.Exec(ctx,
		`UPDATE sets SET exercise_id = $2 WHERE exercise_id = $1 AND user_id = $3`,
		sourceID, targetID, userID,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to move sets: %w", err)
	}
	return tag.RowsAffected(), nil
}

func (r *repository) DeleteExercise(ctx context.Context, exerciseID int64, userID int64) error {
	tag, err := r.db.Exec(ctx,
		`DELETE FROM exercises WHERE id = $1 AND user_id = $2`,
		exerciseID, userID,
	)
	if err != nil {
		return fmt.Errorf("failed to delete exercise: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("exercise not found")
	}
	return nil
}

func (r *repository) SearchExercises(ctx context.Context, userID int64, query string) ([]domain.Exercise, error) {
	sql := `
		SELECT e.id, e.user_id, e.name, e.equipment_type, e.created_at,
		       MAX(s.created_at) as last_used_at
		FROM exercises e
		LEFT JOIN sets s ON e.id = s.exercise_id AND s.user_id = $1
		WHERE e.user_id = $1 AND e.name ILIKE $2
		GROUP BY e.id, e.user_id, e.name, e.equipment_type, e.created_at
		ORDER BY last_used_at DESC NULLS LAST, e.name ASC`

	rows, err := r.db.Query(ctx, sql, userID, "%"+query+"%")
	if err != nil {
		return nil, fmt.Errorf("failed to search exercises: %w", err)
	}
	defer rows.Close()

	var exercises []domain.Exercise
	for rows.Next() {
		var ex domain.Exercise
		if err := rows.Scan(&ex.ID, &ex.UserID, &ex.Name, &ex.EquipmentType, &ex.CreatedAt, &ex.LastUsedAt); err != nil {
			return nil, fmt.Errorf("failed to scan exercise: %w", err)
		}
		exercises = append(exercises, ex)
	}

	return exercises, rows.Err()
}

func (r *repository) CreateWorkout(ctx context.Context, workout *domain.Workout) (int64, error) {
	query := `
		INSERT INTO workouts (user_id, started_at, completed_at)
		VALUES ($1, $2, $3)
		RETURNING id`

	var id int64
	err := r.db.QueryRow(ctx, query,
		workout.UserID,
		workout.StartedAt,
		workout.CompletedAt,
	).Scan(&id)

	return id, err
}

func (r *repository) CloseWorkout(ctx context.Context, workoutID int64, completedAt time.Time) error {
	query := `
		UPDATE workouts
		SET completed_at = $1
		WHERE id = $2`

	_, err := r.db.Exec(ctx, query, completedAt, workoutID)
	return err
}

func (r *repository) ListWorkouts(ctx context.Context, userID int64) ([]domain.Workout, error) {
	query := `
		SELECT id, user_id, started_at, completed_at
		FROM workouts
		WHERE user_id = $1
		ORDER BY started_at DESC`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query workouts: %w", err)
	}
	defer rows.Close()

	var workouts []domain.Workout
	for rows.Next() {
		var w domain.Workout
		err := rows.Scan(
			&w.ID,
			&w.UserID,
			&w.StartedAt,
			&w.CompletedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan workout: %w", err)
		}
		workouts = append(workouts, w)
	}

	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return workouts, nil
}

func (r *repository) GetWorkoutByDate(ctx context.Context, userID int64, date time.Time) (*domain.Workout, error) {
	query := `
		SELECT id, user_id, started_at, completed_at
		FROM workouts
		WHERE user_id = $1
		  AND started_at >= $2
		  AND started_at < $3
		LIMIT 1`

	dayStart := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	dayEnd := dayStart.AddDate(0, 0, 1)

	var w domain.Workout
	err := r.db.QueryRow(ctx, query, userID, dayStart, dayEnd).Scan(
		&w.ID, &w.UserID, &w.StartedAt, &w.CompletedAt,
	)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, nil
		}
		return nil, err
	}

	return &w, nil
}

func (r *repository) CreateSet(ctx context.Context, set *domain.Set) (int64, error) {
	query := `
		INSERT INTO sets (user_id, workout_id, exercise_id, reps, duration_seconds, weight_kg, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id`

	var id int64
	err := r.db.QueryRow(ctx, query,
		set.UserID,
		set.WorkoutID,
		set.ExerciseID,
		util.NullIfZero(set.Reps),
		util.NullIfZero(set.DurationSeconds),
		util.NullIfZero(set.WeightKg),
		set.CreatedAt,
	).Scan(&id)

	return id, err
}

func (r *repository) GetLastSet(ctx context.Context, userID int64) (*domain.WorkoutSet, error) {
	query := `
		SELECT
			w.id, w.user_id, w.started_at, w.completed_at,
			s.id, s.user_id, s.workout_id, s.exercise_id,
			COALESCE(s.reps, 0), COALESCE(s.duration_seconds, 0), COALESCE(s.weight_kg, 0),
			s.created_at
		FROM sets s
		JOIN workouts w ON s.workout_id = w.id
		WHERE s.user_id = $1
		ORDER BY s.created_at DESC
		LIMIT 1`

	var ws domain.WorkoutSet
	err := r.db.QueryRow(ctx, query, userID).Scan(
		&ws.Workout.ID,
		&ws.Workout.UserID,
		&ws.Workout.StartedAt,
		&ws.Workout.CompletedAt,
		&ws.Set.ID,
		&ws.Set.UserID,
		&ws.Set.WorkoutID,
		&ws.Set.ExerciseID,
		&ws.Set.Reps,
		&ws.Set.DurationSeconds,
		&ws.Set.WeightKg,
		&ws.Set.CreatedAt,
	)

	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, nil
		}
		return nil, err
	}

	return &ws, nil
}

func (r *repository) GetSetByID(ctx context.Context, setID int64, userID int64) (*domain.SetWithExercise, error) {
	query := `
		SELECT s.id, s.user_id, s.workout_id, s.exercise_id,
		       COALESCE(s.reps, 0), COALESCE(s.duration_seconds, 0), COALESCE(s.weight_kg, 0),
		       s.created_at, e.name
		FROM sets s
		JOIN exercises e ON s.exercise_id = e.id
		WHERE s.id = $1 AND s.user_id = $2`

	var s domain.SetWithExercise
	err := r.db.QueryRow(ctx, query, setID, userID).Scan(
		&s.ID,
		&s.UserID,
		&s.WorkoutID,
		&s.ExerciseID,
		&s.Reps,
		&s.DurationSeconds,
		&s.WeightKg,
		&s.CreatedAt,
		&s.ExerciseName,
	)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, fmt.Errorf("set not found")
		}
		return nil, err
	}

	return &s, nil
}

func (r *repository) DeleteSet(ctx context.Context, setID int64, userID int64) error {
	result, err := r.db.Exec(ctx, `DELETE FROM sets WHERE id = $1 AND user_id = $2`, setID, userID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("set not found")
	}
	return nil
}

func (r *repository) ListSets(ctx context.Context, userID int64, from time.Time, to time.Time) ([]domain.Set, error) {
	query := `
		SELECT id, user_id, workout_id, exercise_id,
		       COALESCE(reps, 0), COALESCE(duration_seconds, 0), COALESCE(weight_kg, 0),
		       created_at
		FROM sets
		WHERE user_id = $1
		  AND created_at >= $2
		  AND created_at <= $3
		ORDER BY created_at DESC`

	rows, err := r.db.Query(ctx, query, userID, from, to)
	if err != nil {
		return nil, fmt.Errorf("failed to query sets: %w", err)
	}
	defer rows.Close()

	var sets []domain.Set
	for rows.Next() {
		var s domain.Set
		err := rows.Scan(
			&s.ID,
			&s.UserID,
			&s.WorkoutID,
			&s.ExerciseID,
			&s.Reps,
			&s.DurationSeconds,
			&s.WeightKg,
			&s.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan set: %w", err)
		}
		sets = append(sets, s)
	}

	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return sets, nil
}

func (r *repository) GetExercisesByIDs(ctx context.Context, userID int64, exerciseIDs []int64) ([]domain.Exercise, error) {
	if len(exerciseIDs) == 0 {
		return []domain.Exercise{}, nil
	}

	// Build query using squirrel for proper IN clause
	psql := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)
	selectBuilder := psql.Select(
		"id", "user_id", "name", "equipment_type", "created_at",
	).From("exercises").
		Where(squirrel.Eq{"user_id": userID}).
		Where(squirrel.Eq{"id": exerciseIDs})

	query, args, err := selectBuilder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build query: %w", err)
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query exercises: %w", err)
	}
	defer rows.Close()

	var exercises []domain.Exercise
	for rows.Next() {
		var ex domain.Exercise
		err := rows.Scan(
			&ex.ID,
			&ex.UserID,
			&ex.Name,
			&ex.EquipmentType,
			&ex.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan exercise: %w", err)
		}
		exercises = append(exercises, ex)
	}

	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return exercises, nil
}

func (r *repository) GetWorkoutsByIDs(ctx context.Context, userID int64, workoutIDs []int64) ([]domain.Workout, error) {
	if len(workoutIDs) == 0 {
		return []domain.Workout{}, nil
	}

	// Build query using squirrel for proper IN clause
	psql := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)
	selectBuilder := psql.Select(
		"id", "user_id", "started_at", "completed_at",
	).From("workouts").
		Where(squirrel.Eq{"user_id": userID}).
		Where(squirrel.Eq{"id": workoutIDs})

	query, args, err := selectBuilder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build query: %w", err)
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query workouts: %w", err)
	}
	defer rows.Close()

	var workouts []domain.Workout
	for rows.Next() {
		var w domain.Workout
		err := rows.Scan(
			&w.ID,
			&w.UserID,
			&w.StartedAt,
			&w.CompletedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan workout: %w", err)
		}
		workouts = append(workouts, w)
	}

	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return workouts, nil
}

func (r *repository) GetExerciseHistory(ctx context.Context, userID int64, exerciseID int64, limit int, offset int) ([]domain.Workout, error) {
	query := `
		SELECT DISTINCT w.id, w.user_id, w.started_at, w.completed_at
		FROM workouts w
		JOIN sets s ON s.workout_id = w.id
		WHERE s.user_id = $1 AND s.exercise_id = $2
		ORDER BY w.started_at DESC
		LIMIT $3 OFFSET $4`

	rows, err := r.db.Query(ctx, query, userID, exerciseID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query exercise history: %w", err)
	}
	defer rows.Close()

	var workouts []domain.Workout
	for rows.Next() {
		var w domain.Workout
		if err := rows.Scan(&w.ID, &w.UserID, &w.StartedAt, &w.CompletedAt); err != nil {
			return nil, fmt.Errorf("failed to scan workout: %w", err)
		}
		workouts = append(workouts, w)
	}

	return workouts, rows.Err()
}

func (r *repository) GetPersonalRecords(ctx context.Context, userID int64, exerciseID int64) (*domain.PersonalRecords, error) {
	records := &domain.PersonalRecords{}

	scanSetRecord := func(query string) (*domain.SetRecord, error) {
		rec := &domain.SetRecord{}
		err := r.db.QueryRow(ctx, query, exerciseID, userID).Scan(&rec.WeightKg, &rec.Reps, &rec.CreatedAt)
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return rec, err
	}

	var err error
	records.MaxWeight, err = scanSetRecord(
		`SELECT weight_kg, reps, created_at FROM sets
		 WHERE exercise_id=$1 AND user_id=$2 AND reps>0 AND weight_kg>0
		 ORDER BY weight_kg DESC, reps DESC LIMIT 1`)
	if err != nil {
		return nil, fmt.Errorf("failed to query max_weight: %w", err)
	}

	records.MaxReps, err = scanSetRecord(
		`SELECT weight_kg, reps, created_at FROM sets
		 WHERE exercise_id=$1 AND user_id=$2 AND reps>0
		 ORDER BY reps DESC, weight_kg DESC LIMIT 1`)
	if err != nil {
		return nil, fmt.Errorf("failed to query max_reps: %w", err)
	}

	vr := &domain.VolumeRecord{}
	err = r.db.QueryRow(ctx,
		`SELECT SUM(s.weight_kg*s.reps) AS vol, w.started_at
		 FROM sets s JOIN workouts w ON s.workout_id=w.id
		 WHERE s.exercise_id=$1 AND s.user_id=$2 AND s.reps>0 AND s.weight_kg>0
		 GROUP BY s.workout_id, w.started_at
		 ORDER BY vol DESC LIMIT 1`,
		exerciseID, userID,
	).Scan(&vr.Volume, &vr.StartedAt)
	if err != nil && err != pgx.ErrNoRows {
		return nil, fmt.Errorf("failed to query max_volume: %w", err)
	}
	if err == nil {
		records.MaxVolume = vr
	}

	return records, nil
}

func (r *repository) ListSetsByExerciseAndWorkouts(ctx context.Context, userID int64, exerciseID int64, workoutIDs []int64) ([]domain.Set, error) {
	if len(workoutIDs) == 0 {
		return []domain.Set{}, nil
	}

	psql := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)
	q, args, err := psql.Select(
		"id", "user_id", "workout_id", "exercise_id",
		"COALESCE(reps, 0)", "COALESCE(duration_seconds, 0)", "COALESCE(weight_kg, 0)", "created_at",
	).From("sets").
		Where(squirrel.Eq{"user_id": userID, "exercise_id": exerciseID, "workout_id": workoutIDs}).
		OrderBy("created_at ASC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build query: %w", err)
	}

	rows, err := r.db.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query sets: %w", err)
	}
	defer rows.Close()

	var sets []domain.Set
	for rows.Next() {
		var s domain.Set
		if err := rows.Scan(&s.ID, &s.UserID, &s.WorkoutID, &s.ExerciseID, &s.Reps, &s.DurationSeconds, &s.WeightKg, &s.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan set: %w", err)
		}
		sets = append(sets, s)
	}

	return sets, rows.Err()
}

func (r *repository) CreateActivity(ctx context.Context, activity *domain.Activity) (int64, error) {
	query := `
		INSERT INTO activities (user_id, life_part_ids, name, description, progress_type, frequency_days, started_at, ended_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id`

	now := time.Now()
	activity.CreatedAt = now

	var id int64
	err := r.db.QueryRow(ctx, query,
		activity.UserID,
		activity.LifePartIDs,
		activity.Name,
		activity.Description,
		activity.ProgressType,
		activity.FrequencyDays,
		activity.StartedAt,
		activity.EndedAt,
		activity.CreatedAt,
	).Scan(&id)

	return id, err
}

func (r *repository) ListActivities(ctx context.Context, filter domain.ActivityFilter) ([]domain.Activity, error) {
	psql := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)

	query := psql.Select(
		"id", "user_id", "life_part_ids", "name", "description",
		"progress_type", "frequency_days", "started_at", "ended_at", "created_at", "last_point_at",
	).From("activities").
		Where(squirrel.Eq{"user_id": filter.UserID})

	if filter.ActiveOnly {
		query = query.Where(squirrel.Eq{"ended_at": nil})
	} else {
		query = query.Where("ended_at IS NOT NULL")
	}

	if len(filter.LifePartIDs) > 0 {
		query = query.Where("life_part_ids && ?", filter.LifePartIDs)
	}

	// Фильтр: не показывать активности, которые еще не начались
	query = query.Where("started_at <= NOW()")

	query = query.OrderBy("COALESCE((last_point_at::date + frequency_days) - CURRENT_DATE, 999999) ASC")

	sql, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build query: %w", err)
	}

	rows, err := r.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query activities: %w", err)
	}
	defer rows.Close()

	var activities []domain.Activity
	for rows.Next() {
		var a domain.Activity
		err := rows.Scan(
			&a.ID,
			&a.UserID,
			&a.LifePartIDs,
			&a.Name,
			&a.Description,
			&a.ProgressType,
			&a.FrequencyDays,
			&a.StartedAt,
			&a.EndedAt,
			&a.CreatedAt,
			&a.LastPointAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan activity: %w", err)
		}
		activities = append(activities, a)
	}

	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return activities, nil
}

func (r *repository) GetActivity(ctx context.Context, activityID int64, userID int64) (*domain.Activity, error) {
	query := `
		SELECT id, user_id, life_part_ids, name, description,
		       progress_type, frequency_days, started_at, ended_at, created_at
		FROM activities
		WHERE id = $1 AND user_id = $2`

	var a domain.Activity
	err := r.db.QueryRow(ctx, query, activityID, userID).Scan(
		&a.ID,
		&a.UserID,
		&a.LifePartIDs,
		&a.Name,
		&a.Description,
		&a.ProgressType,
		&a.FrequencyDays,
		&a.StartedAt,
		&a.EndedAt,
		&a.CreatedAt,
	)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, nil
		}
		return nil, err
	}

	return &a, nil
}

func (r *repository) UpdateActivity(ctx context.Context, activity *domain.Activity) error {
	query := `
		UPDATE activities
		SET name = $1, description = $2, frequency_days = $3, life_part_ids = $4
		WHERE id = $5 AND user_id = $6`

	result, err := r.db.Exec(ctx, query,
		activity.Name,
		activity.Description,
		activity.FrequencyDays,
		activity.LifePartIDs,
		activity.ID,
		activity.UserID,
	)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("activity not found")
	}

	return nil
}

func (r *repository) FinishActivity(ctx context.Context, activityID int64, userID int64, endedAt time.Time) error {
	query := `
		UPDATE activities
		SET ended_at = $1
		WHERE id = $2 AND user_id = $3 AND ended_at IS NULL`

	result, err := r.db.Exec(ctx, query, endedAt, activityID, userID)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("activity not found or already finished")
	}

	return nil
}

func (r *repository) CreateProgress(ctx context.Context, progress *domain.ActivityPoint) (int64, error) {
	query := `
		INSERT INTO activity_progress (activity_id, user_id, value, hours_left, note, progress_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id`

	now := time.Now()
	progress.CreatedAt = now

	var id int64
	err := r.db.QueryRow(ctx, query,
		progress.ActivityID,
		progress.UserID,
		progress.Value,
		progress.HoursLeft,
		progress.Note,
		progress.ProgressAt,
		progress.CreatedAt,
	).Scan(&id)

	if err != nil {
		return 0, err
	}

	// Update activity's last_point_at
	updateQuery := `UPDATE activities SET last_point_at = $1 WHERE id = $2`
	_, err = r.db.Exec(ctx, updateQuery, progress.ProgressAt, progress.ActivityID)
	if err != nil {
		return id, fmt.Errorf("failed to update last_point_at: %w", err)
	}

	return id, nil
}

func (r *repository) ListProgress(ctx context.Context, filter domain.ProgressFilter) ([]domain.ActivityPoint, error) {
	psql := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)

	query := psql.Select(
		"id", "activity_id", "user_id", "value", "hours_left", "note", "progress_at", "created_at",
	).From("activity_progress").
		Where(squirrel.Eq{"user_id": filter.UserID}).
		OrderBy("progress_at DESC")

	if filter.ActivityID != 0 {
		query = query.Where(squirrel.Eq{"activity_id": filter.ActivityID})
	}

	if !filter.From.IsZero() {
		query = query.Where(squirrel.GtOrEq{"progress_at": filter.From})
	}

	if !filter.To.IsZero() {
		query = query.Where(squirrel.LtOrEq{"progress_at": filter.To})
	}

	if filter.Limit > 0 {
		query = query.Limit(uint64(filter.Limit))
	}

	sql, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build query: %w", err)
	}

	rows, err := r.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query progress: %w", err)
	}
	defer rows.Close()

	var points []domain.ActivityPoint
	for rows.Next() {
		var p domain.ActivityPoint
		err := rows.Scan(
			&p.ID,
			&p.ActivityID,
			&p.UserID,
			&p.Value,
			&p.HoursLeft,
			&p.Note,
			&p.ProgressAt,
			&p.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan progress point: %w", err)
		}
		points = append(points, p)
	}

	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return points, nil
}

func (r *repository) SearchProgressNotes(ctx context.Context, filter domain.ProgressNoteSearchFilter) ([]domain.ActivityPointWithActivity, error) {
	psql := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)

	query := psql.Select(
		"ap.id", "ap.activity_id", "ap.user_id", "ap.value", "ap.hours_left", "ap.note", "ap.progress_at", "ap.created_at",
		"a.name AS activity_name",
	).
		From("activity_progress ap").
		Join("activities a ON a.id = ap.activity_id").
		Where(squirrel.Eq{"ap.user_id": filter.UserID}).
		Where(squirrel.ILike{"ap.note": "%" + filter.Query + "%"}).
		OrderBy("ap.progress_at DESC")

	if filter.ActivityID != 0 {
		query = query.Where(squirrel.Eq{"ap.activity_id": filter.ActivityID})
	}

	if !filter.From.IsZero() {
		query = query.Where(squirrel.GtOrEq{"ap.progress_at": filter.From})
	}

	if !filter.To.IsZero() {
		query = query.Where(squirrel.LtOrEq{"ap.progress_at": filter.To})
	}

	if filter.ValueMin != nil {
		query = query.Where(squirrel.GtOrEq{"ap.value": *filter.ValueMin})
	}

	if filter.ValueMax != nil {
		query = query.Where(squirrel.LtOrEq{"ap.value": *filter.ValueMax})
	}

	sql, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build query: %w", err)
	}

	rows, err := r.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query progress notes: %w", err)
	}
	defer rows.Close()

	var results []domain.ActivityPointWithActivity
	for rows.Next() {
		var p domain.ActivityPointWithActivity
		err := rows.Scan(
			&p.ID,
			&p.ActivityID,
			&p.UserID,
			&p.Value,
			&p.HoursLeft,
			&p.Note,
			&p.ProgressAt,
			&p.CreatedAt,
			&p.ActivityName,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan progress note: %w", err)
		}
		results = append(results, p)
	}

	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return results, nil
}

func (r *repository) GetTrendStats(ctx context.Context, activityID int64, userID int64, from time.Time, to time.Time) (domain.TrendStats, error) {
	query := `
		SELECT COUNT(*) as count,
		       COALESCE(AVG(value), 0) as average,
		       COALESCE(PERCENTILE_CONT(0.8) WITHIN GROUP (ORDER BY value), 0) as percentile_80
		FROM activity_progress
		WHERE activity_id = $1 AND user_id = $2 AND progress_at >= $3 AND progress_at <= $4`

	var stats domain.TrendStats
	err := r.db.QueryRow(ctx, query, activityID, userID, from, to).Scan(
		&stats.Count,
		&stats.Average,
		&stats.Percentile80,
	)
	if err != nil {
		return domain.TrendStats{}, err
	}

	return stats, nil
}
