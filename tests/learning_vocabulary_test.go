package tests

// Covers add_vocabulary, edit_vocabulary, delete_vocabulary,
// list_vocabulary, search_vocabulary, word mastery and the uncertainty
// debuff from docs/functions/learning-spec.md.

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

func wordResultIDs(results []learning.WordResult) []int64 {
	ids := make([]int64, len(results))
	for i, r := range results {
		ids[i] = r.ID
	}
	return ids
}

func (s *IntegrationTestSuite) countPlanWords(ctx context.Context, planID int64) int {
	n, err := s.Repo().CountVocabulary(ctx, domain.VocabularyFilter{UserID: gateways.UserIDFromContext(ctx), LearningPlanID: &planID})
	require.NoError(s.T(), err)
	return n
}

func (s *IntegrationTestSuite) TestAddVocabulary() {
	ctx := s.Context()
	planID := s.seedLearningPlan(ctx, "Greek A1", domain.LearningPlanActive)
	otherPlan := s.seedLearningPlan(ctx, "Greek A2", domain.LearningPlanActive)
	skill := s.seedSkill(ctx, planID, "food")
	otherPlanSkill := s.seedSkill(ctx, otherPlan, "food")

	otherCtx := s.otherUserContext(ctx)
	foreignPlan := s.seedLearningPlan(otherCtx, "Other", domain.LearningPlanActive)

	s.Run("several words, 0% mastery and no reviews", func() {
		_, out, err := learning.AddVocabulary(ctx, nil, learning.AddVocabularyInput{LearningPlanID: planID, Words: []learning.WordInput{
			{Term: "νερό", Translation: "water", Notes: util.Ptr("neuter"), SkillID: &skill},
			{Term: "ψωμί", Translation: "bread"},
		}})
		require.NoError(s.T(), err)
		require.Len(s.T(), out.Words, 2)
		assert.Equal(s.T(), "νερό", out.Words[0].Term)
		assert.Equal(s.T(), "neuter", out.Words[0].Notes)
		assert.Equal(s.T(), skill, *out.Words[0].SkillID)
		assert.Equal(s.T(), 0, out.Words[0].Mastery)
		assert.Equal(s.T(), 0, out.Words[0].ReviewCount)
		assert.Empty(s.T(), s.wordReviews(ctx, out.Words[0].ID))
	})

	s.Run("same term in another plan succeeds", func() {
		_, _, err := learning.AddVocabulary(ctx, nil, learning.AddVocabularyInput{LearningPlanID: otherPlan, Words: []learning.WordInput{{Term: "νερό", Translation: "water"}}})
		require.NoError(s.T(), err)
	})

	tooMany := make([]learning.WordInput, 51)
	for i := range tooMany {
		tooMany[i] = learning.WordInput{Term: string(rune('a' + i)), Translation: "x"}
	}
	failures := []struct {
		name    string
		input   learning.AddVocabularyInput
		wantErr string
	}{
		{name: "term already in the plan, other case", input: learning.AddVocabularyInput{LearningPlanID: planID, Words: []learning.WordInput{{Term: "γάλα", Translation: "milk"}, {Term: "ΝΕΡΌ", Translation: "water"}}}, wantErr: "terms already in the plan: νερό"},
		{name: "term repeated in the call", input: learning.AddVocabularyInput{LearningPlanID: planID, Words: []learning.WordInput{{Term: "γάλα", Translation: "milk"}, {Term: "Γάλα", Translation: "milk"}}}, wantErr: "repeated in the call: Γάλα"},
		{name: "skill of another plan", input: learning.AddVocabularyInput{LearningPlanID: planID, Words: []learning.WordInput{{Term: "γάλα", Translation: "milk", SkillID: &otherPlanSkill}}}, wantErr: "not found in this plan"},
		{name: "empty term", input: learning.AddVocabularyInput{LearningPlanID: planID, Words: []learning.WordInput{{Term: " ", Translation: "milk"}}}, wantErr: "term is required"},
		{name: "empty translation", input: learning.AddVocabularyInput{LearningPlanID: planID, Words: []learning.WordInput{{Term: "γάλα", Translation: ""}}}, wantErr: "translation is required"},
		{name: "no words", input: learning.AddVocabularyInput{LearningPlanID: planID}, wantErr: "1-50"},
		{name: "more than 50", input: learning.AddVocabularyInput{LearningPlanID: planID, Words: tooMany}, wantErr: "1-50"},
		{name: "another user's plan", input: learning.AddVocabularyInput{LearningPlanID: foreignPlan, Words: []learning.WordInput{{Term: "γάλα", Translation: "milk"}}}, wantErr: "not found"},
	}
	for _, tt := range failures {
		s.Run(tt.name, func() {
			before := s.countPlanWords(ctx, planID)
			_, _, err := learning.AddVocabulary(ctx, nil, tt.input)
			require.ErrorContains(s.T(), err, tt.wantErr)
			assert.Equal(s.T(), before, s.countPlanWords(ctx, planID))
		})
	}
}

