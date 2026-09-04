package tests

// Covers the cross-domain Goals feature from docs/functions/goals-spec.md:
// create_goal, update_goal, refresh_goals, get_goal_progress, and
// log_goal_progress — the five MCP tools, one per goal_type's own
// derivation where it matters (money_saving/money_spend/exercise_max_weight/
// exercise_total_volume/activity_occurrence_count/activity_streak_count/manual).

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"personal/action/goals"
	"personal/action/money"
	"personal/domain"
)

// --- create_goal -------------------------------------------------------------

func (s *IntegrationTestSuite) TestCreateGoal_MoneySpend_ComputesInitialSpendFromCategory() {
	ctx := s.Context()
	now := time.Now().UTC()

	_, _, err := money.AddTransactions(ctx, nil, money.AddTransactionsInput{
		Transactions: []money.TransactionInput{
			{Type: "expense", AmountOriginal: 150, Currency: "EUR", AmountEUR: 150, Account: "Revolut", Category: "food/cafe", Merchant: "Starbucks", TransactedAt: now.Add(-time.Hour)},
			{Type: "expense", AmountOriginal: 30, Currency: "EUR", AmountEUR: 30, Account: "Revolut", Category: "transport", Merchant: "Bolt", TransactedAt: now.Add(-time.Hour)},
		},
	})
	require.NoError(s.T(), err)

	category := "food"
	starts := now.Add(-2 * time.Hour)
	_, out, err := goals.CreateGoal(ctx, nil, goals.CreateGoalInput{
		Name: "Food Budget", GoalType: "money_spend", TargetValue: 500, Category: &category, StartsAt: &starts,
	})
	require.NoError(s.T(), err)
	require.Empty(s.T(), out.Error)
	assert.NotZero(s.T(), out.ID)
	assert.InDelta(s.T(), 150.0, out.Goal.CurrentValue, 0.01, "only food-prefixed spend counts")
	assert.InDelta(s.T(), 350.0, out.Goal.RemainingValue, 0.01)
}

func (s *IntegrationTestSuite) TestCreateGoal_MoneySaving_CapturesBaselineAndStartsAtZero() {
	ctx := s.Context()
	now := time.Now().UTC()

	_, _, err := money.AddTransactions(ctx, nil, money.AddTransactionsInput{
		Transactions: []money.TransactionInput{
			{Type: "income", AmountOriginal: 1000, Currency: "EUR", AmountEUR: 1000, Account: "Revolut", Category: "salary", Merchant: "Employer", TransactedAt: now.Add(-10 * 24 * time.Hour)},
		},
	})
	require.NoError(s.T(), err)

	_, out, err := goals.CreateGoal(ctx, nil, goals.CreateGoalInput{
		Name: "Emergency Fund", GoalType: "money_saving", TargetValue: 500,
	})
	require.NoError(s.T(), err)
	require.Empty(s.T(), out.Error)
	require.NotNil(s.T(), out.Goal.BaselineBalanceEUR)
	assert.InDelta(s.T(), 1000.0, *out.Goal.BaselineBalanceEUR, 0.01, "baseline is the balance as of goal creation")
	assert.InDelta(s.T(), 0.0, out.Goal.CurrentValue, 0.01, "no saving has happened yet, relative to the baseline")
}

func (s *IntegrationTestSuite) TestCreateGoal_ExerciseMaxWeight_AlreadyMetIfBackdated() {
	ctx := s.Context()
	exerciseID := s.createExercise(ctx, "Bench Press", "barbell")

	wID := s.createWorkout(ctx, time.Now().Add(-48*time.Hour))
	s.createSet(ctx, wID, exerciseID, 3, 100, time.Now().Add(-48*time.Hour))

	_, out, err := goals.CreateGoal(ctx, nil, goals.CreateGoalInput{
		Name: "Bench 90kg", GoalType: "exercise_max_weight", TargetValue: 90, ExerciseID: &exerciseID,
	})
	require.NoError(s.T(), err)
	require.Empty(s.T(), out.Error)
	assert.InDelta(s.T(), 100.0, out.Goal.CurrentValue, 0.01, "a goal is already met if the max weight predates it")
	assert.Less(s.T(), out.Goal.RemainingValue, 0.0, "remaining goes negative once exceeded")
}

