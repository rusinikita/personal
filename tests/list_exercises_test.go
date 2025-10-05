package tests

import (
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"personal/action/create_exercise"
	"personal/action/list_exercises"
	"personal/action/log_workout_set"
)

func (s *IntegrationTestSuite) TestListExercises_SortedByLastUsed() {
	ctx := s.Context()

	// Create 3 exercises via create_exercise action
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

	exercise3Input := create_exercise.CreateExerciseInput{
		Name:          "Deadlift",
		EquipmentType: "barbell",
	}
	_, exercise3Output, err := create_exercise.CreateExercise(ctx, nil, exercise3Input)
	require.NoError(s.T(), err)

	// Create set for exercise 2 via log_workout_set action (will have older timestamp)
	time.Sleep(10 * time.Millisecond) // Ensure different timestamps
	setInput2 := log_workout_set.LogWorkoutSetInput{
		ExerciseID: exercise2Output.ID,
		Reps:       10,
		WeightKg:   100.0,
	}
	_, _, err = log_workout_set.LogWorkoutSet(ctx, nil, setInput2)
	require.NoError(s.T(), err)

	// Create set for exercise 1 via log_workout_set action (will have newer timestamp)
	time.Sleep(10 * time.Millisecond) // Ensure different timestamps
	setInput1 := log_workout_set.LogWorkoutSetInput{
		ExerciseID: exercise1Output.ID,
		Reps:       8,
		WeightKg:   80.0,
	}
	_, _, err = log_workout_set.LogWorkoutSet(ctx, nil, setInput1)
	require.NoError(s.T(), err)

	// Call MCP tool list_exercises
	var listInput list_exercises.ListExercisesInput
	_, listOutput, err := list_exercises.ListExercises(ctx, nil, listInput)
	require.NoError(s.T(), err)
	require.Len(s.T(), listOutput.Exercises, 3)

	// Verify order: exercise 1 (most recent), exercise 2 (older), exercise 3 (unused)
	assert.Equal(s.T(), exercise1Output.ID, listOutput.Exercises[0].ID, "Exercise 1 should be first (most recently used)")
	assert.Equal(s.T(), exercise2Output.ID, listOutput.Exercises[1].ID, "Exercise 2 should be second (used earlier)")
	assert.Equal(s.T(), exercise3Output.ID, listOutput.Exercises[2].ID, "Exercise 3 should be third (never used)")

	// Verify last_used_at is populated correctly
	assert.NotNil(s.T(), listOutput.Exercises[0].LastUsedAt, "Exercise 1 should have last_used_at")
	assert.NotNil(s.T(), listOutput.Exercises[1].LastUsedAt, "Exercise 2 should have last_used_at")
	assert.Nil(s.T(), listOutput.Exercises[2].LastUsedAt, "Exercise 3 should not have last_used_at")
}
