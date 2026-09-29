package db

import (
	"context"
	"slices"
	"sort"
	"strings"
	"time"

	"personal/domain"
	"personal/gateways"
)

// MockRepository wraps repository with a nil db connection, overriding only
// the methods the /web/* handlers actually call with fixture data. Every
// other gateways.DB method falls through to the embedded repository and
// panics with a nil pointer dereference on first use — a deliberate signal
// that a newly wired web handler needs a mock added here too. See
// cmd/webui-preview.
type MockRepository struct {
	repository
}

var _ gateways.DB = (*MockRepository)(nil)

// NewMockRepository returns a gateways.DB backed by fixture data instead of
// Postgres.
func NewMockRepository() gateways.DB {
	return &MockRepository{repository: repository{db: nil}}
}

func (m *MockRepository) ListActivities(_ context.Context, filter domain.ActivityFilter) ([]domain.Activity, error) {
	now := time.Now()
	yesterday := now.Add(-20 * time.Hour)
	activities := []domain.Activity{
		{ID: 1, UserID: filter.UserID, LifePartIDs: []int64{1}, Name: "Ship personal tracker", Description: "**Refactor** web transport to the shared `action/webui` design system, replace the ad-hoc screenshot dashboards one subdomain at a time, and retire the old per-page CSS/JS once every view has a browse/detail equivalent built on the new components.", ProgressType: domain.ProgressTypeProjectProgress, Status: domain.ActivityStatusActive, FrequencyDays: 1, StartedAt: now.AddDate(0, 0, -14), LastPointAt: &yesterday},
		{ID: 2, UserID: filter.UserID, LifePartIDs: []int64{2}, Name: "Gym", Description: "Push/pull/legs", ProgressType: domain.ProgressTypeHabitProgress, Status: domain.ActivityStatusActive, FrequencyDays: 2, StartedAt: now.AddDate(0, 0, -60), LastPointAt: &yesterday},
		{ID: 3, UserID: filter.UserID, Name: "Mood check-in", ProgressType: domain.ProgressTypeMood, Status: domain.ActivityStatusActive, FrequencyDays: 1, StartedAt: now.AddDate(0, 0, -90), LastPointAt: &now},
		{ID: 4, UserID: filter.UserID, LifePartIDs: []int64{2, 3}, Name: "Call mom", ProgressType: domain.ProgressTypePromiseState, Status: domain.ActivityStatusActive, FrequencyDays: 7, StartedAt: now.AddDate(0, 0, -30), LastPointAt: &yesterday},
	}
	if filter.ProgressType == "" {
		return activities, nil
	}
	filtered := make([]domain.Activity, 0, len(activities))
	for _, a := range activities {
		if a.ProgressType == filter.ProgressType {
			filtered = append(filtered, a)
		}
	}
	return filtered, nil
}

func (m *MockRepository) ListProgress(_ context.Context, filter domain.ProgressFilter) ([]domain.ActivityPoint, error) {
	now := time.Now()
	return []domain.ActivityPoint{
		{ID: 1, ActivityID: 1, UserID: filter.UserID, Value: 1, Note: "Wired up the transport/web package", ProgressAt: now.Add(-20 * time.Hour)},
		{ID: 2, ActivityID: 1, UserID: filter.UserID, Value: 2, Note: "Shipped cookie-based session login", ProgressAt: now.Add(-3 * 24 * time.Hour)},
		{ID: 3, ActivityID: 2, UserID: filter.UserID, Value: 1, Note: "Leg day, felt strong", ProgressAt: now.Add(-20 * time.Hour)},
		{ID: 4, ActivityID: 3, UserID: filter.UserID, Value: 2, Note: "Bright and productive morning", ProgressAt: now},
		{ID: 5, ActivityID: 4, UserID: filter.UserID, Value: 1, Note: "Left a voice message", ProgressAt: now.Add(-20 * time.Hour)},
	}, nil
}

