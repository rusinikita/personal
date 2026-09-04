package goals

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"personal/domain"
	"personal/gateways"
	"personal/util"
)

var LogGoalProgressMCPDefinition = mcp.Tool{
	Name:        "log_goal_progress",
	Description: "Increments a manual goal's current_value by delta (defaults to 1), returning the new value.",
	Annotations: &mcp.ToolAnnotations{
		DestructiveHint: util.Ptr(false),
		Title:           "Log goal progress",
	},
}

// LogGoalProgressInput is the MCP tool input.
type LogGoalProgressInput struct {
	GoalID int64    `json:"goal_id"`
	Delta  *float64 `json:"delta,omitempty"` // defaults to 1
}

// LogGoalProgressOutput is the MCP tool output.
type LogGoalProgressOutput struct {
	Goal  GoalOutput `json:"goal"`
	Error string     `json:"error,omitempty"`
}

func LogGoalProgress(ctx context.Context, _ *mcp.CallToolRequest, input LogGoalProgressInput) (*mcp.CallToolResult, LogGoalProgressOutput, error) {
	db := gateways.DBFromContext(ctx)
	if db == nil {
		return nil, LogGoalProgressOutput{}, fmt.Errorf("database not available in context")
	}
	userID := gateways.UserIDFromContext(ctx)
	if userID == 0 {
		return nil, LogGoalProgressOutput{}, fmt.Errorf("user_id not available in context")
	}

	goal, err := db.GetGoal(ctx, input.GoalID, userID)
	if err != nil {
		return nil, LogGoalProgressOutput{}, fmt.Errorf("database error: %w", err)
	}
	if goal == nil {
		return nil, LogGoalProgressOutput{Error: "goal not found"}, nil
	}
	if goal.GoalType != domain.GoalTypeManual {
		return nil, LogGoalProgressOutput{Error: "log_goal_progress only applies to manual goals"}, nil
	}

	delta := 1.0
	if input.Delta != nil {
		delta = *input.Delta
	}
	newValue := goal.CurrentValue + delta

	if err := db.UpdateGoal(ctx, userID, domain.GoalUpdate{ID: input.GoalID, CurrentValue: &newValue}); err != nil {
		return nil, LogGoalProgressOutput{}, fmt.Errorf("database error: %w", err)
	}
	goal.CurrentValue = newValue

	return nil, LogGoalProgressOutput{Goal: toGoalOutput(*goal)}, nil
}