func (s *IntegrationTestSuite) TestCreateGoal_ExerciseTotalVolume_SumsWeightTimesReps() {
	ctx := s.Context()
	exerciseID := s.createExercise(ctx, "Squat", "barbell")

	wID := s.createWorkout(ctx, time.Now().Add(-time.Hour))
	s.createSet(ctx, wID, exerciseID, 5, 100, time.Now().Add(-time.Hour))
	s.createSet(ctx, wID, exerciseID, 5, 100, time.Now().Add(-time.Hour))

	starts := time.Now().Add(-2 * time.Hour)
	_, out, err := goals.CreateGoal(ctx, nil, goals.CreateGoalInput{
		Name: "5000kg Quarter", GoalType: "exercise_total_volume", TargetValue: 5000, ExerciseID: &exerciseID, StartsAt: &starts,
	})
	require.NoError(s.T(), err)
	require.Empty(s.T(), out.Error)
	assert.InDelta(s.T(), 1000.0, out.Goal.CurrentValue, 0.01)
}

func (s *IntegrationTestSuite) TestCreateGoal_ActivityOccurrenceCount_CountsSinceStartsAt() {
	ctx := s.Context()
	userID := s.UserID()
	now := time.Now().UTC()
	activityID := s.createActivity(ctx, "Meditation", domain.ProgressTypeHabitProgress, now.Add(-30*24*time.Hour))

	starts := now.Add(-10 * 24 * time.Hour)
	// Before the goal's window — must not count.
	_, err := s.Repo().CreateProgress(ctx, &domain.ActivityPoint{UserID: userID, ActivityID: activityID, Value: 1, ProgressAt: now.Add(-20 * 24 * time.Hour)})
	require.NoError(s.T(), err)
	// Inside the goal's window — must count.
	_, err = s.Repo().CreateProgress(ctx, &domain.ActivityPoint{UserID: userID, ActivityID: activityID, Value: 1, ProgressAt: now.Add(-5 * 24 * time.Hour)})
	require.NoError(s.T(), err)
	_, err = s.Repo().CreateProgress(ctx, &domain.ActivityPoint{UserID: userID, ActivityID: activityID, Value: 1, ProgressAt: now.Add(-1 * 24 * time.Hour)})
	require.NoError(s.T(), err)

	_, out, err := goals.CreateGoal(ctx, nil, goals.CreateGoalInput{
		Name: "Meditate 30 Times", GoalType: "activity_occurrence_count", TargetValue: 30, ActivityID: &activityID, StartsAt: &starts,
	})
	require.NoError(s.T(), err)
	require.Empty(s.T(), out.Error)
	assert.InDelta(s.T(), 2.0, out.Goal.CurrentValue, 0.01)
}

func (s *IntegrationTestSuite) TestCreateGoal_ActivityStreakCount_LapsedStreakIsZero() {
	ctx := s.Context()
	userID := s.UserID()
	now := time.Now().UTC()
	activityID := s.createActivity(ctx, "Journaling", domain.ProgressTypeHabitProgress, now.Add(-60*24*time.Hour))

	// Last check-in was 10 days ago — with a 1-day frequency, the streak has
	// already lapsed by the time the goal is created (today).
	_, err := s.Repo().CreateProgress(ctx, &domain.ActivityPoint{UserID: userID, ActivityID: activityID, Value: 1, ProgressAt: now.Add(-10 * 24 * time.Hour)})
	require.NoError(s.T(), err)

	_, out, err := goals.CreateGoal(ctx, nil, goals.CreateGoalInput{
		Name: "30-day Streak", GoalType: "activity_streak_count", TargetValue: 30, ActivityID: &activityID,
	})
	require.NoError(s.T(), err)
	require.Empty(s.T(), out.Error)
	assert.InDelta(s.T(), 0.0, out.Goal.CurrentValue, 0.01)
}