func (m *MockRepository) CountActivities(_ context.Context, _ domain.ActivityFilter) (int, error) {
	return 4, nil
}

// ListLifeParts returns fixture life parts referenced by ListActivities'
// fixture activities (1: "Ship personal tracker", 2: "Gym" and "Call mom",
// 3: "Call mom" also) so the browse list's life_part chips have something to
// render in the preview.
func (m *MockRepository) ListLifeParts(_ context.Context, userID int64) ([]domain.LifePart, error) {
	return []domain.LifePart{
		{ID: 1, UserID: userID, Name: "Career", Description: "Work, side projects, professional growth"},
		{ID: 2, UserID: userID, Name: "Health", Description: "Physical fitness, sleep, nutrition"},
		{ID: 3, UserID: userID, Name: "Family", Description: "Parents, siblings, extended family"},
	}, nil
}

func (m *MockRepository) GetActivity(_ context.Context, activityID int64, userID int64) (*domain.Activity, error) {
	activities, _ := m.ListActivities(context.Background(), domain.ActivityFilter{UserID: userID})
	for _, a := range activities {
		if a.ID == activityID {
			return &a, nil
		}
	}
	return nil, nil
}

func (m *MockRepository) CountProgress(_ context.Context, _ domain.ProgressFilter) (int, error) {
	return 5, nil
}

func (m *MockRepository) GetTrendStats(_ context.Context, _ int64, _ int64, _ time.Time, _ time.Time) (domain.TrendStats, error) {
	return domain.TrendStats{Count: 5, Average: 1.2, Percentile80: 2}, nil
}

// ListSteps returns fixture steps for activities 1 ("Ship personal tracker")
// and 2 ("Gym") so the browse list, drill-down, and "log a point" form all
// have something to render in the preview — filtered the same way the real
// repository filters by ActivityID/Statuses.
func (m *MockRepository) ListSteps(_ context.Context, filter domain.StepFilter) ([]domain.Step, error) {
	now := time.Now()
	lastExecutedAt := now.AddDate(0, 0, -4)
	all := []domain.Step{
		{ID: 1, UserID: filter.UserID, ActivityID: 1, Name: "Wire up steps UI", Type: domain.StepTypeOneTime, Status: domain.StepStatusActive, CreatedAt: now.AddDate(0, 0, -2)},
		{ID: 2, UserID: filter.UserID, ActivityID: 1, Name: "Write E2E tests", Type: domain.StepTypeOneTime, Status: domain.StepStatusActive, CreatedAt: now.AddDate(0, 0, -1)},
		{ID: 3, UserID: filter.UserID, ActivityID: 2, Name: "Buy protein powder", Type: domain.StepTypeRepeatable, Status: domain.StepStatusActive, CreatedAt: now.AddDate(0, 0, -3), LastExecutedAt: &lastExecutedAt, ExecutionsLast30Days: 3},
	}

	steps := make([]domain.Step, 0, len(all))
	for _, st := range all {
		if filter.ActivityID != 0 && st.ActivityID != filter.ActivityID {
			continue
		}
		if len(filter.Statuses) > 0 && !stepStatusIn(st.Status, filter.Statuses) {
			continue
		}
		steps = append(steps, st)
	}
	return steps, nil
}

// CreateProgress, CreateStep and UpdateStep are write no-ops (the mock has
// no state to persist into) — added so BrowseCreatePointWebHandler's POST
// (which calls all three: the point itself, then closing/queuing steps)
// doesn't panic on the embedded repository's nil db connection, same
// "deliberate signal" pattern as every other MockRepository override here.
func (m *MockRepository) CreateProgress(_ context.Context, _ *domain.ActivityPoint) (int64, error) {
	return 999, nil
}

func (m *MockRepository) CreateStep(_ context.Context, _ *domain.Step) (int64, error) {
	return 999, nil
}

func (m *MockRepository) UpdateStep(_ context.Context, _ *domain.Step) error {
	return nil
}

