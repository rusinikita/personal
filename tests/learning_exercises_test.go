package tests

// Covers create_learning_exercises, edit_learning_exercise,
// submit_learning_exercise, review_learning_exercises,
// delete_learning_exercise and list_learning_exercises from
// docs/functions/learning-spec.md.

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

func exerciseResultIDs(results []learning.ExerciseResult) []int64 {
	ids := make([]int64, len(results))
	for i, r := range results {
		ids[i] = r.ID
	}
	return ids
}

func (s *IntegrationTestSuite) countPlanExercises(ctx context.Context, planID int64) int {
	n, err := s.Repo().CountLearningExercises(ctx, domain.LearningExerciseFilter{UserID: gateways.UserIDFromContext(ctx), LearningPlanID: &planID})
	require.NoError(s.T(), err)
	return n
}

func (s *IntegrationTestSuite) TestCreateLearningExercises() {
	ctx := s.Context()
	userID := gateways.UserIDFromContext(ctx)
	planID := s.seedLearningPlan(ctx, "Greek A1", domain.LearningPlanActive)
	skill := s.seedSkill(ctx, planID, "present tense")

	otherCtx := s.otherUserContext(ctx)
	foreignSkill := s.seedSkill(otherCtx, s.seedLearningPlan(otherCtx, "Other", domain.LearningPlanActive), "x")

	s.Run("several saved as planned", func() {
		content := "1. Εγώ ___ (είμαι)\n2. Εσύ ___ (έχω)\n"
		_, out, err := learning.CreateLearningExercises(ctx, nil, learning.CreateLearningExercisesInput{Exercises: []learning.ExerciseInput{
			{SkillID: skill, Content: content, Kind: util.Ptr("put into the correct form"), PlannedFor: learningDate(0)},
			{SkillID: skill, Content: "Translate: I have water", PlannedFor: learningDate(2)},
		}})
		require.NoError(s.T(), err)
		require.Len(s.T(), out.Exercises, 2)

		first, err := s.Repo().GetLearningExercise(ctx, out.Exercises[0].ID, userID)
		require.NoError(s.T(), err)
		assert.Equal(s.T(), content, first.Content)
		assert.Equal(s.T(), "put into the correct form", *first.Kind)
		assert.True(s.T(), learningDay(0).Equal(first.PlannedFor), "planned day is midnight Asia/Nicosia")
		assert.Equal(s.T(), domain.LearningExercisePlanned, first.State())
		assert.Equal(s.T(), "planned", out.Exercises[0].State)
		assert.Equal(s.T(), "present tense", out.Exercises[0].SkillName)

		second, err := s.Repo().GetLearningExercise(ctx, out.Exercises[1].ID, userID)
		require.NoError(s.T(), err)
		assert.Nil(s.T(), second.Kind)
	})

	valid := learning.ExerciseInput{SkillID: skill, Content: "ok", PlannedFor: learningDate(1)}
	tooMany := make([]learning.ExerciseInput, 51)
	for i := range tooMany {
		tooMany[i] = valid
	}
	failures := []struct {
		name      string
		exercises []learning.ExerciseInput
		wantErr   string
	}{
		{name: "empty content", exercises: []learning.ExerciseInput{valid, {SkillID: skill, Content: " ", PlannedFor: learningDate(1)}}, wantErr: "content is required"},
		{name: "past day", exercises: []learning.ExerciseInput{valid, {SkillID: skill, Content: "x", PlannedFor: learningDate(-1)}}, wantErr: "in the past"},
		{name: "no entries", exercises: nil, wantErr: "1-50"},
		{name: "more than 50", exercises: tooMany, wantErr: "1-50"},
		{name: "another user's skill", exercises: []learning.ExerciseInput{valid, {SkillID: foreignSkill, Content: "x", PlannedFor: learningDate(1)}}, wantErr: "not found"},
	}
	for _, tt := range failures {
		s.Run(tt.name, func() {
			before := s.countPlanExercises(ctx, planID)
			_, _, err := learning.CreateLearningExercises(ctx, nil, learning.CreateLearningExercisesInput{Exercises: tt.exercises})
			require.ErrorContains(s.T(), err, tt.wantErr)
			assert.Equal(s.T(), before, s.countPlanExercises(ctx, planID))
		})
	}
}