func (s *IntegrationTestSuite) TestCreateGoal_ActivityStreakCount_CountsConsecutiveCheckIns() {
	ctx := s.Context()
	userID := s.UserID()
	now := time.Now().UTC()
	activityID := s.createActivity(ctx, "Journaling", domain.ProgressTypeHabitProgress, now.Add(-60*24*time.Hour))

	for i := 0; i < 3; i++ {
		_, err := s.Repo().CreateProgress(ctx, &domain.ActivityPoint{
			UserID: userID, ActivityID: activityID, Value: 1, ProgressAt: now.Add(-time.Duration(i) * 24 * time.Hour),
		})
		require.NoError(s.T(), err)
	}

	_, out, err := goals.CreateGoal(ctx, nil, goals.CreateGoalInput{
		Name: "30-day Streak", GoalType: "activity_streak_count", TargetValue: 30, ActivityID: &activityID,
	})
	require.NoError(s.T(), err)
	require.Empty(s.T(), out.Error)
	assert.InDelta(s.T(), 3.0, out.Goal.CurrentValue, 0.01)
}

func (s *IntegrationTestSuite) TestCreateGoal_Manual_StartsAtZero() {
	ctx := s.Context()

	_, out, err := goals.CreateGoal(ctx, nil, goals.CreateGoalInput{Name: "Read 12 Books", GoalType: "manual", TargetValue: 12})
	require.NoError(s.T(), err)
	require.Empty(s.T(), out.Error)
	assert.Equal(s.T(), 0.0, out.Goal.CurrentValue)
}

func (s *IntegrationTestSuite) TestCreateGoal_ValidationErrors() {
	ctx := s.Context()
	category := "food"
	unit := "kg"
	pastEnd := time.Now().Add(-time.Hour)
	badExerciseID := int64(999999)
	badActivityID := int64(999999)

	cases := []struct {
		name  string
		input goals.CreateGoalInput
		want  string
	}{
		{"unknown type", goals.CreateGoalInput{Name: "x", GoalType: "not_a_type", TargetValue: 1}, `unknown goal_type "not_a_type"`},
		{"non-positive target", goals.CreateGoalInput{Name: "x", GoalType: "manual", TargetValue: 0}, "target_value must be greater than 0"},
		{"ends before starts", goals.CreateGoalInput{Name: "x", GoalType: "manual", TargetValue: 1, EndsAt: &pastEnd}, "ends_at must be after starts_at"},
		{"money_spend missing category", goals.CreateGoalInput{Name: "x", GoalType: "money_spend", TargetValue: 1}, "category is required for money_spend goals"},
		{"money_saving rejects unit", goals.CreateGoalInput{Name: "x", GoalType: "money_saving", TargetValue: 1, Unit: &unit}, "unit is not applicable to money_saving goals"},
		{"money_saving rejects category", goals.CreateGoalInput{Name: "x", GoalType: "money_saving", TargetValue: 1, Category: &category}, "category is not applicable to money_saving goals"},
		{"exercise_max_weight missing exercise_id", goals.CreateGoalInput{Name: "x", GoalType: "exercise_max_weight", TargetValue: 1}, "exercise_id is required for exercise_max_weight goals"},
		{"exercise_max_weight not found", goals.CreateGoalInput{Name: "x", GoalType: "exercise_max_weight", TargetValue: 1, ExerciseID: &badExerciseID}, "exercise not found"},
		{"activity_occurrence_count missing activity_id", goals.CreateGoalInput{Name: "x", GoalType: "activity_occurrence_count", TargetValue: 1}, "activity_id is required for activity_occurrence_count goals"},
		{"activity_occurrence_count not found", goals.CreateGoalInput{Name: "x", GoalType: "activity_occurrence_count", TargetValue: 1, ActivityID: &badActivityID}, "activity not found"},
	}

	for _, tc := range cases {
		s.T().Run(tc.name, func(t *testing.T) {
			_, out, err := goals.CreateGoal(ctx, nil, tc.input)
			require.NoError(t, err)
			assert.Equal(t, tc.want, out.Error)
		})
	}
}

