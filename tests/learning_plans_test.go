package tests

// Covers create_learning_plan, edit_learning_plan, list_learning_plans and
// get_learning_plan from docs/functions/learning-spec.md, plus the seed
// helpers shared by every learning test file.

import (
	"context"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"personal/action/learning"
	"personal/domain"
	"personal/gateways"
	"personal/util"
)

var nicosia = func() *time.Location {
	loc, err := time.LoadLocation("Asia/Nicosia")
	if err != nil {
		panic(err)
	}
	return loc
}()

// learningDay is midnight Asia/Nicosia, offset days from today.
func learningDay(offset int) time.Time {
	y, m, d := time.Now().In(nicosia).Date()
	return time.Date(y, m, d+offset, 0, 0, 0, 0, nicosia)
}

// learningDate is learningDay as YYYY-MM-DD.
func learningDate(offset int) string {
	return learningDay(offset).Format("2006-01-02")
}

func (s *IntegrationTestSuite) seedLearningPlan(ctx context.Context, name string, status domain.LearningPlanStatus) int64 {
	id, err := s.Repo().CreateLearningPlan(ctx, &domain.LearningPlan{
		UserID: gateways.UserIDFromContext(ctx),
		Name:   name,
		Status: status,
	})
	require.NoError(s.T(), err)
	return id
}

func (s *IntegrationTestSuite) seedSkill(ctx context.Context, planID int64, name string) int64 {
	id, err := s.Repo().CreateSkill(ctx, &domain.Skill{
		UserID:         gateways.UserIDFromContext(ctx),
		LearningPlanID: planID,
		Name:           name,
	})
	require.NoError(s.T(), err)
	return id
}

func (s *IntegrationTestSuite) seedExercise(ctx context.Context, skillID int64, content string, plannedFor time.Time) int64 {
	ids, err := s.Repo().CreateLearningExercises(ctx, []domain.LearningExercise{{
		UserID:     gateways.UserIDFromContext(ctx),
		SkillID:    skillID,
		Content:    content,
		PlannedFor: plannedFor,
	}})
	require.NoError(s.T(), err)
	return ids[0]
}

func (s *IntegrationTestSuite) seedAnswer(ctx context.Context, exerciseID int64, answer string, at time.Time) {
	require.NoError(s.T(), s.Repo().SubmitLearningExerciseAnswer(ctx, exerciseID, gateways.UserIDFromContext(ctx), answer, at))
}

func (s *IntegrationTestSuite) seedExerciseScore(ctx context.Context, exerciseID int64, score int, at time.Time) {
	require.NoError(s.T(), s.Repo().ReviewLearningExercises(ctx, gateways.UserIDFromContext(ctx), []domain.LearningExerciseReview{
		{ExerciseID: exerciseID, Score: score, ReviewedAt: at},
	}))
}

// seedReviewedExercise creates an exercise answered and scored at the given time.
func (s *IntegrationTestSuite) seedReviewedExercise(ctx context.Context, skillID int64, score int, at time.Time) int64 {
	id := s.seedExercise(ctx, skillID, "exercise", learningDay(-30))
	s.seedAnswer(ctx, id, "answer", at)
	s.seedExerciseScore(ctx, id, score, at)
	return id
}

func (s *IntegrationTestSuite) seedWord(ctx context.Context, planID int64, term, translation string) int64 {
	return s.seedWordAt(ctx, planID, term, translation, time.Time{})
}

func (s *IntegrationTestSuite) seedWordAt(ctx context.Context, planID int64, term, translation string, createdAt time.Time) int64 {
	ids, err := s.Repo().CreateVocabulary(ctx, []domain.VocabularyWord{{
		UserID:         gateways.UserIDFromContext(ctx),
		LearningPlanID: planID,
		Term:           term,
		Translation:    translation,
		CreatedAt:      createdAt,
	}})
	require.NoError(s.T(), err)
	return ids[0]
}

func (s *IntegrationTestSuite) seedPendingReview(ctx context.Context, wordID int64, plannedFor time.Time) int64 {
	ids, err := s.Repo().CreateVocabularyReviews(ctx, []domain.VocabularyReview{{
		UserID:       gateways.UserIDFromContext(ctx),
		VocabularyID: wordID,
		PlannedFor:   plannedFor,
	}})
	require.NoError(s.T(), err)
	return ids[0]
}