func (s *IntegrationTestSuite) TestEditVocabulary() {
	ctx := s.Context()
	planID := s.seedLearningPlan(ctx, "Greek A1", domain.LearningPlanActive)
	skill := s.seedSkill(ctx, planID, "food")
	otherPlanSkill := s.seedSkill(ctx, s.seedLearningPlan(ctx, "Greek A2", domain.LearningPlanActive), "food")
	word := s.seedWord(ctx, planID, "νερο", "water")
	s.seedWord(ctx, planID, "ψωμί", "bread")
	s.seedScoredReview(ctx, word, 5, time.Now().Add(-time.Hour))

	otherCtx := s.otherUserContext(ctx)
	foreign := s.seedWord(otherCtx, s.seedLearningPlan(otherCtx, "Other", domain.LearningPlanActive), "x", "y")

	s.Run("term alone", func() {
		_, out, err := learning.EditVocabulary(ctx, nil, learning.EditVocabularyInput{WordID: word, Term: util.Ptr("νερό")})
		require.NoError(s.T(), err)
		assert.Equal(s.T(), "νερό", out.Word.Term)
		assert.Equal(s.T(), "water", out.Word.Translation)
	})

	s.Run("translation alone", func() {
		_, out, err := learning.EditVocabulary(ctx, nil, learning.EditVocabularyInput{WordID: word, Translation: util.Ptr("water (n.)")})
		require.NoError(s.T(), err)
		assert.Equal(s.T(), "water (n.)", out.Word.Translation)
	})

	s.Run("notes and skill set, then cleared", func() {
		_, out, err := learning.EditVocabulary(ctx, nil, learning.EditVocabularyInput{WordID: word, Notes: util.Ptr("το νερό"), SkillID: &skill})
		require.NoError(s.T(), err)
		assert.Equal(s.T(), "το νερό", out.Word.Notes)
		assert.Equal(s.T(), skill, *out.Word.SkillID)

		_, _, err = learning.EditVocabulary(ctx, nil, learning.EditVocabularyInput{WordID: word, Notes: util.Ptr(""), SkillID: util.Ptr(int64(0))})
		require.NoError(s.T(), err)
		stored := s.learningWord(ctx, word)
		assert.Nil(s.T(), stored.Notes)
		assert.Nil(s.T(), stored.SkillID)
	})

	s.Run("reviews unchanged", func() {
		reviews := s.wordReviews(ctx, word)
		require.Len(s.T(), reviews, 1)
		assert.Equal(s.T(), 5, *reviews[0].Score)
		assert.Equal(s.T(), 20, s.learningWord(ctx, word).Mastery)
	})

	failures := []struct {
		name    string
		input   learning.EditVocabularyInput
		wantErr string
	}{
		{name: "no fields", input: learning.EditVocabularyInput{WordID: word}, wantErr: "at least one"},
		{name: "duplicate term", input: learning.EditVocabularyInput{WordID: word, Term: util.Ptr("ΨΩΜΊ")}, wantErr: "already in the plan"},
		{name: "skill of another plan", input: learning.EditVocabularyInput{WordID: word, SkillID: &otherPlanSkill}, wantErr: "not found in this plan"},
		{name: "another user's word", input: learning.EditVocabularyInput{WordID: foreign, Term: util.Ptr("z")}, wantErr: "not found"},
	}
	for _, tt := range failures {
		s.Run(tt.name, func() {
			_, _, err := learning.EditVocabulary(ctx, nil, tt.input)
			require.ErrorContains(s.T(), err, tt.wantErr)
		})
	}
}

