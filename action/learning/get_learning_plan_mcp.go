package learning

import (
	"context"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"personal/gateways"
)

var GetLearningPlanMCPDefinition = mcp.Tool{
	Name: "get_learning_plan",
	Annotations: &mcp.ToolAnnotations{
		ReadOnlyHint:   true,
		IdempotentHint: true,
		Title:          "Get a learning plan",
	},
	Description: `Return one learning plan with its skills, exercise stats and vocabulary stats.

Each skill has mastery, uncertainty, debuff and effective_mastery (80+ is confidently mastered), answered and planned exercise counts, last practiced date and last score.
Start of the weekly planning: read this first, then list_learning_exercises and list_vocabulary_reviews.

Inputs:
- learning_plan_id: required`,
}

type GetLearningPlanInput struct {
	LearningPlanID int64 `json:"learning_plan_id" jsonschema:"Plan ID"`
}

type PlanDetailResult struct {
	PlanResult
	Skills          []SkillResult         `json:"skills" jsonschema:"Skills, oldest first"`
	ExerciseStats   ExerciseStatsResult   `json:"exercise_stats" jsonschema:"Exercise stats"`
	VocabularyStats VocabularyStatsResult `json:"vocabulary_stats" jsonschema:"Vocabulary stats"`
}

type GetLearningPlanOutput struct {
	Plan PlanDetailResult `json:"plan" jsonschema:"The plan"`
}

func GetLearningPlan(ctx context.Context, _ *mcp.CallToolRequest, input GetLearningPlanInput) (*mcp.CallToolResult, GetLearningPlanOutput, error) {
	db, userID, err := dbAndUser(ctx)
	if err != nil {
		return nil, GetLearningPlanOutput{}, err
	}

	detail, err := planDetail(ctx, db, userID, input.LearningPlanID, time.Now())
	if err != nil {
		return nil, GetLearningPlanOutput{}, err
	}

	return nil, GetLearningPlanOutput{Plan: *detail}, nil
}

// planDetail is shared by get_learning_plan and GET /web/learning/:id.
func planDetail(ctx context.Context, db gateways.DB, userID, planID int64, now time.Time) (*PlanDetailResult, error) {
	plan, err := loadPlan(ctx, db, userID, planID)
	if err != nil {
		return nil, err
	}

	skills, err := db.ListSkills(ctx, plan.ID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list skills: %w", err)
	}
	skillResults := make([]SkillResult, 0, len(skills))
	for _, s := range withSkillMastery(skills, now) {
		skillResults = append(skillResults, skillToResult(s))
	}

	exStats, err := exerciseStats(ctx, db, userID, []int64{plan.ID}, now)
	if err != nil {
		return nil, err
	}
	vocStats, err := vocabularyStats(ctx, db, userID, []int64{plan.ID}, now)
	if err != nil {
		return nil, err
	}

	return &PlanDetailResult{
		PlanResult:      planToResult(*plan),
		Skills:          skillResults,
		ExerciseStats:   exStats[plan.ID],
		VocabularyStats: vocStats[plan.ID],
	}, nil
}
