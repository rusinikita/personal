package tests

import (
	"context"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"personal/action/create_exercise"
	"personal/action/delete_workout_set"
	"personal/domain"
	"personal/gateways"
)

func (s *IntegrationTestSuite) TestDeleteWorkoutSet_Successfully() {
	ctx := s.Context()

	// Create exercise
	_, exerciseOutput, err := create_exercise.CreateExercise(ctx, nil, create_exercise.CreateExerciseInput{
		Name:          "Bench Press",
		EquipmentType: "barbell",
	})
	require.NoError(s.T(), err)

	// Create workout
	workoutID, err := s.Repo().CreateWorkout(ctx, &domain.Workout{
		UserID:    s.UserID(),
		StartedAt: time.Now(),
	})
	require.NoError(s.T(), err)

	// Create set
	setID, err := s.Repo().CreateSet(ctx, &domain.Set{
		UserID:     s.UserID(),
		WorkoutID:  workoutID,
		ExerciseID: exerciseOutput.ID,
		Reps:       8,
		WeightKg:   80.0,
		CreatedAt:  time.Now(),
	})
	require.NoError(s.T(), err)

	// Call MCP tool delete_workout_set with set_id
	_, output, err := delete_workout_set.DeleteWorkoutSet(ctx, nil, delete_workout_set.DeleteWorkoutSetInput{
		SetID: setID,
	})
	require.NoError(s.T(), err)
	assert.Equal(s.T(), setID, output.SetID)
	assert.Equal(s.T(), "Bench Press", output.ExerciseName)
	assert.Equal(s.T(), 80.0, output.WeightKg)
	assert.Equal(s.T(), int64(8), output.Reps)

	// Verify set no longer exists
	_, err = s.Repo().GetSetByID(ctx, setID, s.UserID())
	require.Error(s.T(), err, "set should not exist after deletion")
}

func (s *IntegrationTestSuite) TestDeleteWorkoutSet_NotFound() {
	ctx := s.Context()

	_, _, err := delete_workout_set.DeleteWorkoutSet(ctx, nil, delete_workout_set.DeleteWorkoutSetInput{
		SetID: 999999999,
	})
	require.Error(s.T(), err)
}

func (s *IntegrationTestSuite) TestDeleteWorkoutSet_OtherUsersSet() {
	ctx := s.Context()

	// Create exercise and set for user A (current context user)
	_, exerciseOutput, err := create_exercise.CreateExercise(ctx, nil, create_exercise.CreateExerciseInput{
		Name:          "Squat",
		EquipmentType: "barbell",
	})
	require.NoError(s.T(), err)

	workoutID, err := s.Repo().CreateWorkout(ctx, &domain.Workout{
		UserID:    s.UserID(),
		StartedAt: time.Now(),
	})
	require.NoError(s.T(), err)

	setID, err := s.Repo().CreateSet(ctx, &domain.Set{
		UserID:     s.UserID(),
		WorkoutID:  workoutID,
		ExerciseID: exerciseOutput.ID,
		Reps:       5,
		WeightKg:   100.0,
		CreatedAt:  time.Now(),
	})
	require.NoError(s.T(), err)

	// Attempt deletion as a different user (user B)
	ctxB := gateways.WithUserID(context.Background(), 99999)
	ctxB = gateways.WithDB(ctxB, s.Repo())
	_, _, err = delete_workout_set.DeleteWorkoutSet(ctxB, nil, delete_workout_set.DeleteWorkoutSetInput{
		SetID: setID,
	})
	require.Error(s.T(), err)
}
