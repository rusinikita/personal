package tests

// Covers create_skill, edit_skill and skill mastery from
// docs/functions/learning-spec.md.

import (
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"personal/action/learning"
	"personal/domain"
)

func (s *IntegrationTestSuite) TestCreateSkill() {
	ctx := s.Context()
	planID := s.seedLearningPlan(ctx, "Greek A1", domain.LearningPlanActive)
	otherPlanOfUser := s.seedLearningPlan(ctx, "Greek A2", domain.LearningPlanActive)

	otherCtx := s.otherUserContext(ctx)
	foreignPlan := s.seedLearningPlan(otherCtx, "Other", domain.LearningPlanActive)

	s.Run("created with 0% mastery", func() {
		_, out, err := learning.CreateSkill(ctx, nil, learning.CreateSkillInput{LearningPlanID: planID, Name: "Present tense"})
		require.NoError(s.T(), err)
		assert.Equal(s.T(), "Present tense", out.Skill.Name)
		assert.Equal(s.T(), planID, out.Skill.LearningPlanID)
		assert.Equal(s.T(), 0, out.Skill.Mastery)
		assert.Equal(s.T(), 0, out.Skill.EffectiveMastery)
	})

	s.Run("same name in another plan succeeds", func() {
		_, _, err := learning.CreateSkill(ctx, nil, learning.CreateSkillInput{LearningPlanID: otherPlanOfUser, Name: "Present tense"})
		require.NoError(s.T(), err)
	})

	failures := []struct {
		name    string
		input   learning.CreateSkillInput
		wantErr string
	}{
		{name: "duplicate name in another case", input: learning.CreateSkillInput{LearningPlanID: planID, Name: "PRESENT TENSE"}, wantErr: "already exists"},
		{name: "empty name", input: learning.CreateSkillInput{LearningPlanID: planID, Name: " "}, wantErr: "name is required"},
		{name: "another user's plan", input: learning.CreateSkillInput{LearningPlanID: foreignPlan, Name: "x"}, wantErr: "not found"},
	}
	for _, tt := range failures {
		s.Run(tt.name, func() {
			_, _, err := learning.CreateSkill(ctx, nil, tt.input)
			require.ErrorContains(s.T(), err, tt.wantErr)
		})
	}
}

func (s *IntegrationTestSuite) TestEditSkill() {
	ctx := s.Context()
	planID := s.seedLearningPlan(ctx, "Greek A1", domain.LearningPlanActive)
	skill := s.seedSkill(ctx, planID, "present tense")
	s.seedSkill(ctx, planID, "articles")

	otherCtx := s.otherUserContext(ctx)
	foreignSkill := s.seedSkill(otherCtx, s.seedLearningPlan(otherCtx, "Other", domain.LearningPlanActive), "x")

	s.Run("renames", func() {
		_, out, err := learning.EditSkill(ctx, nil, learning.EditSkillInput{SkillID: skill, Name: "Present Tense"})
		require.NoError(s.T(), err)
		assert.Equal(s.T(), "Present Tense", out.Skill.Name)
	})

	failures := []struct {
		name    string
		input   learning.EditSkillInput
		wantErr string
	}{
		{name: "empty name", input: learning.EditSkillInput{SkillID: skill, Name: ""}, wantErr: "name is required"},
		{name: "duplicate name", input: learning.EditSkillInput{SkillID: skill, Name: "Articles"}, wantErr: "already exists"},
		{name: "another user's skill", input: learning.EditSkillInput{SkillID: foreignSkill, Name: "y"}, wantErr: "not found"},
	}
	for _, tt := range failures {
		s.Run(tt.name, func() {
			_, _, err := learning.EditSkill(ctx, nil, tt.input)
			require.ErrorContains(s.T(), err, tt.wantErr)
		})
	}
}

func (s *IntegrationTestSuite) TestSkillMastery() {
	ctx := s.Context()
	now := time.Now()
	planID := s.seedLearningPlan(ctx, "Greek A1", domain.LearningPlanActive)

	skillMastery := func(skillID int64) learning.SkillResult {
		_, out, err := learning.GetLearningPlan(ctx, nil, learning.GetLearningPlanInput{LearningPlanID: planID})
		require.NoError(s.T(), err)
		for _, sk := range out.Plan.Skills {
			if sk.ID == skillID {
				return sk
			}
		}
		s.T().Fatalf("skill %d not found", skillID)
		return learning.SkillResult{}
	}

	tests := []struct {
		name   string
		scores []int
		want   int
	}{
		{name: "four 5s reach 80", scores: []int{5, 5, 5, 5}, want: 80},
		{name: "5 then 1 back to 0", scores: []int{5, 1}, want: 0},
		{name: "1 at 0% stays 0", scores: []int{1}, want: 0},
		{name: "six 5s stay at 100", scores: []int{5, 5, 5, 5, 5, 5}, want: 100},
		{name: "3 changes nothing", scores: []int{5, 3}, want: 20},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			skill := s.seedSkill(ctx, planID, tt.name)
			for i, score := range tt.scores {
				s.seedReviewedExercise(ctx, skill, score, now.Add(time.Duration(i-len(tt.scores))*time.Minute))
			}
			assert.Equal(s.T(), tt.want, skillMastery(skill).Mastery)
		})
	}

	s.Run("answered but unreviewed exercises not counted", func() {
		skill := s.seedSkill(ctx, planID, "unreviewed")
		s.seedReviewedExercise(ctx, skill, 5, now.Add(-time.Hour))
		answered := s.seedExercise(ctx, skill, "x", learningDay(0))
		s.seedAnswer(ctx, answered, "y", now)
		assert.Equal(s.T(), 20, skillMastery(skill).Mastery)
	})

	s.Run("re-reviewing an exercise recomputes with the new score", func() {
		skill := s.seedSkill(ctx, planID, "re-review")
		exercise := s.seedExercise(ctx, skill, "x", learningDay(0))
		s.seedAnswer(ctx, exercise, "y", now)

		_, _, err := learning.ReviewLearningExercises(ctx, nil, learning.ReviewLearningExercisesInput{Reviews: []learning.ExerciseReviewInput{{ExerciseID: exercise, Score: 5}}})
		require.NoError(s.T(), err)
		assert.Equal(s.T(), 20, skillMastery(skill).Mastery)

		_, _, err = learning.ReviewLearningExercises(ctx, nil, learning.ReviewLearningExercisesInput{Reviews: []learning.ExerciseReviewInput{{ExerciseID: exercise, Score: 4}}})
		require.NoError(s.T(), err)
		assert.Equal(s.T(), 10, skillMastery(skill).Mastery)
	})

	s.Run("uncertainty counted from the latest answered exercise", func() {
		skill := s.seedSkill(ctx, planID, "uncertainty")
		for i := range 4 {
			s.seedReviewedExercise(ctx, skill, 5, now.AddDate(0, 0, -20).Add(time.Duration(i)*time.Minute))
		}
		// mastery 80, S = 51 days, last practice ~20 days ago
		before := skillMastery(skill)
		assert.Equal(s.T(), 28, before.Uncertainty)

		answered := s.seedExercise(ctx, skill, "x", learningDay(0))
		s.seedAnswer(ctx, answered, "y", now.AddDate(0, 0, -14))
		after := skillMastery(skill)
		assert.Equal(s.T(), 80, after.Mastery)
		assert.Equal(s.T(), 22, after.Uncertainty)
		assert.Equal(s.T(), 9, after.Debuff)
		assert.Equal(s.T(), 71, after.EffectiveMastery)
	})
}