// --- update_goal ---------------------------------------------------------

func (s *IntegrationTestSuite) TestUpdateGoal_NameTargetAndDeadline() {
	ctx := s.Context()

	_, created, err := goals.CreateGoal(ctx, nil, goals.CreateGoalInput{Name: "Read 12 Books", GoalType: "manual", TargetValue: 12})
	require.NoError(s.T(), err)

	newName := "Read 20 Books"
	newTarget := 20.0
	newEnd := time.Now().Add(90 * 24 * time.Hour)
	_, out, err := goals.UpdateGoal(ctx, nil, goals.UpdateGoalInput{
		GoalID: created.ID, Name: &newName, TargetValue: &newTarget, EndsAt: &newEnd,
	})
	require.NoError(s.T(), err)
	require.Empty(s.T(), out.Error)
	assert.Equal(s.T(), "Read 20 Books", out.Goal.Name)
	assert.Equal(s.T(), 20.0, out.Goal.TargetValue)
	require.NotNil(s.T(), out.Goal.EndsAt)
	assert.WithinDuration(s.T(), newEnd, *out.Goal.EndsAt, time.Second)
}

func (s *IntegrationTestSuite) TestUpdateGoal_ClearEndsAt() {
	ctx := s.Context()
	end := time.Now().Add(30 * 24 * time.Hour)

	_, created, err := goals.CreateGoal(ctx, nil, goals.CreateGoalInput{Name: "x", GoalType: "manual", TargetValue: 1, EndsAt: &end})
	require.NoError(s.T(), err)
	require.NotNil(s.T(), created.Goal.EndsAt)

	_, out, err := goals.UpdateGoal(ctx, nil, goals.UpdateGoalInput{GoalID: created.ID, ClearEndsAt: true})
	require.NoError(s.T(), err)
	require.Empty(s.T(), out.Error)
	assert.Nil(s.T(), out.Goal.EndsAt)
}

func (s *IntegrationTestSuite) TestUpdateGoal_CurrentValue_OnlyManualCanBeCorrected() {
	ctx := s.Context()
	exerciseID := s.createExercise(ctx, "Deadlift", "barbell")

	_, manualGoal, err := goals.CreateGoal(ctx, nil, goals.CreateGoalInput{Name: "Read 12 Books", GoalType: "manual", TargetValue: 12})
	require.NoError(s.T(), err)
	newValue := 5.0
	_, out, err := goals.UpdateGoal(ctx, nil, goals.UpdateGoalInput{GoalID: manualGoal.ID, CurrentValue: &newValue})
	require.NoError(s.T(), err)
	require.Empty(s.T(), out.Error)
	assert.Equal(s.T(), 5.0, out.Goal.CurrentValue)

	_, exerciseGoal, err := goals.CreateGoal(ctx, nil, goals.CreateGoalInput{Name: "Deadlift 150kg", GoalType: "exercise_max_weight", TargetValue: 150, ExerciseID: &exerciseID})
	require.NoError(s.T(), err)
	_, rejected, err := goals.UpdateGoal(ctx, nil, goals.UpdateGoalInput{GoalID: exerciseGoal.ID, CurrentValue: &newValue})
	require.NoError(s.T(), err)
	assert.Equal(s.T(), "current_value can only be corrected on manual goals", rejected.Error)
}