func stepStatusIn(status domain.StepStatus, statuses []domain.StepStatus) bool {
	for _, s := range statuses {
		if s == status {
			return true
		}
	}
	return false
}

func (m *MockRepository) GetStep(ctx context.Context, stepID int64, userID int64) (*domain.Step, error) {
	steps, _ := m.ListSteps(ctx, domain.StepFilter{UserID: userID})
	for _, st := range steps {
		if st.ID == stepID {
			return &st, nil
		}
	}
	return nil, nil
}

// mockExercises is the fixed set of exercises the workouts dashboard mock
// pages render, shared by ListPersonalRecords and GetExercise.
func mockExercises(userID int64) []domain.Exercise {
	return []domain.Exercise{
		{ID: 1, UserID: userID, Name: "Bench Press", EquipmentType: domain.EquipmentBarbell},
		{ID: 2, UserID: userID, Name: "Squat", EquipmentType: domain.EquipmentBarbell},
		{ID: 3, UserID: userID, Name: "Pull-up", EquipmentType: domain.EquipmentBodyweight},
	}
}

// mockSetsForExercise returns fixture sets for one exercise, oldest first,
// spaced a week apart, with weight/reps progressing over time — enough to
// draw a meaningful trend line on the drill-down charts.
func mockSetsForExercise(userID int64, exerciseID int64) []domain.Set {
	now := time.Now()
	week := func(n int) time.Time { return now.AddDate(0, 0, -7*n) }

	switch exerciseID {
	case 1: // Bench Press
		return []domain.Set{
			{ID: 101, UserID: userID, WorkoutID: 1001, ExerciseID: 1, Reps: 10, WeightKg: 60, CreatedAt: week(4)},
			{ID: 102, UserID: userID, WorkoutID: 1002, ExerciseID: 1, Reps: 8, WeightKg: 65, CreatedAt: week(3)},
			{ID: 103, UserID: userID, WorkoutID: 1003, ExerciseID: 1, Reps: 6, WeightKg: 70, CreatedAt: week(2)},
			{ID: 104, UserID: userID, WorkoutID: 1004, ExerciseID: 1, Reps: 5, WeightKg: 82.5, CreatedAt: week(1)},
		}
	case 2: // Squat
		return []domain.Set{
			{ID: 201, UserID: userID, WorkoutID: 2001, ExerciseID: 2, Reps: 10, WeightKg: 80, CreatedAt: week(4)},
			{ID: 202, UserID: userID, WorkoutID: 2002, ExerciseID: 2, Reps: 8, WeightKg: 85, CreatedAt: week(3)},
			{ID: 203, UserID: userID, WorkoutID: 2003, ExerciseID: 2, Reps: 6, WeightKg: 90, CreatedAt: week(2)},
			{ID: 204, UserID: userID, WorkoutID: 2004, ExerciseID: 2, Reps: 5, WeightKg: 100, CreatedAt: week(1)},
		}
	case 3: // Pull-up (bodyweight — weight_kg stays 0)
		return []domain.Set{
			{ID: 301, UserID: userID, WorkoutID: 3001, ExerciseID: 3, Reps: 8, CreatedAt: week(3)},
			{ID: 302, UserID: userID, WorkoutID: 3002, ExerciseID: 3, Reps: 10, CreatedAt: week(2)},
			{ID: 303, UserID: userID, WorkoutID: 3003, ExerciseID: 3, Reps: 12, CreatedAt: week(1)},
		}
	default:
		return nil
	}
}

func (m *MockRepository) GetExercise(_ context.Context, exerciseID int64, userID int64) (*domain.Exercise, error) {
	for _, ex := range mockExercises(userID) {
		if ex.ID == exerciseID {
			return &ex, nil
		}
	}
	return nil, nil
}

