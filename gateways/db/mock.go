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

func (m *MockRepository) AddTransactions(_ context.Context, txs []*domain.Transaction) ([]*domain.Transaction, error) {
	now := time.Now()
	for i, tx := range txs {
		tx.ID = int64(i + 1)
		tx.CreatedAt = now
	}
	return txs, nil
}