func (s *IntegrationTestSuite) TestUpdateGoal_Category_OnlyMoneySpend() {
	ctx := s.Context()
	category := "food"

	_, spendGoal, err := goals.CreateGoal(ctx, nil, goals.CreateGoalInput{Name: "Food", GoalType: "money_spend", TargetValue: 100, Category: &category})
	require.NoError(s.T(), err)
	newCategory := "groceries"
	_, out, err := goals.UpdateGoal(ctx, nil, goals.UpdateGoalInput{GoalID: spendGoal.ID, Category: &newCategory})
	require.NoError(s.T(), err)
	require.Empty(s.T(), out.Error)
	require.NotNil(s.T(), out.Goal.Category)
	assert.Equal(s.T(), "groceries", *out.Goal.Category)

	_, manualGoal, err := goals.CreateGoal(ctx, nil, goals.CreateGoalInput{Name: "x", GoalType: "manual", TargetValue: 1})
	require.NoError(s.T(), err)
	_, rejected, err := goals.UpdateGoal(ctx, nil, goals.UpdateGoalInput{GoalID: manualGoal.ID, Category: &newCategory})
	require.NoError(s.T(), err)
	assert.Equal(s.T(), "category can only be set on money_spend goals", rejected.Error)
}

func (s *IntegrationTestSuite) TestUpdateGoal_NotFound() {
	ctx := s.Context()
	newName := "x"
	_, out, err := goals.UpdateGoal(ctx, nil, goals.UpdateGoalInput{GoalID: 999999, Name: &newName})
	require.NoError(s.T(), err)
	assert.Equal(s.T(), "goal not found", out.Error)
}

// --- refresh_goals ---------------------------------------------------------

func (s *IntegrationTestSuite) TestRefreshGoals_SingleGoal_RecomputesOnDemandOnly() {
	ctx := s.Context()
	now := time.Now().UTC()

	category := "food"
	starts := now.Add(-time.Hour)
	_, created, err := goals.CreateGoal(ctx, nil, goals.CreateGoalInput{Name: "Food", GoalType: "money_spend", TargetValue: 500, Category: &category, StartsAt: &starts})
	require.NoError(s.T(), err)
	assert.Equal(s.T(), 0.0, created.Goal.CurrentValue, "no spend yet at creation time")

	_, _, err = money.AddTransactions(ctx, nil, money.AddTransactionsInput{
		Transactions: []money.TransactionInput{
			{Type: "expense", AmountOriginal: 75, Currency: "EUR", AmountEUR: 75, Account: "Revolut", Category: "food/cafe", Merchant: "Starbucks", TransactedAt: now},
		},
	})
	require.NoError(s.T(), err)

	// get_goal_progress must still show the stale cached value — reads never
	// recompute (see goals-spec.md Best Practices).
	_, progressOut, err := goals.GetGoalProgress(ctx, nil, goals.GetGoalProgressInput{})
	require.NoError(s.T(), err)
	require.Len(s.T(), progressOut.Goals, 1)
	assert.Equal(s.T(), 0.0, progressOut.Goals[0].CurrentValue, "current_value is cached, not live")

	goalID := created.ID
	_, refreshOut, err := goals.RefreshGoals(ctx, nil, goals.RefreshGoalsInput{GoalID: &goalID})
	require.NoError(s.T(), err)
	require.Empty(s.T(), refreshOut.Error)
	require.NotNil(s.T(), refreshOut.Goal)
	assert.InDelta(s.T(), 75.0, refreshOut.Goal.CurrentValue, 0.01)
	assert.Equal(s.T(), 1, refreshOut.Refreshed)
}

func (s *IntegrationTestSuite) TestRefreshGoals_AllActive_SkipsManual() {
	ctx := s.Context()

	_, spendGoal, err := goals.CreateGoal(ctx, nil, goals.CreateGoalInput{Name: "Spend", GoalType: "money_spend", TargetValue: 100, Category: strPtr("food")})
	require.NoError(s.T(), err)
	_, manualGoal, err := goals.CreateGoal(ctx, nil, goals.CreateGoalInput{Name: "Manual", GoalType: "manual", TargetValue: 10})
	require.NoError(s.T(), err)

	_, out, err := goals.RefreshGoals(ctx, nil, goals.RefreshGoalsInput{})
	require.NoError(s.T(), err)
	require.Empty(s.T(), out.Error)
	assert.Equal(s.T(), 1, out.Refreshed, "manual must be skipped")
	assert.NotZero(s.T(), spendGoal.ID)
	assert.NotZero(s.T(), manualGoal.ID)
}