func (m *MockRepository) GetPersonalRecords(_ context.Context, userID int64, exerciseID int64) (*domain.PersonalRecords, error) {
	sets := mockSetsForExercise(userID, exerciseID)
	records := &domain.PersonalRecords{}
	for _, s := range sets {
		if s.Reps <= 0 {
			continue
		}
		if s.WeightKg > 0 {
			if records.MaxWeight == nil || s.WeightKg > records.MaxWeight.WeightKg {
				records.MaxWeight = &domain.SetRecord{WeightKg: s.WeightKg, Reps: s.Reps, CreatedAt: s.CreatedAt}
			}
			volume := s.WeightKg * float64(s.Reps)
			if records.MaxVolume == nil || volume > records.MaxVolume.Volume {
				records.MaxVolume = &domain.VolumeRecord{Volume: volume, StartedAt: s.CreatedAt}
			}
		}
		if records.MaxReps == nil || s.Reps > records.MaxReps.Reps {
			records.MaxReps = &domain.SetRecord{WeightKg: s.WeightKg, Reps: s.Reps, CreatedAt: s.CreatedAt}
		}
	}
	return records, nil
}

func (m *MockRepository) ListPersonalRecords(ctx context.Context, userID int64) ([]domain.ExercisePersonalRecords, error) {
	exercises := mockExercises(userID)
	results := make([]domain.ExercisePersonalRecords, 0, len(exercises))
	for _, ex := range exercises {
		records, _ := m.GetPersonalRecords(ctx, userID, ex.ID)
		results = append(results, domain.ExercisePersonalRecords{
			Exercise: ex,
			SetCount: int64(len(mockSetsForExercise(userID, ex.ID))),
			Records:  *records,
		})
	}
	return results, nil
}

func (m *MockRepository) GetExerciseHistory(_ context.Context, userID int64, exerciseID int64, limit int, offset int) ([]domain.Workout, error) {
	sets := mockSetsForExercise(userID, exerciseID)
	workouts := make([]domain.Workout, len(sets))
	for i, s := range sets {
		// Newest-first, matching the real GetExerciseHistory's ORDER BY started_at DESC.
		workouts[len(sets)-1-i] = domain.Workout{ID: s.WorkoutID, UserID: userID, StartedAt: s.CreatedAt}
	}
	if offset >= len(workouts) {
		return []domain.Workout{}, nil
	}
	end := len(workouts)
	if limit > 0 && offset+limit < end {
		end = offset + limit
	}
	return workouts[offset:end], nil
}

func (m *MockRepository) ListSetsByExerciseAndWorkouts(_ context.Context, userID int64, exerciseID int64, workoutIDs []int64) ([]domain.Set, error) {
	wanted := make(map[int64]bool, len(workoutIDs))
	for _, id := range workoutIDs {
		wanted[id] = true
	}
	var result []domain.Set
	for _, s := range mockSetsForExercise(userID, exerciseID) {
		if wanted[s.WorkoutID] {
			result = append(result, s)
		}
	}
	return result, nil
}

func (m *MockRepository) AddTransactions(_ context.Context, txs []*domain.Transaction) ([]*domain.Transaction, error) {
	now := time.Now()
	for i, tx := range txs {
		tx.ID = int64(i + 1)
		tx.CreatedAt = now
	}
	return txs, nil
}