func (s *IntegrationTestSuite) TestDeleteVocabulary() {
	ctx := s.Context()
	planID := s.seedLearningPlan(ctx, "Greek A1", domain.LearningPlanActive)
	word := s.seedWord(ctx, planID, "νερό", "water")
	s.seedScoredReview(ctx, word, 5, time.Now())
	s.seedPendingReview(ctx, word, learningDay(1))

	otherCtx := s.otherUserContext(ctx)
	foreign := s.seedWord(otherCtx, s.seedLearningPlan(otherCtx, "Other", domain.LearningPlanActive), "x", "y")

	s.Run("word and its reviews deleted and returned", func() {
		_, out, err := learning.DeleteVocabulary(ctx, nil, learning.DeleteVocabularyInput{WordID: word})
		require.NoError(s.T(), err)
		assert.Equal(s.T(), "νερό", out.Word.Term)
		assert.Equal(s.T(), 0, s.countPlanWords(ctx, planID))
		assert.Empty(s.T(), s.wordReviews(ctx, word))
	})

	s.Run("unknown id fails", func() {
		_, _, err := learning.DeleteVocabulary(ctx, nil, learning.DeleteVocabularyInput{WordID: word})
		require.ErrorContains(s.T(), err, "not found")
	})

	s.Run("another user's word fails", func() {
		_, _, err := learning.DeleteVocabulary(ctx, nil, learning.DeleteVocabularyInput{WordID: foreign})
		require.ErrorContains(s.T(), err, "not found")
	})
}

