package learning

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"personal/domain"
)

var CreateLearningPlanMCPDefinition = mcp.Tool{
	Name:        "create_learning_plan",
	Annotations: &mcp.ToolAnnotations{Title: "Create a learning plan"},
	Description: `Create a learning plan (a course or track, e.g. "Greek A1", "Kubernetes fundamentals") in status active.

Inputs:
- name: required
- description: (optional) the goal of the plan`,
}

type CreateLearningPlanInput struct {
	Name        string  `json:"name" jsonschema:"Plan name"`
	Description *string `json:"description,omitempty" jsonschema:"Goal of the plan"`
}

type CreateLearningPlanOutput struct {
	Plan PlanResult `json:"plan" jsonschema:"Created plan"`
}

func CreateLearningPlan(ctx context.Context, _ *mcp.CallToolRequest, input CreateLearningPlanInput) (*mcp.CallToolResult, CreateLearningPlanOutput, error) {
	db, userID, err := dbAndUser(ctx)
	if err != nil {
		return nil, CreateLearningPlanOutput{}, err
	}

	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, CreateLearningPlanOutput{}, fmt.Errorf("name is required")
	}
	if len([]rune(name)) > maxTextLength {
		return nil, CreateLearningPlanOutput{}, fmt.Errorf("name must be at most %d characters", maxTextLength)
	}

	id, err := db.CreateLearningPlan(ctx, &domain.LearningPlan{
		UserID:      userID,
		Name:        name,
		Description: optionalText(input.Description),
		Status:      domain.LearningPlanActive,
	})
	if err != nil {
		return nil, CreateLearningPlanOutput{}, fmt.Errorf("failed to create learning plan: %w", err)
	}

	plan, err := loadPlan(ctx, db, userID, id)
	if err != nil {
		return nil, CreateLearningPlanOutput{}, err
	}

	return nil, CreateLearningPlanOutput{Plan: planToResult(*plan)}, nil
}
