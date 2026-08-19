package tests

import (
	"personal/action/workout"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"personal/domain"
)

func (s *IntegrationTestSuite) TestLogWorkoutSet_WithRepsCreatesActiveWorkout() {
	ctx := s.Context()

	// Create exercise via create_exercise action
	exerciseInput := workout.CreateExerciseInput{
		Name:          "Bench Press",
		EquipmentType: "barbell",
	}
	_, exerciseOutput, err := workout.CreateExercise(ctx, nil, exerciseInput)
	require.NoError(s.T(), err)

	// Call MCP tool log_workout_set with exercise_id, reps, weight_kg
	input := workout.LogWorkoutSetInput{
		ExerciseID: exerciseOutput.ID,
		Reps:       10,
		WeightKg:   80.5,
	}

	_, output, err := workout.LogWorkoutSet(ctx, nil, input)
	require.NoError(s.T(), err)
	require.NotZero(s.T(), output.SetID)
	require.NotZero(s.T(), output.WorkoutID)
	assert.True(s.T(), output.IsNewWorkout)

	workoutSet, err := s.Repo().GetLastSet(ctx, s.UserID())
	require.NoError(s.T(), err)
	assert.Equal(s.T(), output.SetID, workoutSet.Set.ID)
	assert.Equal(s.T(), output.WorkoutID, workoutSet.Workout.ID)
	assert.Equal(s.T(), exerciseOutput.ID, workoutSet.Set.ExerciseID)
	assert.Equal(s.T(), int64(10), workoutSet.Set.Reps)
	assert.Equal(s.T(), 80.5, workoutSet.Set.WeightKg)
	assert.Equal(s.T(), int64(0), workoutSet.Set.DurationSeconds)
	assert.Nil(s.T(), workoutSet.Workout.CompletedAt)
}

func (s *IntegrationTestSuite) TestLogWorkoutSet_WithDuration() {
	ctx := s.Context()

	// Create exercise via create_exercise action
	exerciseInput := workout.CreateExerciseInput{
		Name:          "Plank",
		EquipmentType: "bodyweight",
	}
	_, exerciseOutput, err := workout.CreateExercise(ctx, nil, exerciseInput)
	require.NoError(s.T(), err)

	// Call MCP tool log_workout_set with exercise_id, duration_seconds
	input := workout.LogWorkoutSetInput{
		ExerciseID:      exerciseOutput.ID,
		DurationSeconds: 60,
	}

	_, output, err := workout.LogWorkoutSet(ctx, nil, input)
	require.NoError(s.T(), err)
	require.NotZero(s.T(), output.SetID)

	workoutSet, err := s.Repo().GetLastSet(ctx, s.UserID())
	require.NoError(s.T(), err)
	assert.Equal(s.T(), int64(60), workoutSet.Set.DurationSeconds)
	assert.Equal(s.T(), int64(0), workoutSet.Set.Reps)
}

func (s *IntegrationTestSuite) TestLogWorkoutSet_ReusesActiveWorkout() {
	ctx := s.Context()

	// Create exercise via create_exercise action
	exerciseInput := workout.CreateExerciseInput{
		Name:          "Squat",
		EquipmentType: "barbell",
	}
	_, exerciseOutput, err := workout.CreateExercise(ctx, nil, exerciseInput)
	require.NoError(s.T(), err)

	// Create active workout
	activeWorkout := domain.Workout{
		UserID:      s.UserID(),
		StartedAt:   time.Now(),
		CompletedAt: nil,
	}
	workoutID, err := s.Repo().CreateWorkout(ctx, &activeWorkout)
	require.NoError(s.T(), err)

	// Create a set in the active workout (less than 2 hours ago)
	set := domain.Set{
		UserID:     s.UserID(),
		WorkoutID:  workoutID,
		ExerciseID: exerciseOutput.ID,
		Reps:       5,
		WeightKg:   100.0,
		CreatedAt:  time.Now().Add(-30 * time.Minute),
	}
	_, err = s.Repo().CreateSet(ctx, &set)
	require.NoError(s.T(), err)

	// Call MCP tool log_workout_set
	input := workout.LogWorkoutSetInput{
		ExerciseID: exerciseOutput.ID,
		Reps:       8,
		WeightKg:   100.0,
	}

	_, output, err := workout.LogWorkoutSet(ctx, nil, input)
	require.NoError(s.T(), err)
	assert.False(s.T(), output.IsNewWorkout)
	assert.Equal(s.T(), workoutID, output.WorkoutID)

	workoutSet, err := s.Repo().GetLastSet(ctx, s.UserID())
	require.NoError(s.T(), err)
	assert.Equal(s.T(), workoutID, workoutSet.Workout.ID)
	assert.Equal(s.T(), int64(8), workoutSet.Set.Reps)
}

