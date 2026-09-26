package tests

// Covers the cross-domain Achievements feature from docs/functions/achievements-spec.md:
// create_achievement, update_achievement, refresh_achievements, get_achievement_progress, and
// log_achievement_progress — the five MCP tools, one per achievement_type's own
// derivation where it matters (money_saving/money_spend/exercise_max_weight/
// exercise_total_volume/activity_occurrence_count/activity_streak_count/manual).

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"personal/action/achievements"
	"personal/action/money"
	"personal/domain"
)

// --- create_achievement -------------------------------------------------------------

func (s *IntegrationTestSuite) TestCreateAchievement_MoneySpend_ComputesInitialSpendFromCategory() {
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
	_, out, err := achievements.CreateAchievement(ctx, nil, achievements.CreateAchievementInput{
		Name: "Food Budget", AchievementType: "money_spend", TargetValue: 500, Category: &category, StartsAt: &starts,
	})
	require.NoError(s.T(), err)
	require.Empty(s.T(), out.Error)
	assert.NotZero(s.T(), out.ID)
	assert.InDelta(s.T(), 150.0, out.Achievement.CurrentValue, 0.01, "only food-prefixed spend counts")
	assert.InDelta(s.T(), 350.0, out.Achievement.RemainingValue, 0.01)
}

func (s *IntegrationTestSuite) TestCreateAchievement_MoneySaving_CapturesBaselineAndStartsAtZero() {
	ctx := s.Context()
	now := time.Now().UTC()

	_, _, err := money.AddTransactions(ctx, nil, money.AddTransactionsInput{
		Transactions: []money.TransactionInput{
			{Type: "income", AmountOriginal: 1000, Currency: "EUR", AmountEUR: 1000, Account: "Revolut", Category: "salary", Merchant: "Employer", TransactedAt: now.Add(-10 * 24 * time.Hour)},
		},
	})
	require.NoError(s.T(), err)

	_, out, err := achievements.CreateAchievement(ctx, nil, achievements.CreateAchievementInput{
		Name: "Emergency Fund", AchievementType: "money_saving", TargetValue: 500,
	})
	require.NoError(s.T(), err)
	require.Empty(s.T(), out.Error)
	require.NotNil(s.T(), out.Achievement.BaselineBalanceEUR)
	assert.InDelta(s.T(), 1000.0, *out.Achievement.BaselineBalanceEUR, 0.01, "baseline is the balance as of achievement creation")
	assert.InDelta(s.T(), 0.0, out.Achievement.CurrentValue, 0.01, "no saving has happened yet, relative to the baseline")
}

func (s *IntegrationTestSuite) TestCreateAchievement_ExerciseMaxWeight_AlreadyMetIfBackdated() {
	ctx := s.Context()
	exerciseID := s.createExercise(ctx, "Bench Press", "barbell")

	wID := s.createWorkout(ctx, time.Now().Add(-48*time.Hour))
	s.createSet(ctx, wID, exerciseID, 3, 100, time.Now().Add(-48*time.Hour))

	_, out, err := achievements.CreateAchievement(ctx, nil, achievements.CreateAchievementInput{
		Name: "Bench 90kg", AchievementType: "exercise_max_weight", TargetValue: 90, ExerciseID: &exerciseID,
	})
	require.NoError(s.T(), err)
	require.Empty(s.T(), out.Error)
	assert.InDelta(s.T(), 100.0, out.Achievement.CurrentValue, 0.01, "an achievement is already met if the max weight predates it")
	assert.Less(s.T(), out.Achievement.RemainingValue, 0.0, "remaining goes negative once exceeded")
}

