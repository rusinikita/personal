package tests

// Covers plan_vocabulary, cancel_vocabulary_reviews, log_vocabulary_reviews
// and list_vocabulary_reviews from docs/functions/learning-spec.md.

import (
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"personal/action/learning"
	"personal/domain"
)

func reviewResultIDs(results []learning.ReviewResult) []int64 {
	ids := make([]int64, len(results))
	for i, r := range results {
		ids[i] = r.ID
	}
	return ids
}

func (s *IntegrationTestSuite) TestPlanVocabulary() {
	ctx := s.Context()
	planID := s.seedLearningPlan(ctx, "Greek A1", domain.LearningPlanActive)
	water := s.seedWord(ctx, planID, "νερό", "water")
	bread := s.seedWord(ctx, planID, "ψωμί", "bread")
	milk := s.seedWord(ctx, planID, "γάλα", "milk")
	s.seedPendingReview(ctx, milk, learningDay(2))
	s.seedScoredReview(ctx, milk, 5, learningDay(3).Add(time.Hour))

	otherCtx := s.otherUserContext(ctx)
	foreign := s.seedWord(otherCtx, s.seedLearningPlan(otherCtx, "Other", domain.LearningPlanActive), "x", "y")

	s.Run("one and several dates, several words", func() {
		_, out, err := learning.PlanVocabulary(ctx, nil, learning.PlanVocabularyInput{Entries: []learning.PlanVocabularyEntry{
			{WordID: water, ReviewDates: []string{learningDate(0)}},
			{WordID: bread, ReviewDates: []string{learningDate(1), learningDate(4)}},
		}})
		require.NoError(s.T(), err)
		require.Len(s.T(), out.Reviews, 3)

		reviews := s.wordReviews(ctx, bread)
		require.Len(s.T(), reviews, 2)
		for _, r := range reviews {
			assert.Nil(s.T(), r.Score)
			assert.Nil(s.T(), r.ReviewedAt)
		}
		assert.True(s.T(), learningDay(1).Equal(reviews[0].PlannedFor), "planned at midnight Asia/Nicosia")
		assert.True(s.T(), learningDay(4).Equal(reviews[1].PlannedFor))
	})

	s.Run("a scored review on that day does not block planning", func() {
		_, _, err := learning.PlanVocabulary(ctx, nil, learning.PlanVocabularyInput{Entries: []learning.PlanVocabularyEntry{
			{WordID: milk, ReviewDates: []string{learningDate(3)}},
		}})
		require.NoError(s.T(), err)
	})

	tooMany := make([]learning.PlanVocabularyEntry, 101)
	for i := range tooMany {
		tooMany[i] = learning.PlanVocabularyEntry{WordID: water, ReviewDates: []string{learningDate(10)}}
	}
	failures := []struct {
		name    string
		entries []learning.PlanVocabularyEntry
		wantErr string
	}{
		{name: "no dates", entries: []learning.PlanVocabularyEntry{{WordID: water, ReviewDates: []string{learningDate(9)}}, {WordID: bread}}, wantErr: "review_dates cannot be empty"},
		{name: "past date", entries: []learning.PlanVocabularyEntry{{WordID: water, ReviewDates: []string{learningDate(9)}}, {WordID: bread, ReviewDates: []string{learningDate(-1)}}}, wantErr: "in the past"},
		{name: "pending review on that day", entries: []learning.PlanVocabularyEntry{{WordID: water, ReviewDates: []string{learningDate(9)}}, {WordID: milk, ReviewDates: []string{learningDate(2)}}}, wantErr: "already has a pending review"},
		{name: "same day twice in the call", entries: []learning.PlanVocabularyEntry{{WordID: water, ReviewDates: []string{learningDate(9), learningDate(9)}}}, wantErr: "already has a pending review"},
		{name: "unknown word", entries: []learning.PlanVocabularyEntry{{WordID: water, ReviewDates: []string{learningDate(9)}}, {WordID: -1, ReviewDates: []string{learningDate(9)}}}, wantErr: "not found"},
		{name: "another user's word", entries: []learning.PlanVocabularyEntry{{WordID: water, ReviewDates: []string{learningDate(9)}}, {WordID: foreign, ReviewDates: []string{learningDate(9)}}}, wantErr: "not found"},
		{name: "no entries", entries: nil, wantErr: "1-100"},
		{name: "more than 100", entries: tooMany, wantErr: "1-100"},
	}
	for _, tt := range failures {
		s.Run(tt.name, func() {
			before := len(s.wordReviews(ctx, water))
			_, _, err := learning.PlanVocabulary(ctx, nil, learning.PlanVocabularyInput{Entries: tt.entries})
			require.ErrorContains(s.T(), err, tt.wantErr)
			assert.Len(s.T(), s.wordReviews(ctx, water), before, "nothing changes in the whole call")
		})
	}
}

