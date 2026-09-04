package goals

import (
	"context"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"personal/domain"
	"personal/gateways"
)

var GetGoalProgressMCPDefinition = mcp.Tool{
	Name: "get_goal_progress",
	Description: "Returns every goal active as of a given date (defaults to now), each with remaining_value. " +
		"current_value is a cached column, not recomputed here — call refresh_goals first if the numbers might be stale.",
}

// GetGoalProgressInput is the MCP tool input.
type GetGoalProgressInput struct {
	Date *time.Time `json:"date,omitempty"`
}

// GetGoalProgressOutput is the MCP tool output.
type GetGoalProgressOutput struct {
	Goals []GoalOutput `json:"goals"`
}

func GetGoalProgress(ctx context.Context, _ *mcp.CallToolRequest, input GetGoalProgressInput) (*mcp.CallToolResult, GetGoalProgressOutput, error) {
	db := gateways.DBFromContext(ctx)
	if db == nil {
		return nil, GetGoalProgressOutput{}, fmt.Errorf("database not available in context")
	}
	userID := gateways.UserIDFromContext(ctx)
	if userID == 0 {
		return nil, GetGoalProgressOutput{}, fmt.Errorf("user_id not available in context")
	}

	date := time.Now().UTC()
	if input.Date != nil {
		date = *input.Date
	}

	goalList, err := db.ListGoals(ctx, domain.GoalFilter{UserID: userID, ActiveOnly: true, At: date})
	if err != nil {
		return nil, GetGoalProgressOutput{}, fmt.Errorf("database error: %w", err)
	}

	out := make([]GoalOutput, len(goalList))
	for i, g := range goalList {
		out[i] = toGoalOutput(g)
	}
	return nil, GetGoalProgressOutput{Goals: out}, nil
}