func (s *IntegrationTestSuite) TestCreateAchievement_ExerciseTotalVolume_SumsWeightTimesReps() {
	ctx := s.Context()
	exerciseID := s.createExercise(ctx, "Squat", "barbell")

	wID := s.createWorkout(ctx, time.Now().Add(-time.Hour))
	s.createSet(ctx, wID, exerciseID, 5, 100, time.Now().Add(-time.Hour))
	s.createSet(ctx, wID, exerciseID, 5, 100, time.Now().Add(-time.Hour))

	starts := time.Now().Add(-2 * time.Hour)
	_, out, err := achievements.CreateAchievement(ctx, nil, achievements.CreateAchievementInput{
		Name: "5000kg Quarter", AchievementType: "exercise_total_volume", TargetValue: 5000, ExerciseID: &exerciseID, StartsAt: &starts,
	})
	require.NoError(s.T(), err)
	require.Empty(s.T(), out.Error)
	assert.InDelta(s.T(), 1000.0, out.Achievement.CurrentValue, 0.01)
}

func (s *IntegrationTestSuite) TestCreateAchievement_ActivityOccurrenceCount_CountsSinceStartsAt() {
	ctx := s.Context()
	userID := s.UserID()
	now := time.Now().UTC()
	activityID := s.createActivity(ctx, "Meditation", domain.ProgressTypeHabitProgress, now.Add(-30*24*time.Hour))

	starts := now.Add(-10 * 24 * time.Hour)
	// Before the achievement's window — must not count.
	_, err := s.Repo().CreateProgress(ctx, &domain.ActivityPoint{UserID: userID, ActivityID: activityID, Value: 1, ProgressAt: now.Add(-20 * 24 * time.Hour)})
	require.NoError(s.T(), err)
	// Inside the achievement's window — must count.
	_, err = s.Repo().CreateProgress(ctx, &domain.ActivityPoint{UserID: userID, ActivityID: activityID, Value: 1, ProgressAt: now.Add(-5 * 24 * time.Hour)})
	require.NoError(s.T(), err)
	_, err = s.Repo().CreateProgress(ctx, &domain.ActivityPoint{UserID: userID, ActivityID: activityID, Value: 1, ProgressAt: now.Add(-1 * 24 * time.Hour)})
	require.NoError(s.T(), err)

	_, out, err := achievements.CreateAchievement(ctx, nil, achievements.CreateAchievementInput{
		Name: "Meditate 30 Times", AchievementType: "activity_occurrence_count", TargetValue: 30, ActivityID: &activityID, StartsAt: &starts,
	})
	require.NoError(s.T(), err)
	require.Empty(s.T(), out.Error)
	assert.InDelta(s.T(), 2.0, out.Achievement.CurrentValue, 0.01)
}

func (s *IntegrationTestSuite) TestCreateAchievement_ActivityStreakCount_LapsedStreakIsZero() {
	ctx := s.Context()
	userID := s.UserID()
	now := time.Now().UTC()
	activityID := s.createActivity(ctx, "Journaling", domain.ProgressTypeHabitProgress, now.Add(-60*24*time.Hour))

	// Last check-in was 10 days ago — with a 1-day frequency, the streak has
	// already lapsed by the time the achievement is created (today).
	_, err := s.Repo().CreateProgress(ctx, &domain.ActivityPoint{UserID: userID, ActivityID: activityID, Value: 1, ProgressAt: now.Add(-10 * 24 * time.Hour)})
	require.NoError(s.T(), err)

	_, out, err := achievements.CreateAchievement(ctx, nil, achievements.CreateAchievementInput{
		Name: "30-day Streak", AchievementType: "activity_streak_count", TargetValue: 30, ActivityID: &activityID,
	})
	require.NoError(s.T(), err)
	require.Empty(s.T(), out.Error)
	assert.InDelta(s.T(), 0.0, out.Achievement.CurrentValue, 0.01)
}

func (s *IntegrationTestSuite) TestCreateAchievement_ActivityStreakCount_CountsConsecutiveCheckIns() {
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

	_, out, err := achievements.CreateAchievement(ctx, nil, achievements.CreateAchievementInput{
		Name: "30-day Streak", AchievementType: "activity_streak_count", TargetValue: 30, ActivityID: &activityID,
	})
	require.NoError(s.T(), err)
	require.Empty(s.T(), out.Error)
	assert.InDelta(s.T(), 3.0, out.Achievement.CurrentValue, 0.01)
}

