package tests

// Covers the Goals web dashboard from docs/functions/goals-spec.md:
// GET /web/goals (active goals as a tile grid, past/completed as a table)
// and POST /web/goals/refresh (the one write route), plus the embedded goal
// tile grids on Money/Progress-browse/Workouts.

import (
	"context"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"personal/action/goals"
	"personal/action/money"
	"personal/domain"
	"personal/gateways"
)

// goalsDashboardRouter builds a minimal gin engine wired to the test
// suite's DB and user_id, serving the two goals dashboard routes the same
// way transport/web does.
func (s *IntegrationTestSuite) goalsDashboardRouter(ctx context.Context) *gin.Engine {
	userID := gateways.UserIDFromContext(ctx)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		reqCtx := gateways.WithDB(c.Request.Context(), s.Repo())
		reqCtx = gateways.WithUserID(reqCtx, userID)
		c.Request = c.Request.WithContext(reqCtx)
		c.Next()
	})
	r.GET("/web/goals", goals.DashboardWebHandler)
	r.POST("/web/goals/refresh", goals.RefreshWebHandler)
	r.GET("/web/goals/eink", goals.EinkDashboardWebHandler)
	return r
}

// --- GET /web/goals --------------------------------------------------------

func (s *IntegrationTestSuite) TestGoalsDashboard_ActiveGoalsShowAsTiles() {
	ctx := s.Context()

	_, _, err := goals.CreateGoal(ctx, nil, goals.CreateGoalInput{Name: "Read 12 Books", GoalType: "manual", TargetValue: 12})
	require.NoError(s.T(), err)

	r := s.goalsDashboardRouter(ctx)
	req := httptest.NewRequest(http.MethodGet, "/web/goals", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code)
	body := w.Body.String()
	assert.Contains(s.T(), body, "Read 12 Books")
	assert.Contains(s.T(), body, `<article class="webui-goal-tile`)
	assert.Contains(s.T(), body, `action="/web/goals/refresh"`, "must show the Refresh form")
}

func (s *IntegrationTestSuite) TestGoalsDashboard_EmptyState() {
	r := s.goalsDashboardRouter(s.Context())
	req := httptest.NewRequest(http.MethodGet, "/web/goals", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code)
	assert.Contains(s.T(), w.Body.String(), "No active goals yet")
}

func (s *IntegrationTestSuite) TestGoalsDashboard_PastGoalsListedInTable_NotAsTiles() {
	ctx := s.Context()
	now := time.Now().UTC()
	pastStart := now.Add(-48 * time.Hour)
	pastEnd := now.Add(-24 * time.Hour)

	_, _, err := goals.CreateGoal(ctx, nil, goals.CreateGoalInput{
		Name: "Old Goal", GoalType: "manual", TargetValue: 12, StartsAt: &pastStart, EndsAt: &pastEnd,
	})
	require.NoError(s.T(), err)

	r := s.goalsDashboardRouter(ctx)
	req := httptest.NewRequest(http.MethodGet, "/web/goals", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code)
	body := w.Body.String()
	assert.Contains(s.T(), body, "Old Goal")
	assert.Contains(s.T(), body, "No active goals yet", "the finished goal must not count as active")
}

// --- POST /web/goals/refresh ------------------------------------------------

func (s *IntegrationTestSuite) TestGoalsRefresh_RecomputesActiveGoalsAndRedirects() {
	ctx := s.Context()
	now := time.Now().UTC()
	category := "food"
	starts := now.Add(-time.Hour)

	_, created, err := goals.CreateGoal(ctx, nil, goals.CreateGoalInput{
		Name: "Food Budget", GoalType: "money_spend", TargetValue: 500, Category: &category, StartsAt: &starts,
	})
	require.NoError(s.T(), err)
	assert.Equal(s.T(), 0.0, created.Goal.CurrentValue)

	_, _, err = money.AddTransactions(ctx, nil, money.AddTransactionsInput{
		Transactions: []money.TransactionInput{
			{Type: "expense", AmountOriginal: 80, Currency: "EUR", AmountEUR: 80, Account: "Revolut", Category: "food/cafe", Merchant: "Starbucks", TransactedAt: now},
		},
	})
	require.NoError(s.T(), err)

	r := s.goalsDashboardRouter(ctx)
	req := httptest.NewRequest(http.MethodPost, "/web/goals/refresh", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusFound, w.Code)
	assert.Equal(s.T(), "/web/goals", w.Header().Get("Location"))

	updated, err := s.Repo().GetGoal(ctx, created.ID, gateways.UserIDFromContext(ctx))
	require.NoError(s.T(), err)
	assert.InDelta(s.T(), 80.0, updated.CurrentValue, 0.01)
}

