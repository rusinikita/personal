package achievements

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"personal/domain"
	"personal/gateways"
	"personal/util"
)

var LogAchievementProgressMCPDefinition = mcp.Tool{
	Name:        "log_achievement_progress",
	Description: "Increments a manual achievement's current_value by delta (defaults to 1), returning the new value.",
	Annotations: &mcp.ToolAnnotations{
		DestructiveHint: util.Ptr(false),
		Title:           "Log achievement progress",
	},
}

// LogAchievementProgressInput is the MCP tool input.
type LogAchievementProgressInput struct {
	AchievementID int64    `json:"achievement_id"`
	Delta         *float64 `json:"delta,omitempty"` // defaults to 1
}

// LogAchievementProgressOutput is the MCP tool output.
type LogAchievementProgressOutput struct {
	Achievement AchievementOutput `json:"achievement"`
	Error       string            `json:"error,omitempty"`
}

func LogAchievementProgress(ctx context.Context, _ *mcp.CallToolRequest, input LogAchievementProgressInput) (*mcp.CallToolResult, LogAchievementProgressOutput, error) {
	db := gateways.DBFromContext(ctx)
	if db == nil {
		return nil, LogAchievementProgressOutput{}, fmt.Errorf("database not available in context")
	}
	userID := gateways.UserIDFromContext(ctx)
	if userID == 0 {
		return nil, LogAchievementProgressOutput{}, fmt.Errorf("user_id not available in context")
	}

	achievement, err := db.GetAchievement(ctx, input.AchievementID, userID)
	if err != nil {
		return nil, LogAchievementProgressOutput{}, fmt.Errorf("database error: %w", err)
	}
	if achievement == nil {
		return nil, LogAchievementProgressOutput{Error: "achievement not found"}, nil
	}
	if achievement.AchievementType != domain.AchievementTypeManual {
		return nil, LogAchievementProgressOutput{Error: "log_achievement_progress only applies to manual achievements"}, nil
	}

	delta := 1.0
	if input.Delta != nil {
		delta = *input.Delta
	}
	newValue := achievement.CurrentValue + delta

	if err := db.UpdateAchievement(ctx, userID, domain.AchievementUpdate{ID: input.AchievementID, CurrentValue: &newValue}); err != nil {
		return nil, LogAchievementProgressOutput{}, fmt.Errorf("database error: %w", err)
	}
	achievement.CurrentValue = newValue

	return nil, LogAchievementProgressOutput{Achievement: toAchievementOutput(*achievement)}, nil
}