func (s *IntegrationTestSuite) TestEditLearningExercise() {
	ctx := s.Context()
	userID := gateways.UserIDFromContext(ctx)
	planID := s.seedLearningPlan(ctx, "Greek A1", domain.LearningPlanActive)
	skill := s.seedSkill(ctx, planID, "present tense")
	exercise := s.seedExercise(ctx, skill, "original", learningDay(-2))
	answered := s.seedExercise(ctx, skill, "answered", learningDay(0))
	s.seedAnswer(ctx, answered, "a", time.Now())

	otherCtx := s.otherUserContext(ctx)
	foreign := s.seedExercise(otherCtx, s.seedSkill(otherCtx, s.seedLearningPlan(otherCtx, "Other", domain.LearningPlanActive), "x"), "x", learningDay(0))

	s.Run("kind alone", func() {
		_, out, err := learning.EditLearningExercise(ctx, nil, learning.EditLearningExerciseInput{ExerciseID: exercise, Kind: util.Ptr("choose the ending")})
		require.NoError(s.T(), err)
		assert.Equal(s.T(), "choose the ending", out.Exercise.Kind)
		assert.Equal(s.T(), "original", out.Exercise.Content)
	})

	s.Run("content alone", func() {
		_, out, err := learning.EditLearningExercise(ctx, nil, learning.EditLearningExerciseInput{ExerciseID: exercise, Content: util.Ptr("new text")})
		require.NoError(s.T(), err)
		assert.Equal(s.T(), "new text", out.Exercise.Content)
		assert.Equal(s.T(), "choose the ending", out.Exercise.Kind)
	})

	s.Run("moves a missed exercise to a later day", func() {
		_, out, err := learning.EditLearningExercise(ctx, nil, learning.EditLearningExerciseInput{ExerciseID: exercise, PlannedFor: util.Ptr(learningDate(3))})
		require.NoError(s.T(), err)
		assert.Equal(s.T(), learningDate(3), out.Exercise.PlannedFor)

		stored, err := s.Repo().GetLearningExercise(ctx, exercise, userID)
		require.NoError(s.T(), err)
		assert.True(s.T(), learningDay(3).Equal(stored.PlannedFor))
	})

	failures := []struct {
		name    string
		input   learning.EditLearningExerciseInput
		wantErr string
	}{
		{name: "no fields", input: learning.EditLearningExerciseInput{ExerciseID: exercise}, wantErr: "at least one"},
		{name: "answered exercise", input: learning.EditLearningExerciseInput{ExerciseID: answered, Content: util.Ptr("x")}, wantErr: "already answered"},
		{name: "another user's exercise", input: learning.EditLearningExerciseInput{ExerciseID: foreign, Content: util.Ptr("x")}, wantErr: "not found"},
	}
	for _, tt := range failures {
		s.Run(tt.name, func() {
			_, _, err := learning.EditLearningExercise(ctx, nil, tt.input)
			require.ErrorContains(s.T(), err, tt.wantErr)
		})
	}
}

