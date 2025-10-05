package log_workout_set

import (
	"context"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"personal/domain"
	"personal/gateways"
	"personal/util"
)

var MCPDefinition = mcp.Tool{
	Name: "log_workout_set",
	Annotations: &mcp.ToolAnnotations{
		DestructiveHint: util.Ptr(true),
		IdempotentHint:  false,
		Title:           "Log workout set",
	},
	Description: `Log a workout set (подход) for an exercise.

This tool logs a set of an exercise with repetitions, duration, and/or weight.
If there is no active workout or the last set was logged more than 2 hours ago,
a new workout will be automatically created.

At least one of reps or duration_seconds must be provided.

Parameters:
- exercise_id: ID of the exercise
- reps: Number of repetitions (optional, for rep-based exercises)
- duration_seconds: Duration in seconds (optional, for static exercises like plank)
- weight_kg: Weight in kilograms (optional, for weighted exercises)

Returns:
- set_id: ID of the created set
- workout_id: ID of the workout (new or existing)
- is_new_workout: Whether a new workout was created`,
}

type LogWorkoutSetInput struct {
	ExerciseID      int64   `json:"exercise_id" jsonschema:"Exercise ID"`
	Reps            int64   `json:"reps,omitempty" jsonschema:"Number of repetitions (optional, 0 if not provided)"`
	DurationSeconds int64   `json:"duration_seconds,omitempty" jsonschema:"Duration in seconds (optional, 0 if not provided)"`
	WeightKg        float64 `json:"weight_kg,omitempty" jsonschema:"Weight in kilograms (optional, 0 if not provided)"`
}

type LogWorkoutSetOutput struct {
	SetID        int64 `json:"set_id" jsonschema:"Created set ID"`
	WorkoutID    int64 `json:"workout_id" jsonschema:"Workout ID"`
	IsNewWorkout bool  `json:"is_new_workout" jsonschema:"Whether a new workout was created"`
}

func LogWorkoutSet(ctx context.Context, _ *mcp.CallToolRequest, input LogWorkoutSetInput) (*mcp.CallToolResult, LogWorkoutSetOutput, error) {
	// Get database from context
	db := gateways.DBFromContext(ctx)
	if db == nil {
		return nil, LogWorkoutSetOutput{}, fmt.Errorf("database not available in context")
	}

	// Get user ID from context
	userID := gateways.UserIDFromContext(ctx)
	if userID == 0 {
		return nil, LogWorkoutSetOutput{}, fmt.Errorf("user_id not available in context")
	}

	// 1. Validate input
	if err := validateInput(input); err != nil {
		return nil, LogWorkoutSetOutput{}, fmt.Errorf("validation error: %w", err)
	}

	// 2. Get last set to determine if we need a new workout
	lastSet, err := db.GetLastSet(ctx, userID)
	var workoutID int64
	var isNewWorkout bool

	// 3. Determine if we need to create a new workout
	now := time.Now()
	twoHoursAgo := now.Add(-2 * time.Hour)

	needNewWorkout := false
	var oldWorkoutID int64

	if err != nil || lastSet == nil {
		// No previous sets, need new workout
		needNewWorkout = true
	} else if lastSet.Workout.CompletedAt != nil {
		// Last workout is completed, need new workout
		needNewWorkout = true
	} else if lastSet.Set.CreatedAt.Before(twoHoursAgo) {
		// Last set was more than 2 hours ago, need new workout
		needNewWorkout = true
		oldWorkoutID = lastSet.Workout.ID
	} else {
		// Reuse existing workout
		workoutID = lastSet.Workout.ID
		isNewWorkout = false
	}

	// 4. Close old workout if needed
	if needNewWorkout && oldWorkoutID != 0 {
		err = db.CloseWorkout(ctx, oldWorkoutID, lastSet.Set.CreatedAt)
		if err != nil {
			return nil, LogWorkoutSetOutput{}, fmt.Errorf("failed to close old workout: %w", err)
		}
	}

	// 5. Create new workout if needed
	if needNewWorkout {
		workout := &domain.Workout{
			UserID:      userID,
			StartedAt:   now,
			CompletedAt: nil,
		}

		workoutID, err = db.CreateWorkout(ctx, workout)
		if err != nil {
			return nil, LogWorkoutSetOutput{}, fmt.Errorf("failed to create workout: %w", err)
		}
		isNewWorkout = true
	}

	// 6. Create set
	set := &domain.Set{
		UserID:          userID,
		WorkoutID:       workoutID,
		ExerciseID:      input.ExerciseID,
		Reps:            input.Reps,
		DurationSeconds: input.DurationSeconds,
		WeightKg:        input.WeightKg,
		CreatedAt:       now,
	}

	setID, err := db.CreateSet(ctx, set)
	if err != nil {
		return nil, LogWorkoutSetOutput{}, fmt.Errorf("failed to create set: %w", err)
	}

	// 7. Return success response
	return nil, LogWorkoutSetOutput{
		SetID:        setID,
		WorkoutID:    workoutID,
		IsNewWorkout: isNewWorkout,
	}, nil
}

func validateInput(input LogWorkoutSetInput) error {
	if input.ExerciseID == 0 {
		return fmt.Errorf("exercise_id is required")
	}

	if input.Reps == 0 && input.DurationSeconds == 0 {
		return fmt.Errorf("at least one of reps or duration_seconds must be provided")
	}

	return nil
}