// mockTransactions is fixture data for the money web dashboard preview
// pages, spanning the last few months across a handful of categories.
func mockTransactions(userID int64) []domain.Transaction {
	now := time.Now()
	day := func(n int) time.Time { return now.AddDate(0, 0, -n) }
	return []domain.Transaction{
		{ID: 1, UserID: userID, Type: domain.TransactionTypeIncome, AmountEUR: 3500, Currency: "EUR", Account: "Revolut", Category: "salary", Merchant: "Employer", TransactedAt: day(95), CreatedAt: day(95)},
		{ID: 2, UserID: userID, Type: domain.TransactionTypeExpense, AmountEUR: 700, Currency: "EUR", Account: "Revolut", Category: "rent", Merchant: "Landlord", TransactedAt: day(90), CreatedAt: day(90)},
		{ID: 3, UserID: userID, Type: domain.TransactionTypeExpense, AmountEUR: 320, Currency: "EUR", Account: "Revolut", Category: "groceries", Merchant: "Lidl", TransactedAt: day(60), CreatedAt: day(60)},
		{ID: 4, UserID: userID, Type: domain.TransactionTypeIncome, AmountEUR: 3500, Currency: "EUR", Account: "Revolut", Category: "salary", Merchant: "Employer", TransactedAt: day(65), CreatedAt: day(65)},
		{ID: 5, UserID: userID, Type: domain.TransactionTypeExpense, AmountEUR: 45, Currency: "EUR", Account: "Revolut", Category: "transport", Merchant: "Bolt", TransactedAt: day(40), CreatedAt: day(40)},
		{ID: 6, UserID: userID, Type: domain.TransactionTypeExpense, AmountEUR: 280, Currency: "EUR", Account: "Revolut", Category: "groceries", Merchant: "Lidl", TransactedAt: day(15), CreatedAt: day(15)},
		{ID: 7, UserID: userID, Type: domain.TransactionTypeIncome, AmountEUR: 3500, Currency: "EUR", Account: "Revolut", Category: "salary", Merchant: "Employer", TransactedAt: day(35), CreatedAt: day(35)},
		{ID: 8, UserID: userID, Type: domain.TransactionTypeExpense, AmountEUR: 32, Currency: "EUR", Account: "Revolut", Category: "food/cafe", Merchant: "Costa Coffee", TransactedAt: day(3), CreatedAt: day(1)},
		{ID: 9, UserID: userID, Type: domain.TransactionTypeExpense, AmountEUR: 18, Currency: "EUR", Account: "Revolut", Category: "transport", Merchant: "Bolt", TransactedAt: day(1), CreatedAt: day(1)},
	}
}

func (m *MockRepository) GetMoneySummary(_ context.Context, userID int64) (domain.MoneySummary, error) {
	txs := mockTransactions(userID)
	if len(txs) == 0 {
		return domain.MoneySummary{}, nil
	}
	first, last := txs[0].TransactedAt, txs[0].CreatedAt
	for _, tx := range txs[1:] {
		if tx.TransactedAt.Before(first) {
			first = tx.TransactedAt
		}
		if tx.CreatedAt.After(last) {
			last = tx.CreatedAt
		}
	}
	return domain.MoneySummary{FirstTransactionAt: &first, LastSyncedAt: &last}, nil
}

func (m *MockRepository) GetBalance(_ context.Context, userID int64, from, to time.Time) (domain.BalanceResult, error) {
	var income, expense float64
	for _, tx := range mockTransactions(userID) {
		if tx.TransactedAt.Before(from) || tx.TransactedAt.After(to) {
			continue
		}
		switch tx.Type {
		case domain.TransactionTypeIncome:
			income += tx.AmountEUR
		case domain.TransactionTypeExpense:
			expense += tx.AmountEUR
		}
	}
	return domain.BalanceResult{From: from, To: to, IncomeEUR: income, ExpenseEUR: expense, BalanceEUR: income - expense}, nil
}

func (m *MockRepository) GetSpendingByCategory(_ context.Context, userID int64, from, to time.Time, depth int) ([]domain.SpendingByCategory, error) {
	if depth < 1 {
		depth = 1
	}
	totals := map[string]*domain.SpendingByCategory{}
	var order []string
	for _, tx := range mockTransactions(userID) {
		if tx.Type != domain.TransactionTypeExpense {
			continue
		}
		if tx.TransactedAt.Before(from) || tx.TransactedAt.After(to) {
			continue
		}
		parts := strings.Split(tx.Category, "/")
		if len(parts) > depth {
			parts = parts[:depth]
		}
		cat := strings.Join(parts, "/")
		if totals[cat] == nil {
			totals[cat] = &domain.SpendingByCategory{Category: cat}
			order = append(order, cat)
		}
		totals[cat].TotalEUR += tx.AmountEUR
		totals[cat].Count++
	}
	result := make([]domain.SpendingByCategory, 0, len(order))
	for _, cat := range order {
		result = append(result, *totals[cat])
	}
	sort.Slice(result, func(i, j int) bool { return result[i].TotalEUR > result[j].TotalEUR })
	return result, nil
}