func (s *IntegrationTestSuite) seedScoredReview(ctx context.Context, wordID int64, score int, reviewedAt time.Time) int64 {
	ids, err := s.Repo().CreateVocabularyReviews(ctx, []domain.VocabularyReview{{
		UserID:       gateways.UserIDFromContext(ctx),
		VocabularyID: wordID,
		PlannedFor:   reviewedAt,
		Score:        util.Ptr(score),
		ReviewedAt:   &reviewedAt,
	}})
	require.NoError(s.T(), err)
	return ids[0]
}

func (s *IntegrationTestSuite) learningWord(ctx context.Context, wordID int64) domain.VocabularyWord {
	words, err := s.Repo().ListVocabulary(ctx, domain.VocabularyFilter{UserID: gateways.UserIDFromContext(ctx), IDs: []int64{wordID}})
	require.NoError(s.T(), err)
	require.Len(s.T(), words, 1)
	w := words[0]
	w.MasteryState = domain.ComputeMastery(w.Scores, w.LastReviewedAt, time.Now())
	return w
}

func (s *IntegrationTestSuite) wordReviews(ctx context.Context, wordID int64) []domain.VocabularyReview {
	reviews, err := s.Repo().ListVocabularyReviews(ctx, domain.VocabularyReviewFilter{UserID: gateways.UserIDFromContext(ctx), VocabularyID: &wordID})
	require.NoError(s.T(), err)
	return reviews
}

func planResultIDs(plans []learning.PlanSummaryResult) []int64 {
	ids := make([]int64, len(plans))
	for i, p := range plans {
		ids[i] = p.ID
	}
	return ids
}

// --- create_learning_plan ---------------------------------------------------

func (s *IntegrationTestSuite) TestCreateLearningPlan() {
	ctx := s.Context()

	s.Run("created as active", func() {
		_, out, err := learning.CreateLearningPlan(ctx, nil, learning.CreateLearningPlanInput{Name: "Greek A1", Description: util.Ptr("Order food in Greek")})
		require.NoError(s.T(), err)
		assert.Equal(s.T(), "active", out.Plan.Status)
		assert.Equal(s.T(), "Order food in Greek", out.Plan.Description)

		plan, err := s.Repo().GetLearningPlan(ctx, out.Plan.ID, gateways.UserIDFromContext(ctx))
		require.NoError(s.T(), err)
		assert.Equal(s.T(), "Greek A1", plan.Name)
		assert.Equal(s.T(), domain.LearningPlanActive, plan.Status)
	})

	s.Run("empty name fails", func() {
		_, _, err := learning.CreateLearningPlan(ctx, nil, learning.CreateLearningPlanInput{Name: "  "})
		require.ErrorContains(s.T(), err, "name is required")
	})
}

// --- edit_learning_plan -----------------------------------------------------

