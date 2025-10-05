package tests

import (
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"personal/action/create_exercise"
	"personal/action/list_workouts"
	"personal/domain"
)

func (s *IntegrationTestSuite) TestListWorkouts_WithSets() {
	ctx := s.Context()

	// Arrange: Create 2 exercises
	exercise1Input := create_exercise.CreateExerciseInput{
		Name:          "Bench Press",
		EquipmentType: "barbell",
	}
	_, exercise1Output, err := create_exercise.CreateExercise(ctx, nil, exercise1Input)
	require.NoError(s.T(), err)

	exercise2Input := create_exercise.CreateExerciseInput{
		Name:          "Squat",
		EquipmentType: "barbell",
	}
	_, exercise2Output, err := create_exercise.CreateExercise(ctx, nil, exercise2Input)
	require.NoError(s.T(), err)

	// Create workout 1 (completed, started 2h ago, completed 1h ago)
	now := time.Now()
	workout1StartedAt := now.Add(-2 * time.Hour)
	workout1CompletedAt := now.Add(-1 * time.Hour)
	workout1 := domain.Workout{
		UserID:      s.UserID(),
		StartedAt:   workout1StartedAt,
		CompletedAt: &workout1CompletedAt,
	}
	workout1ID, err := s.Repo().CreateWorkout(ctx, &workout1)
	require.NoError(s.T(), err)

	// Create 3 sets for workout 1, exercise 1
	for i := 0; i < 3; i++ {
		set := domain.Set{
			UserID:     s.UserID(),
			WorkoutID:  workout1ID,
			ExerciseID: exercise1Output.ID,
			Reps:       10 + int64(i),
			WeightKg:   80.0 + float64(i)*2.5,
			CreatedAt:  workout1StartedAt.Add(time.Duration(i) * 5 * time.Minute),
		}
		_, err = s.Repo().CreateSet(ctx, &set)
		require.NoError(s.T(), err)
	}

	// Create workout 2 (active, started 1h ago)
	workout2StartedAt := now.Add(-1 * time.Hour)
	workout2 := domain.Workout{
		UserID:      s.UserID(),
		StartedAt:   workout2StartedAt,
		CompletedAt: nil,
	}
	workout2ID, err := s.Repo().CreateWorkout(ctx, &workout2)
	require.NoError(s.T(), err)

	// Create 2 sets for workout 2, exercise 2
	for i := 0; i < 2; i++ {
		set := domain.Set{
			UserID:     s.UserID(),
			WorkoutID:  workout2ID,
			ExerciseID: exercise2Output.ID,
			Reps:       8 + int64(i),
			WeightKg:   100.0 + float64(i)*5.0,
			CreatedAt:  workout2StartedAt.Add(time.Duration(i) * 3 * time.Minute),
		}
		_, err = s.Repo().CreateSet(ctx, &set)
		require.NoError(s.T(), err)
	}

	// Act: Call MCP tool list_workouts
	input := list_workouts.ListWorkoutsInput{
		Limit: 10,
	}
	_, output, err := list_workouts.ListWorkouts(ctx, nil, input)
	require.NoError(s.T(), err)

	// Assert: Verify 2 workouts returned, sorted by started_at DESC (workout 2 first, workout 1 second)
	require.Len(s.T(), output.Workouts, 2)

	// Workout 2 should be first (most recent)
	assert.Equal(s.T(), workout2ID, output.Workouts[0].ID)
	assert.Equal(s.T(), s.UserID(), output.Workouts[0].UserID)
	assert.Nil(s.T(), output.Workouts[0].CompletedAt, "Workout 2 should be active")

	// Workout 2 should have 1 exercise with 2 sets
	require.Len(s.T(), output.Workouts[0].Exercises, 1)
	assert.Equal(s.T(), exercise2Output.ID, output.Workouts[0].Exercises[0].ExerciseID)
	assert.Equal(s.T(), exercise2Output.Name, output.Workouts[0].Exercises[0].ExerciseName)
	assert.Equal(s.T(), "barbell", output.Workouts[0].Exercises[0].EquipmentType)
	require.Len(s.T(), output.Workouts[0].Exercises[0].Sets, 2)

	// Workout 1 should be second
	assert.Equal(s.T(), workout1ID, output.Workouts[1].ID)
	assert.NotNil(s.T(), output.Workouts[1].CompletedAt, "Workout 1 should be completed")

	// Workout 1 should have 1 exercise with 3 sets
	require.Len(s.T(), output.Workouts[1].Exercises, 1)
	assert.Equal(s.T(), exercise1Output.ID, output.Workouts[1].Exercises[0].ExerciseID)
	assert.Equal(s.T(), exercise1Output.Name, output.Workouts[1].Exercises[0].ExerciseName)
	assert.Equal(s.T(), "barbell", output.Workouts[1].Exercises[0].EquipmentType)
	require.Len(s.T(), output.Workouts[1].Exercises[0].Sets, 3)
}