func (m *MockRepository) GetTransactions(_ context.Context, filter domain.TransactionFilter) ([]*domain.Transaction, int, error) {
	all := mockTransactions(filter.UserID)
	var filtered []*domain.Transaction
	for i := range all {
		tx := all[i]
		if filter.From != nil && tx.TransactedAt.Before(*filter.From) {
			continue
		}
		if filter.To != nil && tx.TransactedAt.After(*filter.To) {
			continue
		}
		if filter.Category != nil && !strings.HasPrefix(tx.Category, *filter.Category) {
			continue
		}
		filtered = append(filtered, &tx)
	}
	sort.Slice(filtered, func(i, j int) bool { return filtered[i].TransactedAt.After(filtered[j].TransactedAt) })

	total := len(filtered)
	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}
	offset := filter.Offset
	if offset > len(filtered) {
		offset = len(filtered)
	}
	end := offset + limit
	if end > len(filtered) {
		end = len(filtered)
	}
	return filtered[offset:end], total, nil
}

// mockAchievements is fixture data for the achievements dashboard and every page that
// embeds an achievement tile grid (Money/Progress-browse/Workouts), one achievement per
// applicable type on each of those domains, so the preview shows what an
// embedded (and the dedicated) grid actually looks like.
func mockAchievements(userID int64) []domain.Achievement {
	now := time.Now()
	deadline := now.AddDate(0, 3, 0)
	category := "food"
	unit := "sessions"
	baseline := 1000.0
	exerciseID := int64(1)
	activityID := int64(2)
	return []domain.Achievement{
		{ID: 1, UserID: userID, Name: "Emergency Fund", AchievementType: domain.AchievementTypeMoneySaving, TargetValue: 5000, CurrentValue: 2100, BaselineBalanceEUR: &baseline, StartsAt: now.AddDate(0, -2, 0), EndsAt: &deadline},
		{ID: 2, UserID: userID, Name: "Food Budget", AchievementType: domain.AchievementTypeMoneySpend, TargetValue: 400, CurrentValue: 320, Category: &category, StartsAt: now.AddDate(0, 0, -20)},
		{ID: 3, UserID: userID, Name: "Bench Press 100kg", AchievementType: domain.AchievementTypeExerciseMaxWeight, ExerciseID: &exerciseID, TargetValue: 100, CurrentValue: 82.5, StartsAt: now.AddDate(0, -1, 0)},
		{ID: 4, UserID: userID, Name: "Meditate 30 Times", AchievementType: domain.AchievementTypeActivityOccurrenceCount, ActivityID: &activityID, TargetValue: 30, CurrentValue: 12, Unit: &unit, StartsAt: now.AddDate(0, 0, -14)},
	}
}

func (m *MockRepository) ListAchievements(_ context.Context, filter domain.AchievementFilter) ([]domain.Achievement, error) {
	types := make(map[domain.AchievementType]bool, len(filter.Types))
	for _, t := range filter.Types {
		types[t] = true
	}
	var result []domain.Achievement
	for _, g := range mockAchievements(filter.UserID) {
		if len(types) > 0 && !types[g.AchievementType] {
			continue
		}
		result = append(result, g)
	}
	return result, nil
}

func (m *MockRepository) GetAchievement(_ context.Context, achievementID int64, userID int64) (*domain.Achievement, error) {
	for _, g := range mockAchievements(userID) {
		if g.ID == achievementID {
			return &g, nil
		}
	}
	return nil, nil
}