func (s *IntegrationTestSuite) TestEditLearningPlan() {
	ctx := s.Context()
	userID := gateways.UserIDFromContext(ctx)
	planID := s.seedLearningPlan(ctx, "Greek A1", domain.LearningPlanActive)

	otherCtx := s.otherUserContext(ctx)
	otherPlan := s.seedLearningPlan(otherCtx, "Other", domain.LearningPlanActive)

	s.Run("name alone", func() {
		_, out, err := learning.EditLearningPlan(ctx, nil, learning.EditLearningPlanInput{LearningPlanID: planID, Name: util.Ptr("Greek A2")})
		require.NoError(s.T(), err)
		assert.Equal(s.T(), "Greek A2", out.Plan.Name)
		assert.Equal(s.T(), "active", out.Plan.Status)
	})

	s.Run("description alone, then cleared", func() {
		_, out, err := learning.EditLearningPlan(ctx, nil, learning.EditLearningPlanInput{LearningPlanID: planID, Description: util.Ptr("Read menus")})
		require.NoError(s.T(), err)
		assert.Equal(s.T(), "Read menus", out.Plan.Description)
		assert.Equal(s.T(), "Greek A2", out.Plan.Name)

		_, _, err = learning.EditLearningPlan(ctx, nil, learning.EditLearningPlanInput{LearningPlanID: planID, Description: util.Ptr("")})
		require.NoError(s.T(), err)
		plan, err := s.Repo().GetLearningPlan(ctx, planID, userID)
		require.NoError(s.T(), err)
		assert.Nil(s.T(), plan.Description)
	})

	s.Run("status change", func() {
		_, out, err := learning.EditLearningPlan(ctx, nil, learning.EditLearningPlanInput{LearningPlanID: planID, Status: util.Ptr("completed")})
		require.NoError(s.T(), err)
		assert.Equal(s.T(), "completed", out.Plan.Status)

		_, out, err = learning.EditLearningPlan(ctx, nil, learning.EditLearningPlanInput{LearningPlanID: planID, Status: util.Ptr("paused")})
		require.NoError(s.T(), err)
		assert.Equal(s.T(), "paused", out.Plan.Status)
	})

	failures := []struct {
		name    string
		input   learning.EditLearningPlanInput
		wantErr string
	}{
		{name: "no fields", input: learning.EditLearningPlanInput{LearningPlanID: planID}, wantErr: "at least one"},
		{name: "invalid status", input: learning.EditLearningPlanInput{LearningPlanID: planID, Status: util.Ptr("done")}, wantErr: "invalid status"},
		{name: "empty name", input: learning.EditLearningPlanInput{LearningPlanID: planID, Name: util.Ptr(" ")}, wantErr: "name cannot be empty"},
		{name: "another user's plan", input: learning.EditLearningPlanInput{LearningPlanID: otherPlan, Name: util.Ptr("x")}, wantErr: "not found"},
	}
	for _, tt := range failures {
		s.Run(tt.name, func() {
			_, _, err := learning.EditLearningPlan(ctx, nil, tt.input)
			require.ErrorContains(s.T(), err, tt.wantErr)
		})
	}
}

// --- list_learning_plans ----------------------------------------------------

func (s *IntegrationTestSuite) TestListLearningPlans() {
	ctx := s.Context()

	active := s.seedLearningPlan(ctx, "Greek A1", domain.LearningPlanActive)
	paused := s.seedLearningPlan(ctx, "Kubernetes", domain.LearningPlanPaused)
	completed := s.seedLearningPlan(ctx, "Spanish", domain.LearningPlanCompleted)

	skill := s.seedSkill(ctx, active, "present tense")
	s.seedSkill(ctx, active, "articles")
	s.seedExercise(ctx, skill, "due", learningDay(0))
	s.seedExercise(ctx, skill, "future", learningDay(2))
	answered := s.seedExercise(ctx, skill, "answered", learningDay(-1))
	s.seedAnswer(ctx, answered, "answer", time.Now())

	word := s.seedWord(ctx, active, "νερό", "water")
	s.seedWord(ctx, active, "ψωμί", "bread")
	s.seedPendingReview(ctx, word, learningDay(0))

	otherCtx := s.otherUserContext(ctx)
	s.seedLearningPlan(otherCtx, "Other", domain.LearningPlanActive)

	s.Run("default hides completed", func() {
		_, out, err := learning.ListLearningPlans(ctx, nil, learning.ListLearningPlansInput{})
		require.NoError(s.T(), err)
		assert.Equal(s.T(), []int64{active, paused}, planResultIDs(out.Plans))

		greek := out.Plans[0]
		assert.Equal(s.T(), 2, greek.SkillCount)
		assert.Equal(s.T(), learning.ExerciseStatsResult{Planned: 2, Due: 1, AwaitingReview: 1}, greek.ExerciseStats)
		assert.Equal(s.T(), learning.VocabularyStatsResult{Total: 2, Due: 1, Unscheduled: 1}, greek.VocabularyStats)

		assert.Equal(s.T(), learning.ExerciseStatsResult{}, out.Plans[1].ExerciseStats)
		assert.Equal(s.T(), learning.VocabularyStatsResult{}, out.Plans[1].VocabularyStats)
	})

	s.Run("statuses filter", func() {
		_, out, err := learning.ListLearningPlans(ctx, nil, learning.ListLearningPlansInput{Statuses: []string{"completed", "paused"}})
		require.NoError(s.T(), err)
		assert.Equal(s.T(), []int64{paused, completed}, planResultIDs(out.Plans))
	})

	s.Run("invalid status fails", func() {
		_, _, err := learning.ListLearningPlans(ctx, nil, learning.ListLearningPlansInput{Statuses: []string{"done"}})
		require.ErrorContains(s.T(), err, "invalid status")
	})
}

