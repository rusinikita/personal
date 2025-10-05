package list_workouts

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"personal/domain"
	"personal/gateways"
)

var MCPDefinition = mcp.Tool{
	Name: "list_workouts",
	Annotations: &mcp.ToolAnnotations{
		Title: "List workouts",
	},
	Description: `List recent workouts with their exercises and sets.

This tool returns recent workouts (last 30 days) with complete details:
- Workouts sorted by start time (most recent first)
- Each workout includes all sets grouped by exercise
- Exercise details (name, equipment type) included for each set
- Shows active workouts (completed_at = null) and completed workouts

Returns an array of workouts with nested exercises and sets.`,
}

type ListWorkoutsInput struct {
	Limit int `json:"limit,omitempty" jsonschema:"Maximum number of workouts to return (default 10)"`
}

type SetItem struct {
	ID              int64   `json:"id" jsonschema:"Set ID"`
	Reps            int64   `json:"reps,omitempty" jsonschema:"Number of repetitions (0 for static exercises)"`
	DurationSeconds int64   `json:"duration_seconds,omitempty" jsonschema:"Duration in seconds (0 for rep-based exercises)"`
	WeightKg        float64 `json:"weight_kg,omitempty" jsonschema:"Weight in kilograms (0 for bodyweight)"`
	CreatedAt       string  `json:"created_at" jsonschema:"Set completion timestamp (ISO8601)"`
}

type ExerciseWithSets struct {
	ExerciseID    int64     `json:"exercise_id" jsonschema:"Exercise ID"`
	ExerciseName  string    `json:"exercise_name" jsonschema:"Exercise name"`
	EquipmentType string    `json:"equipment_type" jsonschema:"Equipment type"`
	Sets          []SetItem `json:"sets" jsonschema:"List of sets for this exercise"`
}

type WorkoutItem struct {
	ID          int64              `json:"id" jsonschema:"Workout ID"`
	UserID      int64              `json:"user_id" jsonschema:"User ID"`
	StartedAt   string             `json:"started_at" jsonschema:"Workout start timestamp (ISO8601)"`
	CompletedAt *string            `json:"completed_at" jsonschema:"Workout completion timestamp (ISO8601), null if active"`
	Exercises   []ExerciseWithSets `json:"exercises" jsonschema:"List of exercises with their sets"`
}

type ListWorkoutsOutput struct {
	Workouts []WorkoutItem `json:"workouts" jsonschema:"List of workouts"`
}

func ListWorkouts(ctx context.Context, _ *mcp.CallToolRequest, input ListWorkoutsInput) (*mcp.CallToolResult, ListWorkoutsOutput, error) {
	// Get database from context
	db := gateways.DBFromContext(ctx)
	if db == nil {
		return nil, ListWorkoutsOutput{}, fmt.Errorf("database not available in context")
	}

	// Get user ID from context
	userID := gateways.UserIDFromContext(ctx)
	if userID == 0 {
		return nil, ListWorkoutsOutput{}, fmt.Errorf("user_id not available in context")
	}

	// Set default limit
	limit := input.Limit
	if limit == 0 {
		limit = 10
	}

	// Get sets from last 30 days
	now := time.Now()
	from := now.AddDate(0, 0, -30)
	sets, err := db.ListSets(ctx, userID, from, now)
	if err != nil {
		return nil, ListWorkoutsOutput{}, fmt.Errorf("failed to list sets: %w", err)
	}

	// If no sets, return empty workouts
	if len(sets) == 0 {
		return nil, ListWorkoutsOutput{Workouts: []WorkoutItem{}}, nil
	}

	// Extract unique workout IDs and exercise IDs
	workoutIDsMap := make(map[int64]bool)
	exerciseIDsMap := make(map[int64]bool)
	for _, set := range sets {
		workoutIDsMap[set.WorkoutID] = true
		exerciseIDsMap[set.ExerciseID] = true
	}

	// Convert maps to slices
	workoutIDs := make([]int64, 0, len(workoutIDsMap))
	for id := range workoutIDsMap {
		workoutIDs = append(workoutIDs, id)
	}
	exerciseIDs := make([]int64, 0, len(exerciseIDsMap))
	for id := range exerciseIDsMap {
		exerciseIDs = append(exerciseIDs, id)
	}

	// Get workout details
	workouts, err := db.GetWorkoutsByIDs(ctx, userID, workoutIDs)
	if err != nil {
		return nil, ListWorkoutsOutput{}, fmt.Errorf("failed to get workouts: %w", err)
	}

	// Get exercise details
	exercises, err := db.GetExercisesByIDs(ctx, userID, exerciseIDs)
	if err != nil {
		return nil, ListWorkoutsOutput{}, fmt.Errorf("failed to get exercises: %w", err)
	}

	// Create maps for quick lookup
	workoutMap := make(map[int64]domain.Workout)
	for _, w := range workouts {
		workoutMap[w.ID] = w
	}
	exerciseMap := make(map[int64]domain.Exercise)
	for _, e := range exercises {
		exerciseMap[e.ID] = e
	}

	// Group sets by workout, then by exercise
	workoutSets := make(map[int64]map[int64][]domain.Set)
	for _, set := range sets {
		if workoutSets[set.WorkoutID] == nil {
			workoutSets[set.WorkoutID] = make(map[int64][]domain.Set)
		}
		workoutSets[set.WorkoutID][set.ExerciseID] = append(workoutSets[set.WorkoutID][set.ExerciseID], set)
	}

	// Build output
	output := ListWorkoutsOutput{
		Workouts: make([]WorkoutItem, 0, len(workouts)),
	}

	for _, workout := range workouts {
		item := WorkoutItem{
			ID:        workout.ID,
			UserID:    workout.UserID,
			StartedAt: workout.StartedAt.Format("2006-01-02T15:04:05Z07:00"),
			Exercises: make([]ExerciseWithSets, 0),
		}

		if workout.CompletedAt != nil {
			completedAt := workout.CompletedAt.Format("2006-01-02T15:04:05Z07:00")
			item.CompletedAt = &completedAt
		}

		// Add exercises with their sets
		exerciseSets := workoutSets[workout.ID]
		for exerciseID, sets := range exerciseSets {
			exercise := exerciseMap[exerciseID]
			exerciseWithSets := ExerciseWithSets{
				ExerciseID:    exercise.ID,
				ExerciseName:  exercise.Name,
				EquipmentType: string(exercise.EquipmentType),
				Sets:          make([]SetItem, 0, len(sets)),
			}

			for _, set := range sets {
				setItem := SetItem{
					ID:              set.ID,
					Reps:            set.Reps,
					DurationSeconds: set.DurationSeconds,
					WeightKg:        set.WeightKg,
					CreatedAt:       set.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
				}
				exerciseWithSets.Sets = append(exerciseWithSets.Sets, setItem)
			}

			item.Exercises = append(item.Exercises, exerciseWithSets)
		}

		output.Workouts = append(output.Workouts, item)
	}

	// Sort workouts by started_at DESC
	sort.Slice(output.Workouts, func(i, j int) bool {
		return output.Workouts[i].StartedAt > output.Workouts[j].StartedAt
	})

	// Limit results
	if len(output.Workouts) > limit {
		output.Workouts = output.Workouts[:limit]
	}

	return nil, output, nil
}
