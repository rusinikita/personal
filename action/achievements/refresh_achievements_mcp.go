package achievements

import (
	"context"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"personal/domain"
	"personal/gateways"
)

var RefreshAchievementsMCPDefinition = mcp.Tool{
	Name: "refresh_achievements",
	Description: "Recompute and persist current_value for the six derived achievement types (everything but manual) — " +
		"the same per-achievement_type calculation create_achievement uses for its initial snapshot, run again on demand. " +
		"Omit achievement_id to refresh every active achievement owned by the user; provide it to refresh just that one achievement.",
}

// RefreshAchievementsInput is the MCP tool input.
type RefreshAchievementsInput struct {
	AchievementID *int64 `json:"achievement_id,omitempty"` // omitted = every active achievement
}

// RefreshAchievementsOutput is the MCP tool output. Refreshed is always the count
// of achievements actually recomputed; Achievement is set only when AchievementID was given.
type RefreshAchievementsOutput struct {
	Refreshed   int                `json:"refreshed"`
	Achievement *AchievementOutput `json:"achievement,omitempty"`
	Error       string             `json:"error,omitempty"`
}

// refreshOne recomputes and persists current_value for one non-manual achievement,
// returning the value it wrote.
func refreshOne(ctx context.Context, db gateways.DB, userID int64, g domain.Achievement, now time.Time) (float64, error) {
	currentValue, err := computeCurrentValue(ctx, db, userID, g, now)
	if err != nil {
		return 0, err
	}
	if err := db.UpdateAchievement(ctx, userID, domain.AchievementUpdate{ID: g.ID, CurrentValue: &currentValue}); err != nil {
		return 0, err
	}
	return currentValue, nil
}

func RefreshAchievements(ctx context.Context, _ *mcp.CallToolRequest, input RefreshAchievementsInput) (*mcp.CallToolResult, RefreshAchievementsOutput, error) {
	db := gateways.DBFromContext(ctx)
	if db == nil {
		return nil, RefreshAchievementsOutput{}, fmt.Errorf("database not available in context")
	}
	userID := gateways.UserIDFromContext(ctx)
	if userID == 0 {
		return nil, RefreshAchievementsOutput{}, fmt.Errorf("user_id not available in context")
	}

	now := time.Now().UTC()

	if input.AchievementID != nil {
		achievement, err := db.GetAchievement(ctx, *input.AchievementID, userID)
		if err != nil {
			return nil, RefreshAchievementsOutput{}, fmt.Errorf("database error: %w", err)
		}
		if achievement == nil {
			return nil, RefreshAchievementsOutput{Error: "achievement not found"}, nil
		}
		if achievement.AchievementType == domain.AchievementTypeManual {
			return nil, RefreshAchievementsOutput{Error: "manual achievements have no derived progress to refresh; use log_achievement_progress or update_achievement instead"}, nil
		}

		currentValue, err := refreshOne(ctx, db, userID, *achievement, now)
		if err != nil {
			return nil, RefreshAchievementsOutput{}, fmt.Errorf("database error: %w", err)
		}
		achievement.CurrentValue = currentValue
		out := toAchievementOutput(*achievement)
		return nil, RefreshAchievementsOutput{Refreshed: 1, Achievement: &out}, nil
	}

	refreshed, err := refreshAllActive(ctx, db, userID, now)
	if err != nil {
		return nil, RefreshAchievementsOutput{}, fmt.Errorf("database error: %w", err)
	}

	return nil, RefreshAchievementsOutput{Refreshed: refreshed}, nil
}

// refreshAllActive recomputes and persists current_value for every active,
// non-manual achievement owned by userID — shared by refresh_achievements (achievement_id
// omitted) and POST /web/achievements/refresh (see dashboard_web.go).
func refreshAllActive(ctx context.Context, db gateways.DB, userID int64, now time.Time) (int, error) {
	achievementList, err := db.ListAchievements(ctx, domain.AchievementFilter{UserID: userID, ActiveOnly: true})
	if err != nil {
		return 0, err
	}
	refreshed := 0
	for _, g := range achievementList {
		if g.AchievementType == domain.AchievementTypeManual {
			continue
		}
		if _, err := refreshOne(ctx, db, userID, g, now); err != nil {
			return 0, err
		}
		refreshed++
	}
	return refreshed, nil
}