func (s *IntegrationTestSuite) TestSubmitLearningExercise() {
	ctx := s.Context()
	userID := gateways.UserIDFromContext(ctx)
	planID := s.seedLearningPlan(ctx, "Greek A1", domain.LearningPlanActive)
	skill := s.seedSkill(ctx, planID, "present tense")
	exercise := s.seedExercise(ctx, skill, "x", learningDay(0))

	otherCtx := s.otherUserContext(ctx)
	foreign := s.seedExercise(otherCtx, s.seedSkill(otherCtx, s.seedLearningPlan(otherCtx, "Other", domain.LearningPlanActive), "x"), "x", learningDay(0))

	answer := "1. Εγω ειμαι\n2. Εσυ εχις"

	s.Run("empty answer fails", func() {
		_, _, err := learning.SubmitLearningExercise(ctx, nil, learning.SubmitLearningExerciseInput{ExerciseID: exercise, Answer: "  "})
		require.ErrorContains(s.T(), err, "answer is required")
	})

	s.Run("answer stored word for word", func() {
		_, out, err := learning.SubmitLearningExercise(ctx, nil, learning.SubmitLearningExerciseInput{ExerciseID: exercise, Answer: answer})
		require.NoError(s.T(), err)
		assert.Equal(s.T(), "answered", out.Exercise.State)

		stored, err := s.Repo().GetLearningExercise(ctx, exercise, userID)
		require.NoError(s.T(), err)
		assert.Equal(s.T(), answer, *stored.Answer)
		require.NotNil(s.T(), stored.AnsweredAt)
		assert.WithinDuration(s.T(), time.Now(), *stored.AnsweredAt, time.Minute)
	})

	s.Run("second answer fails and keeps the first", func() {
		_, _, err := learning.SubmitLearningExercise(ctx, nil, learning.SubmitLearningExerciseInput{ExerciseID: exercise, Answer: "other"})
		require.ErrorContains(s.T(), err, "already has an answer")

		stored, err := s.Repo().GetLearningExercise(ctx, exercise, userID)
		require.NoError(s.T(), err)
		assert.Equal(s.T(), answer, *stored.Answer)
	})

	s.Run("another user's exercise fails", func() {
		_, _, err := learning.SubmitLearningExercise(ctx, nil, learning.SubmitLearningExerciseInput{ExerciseID: foreign, Answer: "x"})
		require.ErrorContains(s.T(), err, "not found")
	})
}

func (s *IntegrationTestSuite) TestReviewLearningExercises() {
	ctx := s.Context()
	userID := gateways.UserIDFromContext(ctx)
	planID := s.seedLearningPlan(ctx, "Greek A1", domain.LearningPlanActive)
	skill := s.seedSkill(ctx, planID, "present tense")

	answeredExercise := func() int64 {
		id := s.seedExercise(ctx, skill, "x", learningDay(0))
		s.seedAnswer(ctx, id, "y", time.Now())
		return id
	}
	first := answeredExercise()
	second := answeredExercise()
	unanswered := s.seedExercise(ctx, skill, "x", learningDay(0))

	otherCtx := s.otherUserContext(ctx)
	foreign := s.seedExercise(otherCtx, s.seedSkill(otherCtx, s.seedLearningPlan(otherCtx, "Other", domain.LearningPlanActive), "x"), "x", learningDay(0))
	s.seedAnswer(otherCtx, foreign, "y", time.Now())

	s.Run("several in one call", func() {
		_, out, err := learning.ReviewLearningExercises(ctx, nil, learning.ReviewLearningExercisesInput{Reviews: []learning.ExerciseReviewInput{
			{ExerciseID: first, Score: 4, Note: util.Ptr("είμαι, not ειμαι")},
			{ExerciseID: second, Score: 2},
		}})
		require.NoError(s.T(), err)
		require.Len(s.T(), out.Exercises, 2)
		assert.Equal(s.T(), "reviewed", out.Exercises[0].State)

		stored, err := s.Repo().GetLearningExercise(ctx, first, userID)
		require.NoError(s.T(), err)
		assert.Equal(s.T(), 4, *stored.Score)
		assert.Equal(s.T(), "είμαι, not ειμαι", *stored.Note)
		assert.NotNil(s.T(), stored.ReviewedAt)
	})

	s.Run("second review overwrites", func() {
		_, _, err := learning.ReviewLearningExercises(ctx, nil, learning.ReviewLearningExercisesInput{Reviews: []learning.ExerciseReviewInput{
			{ExerciseID: first, Score: 5, Note: util.Ptr("all correct")},
		}})
		require.NoError(s.T(), err)

		stored, err := s.Repo().GetLearningExercise(ctx, first, userID)
		require.NoError(s.T(), err)
		assert.Equal(s.T(), 5, *stored.Score)
		assert.Equal(s.T(), "all correct", *stored.Note)
	})

	fresh := answeredExercise()
	tooMany := make([]learning.ExerciseReviewInput, 51)
	for i := range tooMany {
		tooMany[i] = learning.ExerciseReviewInput{ExerciseID: fresh, Score: 5}
	}
	failures := []struct {
		name    string
		reviews []learning.ExerciseReviewInput
		wantErr string
	}{
		{name: "score outside 1-5", reviews: []learning.ExerciseReviewInput{{ExerciseID: fresh, Score: 5}, {ExerciseID: second, Score: 6}}, wantErr: "score must be 1-5"},
		{name: "unanswered exercise", reviews: []learning.ExerciseReviewInput{{ExerciseID: fresh, Score: 5}, {ExerciseID: unanswered, Score: 3}}, wantErr: "no answer"},
		{name: "no entries", reviews: nil, wantErr: "1-50"},
		{name: "more than 50", reviews: tooMany, wantErr: "1-50"},
		{name: "another user's exercise", reviews: []learning.ExerciseReviewInput{{ExerciseID: fresh, Score: 5}, {ExerciseID: foreign, Score: 3}}, wantErr: "not found"},
	}
	for _, tt := range failures {
		s.Run(tt.name, func() {
			_, _, err := learning.ReviewLearningExercises(ctx, nil, learning.ReviewLearningExercisesInput{Reviews: tt.reviews})
			require.ErrorContains(s.T(), err, tt.wantErr)

			stored, err := s.Repo().GetLearningExercise(ctx, fresh, userID)
			require.NoError(s.T(), err)
			assert.Nil(s.T(), stored.Score, "nothing changes in the whole call")
		})
	}
}

