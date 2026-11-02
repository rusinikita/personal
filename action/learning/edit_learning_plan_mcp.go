package learning

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"personal/domain"
)

var EditLearningPlanMCPDefinition = mcp.Tool{
	Name:        "edit_learning_plan",
	Annotations: &mcp.ToolAnnotations{Title: "Edit a learning plan"},
	Description: `Partial update of a learning plan: only the provided fields change, at least one is required.

Inputs:
- learning_plan_id: required
- name: (optional) new name, not empty
- description: (optional) new goal; "" clears it
- status: (optional) active, paused or completed; any status can change to any other. A completed plan leaves the default lists`,
}

type EditLearningPlanInput struct {
	LearningPlanID int64   `json:"learning_plan_id" jsonschema:"Plan ID"`
	Name           *string `json:"name,omitempty" jsonschema:"New name"`
	Description    *string `json:"description,omitempty" jsonschema:"New goal; empty string clears it"`
	Status         *string `json:"status,omitempty" jsonschema:"active, paused or completed"`
}

type EditLearningPlanOutput struct {
	Plan PlanResult `json:"plan" jsonschema:"Updated plan"`
}

var validPlanStatuses = map[domain.LearningPlanStatus]bool{
	domain.LearningPlanActive:    true,
	domain.LearningPlanPaused:    true,
	domain.LearningPlanCompleted: true,
}

func EditLearningPlan(ctx context.Context, _ *mcp.CallToolRequest, input EditLearningPlanInput) (*mcp.CallToolResult, EditLearningPlanOutput, error) {
	db, userID, err := dbAndUser(ctx)
	if err != nil {
		return nil, EditLearningPlanOutput{}, err
	}

	if input.Name == nil && input.Description == nil && input.Status == nil {
		return nil, EditLearningPlanOutput{}, fmt.Errorf("at least one of name, description or status is required")
	}

	plan, err := loadPlan(ctx, db, userID, input.LearningPlanID)
	if err != nil {
		return nil, EditLearningPlanOutput{}, err
	}

	if input.Name != nil {
		name := strings.TrimSpace(*input.Name)
		if name == "" {
			return nil, EditLearningPlanOutput{}, fmt.Errorf("name cannot be empty")
		}
		if len([]rune(name)) > maxTextLength {
			return nil, EditLearningPlanOutput{}, fmt.Errorf("name must be at most %d characters", maxTextLength)
		}
		plan.Name = name
	}
	if input.Description != nil {
		plan.Description = optionalText(input.Description)
	}
	if input.Status != nil {
		status := domain.LearningPlanStatus(*input.Status)
		if !validPlanStatuses[status] {
			return nil, EditLearningPlanOutput{}, fmt.Errorf("invalid status %q", *input.Status)
		}
		plan.Status = status
	}

	if err := db.UpdateLearningPlan(ctx, plan); err != nil {
		return nil, EditLearningPlanOutput{}, fmt.Errorf("failed to update learning plan: %w", err)
	}

	plan, err = loadPlan(ctx, db, userID, plan.ID)
	if err != nil {
		return nil, EditLearningPlanOutput{}, err
	}

	return nil, EditLearningPlanOutput{Plan: planToResult(*plan)}, nil
}