func (s *IntegrationTestSuite) TestCreateAchievement_Manual_StartsAtZero() {
	ctx := s.Context()

	_, out, err := achievements.CreateAchievement(ctx, nil, achievements.CreateAchievementInput{Name: "Read 12 Books", AchievementType: "manual", TargetValue: 12})
	require.NoError(s.T(), err)
	require.Empty(s.T(), out.Error)
	assert.Equal(s.T(), 0.0, out.Achievement.CurrentValue)
}

func (s *IntegrationTestSuite) TestCreateAchievement_ValidationErrors() {
	ctx := s.Context()
	category := "food"
	unit := "kg"
	pastEnd := time.Now().Add(-time.Hour)
	badExerciseID := int64(999999)
	badActivityID := int64(999999)

	cases := []struct {
		name  string
		input achievements.CreateAchievementInput
		want  string
	}{
		{"unknown type", achievements.CreateAchievementInput{Name: "x", AchievementType: "not_a_type", TargetValue: 1}, `unknown achievement_type "not_a_type"`},
		{"non-positive target", achievements.CreateAchievementInput{Name: "x", AchievementType: "manual", TargetValue: 0}, "target_value must be greater than 0"},
		{"ends before starts", achievements.CreateAchievementInput{Name: "x", AchievementType: "manual", TargetValue: 1, EndsAt: &pastEnd}, "ends_at must be after starts_at"},
		{"money_spend missing category", achievements.CreateAchievementInput{Name: "x", AchievementType: "money_spend", TargetValue: 1}, "category is required for money_spend achievements"},
		{"money_saving rejects unit", achievements.CreateAchievementInput{Name: "x", AchievementType: "money_saving", TargetValue: 1, Unit: &unit}, "unit is not applicable to money_saving achievements"},
		{"money_saving rejects category", achievements.CreateAchievementInput{Name: "x", AchievementType: "money_saving", TargetValue: 1, Category: &category}, "category is not applicable to money_saving achievements"},
		{"exercise_max_weight missing exercise_id", achievements.CreateAchievementInput{Name: "x", AchievementType: "exercise_max_weight", TargetValue: 1}, "exercise_id is required for exercise_max_weight achievements"},
		{"exercise_max_weight not found", achievements.CreateAchievementInput{Name: "x", AchievementType: "exercise_max_weight", TargetValue: 1, ExerciseID: &badExerciseID}, "exercise not found"},
		{"activity_occurrence_count missing activity_id", achievements.CreateAchievementInput{Name: "x", AchievementType: "activity_occurrence_count", TargetValue: 1}, "activity_id is required for activity_occurrence_count achievements"},
		{"activity_occurrence_count not found", achievements.CreateAchievementInput{Name: "x", AchievementType: "activity_occurrence_count", TargetValue: 1, ActivityID: &badActivityID}, "activity not found"},
	}

	for _, tc := range cases {
		s.T().Run(tc.name, func(t *testing.T) {
			_, out, err := achievements.CreateAchievement(ctx, nil, tc.input)
			require.NoError(t, err)
			assert.Equal(t, tc.want, out.Error)
		})
	}
}

// --- update_achievement ---------------------------------------------------------