func (s *IntegrationTestSuite) TestListVocabulary() {
	ctx := s.Context()
	now := time.Now()
	planID := s.seedLearningPlan(ctx, "Greek A1", domain.LearningPlanActive)

	fresh := s.seedWord(ctx, planID, "γάλα", "milk")
	recent := s.seedWord(ctx, planID, "νερό", "water")
	s.seedScoredReview(ctx, recent, 5, now.Add(-2*time.Hour))
	s.seedScoredReview(ctx, recent, 4, now.Add(-time.Hour))
	s.seedPendingReview(ctx, recent, learningDay(3))
	s.seedPendingReview(ctx, recent, learningDay(1))
	old := s.seedWord(ctx, planID, "ψωμί", "bread")
	for i := range 4 {
		s.seedScoredReview(ctx, old, 5, now.Add(-3*time.Hour).Add(time.Duration(i)*time.Minute))
	}

	otherCtx := s.otherUserContext(ctx)
	s.seedWord(otherCtx, s.seedLearningPlan(otherCtx, "Other", domain.LearningPlanActive), "x", "y")

	s.Run("never reviewed first, then least recently reviewed, with computed fields", func() {
		_, out, err := learning.ListVocabulary(ctx, nil, learning.ListVocabularyInput{LearningPlanID: planID})
		require.NoError(s.T(), err)
		assert.Equal(s.T(), []int64{fresh, old, recent}, wordResultIDs(out.Words))
		assert.Equal(s.T(), 3, out.Total)

		r := out.Words[2]
		assert.Equal(s.T(), 30, r.Mastery)
		assert.Equal(s.T(), 2, r.ReviewCount)
		assert.Equal(s.T(), 4, *r.LastScore)
		assert.Equal(s.T(), learningDate(1), r.NextPlannedFor)

		assert.Equal(s.T(), 80, out.Words[1].Mastery)
		assert.Empty(s.T(), out.Words[0].NextPlannedFor)
	})

	s.Run("max_mastery filter", func() {
		_, out, err := learning.ListVocabulary(ctx, nil, learning.ListVocabularyInput{LearningPlanID: planID, MaxMastery: util.Ptr(79)})
		require.NoError(s.T(), err)
		assert.Equal(s.T(), []int64{fresh, recent}, wordResultIDs(out.Words))
		assert.Equal(s.T(), 2, out.Total)
	})

	s.Run("unscheduled filter", func() {
		_, out, err := learning.ListVocabulary(ctx, nil, learning.ListVocabularyInput{LearningPlanID: planID, Unscheduled: true})
		require.NoError(s.T(), err)
		assert.Equal(s.T(), []int64{fresh, old}, wordResultIDs(out.Words))
		assert.Equal(s.T(), 2, out.Total)
	})

	s.Run("limit and offset", func() {
		_, out, err := learning.ListVocabulary(ctx, nil, learning.ListVocabularyInput{LearningPlanID: planID, Limit: 1, Offset: 1})
		require.NoError(s.T(), err)
		assert.Equal(s.T(), []int64{old}, wordResultIDs(out.Words))
		assert.Equal(s.T(), 3, out.Total)

		_, out, err = learning.ListVocabulary(ctx, nil, learning.ListVocabularyInput{LearningPlanID: planID, MaxMastery: util.Ptr(100), Limit: 1, Offset: 2})
		require.NoError(s.T(), err)
		assert.Equal(s.T(), []int64{recent}, wordResultIDs(out.Words))
	})
}

func (s *IntegrationTestSuite) TestSearchVocabulary() {
	ctx := s.Context()
	now := time.Now()
	planID := s.seedLearningPlan(ctx, "Greek A1", domain.LearningPlanActive)
	otherPlan := s.seedLearningPlan(ctx, "Greek A2", domain.LearningPlanActive)

	water := s.seedWordAt(ctx, planID, "νερό", "Water", now.Add(-3*time.Hour))
	s.seedScoredReview(ctx, water, 5, now)
	_, _, err := learning.EditVocabulary(ctx, nil, learning.EditVocabularyInput{WordID: water, Notes: util.Ptr("a drink")})
	require.NoError(s.T(), err)
	milk := s.seedWordAt(ctx, planID, "γάλα", "milk", now.Add(-2*time.Hour))
	_, _, err = learning.EditVocabulary(ctx, nil, learning.EditVocabularyInput{WordID: milk, Notes: util.Ptr("white DRINK")})
	require.NoError(s.T(), err)
	waterfall := s.seedWordAt(ctx, otherPlan, "καταρράκτης", "waterfall", now.Add(-time.Hour))

	otherCtx := s.otherUserContext(ctx)
	s.seedWord(otherCtx, s.seedLearningPlan(otherCtx, "Other", domain.LearningPlanActive), "water", "water")

	ids := func(results []learning.WordSearchResult) []int64 {
		out := make([]int64, len(results))
		for i, r := range results {
			out[i] = r.ID
		}
		return out
	}

	s.Run("matches term, translation and notes ignoring case, ranked by match count then newest", func() {
		_, out, err := learning.SearchVocabulary(ctx, nil, learning.SearchVocabularyInput{QueryVariants: []string{"WATER", "drink"}})
		require.NoError(s.T(), err)
		assert.Equal(s.T(), []int64{water, waterfall, milk}, ids(out.Results))
		assert.Equal(s.T(), 2, out.Results[0].MatchCount)
		assert.Equal(s.T(), 20, out.Results[0].Mastery)
		assert.Equal(s.T(), 1, out.Results[0].ReviewCount)
	})

	s.Run("term match", func() {
		_, out, err := learning.SearchVocabulary(ctx, nil, learning.SearchVocabularyInput{QueryVariants: []string{"ΓΆΛΑ"}})
		require.NoError(s.T(), err)
		assert.Equal(s.T(), []int64{milk}, ids(out.Results))
	})

	s.Run("plan filter", func() {
		_, out, err := learning.SearchVocabulary(ctx, nil, learning.SearchVocabularyInput{QueryVariants: []string{"water"}, LearningPlanID: &planID})
		require.NoError(s.T(), err)
		assert.Equal(s.T(), []int64{water}, ids(out.Results))
	})

	for _, tt := range []struct {
		name     string
		variants []string
	}{
		{name: "no variants", variants: nil},
		{name: "more than 5 variants", variants: []string{"a", "b", "c", "d", "e", "f"}},
		{name: "empty variant", variants: []string{"water", " "}},
	} {
		s.Run(tt.name+" fails", func() {
			_, _, err := learning.SearchVocabulary(ctx, nil, learning.SearchVocabularyInput{QueryVariants: tt.variants})
			require.Error(s.T(), err)
		})
	}
}

