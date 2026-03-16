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
- date: ISO 8601 date string e.g. "2026-02-19" (optional, for backdating a set to a past workout)

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
	Date            string  `json:"date,omitempty" jsonschema:"ISO 8601 date for backdating e.g. 2026-02-19 (optional)"`
}

type LogWorkoutSetOutput struct {
	SetID        int64 `json:"set_id" jsonschema:"Created set ID"`
	WorkoutID    int64 `json:"workout_id" jsonschema:"Workout ID"`
	IsNewWorkout bool  `json:"is_new_workout" jsonschema:"Whether a new workout was created"`
}

func LogWorkoutSet(ctx context.Context, _ *mcp.CallToolRequest, input LogWorkoutSetInput) (*mcp.CallToolResult, LogWorkoutSetOutput, error) {
	db := gateways.DBFromContext(ctx)
	if db == nil {
		return nil, LogWorkoutSetOutput{}, fmt.Errorf("database not available in context")
	}

	userID := gateways.UserIDFromContext(ctx)
	if userID == 0 {
		return nil, LogWorkoutSetOutput{}, fmt.Errorf("user_id not available in context")
	}

	if err := validateInput(input); err != nil {
		return nil, LogWorkoutSetOutput{}, fmt.Errorf("validation error: %w", err)
	}

	if input.Date != "" {
		return logBackdated(ctx, db, userID, input)
	}

	return logCurrent(ctx, db, userID, input)
}

func logBackdated(ctx context.Context, db gateways.DB, userID int64, input LogWorkoutSetInput) (*mcp.CallToolResult, LogWorkoutSetOutput, error) {
	date, err := time.Parse("2006-01-02", input.Date)
	if err != nil {
		return nil, LogWorkoutSetOutput{}, fmt.Errorf("invalid date format, expected YYYY-MM-DD: %w", err)
	}

	var workoutID int64
	isNewWorkout := false

	existing, err := db.GetWorkoutByDate(ctx, userID, date)
	if err != nil {
		return nil, LogWorkoutSetOutput{}, fmt.Errorf("failed to look up workout by date: %w", err)
	}

	if existing != nil {
		workoutID = existing.ID
	} else {
		dayStart := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
		dayEnd := time.Date(date.Year(), date.Month(), date.Day(), 23, 59, 59, 0, time.UTC)
		workoutID, err = db.CreateWorkout(ctx, &domain.Workout{
			UserID:      userID,
			StartedAt:   dayStart,
			CompletedAt: &dayEnd,
		})
		if err != nil {
			return nil, LogWorkoutSetOutput{}, fmt.Errorf("failed to create backdated workout: %w", err)
		}
		isNewWorkout = true
	}

	setTime := time.Date(date.Year(), date.Month(), date.Day(), 12, 0, 0, 0, time.UTC)
	setID, err := db.CreateSet(ctx, &domain.Set{
		UserID:          userID,
		WorkoutID:       workoutID,
		ExerciseID:      input.ExerciseID,
		Reps:            input.Reps,
		DurationSeconds: input.DurationSeconds,
		WeightKg:        input.WeightKg,
		CreatedAt:       setTime,
	})
	if err != nil {
		return nil, LogWorkoutSetOutput{}, fmt.Errorf("failed to create set: %w", err)
	}

	return nil, LogWorkoutSetOutput{SetID: setID, WorkoutID: workoutID, IsNewWorkout: isNewWorkout}, nil
}

func logCurrent(ctx context.Context, db gateways.DB, userID int64, input LogWorkoutSetInput) (*mcp.CallToolResult, LogWorkoutSetOutput, error) {
	lastSet, err := db.GetLastSet(ctx, userID)
	var workoutID int64
	var isNewWorkout bool

	now := time.Now()
	twoHoursAgo := now.Add(-2 * time.Hour)

	needNewWorkout := false
	var oldWorkoutID int64

	if err != nil || lastSet == nil {
		needNewWorkout = true
	} else if lastSet.Workout.CompletedAt != nil {
		needNewWorkout = true
	} else if lastSet.Set.CreatedAt.Before(twoHoursAgo) {
		needNewWorkout = true
		oldWorkoutID = lastSet.Workout.ID
	} else {
		workoutID = lastSet.Workout.ID
	}

	if needNewWorkout && oldWorkoutID != 0 {
		if err = db.CloseWorkout(ctx, oldWorkoutID, lastSet.Set.CreatedAt); err != nil {
			return nil, LogWorkoutSetOutput{}, fmt.Errorf("failed to close old workout: %w", err)
		}
	}

	if needNewWorkout {
		workoutID, err = db.CreateWorkout(ctx, &domain.Workout{
			UserID:    userID,
			StartedAt: now,
		})
		if err != nil {
			return nil, LogWorkoutSetOutput{}, fmt.Errorf("failed to create workout: %w", err)
		}
		isNewWorkout = true
	}

	setID, err := db.CreateSet(ctx, &domain.Set{
		UserID:          userID,
		WorkoutID:       workoutID,
		ExerciseID:      input.ExerciseID,
		Reps:            input.Reps,
		DurationSeconds: input.DurationSeconds,
		WeightKg:        input.WeightKg,
		CreatedAt:       now,
	})
	if err != nil {
		return nil, LogWorkoutSetOutput{}, fmt.Errorf("failed to create set: %w", err)
	}

	return nil, LogWorkoutSetOutput{SetID: setID, WorkoutID: workoutID, IsNewWorkout: isNewWorkout}, nil
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
