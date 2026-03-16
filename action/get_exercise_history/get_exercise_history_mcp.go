package get_exercise_history

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"personal/domain"
	"personal/gateways"
	"personal/util"
)

var MCPDefinition = mcp.Tool{
	Name: "get_exercise_history",
	Annotations: &mcp.ToolAnnotations{
		DestructiveHint: util.Ptr(false),
		IdempotentHint:  true,
		Title:           "Get exercise history",
	},
	Description: `Return workout sessions containing a specific exercise, newest first.

Each session includes the workout date and all sets logged for that exercise.
Use this before starting an exercise to see previous weights and reps.

Parameters:
- exercise_id: ID of the exercise
- limit: max sessions to return (optional, default 20)
- offset: pagination offset (optional, default 0)

Returns:
- sessions: array of {workout_id, date, sets: [{set_id, weight_kg, reps, duration_seconds}]}`,
}

type GetExerciseHistoryInput struct {
	ExerciseID int64 `json:"exercise_id" jsonschema:"Exercise ID"`
	Limit      int   `json:"limit,omitempty" jsonschema:"Max sessions to return (default 20)"`
	Offset     int   `json:"offset,omitempty" jsonschema:"Pagination offset (default 0)"`
}

type SetSummary struct {
	SetID           int64   `json:"set_id"`
	WeightKg        float64 `json:"weight_kg,omitempty"`
	Reps            int64   `json:"reps,omitempty"`
	DurationSeconds int64   `json:"duration_seconds,omitempty"`
}

type ExerciseSession struct {
	WorkoutID int64        `json:"workout_id"`
	Date      string       `json:"date"`
	Sets      []SetSummary `json:"sets"`
}

type GetExerciseHistoryOutput struct {
	Sessions []ExerciseSession `json:"sessions"`
}

func GetExerciseHistory(ctx context.Context, _ *mcp.CallToolRequest, input GetExerciseHistoryInput) (*mcp.CallToolResult, GetExerciseHistoryOutput, error) {
	db := gateways.DBFromContext(ctx)
	if db == nil {
		return nil, GetExerciseHistoryOutput{}, fmt.Errorf("database not available in context")
	}

	userID := gateways.UserIDFromContext(ctx)
	if userID == 0 {
		return nil, GetExerciseHistoryOutput{}, fmt.Errorf("user_id not available in context")
	}

	if input.ExerciseID == 0 {
		return nil, GetExerciseHistoryOutput{}, fmt.Errorf("exercise_id is required")
	}

	limit := input.Limit
	if limit <= 0 {
		limit = 20
	}

	workouts, err := db.GetExerciseHistory(ctx, userID, input.ExerciseID, limit, input.Offset)
	if err != nil {
		return nil, GetExerciseHistoryOutput{}, fmt.Errorf("failed to get exercise history: %w", err)
	}

	if len(workouts) == 0 {
		return nil, GetExerciseHistoryOutput{Sessions: []ExerciseSession{}}, nil
	}

	workoutIDs := make([]int64, len(workouts))
	for i, w := range workouts {
		workoutIDs[i] = w.ID
	}

	sets, err := db.ListSetsByExerciseAndWorkouts(ctx, userID, input.ExerciseID, workoutIDs)
	if err != nil {
		return nil, GetExerciseHistoryOutput{}, fmt.Errorf("failed to get sets: %w", err)
	}

	setsByWorkout := make(map[int64][]domain.Set)
	for _, s := range sets {
		setsByWorkout[s.WorkoutID] = append(setsByWorkout[s.WorkoutID], s)
	}

	sessions := make([]ExerciseSession, 0, len(workouts))
	for _, w := range workouts {
		workoutSets := setsByWorkout[w.ID]
		summaries := make([]SetSummary, len(workoutSets))
		for i, s := range workoutSets {
			summaries[i] = SetSummary{
				SetID:           s.ID,
				WeightKg:        s.WeightKg,
				Reps:            s.Reps,
				DurationSeconds: s.DurationSeconds,
			}
		}
		sessions = append(sessions, ExerciseSession{
			WorkoutID: w.ID,
			Date:      w.StartedAt.Format("2006-01-02"),
			Sets:      summaries,
		})
	}

	return nil, GetExerciseHistoryOutput{Sessions: sessions}, nil
}