func (s *IntegrationTestSuite) TestUpdateAchievement_NameTargetAndDeadline() {
	ctx := s.Context()

	_, created, err := achievements.CreateAchievement(ctx, nil, achievements.CreateAchievementInput{Name: "Read 12 Books", AchievementType: "manual", TargetValue: 12})
	require.NoError(s.T(), err)

	newName := "Read 20 Books"
	newTarget := 20.0
	newEnd := time.Now().Add(90 * 24 * time.Hour)
	_, out, err := achievements.UpdateAchievement(ctx, nil, achievements.UpdateAchievementInput{
		AchievementID: created.ID, Name: &newName, TargetValue: &newTarget, EndsAt: &newEnd,
	})
	require.NoError(s.T(), err)
	require.Empty(s.T(), out.Error)
	assert.Equal(s.T(), "Read 20 Books", out.Achievement.Name)
	assert.Equal(s.T(), 20.0, out.Achievement.TargetValue)
	require.NotNil(s.T(), out.Achievement.EndsAt)
	assert.WithinDuration(s.T(), newEnd, *out.Achievement.EndsAt, time.Second)
}

func (s *IntegrationTestSuite) TestUpdateAchievement_ClearEndsAt() {
	ctx := s.Context()
	end := time.Now().Add(30 * 24 * time.Hour)

	_, created, err := achievements.CreateAchievement(ctx, nil, achievements.CreateAchievementInput{Name: "x", AchievementType: "manual", TargetValue: 1, EndsAt: &end})
	require.NoError(s.T(), err)
	require.NotNil(s.T(), created.Achievement.EndsAt)

	_, out, err := achievements.UpdateAchievement(ctx, nil, achievements.UpdateAchievementInput{AchievementID: created.ID, ClearEndsAt: true})
	require.NoError(s.T(), err)
	require.Empty(s.T(), out.Error)
	assert.Nil(s.T(), out.Achievement.EndsAt)
}

func (s *IntegrationTestSuite) TestUpdateAchievement_CurrentValue_OnlyManualCanBeCorrected() {
	ctx := s.Context()
	exerciseID := s.createExercise(ctx, "Deadlift", "barbell")

	_, manualAchievement, err := achievements.CreateAchievement(ctx, nil, achievements.CreateAchievementInput{Name: "Read 12 Books", AchievementType: "manual", TargetValue: 12})
	require.NoError(s.T(), err)
	newValue := 5.0
	_, out, err := achievements.UpdateAchievement(ctx, nil, achievements.UpdateAchievementInput{AchievementID: manualAchievement.ID, CurrentValue: &newValue})
	require.NoError(s.T(), err)
	require.Empty(s.T(), out.Error)
	assert.Equal(s.T(), 5.0, out.Achievement.CurrentValue)

	_, exerciseAchievement, err := achievements.CreateAchievement(ctx, nil, achievements.CreateAchievementInput{Name: "Deadlift 150kg", AchievementType: "exercise_max_weight", TargetValue: 150, ExerciseID: &exerciseID})
	require.NoError(s.T(), err)
	_, rejected, err := achievements.UpdateAchievement(ctx, nil, achievements.UpdateAchievementInput{AchievementID: exerciseAchievement.ID, CurrentValue: &newValue})
	require.NoError(s.T(), err)
	assert.Equal(s.T(), "current_value can only be corrected on manual achievements", rejected.Error)
}

func (s *IntegrationTestSuite) TestUpdateAchievement_Category_OnlyMoneySpend() {
	ctx := s.Context()
	category := "food"

	_, spendAchievement, err := achievements.CreateAchievement(ctx, nil, achievements.CreateAchievementInput{Name: "Food", AchievementType: "money_spend", TargetValue: 100, Category: &category})
	require.NoError(s.T(), err)
	newCategory := "groceries"
	_, out, err := achievements.UpdateAchievement(ctx, nil, achievements.UpdateAchievementInput{AchievementID: spendAchievement.ID, Category: &newCategory})
	require.NoError(s.T(), err)
	require.Empty(s.T(), out.Error)
	require.NotNil(s.T(), out.Achievement.Category)
	assert.Equal(s.T(), "groceries", *out.Achievement.Category)

	_, manualAchievement, err := achievements.CreateAchievement(ctx, nil, achievements.CreateAchievementInput{Name: "x", AchievementType: "manual", TargetValue: 1})
	require.NoError(s.T(), err)
	_, rejected, err := achievements.UpdateAchievement(ctx, nil, achievements.UpdateAchievementInput{AchievementID: manualAchievement.ID, Category: &newCategory})
	require.NoError(s.T(), err)
	assert.Equal(s.T(), "category can only be set on money_spend achievements", rejected.Error)
}