func (s *IntegrationTestSuite) TestRefreshGoals_ManualGoal_Rejected() {
	ctx := s.Context()
	_, created, err := goals.CreateGoal(ctx, nil, goals.CreateGoalInput{Name: "Manual", GoalType: "manual", TargetValue: 10})
	require.NoError(s.T(), err)

	goalID := created.ID
	_, out, err := goals.RefreshGoals(ctx, nil, goals.RefreshGoalsInput{GoalID: &goalID})
	require.NoError(s.T(), err)
	assert.Equal(s.T(), "manual goals have no derived progress to refresh; use log_goal_progress or update_goal instead", out.Error)
}

// --- get_goal_progress -----------------------------------------------------

func (s *IntegrationTestSuite) TestGetGoalProgress_OnlyActiveGoals_WithRemainingValue() {
	ctx := s.Context()
	now := time.Now().UTC()

	pastEnd := now.Add(-24 * time.Hour)
	pastStart := now.Add(-48 * time.Hour)
	_, _, err := goals.CreateGoal(ctx, nil, goals.CreateGoalInput{Name: "Old goal", GoalType: "manual", TargetValue: 10, StartsAt: &pastStart, EndsAt: &pastEnd})
	require.NoError(s.T(), err)

	_, active, err := goals.CreateGoal(ctx, nil, goals.CreateGoalInput{Name: "Active goal", GoalType: "manual", TargetValue: 10})
	require.NoError(s.T(), err)

	delta := 3.0
	_, _, err = goals.LogGoalProgress(ctx, nil, goals.LogGoalProgressInput{GoalID: active.ID, Delta: &delta})
	require.NoError(s.T(), err)

	_, out, err := goals.GetGoalProgress(ctx, nil, goals.GetGoalProgressInput{})
	require.NoError(s.T(), err)
	require.Len(s.T(), out.Goals, 1, "the finished goal must not show as active")
	assert.Equal(s.T(), "Active goal", out.Goals[0].Name)
	assert.Equal(s.T(), 3.0, out.Goals[0].CurrentValue)
	assert.Equal(s.T(), 7.0, out.Goals[0].RemainingValue)
}

// --- log_goal_progress -------------------------------------------------------

func (s *IntegrationTestSuite) TestLogGoalProgress_DefaultAndCustomDelta() {
	ctx := s.Context()
	_, created, err := goals.CreateGoal(ctx, nil, goals.CreateGoalInput{Name: "Read 12 Books", GoalType: "manual", TargetValue: 12})
	require.NoError(s.T(), err)

	_, out1, err := goals.LogGoalProgress(ctx, nil, goals.LogGoalProgressInput{GoalID: created.ID})
	require.NoError(s.T(), err)
	require.Empty(s.T(), out1.Error)
	assert.Equal(s.T(), 1.0, out1.Goal.CurrentValue)

	delta := 2.0
	_, out2, err := goals.LogGoalProgress(ctx, nil, goals.LogGoalProgressInput{GoalID: created.ID, Delta: &delta})
	require.NoError(s.T(), err)
	assert.Equal(s.T(), 3.0, out2.Goal.CurrentValue)
}

func (s *IntegrationTestSuite) TestLogGoalProgress_NonManualGoalRejected() {
	ctx := s.Context()
	_, created, err := goals.CreateGoal(ctx, nil, goals.CreateGoalInput{Name: "Spend", GoalType: "money_spend", TargetValue: 100, Category: strPtr("food")})
	require.NoError(s.T(), err)

	_, out, err := goals.LogGoalProgress(ctx, nil, goals.LogGoalProgressInput{GoalID: created.ID})
	require.NoError(s.T(), err)
	assert.Equal(s.T(), "log_goal_progress only applies to manual goals", out.Error)
}

func strPtr(s string) *string { return &s }