func (s *IntegrationTestSuite) TestDeleteLearningExercise() {
	ctx := s.Context()
	userID := gateways.UserIDFromContext(ctx)
	planID := s.seedLearningPlan(ctx, "Greek A1", domain.LearningPlanActive)
	skill := s.seedSkill(ctx, planID, "present tense")

	planned := s.seedExercise(ctx, skill, "planned", learningDay(0))
	answered := s.seedExercise(ctx, skill, "answered", learningDay(0))
	s.seedAnswer(ctx, answered, "y", time.Now())
	reviewed := s.seedReviewedExercise(ctx, skill, 4, time.Now())

	otherCtx := s.otherUserContext(ctx)
	foreign := s.seedExercise(otherCtx, s.seedSkill(otherCtx, s.seedLearningPlan(otherCtx, "Other", domain.LearningPlanActive), "x"), "x", learningDay(0))

	for _, tt := range []struct {
		name  string
		id    int64
		state string
	}{
		{name: "planned", id: planned, state: "planned"},
		{name: "answered", id: answered, state: "answered"},
		{name: "reviewed", id: reviewed, state: "reviewed"},
	} {
		s.Run("deletes "+tt.name, func() {
			_, out, err := learning.DeleteLearningExercise(ctx, nil, learning.DeleteLearningExerciseInput{ExerciseID: tt.id})
			require.NoError(s.T(), err)
			assert.Equal(s.T(), tt.id, out.Exercise.ID)
			assert.Equal(s.T(), tt.state, out.Exercise.State)

			stored, err := s.Repo().GetLearningExercise(ctx, tt.id, userID)
			require.NoError(s.T(), err)
			assert.Nil(s.T(), stored)
		})
	}

	s.Run("unknown id fails", func() {
		_, _, err := learning.DeleteLearningExercise(ctx, nil, learning.DeleteLearningExerciseInput{ExerciseID: planned})
		require.ErrorContains(s.T(), err, "not found")
	})

	s.Run("another user's exercise fails", func() {
		_, _, err := learning.DeleteLearningExercise(ctx, nil, learning.DeleteLearningExerciseInput{ExerciseID: foreign})
		require.ErrorContains(s.T(), err, "not found")
		stored, err := s.Repo().GetLearningExercise(otherCtx, foreign, gateways.UserIDFromContext(otherCtx))
		require.NoError(s.T(), err)
		assert.NotNil(s.T(), stored)
	})
}