func (s *IntegrationTestSuite) TestLogWorkoutSet_ClosesOldWorkoutAndCreatesNew() {
	ctx := s.Context()

	// Create exercise via create_exercise action
	exerciseInput := workout.CreateExerciseInput{
		Name:          "Deadlift",
		EquipmentType: "barbell",
	}
	_, exerciseOutput, err := workout.CreateExercise(ctx, nil, exerciseInput)
	require.NoError(s.T(), err)

	// Create active workout started 3 hours ago
	threeHoursAgo := time.Now().Add(-3 * time.Hour)
	activeWorkout := domain.Workout{
		UserID:      s.UserID(),
		StartedAt:   threeHoursAgo,
		CompletedAt: nil,
	}
	oldWorkoutID, err := s.Repo().CreateWorkout(ctx, &activeWorkout)
	require.NoError(s.T(), err)

	// Create set in old workout (more than 2 hours ago)
	set := domain.Set{
		UserID:     s.UserID(),
		WorkoutID:  oldWorkoutID,
		ExerciseID: exerciseOutput.ID,
		Reps:       5,
		WeightKg:   120.0,
		CreatedAt:  threeHoursAgo,
	}
	_, err = s.Repo().CreateSet(ctx, &set)
	require.NoError(s.T(), err)

	// Call MCP tool log_workout_set with exercise_id, reps
	input := workout.LogWorkoutSetInput{
		ExerciseID: exerciseOutput.ID,
		Reps:       5,
		WeightKg:   120.0,
	}

	_, output, err := workout.LogWorkoutSet(ctx, nil, input)
	require.NoError(s.T(), err)
	assert.True(s.T(), output.IsNewWorkout)
	assert.NotEqual(s.T(), oldWorkoutID, output.WorkoutID)

	// Verify new set in new workout
	workoutSet, err := s.Repo().GetLastSet(ctx, s.UserID())
	require.NoError(s.T(), err)
	assert.Equal(s.T(), output.WorkoutID, workoutSet.Workout.ID)
	assert.NotEqual(s.T(), oldWorkoutID, workoutSet.Workout.ID)

	// Verify old workout has completed_at set
	workouts, err := s.Repo().ListWorkouts(ctx, s.UserID())
	require.NoError(s.T(), err)
	var oldWorkout *domain.Workout
	for _, w := range workouts {
		if w.ID == oldWorkoutID {
			oldWorkout = &w
			break
		}
	}
	require.NotNil(s.T(), oldWorkout)
	assert.NotNil(s.T(), oldWorkout.CompletedAt)
	assert.Equal(s.T(), threeHoursAgo.Unix(), oldWorkout.CompletedAt.Unix())
}

func (s *IntegrationTestSuite) TestLogWorkoutSet_Validation() {
	ctx := s.Context()

	// Create exercise via create_exercise action
	exerciseInput := workout.CreateExerciseInput{
		Name:          "Pull-up",
		EquipmentType: "bodyweight",
	}
	_, exerciseOutput, err := workout.CreateExercise(ctx, nil, exerciseInput)
	require.NoError(s.T(), err)

	// Call MCP tool log_workout_set without reps and duration_seconds
	input := workout.LogWorkoutSetInput{
		ExerciseID: exerciseOutput.ID,
		// No reps or duration_seconds
	}

	_, _, err = workout.LogWorkoutSet(ctx, nil, input)
	require.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "reps")
	assert.Contains(s.T(), err.Error(), "duration_seconds")
}

func (s *IntegrationTestSuite) TestLogWorkoutSet_WithDateCreatesBackdatedWorkout() {
	ctx := s.Context()

	_, exerciseOutput, err := workout.CreateExercise(ctx, nil, workout.CreateExerciseInput{
		Name: "Overhead Press", EquipmentType: "barbell",
	})
	require.NoError(s.T(), err)

	input := workout.LogWorkoutSetInput{
		ExerciseID: exerciseOutput.ID,
		Reps:       5,
		WeightKg:   60.0,
		Date:       "2026-01-15",
	}
	_, output, err := workout.LogWorkoutSet(ctx, nil, input)
	require.NoError(s.T(), err)
	assert.True(s.T(), output.IsNewWorkout)

	workoutSet, err := s.Repo().GetLastSet(ctx, s.UserID())
	require.NoError(s.T(), err)
	assert.Equal(s.T(), "2026-01-15", workoutSet.Set.CreatedAt.Format("2006-01-02"))
	assert.Equal(s.T(), "2026-01-15", workoutSet.Workout.StartedAt.Format("2006-01-02"))
	assert.NotNil(s.T(), workoutSet.Workout.CompletedAt, "backdated workout should be completed")
}

func (s *IntegrationTestSuite) TestLogWorkoutSet_WithDateReusesExistingWorkout() {
	ctx := s.Context()

	_, exerciseOutput, err := workout.CreateExercise(ctx, nil, workout.CreateExerciseInput{
		Name: "Incline Press", EquipmentType: "barbell",
	})
	require.NoError(s.T(), err)

	// Create an existing workout for 2026-01-15
	targetDate := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	completedAt := time.Date(2026, 1, 15, 23, 59, 59, 0, time.UTC)
	existingWorkoutID, err := s.Repo().CreateWorkout(ctx, &domain.Workout{
		UserID:      s.UserID(),
		StartedAt:   targetDate,
		CompletedAt: &completedAt,
	})
	require.NoError(s.T(), err)

	input := workout.LogWorkoutSetInput{
		ExerciseID: exerciseOutput.ID,
		Reps:       8,
		WeightKg:   55.0,
		Date:       "2026-01-15",
	}
	_, output, err := workout.LogWorkoutSet(ctx, nil, input)
	require.NoError(s.T(), err)
	assert.False(s.T(), output.IsNewWorkout)
	assert.Equal(s.T(), existingWorkoutID, output.WorkoutID)
}