func (s *IntegrationTestSuite) TestWordMastery() {
	ctx := s.Context()
	now := time.Now()
	planID := s.seedLearningPlan(ctx, "Greek A1", domain.LearningPlanActive)

	tests := []struct {
		name   string
		scores []int
		want   int
	}{
		{name: "four 5s reach 80", scores: []int{5, 5, 5, 5}, want: 80},
		{name: "5 then 1 back to 0", scores: []int{5, 1}, want: 0},
		{name: "2 at 0% stays 0", scores: []int{2}, want: 0},
		{name: "3 changes nothing", scores: []int{5, 3}, want: 20},
		{name: "six 5s stay at 100", scores: []int{5, 5, 5, 5, 5, 5}, want: 100},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			word := s.seedWord(ctx, planID, tt.name, "x")
			for i, score := range tt.scores {
				s.seedScoredReview(ctx, word, score, now.Add(time.Duration(i-len(tt.scores))*time.Minute))
			}
			assert.Equal(s.T(), tt.want, s.learningWord(ctx, word).Mastery)
		})
	}

	s.Run("pending reviews not counted", func() {
		word := s.seedWord(ctx, planID, "pending", "x")
		s.seedScoredReview(ctx, word, 5, now.Add(-time.Hour))
		s.seedPendingReview(ctx, word, learningDay(-1))
		s.seedPendingReview(ctx, word, learningDay(2))
		w := s.learningWord(ctx, word)
		assert.Equal(s.T(), 20, w.Mastery)
		assert.Equal(s.T(), 1, w.ReviewCount)
	})

	s.Run("flashcard and indirect scores counted together in reviewed_at order", func() {
		word := s.seedWord(ctx, planID, "mixed", "x")
		flashcard := s.seedPendingReview(ctx, word, learningDay(0))
		require.NoError(s.T(), s.Repo().RecordVocabularyReview(ctx, flashcard, gateways.UserIDFromContext(ctx), 1, now.Add(-time.Hour)))
		s.seedScoredReview(ctx, word, 5, now.Add(-2*time.Hour))

		// 5 then 1 → 0; the reverse order would give 20
		assert.Equal(s.T(), 0, s.learningWord(ctx, word).Mastery)

		_, _, err := learning.LogVocabularyReviews(ctx, nil, learning.LogVocabularyReviewsInput{Entries: []learning.LogVocabularyReviewEntry{{WordID: word, Score: 5}}})
		require.NoError(s.T(), err)
		assert.Equal(s.T(), 20, s.learningWord(ctx, word).Mastery)
	})
}

