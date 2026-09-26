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

var CreateAchievementMCPDefinition = mcp.Tool{
	Name: "create_achievement",
	Description: "Create a new achievement — one of seven types (money_saving, money_spend, exercise_max_weight, " +
		"exercise_total_volume, activity_occurrence_count, activity_streak_count, manual), each with its own " +
		"required fields. For the six non-manual types, also computes and stores an initial current_value " +
		"snapshot so the achievement isn't stuck at zero until the first refresh.",
	Annotations: &mcp.ToolAnnotations{
		DestructiveHint: util.Ptr(false),
		Title:           "Create achievement",
	},
}

// CreateAchievementInput is the MCP tool input.
type CreateAchievementInput struct {
	Name            string     `json:"name"`
	AchievementType string     `json:"achievement_type"`
	TargetValue     float64    `json:"target_value"`
	Category        *string    `json:"category,omitempty"`    // required for money_spend, rejected otherwise
	Unit            *string    `json:"unit,omitempty"`        // optional for exercise_max_weight|exercise_total_volume|activity_occurrence_count|activity_streak_count|manual, rejected otherwise
	ExerciseID      *int64     `json:"exercise_id,omitempty"` // required for exercise_max_weight|exercise_total_volume
	ActivityID      *int64     `json:"activity_id,omitempty"` // required for activity_occurrence_count|activity_streak_count
	StartsAt        *time.Time `json:"starts_at,omitempty"`   // defaults to now
	EndsAt          *time.Time `json:"ends_at,omitempty"`     // omit for no deadline
}

// CreateAchievementOutput is the MCP tool output.
type CreateAchievementOutput struct {
	ID          int64             `json:"id"`
	Achievement AchievementOutput `json:"achievement"`
	Error       string            `json:"error,omitempty"`
}

func CreateAchievement(ctx context.Context, _ *mcp.CallToolRequest, input CreateAchievementInput) (*mcp.CallToolResult, CreateAchievementOutput, error) {
	db := gateways.DBFromContext(ctx)
	if db == nil {
		return nil, CreateAchievementOutput{}, fmt.Errorf("database not available in context")
	}
	userID := gateways.UserIDFromContext(ctx)
	if userID == 0 {
		return nil, CreateAchievementOutput{}, fmt.Errorf("user_id not available in context")
	}

	achievementType := domain.AchievementType(input.AchievementType)
	if !achievementType.IsValid() {
		return nil, CreateAchievementOutput{Error: fmt.Sprintf("unknown achievement_type %q", input.AchievementType)}, nil
	}
	if input.TargetValue <= 0 {
		return nil, CreateAchievementOutput{Error: "target_value must be greater than 0"}, nil
	}

	startsAt := time.Now().UTC()
	if input.StartsAt != nil {
		startsAt = *input.StartsAt
	}
	if input.EndsAt != nil && !input.EndsAt.After(startsAt) {
		return nil, CreateAchievementOutput{Error: "ends_at must be after starts_at"}, nil
	}

	if categoryApplicable(achievementType) {
		if input.Category == nil || *input.Category == "" {
			return nil, CreateAchievementOutput{Error: fmt.Sprintf("category is required for %s achievements", achievementType)}, nil
		}
	} else if input.Category != nil {
		return nil, CreateAchievementOutput{Error: fmt.Sprintf("category is not applicable to %s achievements", achievementType)}, nil
	}

	if !unitApplicable(achievementType) && input.Unit != nil {
		return nil, CreateAchievementOutput{Error: fmt.Sprintf("unit is not applicable to %s achievements", achievementType)}, nil
	}

	if exerciseApplicable(achievementType) {
		if input.ExerciseID == nil {
			return nil, CreateAchievementOutput{Error: fmt.Sprintf("exercise_id is required for %s achievements", achievementType)}, nil
		}
		exercise, err := db.GetExercise(ctx, *input.ExerciseID, userID)
		if err != nil {
			return nil, CreateAchievementOutput{}, fmt.Errorf("database error: %w", err)
		}
		if exercise == nil {
			return nil, CreateAchievementOutput{Error: "exercise not found"}, nil
		}
	} else if input.ExerciseID != nil {
		return nil, CreateAchievementOutput{Error: fmt.Sprintf("exercise_id is not applicable to %s achievements", achievementType)}, nil
	}

	if activityApplicable(achievementType) {
		if input.ActivityID == nil {
			return nil, CreateAchievementOutput{Error: fmt.Sprintf("activity_id is required for %s achievements", achievementType)}, nil
		}
		activity, err := db.GetActivity(ctx, *input.ActivityID, userID)
		if err != nil {
			return nil, CreateAchievementOutput{}, fmt.Errorf("database error: %w", err)
		}
		if activity == nil {
			return nil, CreateAchievementOutput{Error: "activity not found"}, nil
		}
	} else if input.ActivityID != nil {
		return nil, CreateAchievementOutput{Error: fmt.Sprintf("activity_id is not applicable to %s achievements", achievementType)}, nil
	}

	g := domain.Achievement{
		UserID:          userID,
		Name:            input.Name,
		AchievementType: achievementType,
		ExerciseID:      input.ExerciseID,
		ActivityID:      input.ActivityID,
		TargetValue:     input.TargetValue,
		Category:        input.Category,
		Unit:            input.Unit,
		StartsAt:        startsAt,
		EndsAt:          input.EndsAt,
	}

	if achievementType == domain.AchievementTypeMoneySaving {
		baseline, err := moneyBalanceSince(ctx, db, userID, startsAt)
		if err != nil {
			return nil, CreateAchievementOutput{}, fmt.Errorf("database error: %w", err)
		}
		g.BaselineBalanceEUR = &baseline
	}

	if achievementType == domain.AchievementTypeManual {
		g.CurrentValue = 0
	} else {
		now := time.Now().UTC()
		currentValue, err := computeCurrentValue(ctx, db, userID, g, now)
		if err != nil {
			return nil, CreateAchievementOutput{}, fmt.Errorf("database error: %w", err)
		}
		g.CurrentValue = currentValue
	}

	id, err := db.CreateAchievement(ctx, &g)
	if err != nil {
		return nil, CreateAchievementOutput{}, fmt.Errorf("database error: %w", err)
	}
	g.ID = id

	return nil, CreateAchievementOutput{ID: id, Achievement: toAchievementOutput(g)}, nil
}