func (s *IntegrationTestSuite) TestUpdateAchievement_NotFound() {
	ctx := s.Context()
	newName := "x"
	_, out, err := achievements.UpdateAchievement(ctx, nil, achievements.UpdateAchievementInput{AchievementID: 999999, Name: &newName})
	require.NoError(s.T(), err)
	assert.Equal(s.T(), "achievement not found", out.Error)
}

// --- refresh_achievements ---------------------------------------------------------

func (s *IntegrationTestSuite) TestRefreshAchievements_SingleAchievement_RecomputesOnDemandOnly() {
	ctx := s.Context()
	now := time.Now().UTC()

	category := "food"
	starts := now.Add(-time.Hour)
	_, created, err := achievements.CreateAchievement(ctx, nil, achievements.CreateAchievementInput{Name: "Food", AchievementType: "money_spend", TargetValue: 500, Category: &category, StartsAt: &starts})
	require.NoError(s.T(), err)
	assert.Equal(s.T(), 0.0, created.Achievement.CurrentValue, "no spend yet at creation time")

	_, _, err = money.AddTransactions(ctx, nil, money.AddTransactionsInput{
		Transactions: []money.TransactionInput{
			{Type: "expense", AmountOriginal: 75, Currency: "EUR", AmountEUR: 75, Account: "Revolut", Category: "food/cafe", Merchant: "Starbucks", TransactedAt: now},
		},
	})
	require.NoError(s.T(), err)

	// get_achievement_progress must still show the stale cached value — reads never
	// recompute (see achievements-spec.md Best Practices).
	_, progressOut, err := achievements.GetAchievementProgress(ctx, nil, achievements.GetAchievementProgressInput{})
	require.NoError(s.T(), err)
	require.Len(s.T(), progressOut.Achievements, 1)
	assert.Equal(s.T(), 0.0, progressOut.Achievements[0].CurrentValue, "current_value is cached, not live")

	achievementID := created.ID
	_, refreshOut, err := achievements.RefreshAchievements(ctx, nil, achievements.RefreshAchievementsInput{AchievementID: &achievementID})
	require.NoError(s.T(), err)
	require.Empty(s.T(), refreshOut.Error)
	require.NotNil(s.T(), refreshOut.Achievement)
	assert.InDelta(s.T(), 75.0, refreshOut.Achievement.CurrentValue, 0.01)
	assert.Equal(s.T(), 1, refreshOut.Refreshed)
}

func (s *IntegrationTestSuite) TestRefreshAchievements_AllActive_SkipsManual() {
	ctx := s.Context()

	_, spendAchievement, err := achievements.CreateAchievement(ctx, nil, achievements.CreateAchievementInput{Name: "Spend", AchievementType: "money_spend", TargetValue: 100, Category: strPtr("food")})
	require.NoError(s.T(), err)
	_, manualAchievement, err := achievements.CreateAchievement(ctx, nil, achievements.CreateAchievementInput{Name: "Manual", AchievementType: "manual", TargetValue: 10})
	require.NoError(s.T(), err)

	_, out, err := achievements.RefreshAchievements(ctx, nil, achievements.RefreshAchievementsInput{})
	require.NoError(s.T(), err)
	require.Empty(s.T(), out.Error)
	assert.Equal(s.T(), 1, out.Refreshed, "manual must be skipped")
	assert.NotZero(s.T(), spendAchievement.ID)
	assert.NotZero(s.T(), manualAchievement.ID)
}

