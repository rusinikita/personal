package delete_workout_set

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"personal/gateways"
	"personal/util"
)

var MCPDefinition = mcp.Tool{
	Name: "delete_workout_set",
	Annotations: &mcp.ToolAnnotations{
		DestructiveHint: util.Ptr(true),
		IdempotentHint:  false,
		Title:           "Delete workout set",
	},
	Description: `Delete a single erroneously logged workout set by its ID.

Returns the deleted set's details (exercise name, weight, reps) as confirmation.

Parameters:
- set_id: ID of the set to delete

Returns:
- set_id: ID of the deleted set
- exercise_name: Name of the exercise
- weight_kg: Weight in kilograms (0 if bodyweight)
- reps: Number of repetitions (0 if duration-based)
- duration_seconds: Duration in seconds (0 if rep-based)`,
}

type DeleteWorkoutSetInput struct {
	SetID int64 `json:"set_id" jsonschema:"Set ID to delete"`
}

type DeleteWorkoutSetOutput struct {
	SetID           int64   `json:"set_id"`
	ExerciseName    string  `json:"exercise_name"`
	WeightKg        float64 `json:"weight_kg,omitempty"`
	Reps            int64   `json:"reps,omitempty"`
	DurationSeconds int64   `json:"duration_seconds,omitempty"`
}

func DeleteWorkoutSet(ctx context.Context, _ *mcp.CallToolRequest, input DeleteWorkoutSetInput) (*mcp.CallToolResult, DeleteWorkoutSetOutput, error) {
	db := gateways.DBFromContext(ctx)
	if db == nil {
		return nil, DeleteWorkoutSetOutput{}, fmt.Errorf("database not available in context")
	}

	userID := gateways.UserIDFromContext(ctx)
	if userID == 0 {
		return nil, DeleteWorkoutSetOutput{}, fmt.Errorf("user_id not available in context")
	}

	if input.SetID == 0 {
		return nil, DeleteWorkoutSetOutput{}, fmt.Errorf("set_id is required")
	}

	set, err := db.GetSetByID(ctx, input.SetID, userID)
	if err != nil {
		return nil, DeleteWorkoutSetOutput{}, fmt.Errorf("set not found: %w", err)
	}

	if err := db.DeleteSet(ctx, input.SetID, userID); err != nil {
		return nil, DeleteWorkoutSetOutput{}, fmt.Errorf("failed to delete set: %w", err)
	}

	return nil, DeleteWorkoutSetOutput{
		SetID:           set.Set.ID,
		ExerciseName:    set.ExerciseName,
		WeightKg:        set.Set.WeightKg,
		Reps:            set.Set.Reps,
		DurationSeconds: set.Set.DurationSeconds,
	}, nil
}
