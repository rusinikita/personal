package db

import (
	"context"
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
	return []domain.Activity{
		{ID: 1, UserID: filter.UserID, Name: "Ship personal tracker", Description: "**Refactor** web transport", ProgressType: domain.ProgressTypeProjectProgress, FrequencyDays: 1, StartedAt: now.AddDate(0, 0, -14), LastPointAt: &yesterday},
		{ID: 2, UserID: filter.UserID, Name: "Gym", Description: "Push/pull/legs", ProgressType: domain.ProgressTypeHabitProgress, FrequencyDays: 2, StartedAt: now.AddDate(0, 0, -60), LastPointAt: &yesterday},
		{ID: 3, UserID: filter.UserID, Name: "Mood check-in", ProgressType: domain.ProgressTypeMood, FrequencyDays: 1, StartedAt: now.AddDate(0, 0, -90), LastPointAt: &now},
		{ID: 4, UserID: filter.UserID, Name: "Call mom", ProgressType: domain.ProgressTypePromiseState, FrequencyDays: 7, StartedAt: now.AddDate(0, 0, -30), LastPointAt: &yesterday},
	}, nil
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
