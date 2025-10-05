package tests

import (
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"personal/action/create_exercise"
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
	// TODO: implement list_workouts MCP tool

	// Assert: Verify 2 workouts returned, sorted by started_at DESC (workout 2 first, workout 1 second)
	// TODO: verify workouts order
	// TODO: verify workout 1 has 3 sets with exercise 1 details
	// TODO: verify workout 2 has 2 sets with exercise 2 details and completed_at=null

	_ = workout1ID
	_ = workout2ID
	assert.True(s.T(), true, "Test implementation pending")
}