func (s *IntegrationTestSuite) TestMasteryUncertainty() {
	ctx := s.Context()
	now := time.Now()
	planID := s.seedLearningPlan(ctx, "Greek A1", domain.LearningPlanActive)

	// seedMastered gives the word four 5s (mastery 80), the last one at lastAt.
	seedMastered := func(term string, lastAt time.Time) int64 {
		word := s.seedWord(ctx, planID, term, "x")
		for i := range 4 {
			s.seedScoredReview(ctx, word, 5, lastAt.Add(time.Duration(i-3)*time.Minute))
		}
		return word
	}

	tests := []struct {
		name            string
		word            func() int64
		wantMastery     int
		wantUncertainty int
		wantDebuff      int
	}{
		{name: "80 reviewed today", word: func() int64 { return seedMastered("today", now) }, wantMastery: 80, wantUncertainty: 0, wantDebuff: 0},
		{name: "80 reviewed 14 days ago", word: func() int64 { return seedMastered("14 days", now.AddDate(0, 0, -14)) }, wantMastery: 80, wantUncertainty: 22, wantDebuff: 9},
		{name: "80 reviewed 51 days ago", word: func() int64 { return seedMastered("51 days", now.AddDate(0, 0, -51)) }, wantMastery: 80, wantUncertainty: 50, wantDebuff: 20},
		{
			name: "20 reviewed 14 days ago",
			word: func() int64 {
				word := s.seedWord(ctx, planID, "twenty", "x")
				s.seedScoredReview(ctx, word, 5, now.AddDate(0, 0, -14))
				return word
			},
			wantMastery: 20, wantUncertainty: 48, wantDebuff: 5,
		},
		{name: "no practice", word: func() int64 { return s.seedWord(ctx, planID, "never", "x") }, wantMastery: 0, wantUncertainty: 100, wantDebuff: 0},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			w := s.learningWord(ctx, tt.word())
			assert.Equal(s.T(), tt.wantMastery, w.Mastery)
			assert.Equal(s.T(), tt.wantUncertainty, w.Uncertainty)
			assert.Equal(s.T(), tt.wantDebuff, w.Debuff)
			assert.Equal(s.T(), tt.wantMastery-tt.wantDebuff, w.EffectiveMastery)
		})
	}

	s.Run("an indirect review resets uncertainty", func() {
		word := seedMastered("reset", now.AddDate(0, 0, -14))
		_, _, err := learning.LogVocabularyReviews(ctx, nil, learning.LogVocabularyReviewsInput{Entries: []learning.LogVocabularyReviewEntry{{WordID: word, Score: 3}}})
		require.NoError(s.T(), err)
		w := s.learningWord(ctx, word)
		assert.Equal(s.T(), 80, w.Mastery)
		assert.Equal(s.T(), 0, w.Uncertainty)
		assert.Equal(s.T(), 80, w.EffectiveMastery)
	})

	s.Run("a word at 80% after 51 days is not mastered and comes back as a candidate", func() {
		plan := s.seedLearningPlan(ctx, "Slipping", domain.LearningPlanActive)
		word := s.seedWord(ctx, plan, "slipping", "x")
		for i := range 4 {
			s.seedScoredReview(ctx, word, 5, now.AddDate(0, 0, -51).Add(time.Duration(i-3)*time.Minute))
		}

		_, out, err := learning.GetLearningPlan(ctx, nil, learning.GetLearningPlanInput{LearningPlanID: plan})
		require.NoError(s.T(), err)
		assert.Equal(s.T(), 0, out.Plan.VocabularyStats.Mastered)
		assert.Equal(s.T(), 60, out.Plan.VocabularyStats.AverageMastery)

		_, list, err := learning.ListVocabulary(ctx, nil, learning.ListVocabularyInput{LearningPlanID: plan, MaxMastery: util.Ptr(79)})
		require.NoError(s.T(), err)
		assert.Equal(s.T(), []int64{word}, wordResultIDs(list.Words))
	})
}
