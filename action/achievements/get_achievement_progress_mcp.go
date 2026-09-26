package achievements

import (
	"context"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"personal/domain"
	"personal/gateways"
)

var GetAchievementProgressMCPDefinition = mcp.Tool{
	Name: "get_achievement_progress",
	Description: "Returns every achievement active as of a given date (defaults to now), each with remaining_value. " +
		"current_value is a cached column, not recomputed here — call refresh_achievements first if the numbers might be stale.",
}

// GetAchievementProgressInput is the MCP tool input.
type GetAchievementProgressInput struct {
	Date *time.Time `json:"date,omitempty"`
}

// GetAchievementProgressOutput is the MCP tool output.
type GetAchievementProgressOutput struct {
	Achievements []AchievementOutput `json:"achievements"`
}

func GetAchievementProgress(ctx context.Context, _ *mcp.CallToolRequest, input GetAchievementProgressInput) (*mcp.CallToolResult, GetAchievementProgressOutput, error) {
	db := gateways.DBFromContext(ctx)
	if db == nil {
		return nil, GetAchievementProgressOutput{}, fmt.Errorf("database not available in context")
	}
	userID := gateways.UserIDFromContext(ctx)
	if userID == 0 {
		return nil, GetAchievementProgressOutput{}, fmt.Errorf("user_id not available in context")
	}

	date := time.Now().UTC()
	if input.Date != nil {
		date = *input.Date
	}

	achievementList, err := db.ListAchievements(ctx, domain.AchievementFilter{UserID: userID, ActiveOnly: true, At: date})
	if err != nil {
		return nil, GetAchievementProgressOutput{}, fmt.Errorf("database error: %w", err)
	}

	out := make([]AchievementOutput, len(achievementList))
	for i, g := range achievementList {
		out[i] = toAchievementOutput(g)
	}
	return nil, GetAchievementProgressOutput{Achievements: out}, nil
}
