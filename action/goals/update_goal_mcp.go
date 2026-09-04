package goals

import (
	"context"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"personal/domain"
	"personal/gateways"
	"personal/util"
)

var UpdateGoalMCPDefinition = mcp.Tool{
	Name: "update_goal",
	Description: "Edit mutable fields of an existing goal by ID — name, target_value, ends_at (or clear_ends_at " +
		"to remove the deadline), and, when they apply to the goal's own goal_type, category (money_spend only) " +
		"and unit. current_value is only editable this way for manual goals — for the other six, use refresh_goals " +
		"to bring it up to date instead.",
	Annotations: &mcp.ToolAnnotations{
		DestructiveHint: util.Ptr(true),
		Title:           "Update goal",
	},
}

// UpdateGoalInput is the MCP tool input. All fields but GoalID are optional
// — nil means unchanged.
type UpdateGoalInput struct {
	GoalID       int64      `json:"goal_id"`
	Name         *string    `json:"name,omitempty"`
	TargetValue  *float64   `json:"target_value,omitempty"`
	EndsAt       *time.Time `json:"ends_at,omitempty"`
	ClearEndsAt  bool       `json:"clear_ends_at,omitempty"`
	Category     *string    `json:"category,omitempty"`      // money_spend only
	Unit         *string    `json:"unit,omitempty"`          // applicable goal types only
	CurrentValue *float64   `json:"current_value,omitempty"` // manual only
}

// UpdateGoalOutput is the MCP tool output.
type UpdateGoalOutput struct {
	Goal  GoalOutput `json:"goal"`
	Error string     `json:"error,omitempty"`
}

func UpdateGoal(ctx context.Context, _ *mcp.CallToolRequest, input UpdateGoalInput) (*mcp.CallToolResult, UpdateGoalOutput, error) {
	db := gateways.DBFromContext(ctx)
	if db == nil {
		return nil, UpdateGoalOutput{}, fmt.Errorf("database not available in context")
	}
	userID := gateways.UserIDFromContext(ctx)
	if userID == 0 {
		return nil, UpdateGoalOutput{}, fmt.Errorf("user_id not available in context")
	}

	goal, err := db.GetGoal(ctx, input.GoalID, userID)
	if err != nil {
		return nil, UpdateGoalOutput{}, fmt.Errorf("database error: %w", err)
	}
	if goal == nil {
		return nil, UpdateGoalOutput{Error: "goal not found"}, nil
	}

	if input.TargetValue != nil && *input.TargetValue <= 0 {
		return nil, UpdateGoalOutput{Error: "target_value must be greater than 0"}, nil
	}
	if input.EndsAt != nil && !input.EndsAt.After(goal.StartsAt) {
		return nil, UpdateGoalOutput{Error: "ends_at must be after starts_at"}, nil
	}

	if input.Category != nil {
		if !categoryApplicable(goal.GoalType) {
			return nil, UpdateGoalOutput{Error: "category can only be set on money_spend goals"}, nil
		}
		if *input.Category == "" {
			return nil, UpdateGoalOutput{Error: "category is required for money_spend goals"}, nil
		}
	}
	if input.Unit != nil && !unitApplicable(goal.GoalType) {
		return nil, UpdateGoalOutput{Error: fmt.Sprintf("unit is not applicable to %s goals", goal.GoalType)}, nil
	}
	if input.CurrentValue != nil && goal.GoalType != domain.GoalTypeManual {
		return nil, UpdateGoalOutput{Error: "current_value can only be corrected on manual goals"}, nil
	}

	update := domain.GoalUpdate{
		ID:           input.GoalID,
		Name:         input.Name,
		TargetValue:  input.TargetValue,
		EndsAt:       input.EndsAt,
		ClearEndsAt:  input.ClearEndsAt,
		Category:     input.Category,
		Unit:         input.Unit,
		CurrentValue: input.CurrentValue,
	}
	if err := db.UpdateGoal(ctx, userID, update); err != nil {
		return nil, UpdateGoalOutput{}, fmt.Errorf("database error: %w", err)
	}

	updated, err := db.GetGoal(ctx, input.GoalID, userID)
	if err != nil {
		return nil, UpdateGoalOutput{}, fmt.Errorf("database error: %w", err)
	}
	return nil, UpdateGoalOutput{Goal: toGoalOutput(*updated)}, nil
}