func (s *IntegrationTestSuite) TestRefreshAchievements_ManualAchievement_Rejected() {
	ctx := s.Context()
	_, created, err := achievements.CreateAchievement(ctx, nil, achievements.CreateAchievementInput{Name: "Manual", AchievementType: "manual", TargetValue: 10})
	require.NoError(s.T(), err)

	achievementID := created.ID
	_, out, err := achievements.RefreshAchievements(ctx, nil, achievements.RefreshAchievementsInput{AchievementID: &achievementID})
	require.NoError(s.T(), err)
	assert.Equal(s.T(), "manual achievements have no derived progress to refresh; use log_achievement_progress or update_achievement instead", out.Error)
}

// --- get_achievement_progress -----------------------------------------------------

func (s *IntegrationTestSuite) TestGetAchievementProgress_OnlyActiveAchievements_WithRemainingValue() {
	ctx := s.Context()
	now := time.Now().UTC()

	pastEnd := now.Add(-24 * time.Hour)
	pastStart := now.Add(-48 * time.Hour)
	_, _, err := achievements.CreateAchievement(ctx, nil, achievements.CreateAchievementInput{Name: "Old achievement", AchievementType: "manual", TargetValue: 10, StartsAt: &pastStart, EndsAt: &pastEnd})
	require.NoError(s.T(), err)

	_, active, err := achievements.CreateAchievement(ctx, nil, achievements.CreateAchievementInput{Name: "Active achievement", AchievementType: "manual", TargetValue: 10})
	require.NoError(s.T(), err)

	delta := 3.0
	_, _, err = achievements.LogAchievementProgress(ctx, nil, achievements.LogAchievementProgressInput{AchievementID: active.ID, Delta: &delta})
	require.NoError(s.T(), err)

	_, out, err := achievements.GetAchievementProgress(ctx, nil, achievements.GetAchievementProgressInput{})
	require.NoError(s.T(), err)
	require.Len(s.T(), out.Achievements, 1, "the finished achievement must not show as active")
	assert.Equal(s.T(), "Active achievement", out.Achievements[0].Name)
	assert.Equal(s.T(), 3.0, out.Achievements[0].CurrentValue)
	assert.Equal(s.T(), 7.0, out.Achievements[0].RemainingValue)
}

// --- log_achievement_progress -------------------------------------------------------

func (s *IntegrationTestSuite) TestLogAchievementProgress_DefaultAndCustomDelta() {
	ctx := s.Context()
	_, created, err := achievements.CreateAchievement(ctx, nil, achievements.CreateAchievementInput{Name: "Read 12 Books", AchievementType: "manual", TargetValue: 12})
	require.NoError(s.T(), err)

	_, out1, err := achievements.LogAchievementProgress(ctx, nil, achievements.LogAchievementProgressInput{AchievementID: created.ID})
	require.NoError(s.T(), err)
	require.Empty(s.T(), out1.Error)
	assert.Equal(s.T(), 1.0, out1.Achievement.CurrentValue)

	delta := 2.0
	_, out2, err := achievements.LogAchievementProgress(ctx, nil, achievements.LogAchievementProgressInput{AchievementID: created.ID, Delta: &delta})
	require.NoError(s.T(), err)
	assert.Equal(s.T(), 3.0, out2.Achievement.CurrentValue)
}

func (s *IntegrationTestSuite) TestLogAchievementProgress_NonManualAchievementRejected() {
	ctx := s.Context()
	_, created, err := achievements.CreateAchievement(ctx, nil, achievements.CreateAchievementInput{Name: "Spend", AchievementType: "money_spend", TargetValue: 100, Category: strPtr("food")})
	require.NoError(s.T(), err)

	_, out, err := achievements.LogAchievementProgress(ctx, nil, achievements.LogAchievementProgressInput{AchievementID: created.ID})
	require.NoError(s.T(), err)
	assert.Equal(s.T(), "log_achievement_progress only applies to manual achievements", out.Error)
}

func strPtr(s string) *string { return &s }
