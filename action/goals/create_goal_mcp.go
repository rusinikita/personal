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

var CreateGoalMCPDefinition = mcp.Tool{
	Name: "create_goal",
	Description: "Create a new goal — one of seven types (money_saving, money_spend, exercise_max_weight, " +
		"exercise_total_volume, activity_occurrence_count, activity_streak_count, manual), each with its own " +
		"required fields. For the six non-manual types, also computes and stores an initial current_value " +
		"snapshot so the goal isn't stuck at zero until the first refresh.",
	Annotations: &mcp.ToolAnnotations{
		DestructiveHint: util.Ptr(false),
		Title:           "Create goal",
	},
}

// CreateGoalInput is the MCP tool input.
type CreateGoalInput struct {
	Name        string     `json:"name"`
	GoalType    string     `json:"goal_type"`
	TargetValue float64    `json:"target_value"`
	Category    *string    `json:"category,omitempty"`    // required for money_spend, rejected otherwise
	Unit        *string    `json:"unit,omitempty"`        // optional for exercise_max_weight|exercise_total_volume|activity_occurrence_count|activity_streak_count|manual, rejected otherwise
	ExerciseID  *int64     `json:"exercise_id,omitempty"` // required for exercise_max_weight|exercise_total_volume
	ActivityID  *int64     `json:"activity_id,omitempty"` // required for activity_occurrence_count|activity_streak_count
	StartsAt    *time.Time `json:"starts_at,omitempty"`   // defaults to now
	EndsAt      *time.Time `json:"ends_at,omitempty"`     // omit for no deadline
}

// CreateGoalOutput is the MCP tool output.
type CreateGoalOutput struct {
	ID    int64      `json:"id"`
	Goal  GoalOutput `json:"goal"`
	Error string     `json:"error,omitempty"`
}

func CreateGoal(ctx context.Context, _ *mcp.CallToolRequest, input CreateGoalInput) (*mcp.CallToolResult, CreateGoalOutput, error) {
	db := gateways.DBFromContext(ctx)
	if db == nil {
		return nil, CreateGoalOutput{}, fmt.Errorf("database not available in context")
	}
	userID := gateways.UserIDFromContext(ctx)
	if userID == 0 {
		return nil, CreateGoalOutput{}, fmt.Errorf("user_id not available in context")
	}

	goalType := domain.GoalType(input.GoalType)
	if !goalType.IsValid() {
		return nil, CreateGoalOutput{Error: fmt.Sprintf("unknown goal_type %q", input.GoalType)}, nil
	}
	if input.TargetValue <= 0 {
		return nil, CreateGoalOutput{Error: "target_value must be greater than 0"}, nil
	}

	startsAt := time.Now().UTC()
	if input.StartsAt != nil {
		startsAt = *input.StartsAt
	}
	if input.EndsAt != nil && !input.EndsAt.After(startsAt) {
		return nil, CreateGoalOutput{Error: "ends_at must be after starts_at"}, nil
	}

	if categoryApplicable(goalType) {
		if input.Category == nil || *input.Category == "" {
			return nil, CreateGoalOutput{Error: fmt.Sprintf("category is required for %s goals", goalType)}, nil
		}
	} else if input.Category != nil {
		return nil, CreateGoalOutput{Error: fmt.Sprintf("category is not applicable to %s goals", goalType)}, nil
	}

	if !unitApplicable(goalType) && input.Unit != nil {
		return nil, CreateGoalOutput{Error: fmt.Sprintf("unit is not applicable to %s goals", goalType)}, nil
	}

	if exerciseApplicable(goalType) {
		if input.ExerciseID == nil {
			return nil, CreateGoalOutput{Error: fmt.Sprintf("exercise_id is required for %s goals", goalType)}, nil
		}
		exercise, err := db.GetExercise(ctx, *input.ExerciseID, userID)
		if err != nil {
			return nil, CreateGoalOutput{}, fmt.Errorf("database error: %w", err)
		}
		if exercise == nil {
			return nil, CreateGoalOutput{Error: "exercise not found"}, nil
		}
	} else if input.ExerciseID != nil {
		return nil, CreateGoalOutput{Error: fmt.Sprintf("exercise_id is not applicable to %s goals", goalType)}, nil
	}

	if activityApplicable(goalType) {
		if input.ActivityID == nil {
			return nil, CreateGoalOutput{Error: fmt.Sprintf("activity_id is required for %s goals", goalType)}, nil
		}
		activity, err := db.GetActivity(ctx, *input.ActivityID, userID)
		if err != nil {
			return nil, CreateGoalOutput{}, fmt.Errorf("database error: %w", err)
		}
		if activity == nil {
			return nil, CreateGoalOutput{Error: "activity not found"}, nil
		}
	} else if input.ActivityID != nil {
		return nil, CreateGoalOutput{Error: fmt.Sprintf("activity_id is not applicable to %s goals", goalType)}, nil
	}

	g := domain.Goal{
		UserID:      userID,
		Name:        input.Name,
		GoalType:    goalType,
		ExerciseID:  input.ExerciseID,
		ActivityID:  input.ActivityID,
		TargetValue: input.TargetValue,
		Category:    input.Category,
		Unit:        input.Unit,
		StartsAt:    startsAt,
		EndsAt:      input.EndsAt,
	}

	if goalType == domain.GoalTypeMoneySaving {
		baseline, err := moneyBalanceSince(ctx, db, userID, startsAt)
		if err != nil {
			return nil, CreateGoalOutput{}, fmt.Errorf("database error: %w", err)
		}
		g.BaselineBalanceEUR = &baseline
	}

	if goalType == domain.GoalTypeManual {
		g.CurrentValue = 0
	} else {
		now := time.Now().UTC()
		currentValue, err := computeCurrentValue(ctx, db, userID, g, now)
		if err != nil {
			return nil, CreateGoalOutput{}, fmt.Errorf("database error: %w", err)
		}
		g.CurrentValue = currentValue
	}

	id, err := db.CreateGoal(ctx, &g)
	if err != nil {
		return nil, CreateGoalOutput{}, fmt.Errorf("database error: %w", err)
	}
	g.ID = id

	return nil, CreateGoalOutput{ID: id, Goal: toGoalOutput(g)}, nil
}