func (s *IntegrationTestSuite) TestListLearningExercises() {
	ctx := s.Context()
	planID := s.seedLearningPlan(ctx, "Greek A1", domain.LearningPlanActive)
	tense := s.seedSkill(ctx, planID, "present tense")
	articles := s.seedSkill(ctx, planID, "articles")

	past := s.seedExercise(ctx, tense, "past\nline two", learningDay(-2))
	today := s.seedExercise(ctx, articles, "today", learningDay(0))
	future := s.seedExercise(ctx, tense, "future", learningDay(2))
	answered := s.seedExercise(ctx, tense, "answered", learningDay(-3))
	s.seedAnswer(ctx, answered, "y", time.Now())
	reviewed := s.seedReviewedExercise(ctx, articles, 5, time.Now()) // planned 30 days ago

	otherCtx := s.otherUserContext(ctx)
	s.seedExercise(otherCtx, s.seedSkill(otherCtx, s.seedLearningPlan(otherCtx, "Other", domain.LearningPlanActive), "x"), "x", learningDay(0))

	tests := []struct {
		name      string
		input     learning.ListLearningExercisesInput
		want      []int64
		wantTotal int
	}{
		{name: "by plan, newest first", input: learning.ListLearningExercisesInput{LearningPlanID: &planID}, want: []int64{future, today, past, answered, reviewed}, wantTotal: 5},
		{name: "by skill", input: learning.ListLearningExercisesInput{SkillID: &articles}, want: []int64{today, reviewed}, wantTotal: 2},
		{name: "states filter", input: learning.ListLearningExercisesInput{LearningPlanID: &planID, States: []string{"answered", "reviewed"}}, want: []int64{answered, reviewed}, wantTotal: 2},
		{name: "planned only is earliest first", input: learning.ListLearningExercisesInput{LearningPlanID: &planID, States: []string{"planned"}}, want: []int64{past, today, future}, wantTotal: 3},
		{name: "due_only hides future days", input: learning.ListLearningExercisesInput{LearningPlanID: &planID, States: []string{"planned"}, DueOnly: true}, want: []int64{past, today}, wantTotal: 2},
		{name: "limit and offset", input: learning.ListLearningExercisesInput{LearningPlanID: &planID, Limit: 2, Offset: 1}, want: []int64{today, past}, wantTotal: 5},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			_, out, err := learning.ListLearningExercises(ctx, nil, tt.input)
			require.NoError(s.T(), err)
			assert.Equal(s.T(), tt.want, exerciseResultIDs(out.Exercises))
			assert.Equal(s.T(), tt.wantTotal, out.Total)
		})
	}

	s.Run("full text returned", func() {
		_, out, err := learning.ListLearningExercises(ctx, nil, learning.ListLearningExercisesInput{SkillID: &tense, States: []string{"planned"}})
		require.NoError(s.T(), err)
		assert.Equal(s.T(), "past\nline two", out.Exercises[0].Content)
		assert.Equal(s.T(), "present tense", out.Exercises[0].SkillName)
	})

	for _, tt := range []struct {
		name  string
		input learning.ListLearningExercisesInput
	}{
		{name: "both ids fail", input: learning.ListLearningExercisesInput{LearningPlanID: &planID, SkillID: &tense}},
		{name: "neither id fails", input: learning.ListLearningExercisesInput{}},
	} {
		s.Run(tt.name, func() {
			_, _, err := learning.ListLearningExercises(ctx, nil, tt.input)
			require.ErrorContains(s.T(), err, "exactly one")
		})
	}
}
