package achievements

import (
	"context"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"personal/domain"
	"personal/gateways"
	"personal/util"
)

var UpdateAchievementMCPDefinition = mcp.Tool{
	Name: "update_achievement",
	Description: "Edit mutable fields of an existing achievement by ID — name, target_value, ends_at (or clear_ends_at " +
		"to remove the deadline), and, when they apply to the achievement's own achievement_type, category (money_spend only) " +
		"and unit. current_value is only editable this way for manual achievements — for the other six, use refresh_achievements " +
		"to bring it up to date instead.",
	Annotations: &mcp.ToolAnnotations{
		DestructiveHint: util.Ptr(true),
		Title:           "Update achievement",
	},
}

// UpdateAchievementInput is the MCP tool input. All fields but AchievementID are optional
// — nil means unchanged.
type UpdateAchievementInput struct {
	AchievementID int64      `json:"achievement_id"`
	Name          *string    `json:"name,omitempty"`
	TargetValue   *float64   `json:"target_value,omitempty"`
	EndsAt        *time.Time `json:"ends_at,omitempty"`
	ClearEndsAt   bool       `json:"clear_ends_at,omitempty"`
	Category      *string    `json:"category,omitempty"`      // money_spend only
	Unit          *string    `json:"unit,omitempty"`          // applicable achievement types only
	CurrentValue  *float64   `json:"current_value,omitempty"` // manual only
}

// UpdateAchievementOutput is the MCP tool output.
type UpdateAchievementOutput struct {
	Achievement AchievementOutput `json:"achievement"`
	Error       string            `json:"error,omitempty"`
}

func UpdateAchievement(ctx context.Context, _ *mcp.CallToolRequest, input UpdateAchievementInput) (*mcp.CallToolResult, UpdateAchievementOutput, error) {
	db := gateways.DBFromContext(ctx)
	if db == nil {
		return nil, UpdateAchievementOutput{}, fmt.Errorf("database not available in context")
	}
	userID := gateways.UserIDFromContext(ctx)
	if userID == 0 {
		return nil, UpdateAchievementOutput{}, fmt.Errorf("user_id not available in context")
	}

	achievement, err := db.GetAchievement(ctx, input.AchievementID, userID)
	if err != nil {
		return nil, UpdateAchievementOutput{}, fmt.Errorf("database error: %w", err)
	}
	if achievement == nil {
		return nil, UpdateAchievementOutput{Error: "achievement not found"}, nil
	}

	if input.TargetValue != nil && *input.TargetValue <= 0 {
		return nil, UpdateAchievementOutput{Error: "target_value must be greater than 0"}, nil
	}
	if input.EndsAt != nil && !input.EndsAt.After(achievement.StartsAt) {
		return nil, UpdateAchievementOutput{Error: "ends_at must be after starts_at"}, nil
	}

	if input.Category != nil {
		if !categoryApplicable(achievement.AchievementType) {
			return nil, UpdateAchievementOutput{Error: "category can only be set on money_spend achievements"}, nil
		}
		if *input.Category == "" {
			return nil, UpdateAchievementOutput{Error: "category is required for money_spend achievements"}, nil
		}
	}
	if input.Unit != nil && !unitApplicable(achievement.AchievementType) {
		return nil, UpdateAchievementOutput{Error: fmt.Sprintf("unit is not applicable to %s achievements", achievement.AchievementType)}, nil
	}
	if input.CurrentValue != nil && achievement.AchievementType != domain.AchievementTypeManual {
		return nil, UpdateAchievementOutput{Error: "current_value can only be corrected on manual achievements"}, nil
	}

	update := domain.AchievementUpdate{
		ID:           input.AchievementID,
		Name:         input.Name,
		TargetValue:  input.TargetValue,
		EndsAt:       input.EndsAt,
		ClearEndsAt:  input.ClearEndsAt,
		Category:     input.Category,
		Unit:         input.Unit,
		CurrentValue: input.CurrentValue,
	}
	if err := db.UpdateAchievement(ctx, userID, update); err != nil {
		return nil, UpdateAchievementOutput{}, fmt.Errorf("database error: %w", err)
	}

	updated, err := db.GetAchievement(ctx, input.AchievementID, userID)
	if err != nil {
		return nil, UpdateAchievementOutput{}, fmt.Errorf("database error: %w", err)
	}
	return nil, UpdateAchievementOutput{Achievement: toAchievementOutput(*updated)}, nil
}
