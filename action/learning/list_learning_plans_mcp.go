package learning

import (
	"context"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"personal/domain"
	"personal/gateways"
)

var ListLearningPlansMCPDefinition = mcp.Tool{
	Name: "list_learning_plans",
	Annotations: &mcp.ToolAnnotations{
		ReadOnlyHint:   true,
		IdempotentHint: true,
		Title:          "List learning plans",
	},
	Description: `List learning plans with their skill count, exercise stats (planned, due, awaiting review) and vocabulary stats (total, due, unscheduled, mastered, average mastery).

Active plans first, then paused, then completed; newest first inside a status.

Parameters:
- statuses: (optional) active, paused, completed; default active and paused`,
}

type ListLearningPlansInput struct {
	Statuses []string `json:"statuses,omitempty" jsonschema:"active, paused, completed; default active and paused"`
}

type PlanSummaryResult struct {
	PlanResult
	SkillCount      int                   `json:"skill_count" jsonschema:"Skills in the plan"`
	ExerciseStats   ExerciseStatsResult   `json:"exercise_stats" jsonschema:"Exercise stats"`
	VocabularyStats VocabularyStatsResult `json:"vocabulary_stats" jsonschema:"Vocabulary stats"`
}

type ListLearningPlansOutput struct {
	Plans []PlanSummaryResult `json:"plans" jsonschema:"Matching plans"`
}

func ListLearningPlans(ctx context.Context, _ *mcp.CallToolRequest, input ListLearningPlansInput) (*mcp.CallToolResult, ListLearningPlansOutput, error) {
	db, userID, err := dbAndUser(ctx)
	if err != nil {
		return nil, ListLearningPlansOutput{}, err
	}

	statuses, err := parsePlanStatuses(input.Statuses)
	if err != nil {
		return nil, ListLearningPlansOutput{}, err
	}

	plans, err := listPlanSummaries(ctx, db, userID, statuses, time.Now())
	if err != nil {
		return nil, ListLearningPlansOutput{}, err
	}

	return nil, ListLearningPlansOutput{Plans: plans}, nil
}

func parsePlanStatuses(raw []string) ([]domain.LearningPlanStatus, error) {
	var statuses []domain.LearningPlanStatus
	for _, s := range raw {
		status := domain.LearningPlanStatus(s)
		if !validPlanStatuses[status] {
			return nil, fmt.Errorf("invalid status %q", s)
		}
		statuses = append(statuses, status)
	}
	return statuses, nil
}

// listPlanSummaries is shared by list_learning_plans and GET /web/learning.
func listPlanSummaries(ctx context.Context, db gateways.DB, userID int64, statuses []domain.LearningPlanStatus, now time.Time) ([]PlanSummaryResult, error) {
	plans, err := db.ListLearningPlans(ctx, domain.LearningPlanFilter{UserID: userID, Statuses: statuses})
	if err != nil {
		return nil, fmt.Errorf("failed to list learning plans: %w", err)
	}

	planIDs := make([]int64, len(plans))
	for i, p := range plans {
		planIDs[i] = p.ID
	}

	exStats, err := exerciseStats(ctx, db, userID, planIDs, now)
	if err != nil {
		return nil, err
	}
	vocStats, err := vocabularyStats(ctx, db, userID, planIDs, now)
	if err != nil {
		return nil, err
	}

	results := make([]PlanSummaryResult, len(plans))
	for i, p := range plans {
		skills, err := db.ListSkills(ctx, p.ID, userID)
		if err != nil {
			return nil, fmt.Errorf("failed to list skills: %w", err)
		}
		results[i] = PlanSummaryResult{
			PlanResult:      planToResult(p),
			SkillCount:      len(skills),
			ExerciseStats:   exStats[p.ID],
			VocabularyStats: vocStats[p.ID],
		}
	}
	return results, nil
}