func (s *IntegrationTestSuite) TestCancelVocabularyReviews() {
	ctx := s.Context()
	planID := s.seedLearningPlan(ctx, "Greek A1", domain.LearningPlanActive)
	word := s.seedWord(ctx, planID, "νερό", "water")
	missed := s.seedPendingReview(ctx, word, learningDay(-2))
	upcoming := s.seedPendingReview(ctx, word, learningDay(2))
	kept := s.seedPendingReview(ctx, word, learningDay(5))
	scored := s.seedScoredReview(ctx, word, 4, time.Now())

	otherCtx := s.otherUserContext(ctx)
	foreignWord := s.seedWord(otherCtx, s.seedLearningPlan(otherCtx, "Other", domain.LearningPlanActive), "x", "y")
	foreign := s.seedPendingReview(otherCtx, foreignWord, learningDay(1))

	s.Run("pending reviews deleted", func() {
		_, out, err := learning.CancelVocabularyReviews(ctx, nil, learning.CancelVocabularyReviewsInput{ReviewIDs: []int64{missed, upcoming}})
		require.NoError(s.T(), err)
		assert.ElementsMatch(s.T(), []int64{missed, upcoming}, reviewResultIDs(out.Reviews))
		var left []int64
		for _, r := range s.wordReviews(ctx, word) {
			left = append(left, r.ID)
		}
		assert.ElementsMatch(s.T(), []int64{kept, scored}, left)
	})

	tooMany := make([]int64, 101)
	for i := range tooMany {
		tooMany[i] = int64(i + 1)
	}
	failures := []struct {
		name    string
		ids     []int64
		wantErr string
	}{
		{name: "review with a result", ids: []int64{kept, scored}, wantErr: "already has a result"},
		{name: "unknown review", ids: []int64{kept, -1}, wantErr: "not found"},
		{name: "another user's review", ids: []int64{kept, foreign}, wantErr: "not found"},
		{name: "no ids", ids: nil, wantErr: "1-100"},
		{name: "more than 100", ids: tooMany, wantErr: "1-100"},
	}
	for _, tt := range failures {
		s.Run(tt.name, func() {
			_, _, err := learning.CancelVocabularyReviews(ctx, nil, learning.CancelVocabularyReviewsInput{ReviewIDs: tt.ids})
			require.ErrorContains(s.T(), err, tt.wantErr)
			assert.Len(s.T(), s.wordReviews(ctx, word), 2, "nothing deleted in the whole call")
		})
	}
}

func (s *IntegrationTestSuite) TestLogVocabularyReviews() {
	ctx := s.Context()
	now := time.Now()
	planID := s.seedLearningPlan(ctx, "Greek A1", domain.LearningPlanActive)
	water := s.seedWord(ctx, planID, "νερό", "water")
	bread := s.seedWord(ctx, planID, "ψωμί", "bread")
	s.seedScoredReview(ctx, bread, 5, now.Add(-2*time.Hour))
	s.seedScoredReview(ctx, bread, 5, now.Add(-time.Hour))
	pending := s.seedPendingReview(ctx, water, learningDay(2))

	otherCtx := s.otherUserContext(ctx)
	foreign := s.seedWord(otherCtx, s.seedLearningPlan(otherCtx, "Other", domain.LearningPlanActive), "x", "y")

	s.Run("scores 4 and 2 create scored reviews dated today", func() {
		_, out, err := learning.LogVocabularyReviews(ctx, nil, learning.LogVocabularyReviewsInput{Entries: []learning.LogVocabularyReviewEntry{
			{WordID: water, Score: 4},
			{WordID: bread, Score: 2},
		}})
		require.NoError(s.T(), err)
		require.Len(s.T(), out.Reviews, 2)
		assert.Equal(s.T(), learningDate(0), out.Reviews[0].PlannedFor)

		assert.Equal(s.T(), 10, s.learningWord(ctx, water).Mastery)
		assert.Equal(s.T(), 30, s.learningWord(ctx, bread).Mastery)

		reviews := s.wordReviews(ctx, water)
		require.Len(s.T(), reviews, 2)
		assert.WithinDuration(s.T(), now, *reviews[0].ReviewedAt, time.Minute)
		assert.Equal(s.T(), pending, reviews[1].ID)
		assert.Nil(s.T(), reviews[1].Score, "pending review of the word untouched")
	})

	tooMany := make([]learning.LogVocabularyReviewEntry, 101)
	for i := range tooMany {
		tooMany[i] = learning.LogVocabularyReviewEntry{WordID: water, Score: 5}
	}
	failures := []struct {
		name    string
		entries []learning.LogVocabularyReviewEntry
		wantErr string
	}{
		{name: "score outside 1-5", entries: []learning.LogVocabularyReviewEntry{{WordID: water, Score: 5}, {WordID: bread, Score: 0}}, wantErr: "score must be 1-5"},
		{name: "unknown word", entries: []learning.LogVocabularyReviewEntry{{WordID: water, Score: 5}, {WordID: -1, Score: 3}}, wantErr: "not found"},
		{name: "another user's word", entries: []learning.LogVocabularyReviewEntry{{WordID: water, Score: 5}, {WordID: foreign, Score: 3}}, wantErr: "not found"},
		{name: "no entries", entries: nil, wantErr: "1-100"},
		{name: "more than 100", entries: tooMany, wantErr: "1-100"},
	}
	for _, tt := range failures {
		s.Run(tt.name, func() {
			_, _, err := learning.LogVocabularyReviews(ctx, nil, learning.LogVocabularyReviewsInput{Entries: tt.entries})
			require.ErrorContains(s.T(), err, tt.wantErr)
			assert.Len(s.T(), s.wordReviews(ctx, water), 2, "nothing created in the whole call")
		})
	}
}