// GetDailyTransactionSummary mirrors the real repository's grouping by
// calendar day in the display timezone (Asia/Nicosia).
func (m *MockRepository) GetDailyTransactionSummary(_ context.Context, userID int64, from, to time.Time) ([]domain.DailySummary, error) {
	loc, err := time.LoadLocation("Asia/Nicosia")
	if err != nil {
		return nil, err
	}
	totals := map[string]*domain.DailySummary{}
	var order []string
	for _, tx := range mockTransactions(userID) {
		if tx.TransactedAt.Before(from) || tx.TransactedAt.After(to) {
			continue
		}
		local := tx.TransactedAt.In(loc)
		key := local.Format("2006-01-02")
		if totals[key] == nil {
			totals[key] = &domain.DailySummary{Date: time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, time.UTC)}
			order = append(order, key)
		}
		totals[key].Count++
		if tx.Type == domain.TransactionTypeExpense {
			totals[key].SpendEUR += tx.AmountEUR
		}
	}
	sort.Strings(order)
	result := make([]domain.DailySummary, 0, len(order))
	for _, key := range order {
		result = append(result, *totals[key])
	}
	return result, nil
}

func (m *MockRepository) ListIdeas(_ context.Context, filter domain.IdeaFilter) ([]domain.Idea, error) {
	now := time.Now()
	blocked := domain.IdeaResolutionBlocked
	ideas := []domain.Idea{
		{ID: 1, UserID: filter.UserID, Body: "Learn to sail", Status: domain.IdeaStatusSpike, CreatedAt: now.AddDate(0, -2, 0), SurfaceCount: 3, LastSurfacedAt: now.AddDate(0, 0, -3)},
		{ID: 2, UserID: filter.UserID, Body: "Build a bookshelf for the hallway", Status: domain.IdeaStatusInbox, CreatedAt: now.AddDate(0, 0, -1), SurfaceCount: 1, LastSurfacedAt: now.AddDate(0, 0, -1)},
		{ID: 3, UserID: filter.UserID, Body: "Write a blog post about the personal tracker", Status: domain.IdeaStatusSomeday, CreatedAt: now.AddDate(0, -1, 0), SurfaceCount: 2, LastSurfacedAt: now.AddDate(0, 0, -10)},
		{ID: 4, UserID: filter.UserID, Body: "Start a running club", Status: domain.IdeaStatusResolved, Resolution: &blocked, CreatedAt: now.AddDate(0, -3, 0), SurfaceCount: 1, LastSurfacedAt: now.AddDate(0, -3, 0)},
	}
	var result []domain.Idea
	for _, idea := range ideas {
		switch {
		case len(filter.Statuses) > 0:
			if slices.Contains(filter.Statuses, idea.Status) {
				result = append(result, idea)
			}
		case len(filter.Resolutions) > 0:
			if idea.Resolution != nil && slices.Contains(filter.Resolutions, *idea.Resolution) {
				result = append(result, idea)
			}
		case idea.Status != domain.IdeaStatusResolved:
			result = append(result, idea)
		}
	}
	if filter.Limit > 0 {
		slices.Reverse(result)
		result = result[:min(filter.Limit, len(result))]
	}
	return result, nil
}

func (m *MockRepository) SearchIdeas(ctx context.Context, filter domain.IdeaSearchFilter) ([]domain.Idea, error) {
	all, _ := m.ListIdeas(ctx, domain.IdeaFilter{UserID: filter.UserID, Statuses: domain.IdeaStatuses})
	var result []domain.Idea
	for _, idea := range all {
		if strings.Contains(strings.ToLower(idea.Body), strings.ToLower(filter.Query)) {
			result = append(result, idea)
		}
	}
	return result, nil
}

func (m *MockRepository) CreateIdea(_ context.Context, _ *domain.Idea) (int64, error) {
	return 100, nil
}

func (m *MockRepository) GetIdea(_ context.Context, ideaID int64, userID int64) (*domain.Idea, error) {
	return &domain.Idea{ID: ideaID, UserID: userID, Status: domain.IdeaStatusInbox, CreatedAt: time.Now(), UpdatedAt: time.Now(), SurfaceCount: 1, LastSurfacedAt: time.Now()}, nil
}