// --- GET /web/goals/eink ----------------------------------------------------

func (s *IntegrationTestSuite) TestGoalsEink_ActiveGoalsShowAsTiles_NoRefreshFormOrPastTable() {
	ctx := s.Context()
	now := time.Now().UTC()
	pastStart := now.Add(-48 * time.Hour)
	pastEnd := now.Add(-24 * time.Hour)

	_, _, err := goals.CreateGoal(ctx, nil, goals.CreateGoalInput{Name: "Read 12 Books", GoalType: "manual", TargetValue: 12})
	require.NoError(s.T(), err)
	_, _, err = goals.CreateGoal(ctx, nil, goals.CreateGoalInput{
		Name: "Old Goal", GoalType: "manual", TargetValue: 12, StartsAt: &pastStart, EndsAt: &pastEnd,
	})
	require.NoError(s.T(), err)

	r := s.goalsDashboardRouter(ctx)
	req := httptest.NewRequest(http.MethodGet, "/web/goals/eink", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code)
	assert.Equal(s.T(), "text/html; charset=utf-8", w.Header().Get("Content-Type"))
	body := w.Body.String()
	assert.Contains(s.T(), body, `<div class="page-title">Goals</div>`)
	assert.Contains(s.T(), body, "Read 12 Books")
	assert.Contains(s.T(), body, `<article class="webui-goal-tile`)
	assert.NotContains(s.T(), body, "Old Goal", "past/completed goals must not appear on the e-ink page")
	assert.NotContains(s.T(), body, `action="/web/goals/refresh"`, "e-ink page has no Refresh form")
	assert.Contains(s.T(), body, "100vw")
	assert.Contains(s.T(), body, "100vh")
}

func (s *IntegrationTestSuite) TestGoalsEink_EmptyState() {
	r := s.goalsDashboardRouter(s.Context())
	req := httptest.NewRequest(http.MethodGet, "/web/goals/eink", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code)
	assert.Contains(s.T(), w.Body.String(), "No active goals")
}

// --- Embedded goal tiles on Money/Progress-browse/Workouts ------------------

func (s *IntegrationTestSuite) TestMoneyDashboard_EmbedsOwnGoalTiles() {
	ctx := s.Context()
	category := "food"
	_, _, err := goals.CreateGoal(ctx, nil, goals.CreateGoalInput{Name: "Food Budget", GoalType: "money_spend", TargetValue: 500, Category: &category})
	require.NoError(s.T(), err)

	r := s.moneyDashboardRouter(ctx)
	req := httptest.NewRequest(http.MethodGet, "/web/money", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code)
	assert.Contains(s.T(), w.Body.String(), "Food Budget")
}

func (s *IntegrationTestSuite) TestMoneyDashboard_NoGoalsMeansNoGoalTilesSection() {
	r := s.moneyDashboardRouter(s.Context())
	req := httptest.NewRequest(http.MethodGet, "/web/money", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code)
	assert.NotContains(s.T(), w.Body.String(), `<article class="webui-goal-tile`, "no tile markup must be rendered, not just an empty grid")
}

func (s *IntegrationTestSuite) TestWorkoutsDashboard_EmbedsOwnGoalTiles() {
	ctx := s.Context()
	exerciseID := s.createExercise(ctx, "Bench Press", "barbell")

	_, _, err := goals.CreateGoal(ctx, nil, goals.CreateGoalInput{Name: "Bench Press 100kg", GoalType: "exercise_max_weight", TargetValue: 100, ExerciseID: &exerciseID})
	require.NoError(s.T(), err)

	r := s.workoutDashboardRouter(ctx)
	req := httptest.NewRequest(http.MethodGet, "/web/workouts", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code)
	assert.Contains(s.T(), w.Body.String(), "Bench Press 100kg")
}

func (s *IntegrationTestSuite) TestProgressBrowse_EmbedsOwnGoalTiles() {
	ctx := s.Context()
	activityID := s.createActivity(ctx, "Meditation", domain.ProgressTypeHabitProgress, time.Now().AddDate(0, 0, -30))

	_, _, err := goals.CreateGoal(ctx, nil, goals.CreateGoalInput{Name: "Meditate 30 Times", GoalType: "activity_occurrence_count", TargetValue: 30, ActivityID: &activityID})
	require.NoError(s.T(), err)

	r := s.browseRouter(ctx)
	req := httptest.NewRequest(http.MethodGet, "/web/progress/browse", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code)
	assert.Contains(s.T(), w.Body.String(), "Meditate 30 Times")
}