func (s *IntegrationTestSuite) TestListVocabularyReviews() {
	ctx := s.Context()
	now := time.Now()
	planID := s.seedLearningPlan(ctx, "Greek A1", domain.LearningPlanActive)
	water := s.seedWord(ctx, planID, "νερό", "water")
	bread := s.seedWord(ctx, planID, "ψωμί", "bread")

	old := s.seedScoredReview(ctx, water, 5, now.AddDate(0, 0, -10))
	mid := s.seedScoredReview(ctx, bread, 3, now.AddDate(0, 0, -3))
	recent := s.seedScoredReview(ctx, water, 4, now.AddDate(0, 0, -1))
	later := s.seedPendingReview(ctx, water, learningDay(3))
	missed := s.seedPendingReview(ctx, bread, learningDay(-2))

	otherPlan := s.seedLearningPlan(ctx, "Greek A2", domain.LearningPlanActive)
	s.seedPendingReview(ctx, s.seedWord(ctx, otherPlan, "γάλα", "milk"), learningDay(1))

	otherCtx := s.otherUserContext(ctx)
	foreignPlan := s.seedLearningPlan(otherCtx, "Other", domain.LearningPlanActive)
	s.seedPendingReview(otherCtx, s.seedWord(otherCtx, foreignPlan, "x", "y"), learningDay(1))

	pendingOnly, scoredOnly := true, false
	tests := []struct {
		name      string
		input     learning.ListVocabularyReviewsInput
		want      []int64
		wantTotal int
	}{
		{
			name:  "scored reviews in a window, newest first",
			input: learning.ListVocabularyReviewsInput{LearningPlanID: planID, ReviewedFrom: now.AddDate(0, 0, -7).Format(time.RFC3339), ReviewedTo: now.Format(time.RFC3339)},
			want:  []int64{recent, mid}, wantTotal: 2,
		},
		{name: "scored only", input: learning.ListVocabularyReviewsInput{LearningPlanID: planID, Pending: &scoredOnly}, want: []int64{recent, mid, old}, wantTotal: 3},
		{name: "pending earliest first", input: learning.ListVocabularyReviewsInput{LearningPlanID: planID, Pending: &pendingOnly}, want: []int64{missed, later}, wantTotal: 2},
		{name: "one word's full history", input: learning.ListVocabularyReviewsInput{LearningPlanID: planID, VocabularyID: &water}, want: []int64{recent, old, later}, wantTotal: 3},
		{name: "limit and offset", input: learning.ListVocabularyReviewsInput{LearningPlanID: planID, Pending: &scoredOnly, Limit: 1, Offset: 1}, want: []int64{mid}, wantTotal: 3},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			_, out, err := learning.ListVocabularyReviews(ctx, nil, tt.input)
			require.NoError(s.T(), err)
			assert.Equal(s.T(), tt.want, reviewResultIDs(out.Reviews))
			assert.Equal(s.T(), tt.wantTotal, out.Total)
		})
	}

	s.Run("term and translation joined", func() {
		_, out, err := learning.ListVocabularyReviews(ctx, nil, learning.ListVocabularyReviewsInput{LearningPlanID: planID, VocabularyID: &bread})
		require.NoError(s.T(), err)
		assert.Equal(s.T(), "ψωμί", out.Reviews[0].Term)
		assert.Equal(s.T(), "bread", out.Reviews[0].Translation)
	})

	s.Run("another user's plan fails", func() {
		_, _, err := learning.ListVocabularyReviews(ctx, nil, learning.ListVocabularyReviewsInput{LearningPlanID: foreignPlan})
		require.ErrorContains(s.T(), err, "not found")
	})

}