// --- get_learning_plan ------------------------------------------------------

func (s *IntegrationTestSuite) TestGetLearningPlan() {
	ctx := s.Context()
	now := time.Now()

	planID := s.seedLearningPlan(ctx, "Greek A1", domain.LearningPlanActive)
	tense := s.seedSkill(ctx, planID, "present tense")
	articles := s.seedSkill(ctx, planID, "articles")

	for i := range 4 {
		s.seedReviewedExercise(ctx, tense, 5, now.Add(time.Duration(i-4)*time.Minute))
	}
	s.seedExercise(ctx, tense, "planned", learningDay(0))
	awaiting := s.seedExercise(ctx, tense, "awaiting", learningDay(-1))
	s.seedAnswer(ctx, awaiting, "answer", now)

	mastered := s.seedWord(ctx, planID, "νερό", "water")
	for i := range 5 {
		s.seedScoredReview(ctx, mastered, 5, now.Add(time.Duration(i-5)*time.Minute))
	}
	scheduled := s.seedWord(ctx, planID, "ψωμί", "bread")
	s.seedPendingReview(ctx, scheduled, learningDay(-1))
	s.seedPendingReview(ctx, scheduled, learningDay(3))

	emptyPlan := s.seedLearningPlan(ctx, "Kubernetes", domain.LearningPlanActive)

	otherCtx := s.otherUserContext(ctx)
	otherPlan := s.seedLearningPlan(otherCtx, "Other", domain.LearningPlanActive)

	s.Run("skills and stats", func() {
		_, out, err := learning.GetLearningPlan(ctx, nil, learning.GetLearningPlanInput{LearningPlanID: planID})
		require.NoError(s.T(), err)
		require.Len(s.T(), out.Plan.Skills, 2)

		t := out.Plan.Skills[0]
		assert.Equal(s.T(), tense, t.ID)
		assert.Equal(s.T(), 80, t.Mastery)
		assert.Equal(s.T(), 5, t.AnsweredCount)
		assert.Equal(s.T(), 1, t.PlannedCount)
		assert.Equal(s.T(), 5, *t.LastScore)
		assert.NotEmpty(s.T(), t.LastPracticedAt)
		assert.Equal(s.T(), 0, t.Uncertainty)

		a := out.Plan.Skills[1]
		assert.Equal(s.T(), articles, a.ID)
		assert.Equal(s.T(), 0, a.Mastery)
		assert.Nil(s.T(), a.LastScore)
		assert.Empty(s.T(), a.LastPracticedAt)

		assert.Equal(s.T(), learning.ExerciseStatsResult{Planned: 1, Due: 1, AwaitingReview: 1}, out.Plan.ExerciseStats)
		// νερό at 100%, ψωμί at 0%: one mastered, average 50; ψωμί's missed review is due
		assert.Equal(s.T(), learning.VocabularyStatsResult{Total: 2, Due: 1, Unscheduled: 1, Mastered: 1, AverageMastery: 50}, out.Plan.VocabularyStats)
	})

	s.Run("plan without words or skills", func() {
		_, out, err := learning.GetLearningPlan(ctx, nil, learning.GetLearningPlanInput{LearningPlanID: emptyPlan})
		require.NoError(s.T(), err)
		assert.Empty(s.T(), out.Plan.Skills)
		assert.Equal(s.T(), learning.VocabularyStatsResult{}, out.Plan.VocabularyStats)
	})

	s.Run("another user's plan fails", func() {
		_, _, err := learning.GetLearningPlan(ctx, nil, learning.GetLearningPlanInput{LearningPlanID: otherPlan})
		require.ErrorContains(s.T(), err, "not found")
	})
}
