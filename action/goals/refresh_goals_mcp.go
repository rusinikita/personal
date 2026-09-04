package goals

import (
	"context"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"personal/domain"
	"personal/gateways"
)

var RefreshGoalsMCPDefinition = mcp.Tool{
	Name: "refresh_goals",
	Description: "Recompute and persist current_value for the six derived goal types (everything but manual) — " +
		"the same per-goal_type calculation create_goal uses for its initial snapshot, run again on demand. " +
		"Omit goal_id to refresh every active goal owned by the user; provide it to refresh just that one goal.",
}

// RefreshGoalsInput is the MCP tool input.
type RefreshGoalsInput struct {
	GoalID *int64 `json:"goal_id,omitempty"` // omitted = every active goal
}

// RefreshGoalsOutput is the MCP tool output. Refreshed is always the count
// of goals actually recomputed; Goal is set only when GoalID was given.
type RefreshGoalsOutput struct {
	Refreshed int         `json:"refreshed"`
	Goal      *GoalOutput `json:"goal,omitempty"`
	Error     string      `json:"error,omitempty"`
}

// refreshOne recomputes and persists current_value for one non-manual goal,
// returning the value it wrote.
func refreshOne(ctx context.Context, db gateways.DB, userID int64, g domain.Goal, now time.Time) (float64, error) {
	currentValue, err := computeCurrentValue(ctx, db, userID, g, now)
	if err != nil {
		return 0, err
	}
	if err := db.UpdateGoal(ctx, userID, domain.GoalUpdate{ID: g.ID, CurrentValue: &currentValue}); err != nil {
		return 0, err
	}
	return currentValue, nil
}

func RefreshGoals(ctx context.Context, _ *mcp.CallToolRequest, input RefreshGoalsInput) (*mcp.CallToolResult, RefreshGoalsOutput, error) {
	db := gateways.DBFromContext(ctx)
	if db == nil {
		return nil, RefreshGoalsOutput{}, fmt.Errorf("database not available in context")
	}
	userID := gateways.UserIDFromContext(ctx)
	if userID == 0 {
		return nil, RefreshGoalsOutput{}, fmt.Errorf("user_id not available in context")
	}

	now := time.Now().UTC()

	if input.GoalID != nil {
		goal, err := db.GetGoal(ctx, *input.GoalID, userID)
		if err != nil {
			return nil, RefreshGoalsOutput{}, fmt.Errorf("database error: %w", err)
		}
		if goal == nil {
			return nil, RefreshGoalsOutput{Error: "goal not found"}, nil
		}
		if goal.GoalType == domain.GoalTypeManual {
			return nil, RefreshGoalsOutput{Error: "manual goals have no derived progress to refresh; use log_goal_progress or update_goal instead"}, nil
		}

		currentValue, err := refreshOne(ctx, db, userID, *goal, now)
		if err != nil {
			return nil, RefreshGoalsOutput{}, fmt.Errorf("database error: %w", err)
		}
		goal.CurrentValue = currentValue
		out := toGoalOutput(*goal)
		return nil, RefreshGoalsOutput{Refreshed: 1, Goal: &out}, nil
	}

	refreshed, err := refreshAllActive(ctx, db, userID, now)
	if err != nil {
		return nil, RefreshGoalsOutput{}, fmt.Errorf("database error: %w", err)
	}

	return nil, RefreshGoalsOutput{Refreshed: refreshed}, nil
}

// refreshAllActive recomputes and persists current_value for every active,
// non-manual goal owned by userID — shared by refresh_goals (goal_id
// omitted) and POST /web/goals/refresh (see dashboard_web.go).
func refreshAllActive(ctx context.Context, db gateways.DB, userID int64, now time.Time) (int, error) {
	goalList, err := db.ListGoals(ctx, domain.GoalFilter{UserID: userID, ActiveOnly: true})
	if err != nil {
		return 0, err
	}
	refreshed := 0
	for _, g := range goalList {
		if g.GoalType == domain.GoalTypeManual {
			continue
		}
		if _, err := refreshOne(ctx, db, userID, g, now); err != nil {
			return 0, err
		}
		refreshed++
	}
	return refreshed, nil
}
