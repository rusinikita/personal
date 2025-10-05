package tests

import (
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"personal/action/create_exercise"
	"personal/action/log_workout_set"
	"personal/domain"
	"personal/util"
)

func (s *IntegrationTestSuite) TestLogWorkoutSet_WithRepsCreatesActiveWorkout() {
	ctx := s.Context()

	// Create exercise via create_exercise action
	exerciseInput := create_exercise.CreateExerciseInput{
		Name:          "Bench Press",
		EquipmentType: "barbell",
	}
	_, exerciseOutput, err := create_exercise.CreateExercise(ctx, nil, exerciseInput)
	require.NoError(s.T(), err)

	// Call MCP tool log_workout_set with exercise_id, reps, weight_kg
	input := log_workout_set.LogWorkoutSetInput{
		ExerciseID: exerciseOutput.ID,
		Reps:       util.Ptr(int64(10)),
		WeightKg:   util.Ptr(80.5),
	}

	// TODO: implement log_workout_set.LogWorkoutSet action
	// _, output, err := log_workout_set.LogWorkoutSet(ctx, nil, input)
	// require.NoError(s.T(), err)
	// require.NotZero(s.T(), output.SetID)
	// require.NotZero(s.T(), output.WorkoutID)
	// assert.True(s.T(), output.IsNewWorkout)

	// TODO: implement DB.GetLastSet method
	// workoutSet, err := s.Repo().GetLastSet(ctx, s.UserID())
	// require.NoError(s.T(), err)
	// assert.Equal(s.T(), output.SetID, workoutSet.Set.ID)
	// assert.Equal(s.T(), output.WorkoutID, workoutSet.Workout.ID)
	// assert.Equal(s.T(), exerciseOutput.ID, workoutSet.Set.ExerciseID)
	// assert.Equal(s.T(), int64(10), *workoutSet.Set.Reps)
	// assert.Equal(s.T(), 80.5, *workoutSet.Set.WeightKg)
	// assert.Nil(s.T(), workoutSet.Set.DurationSeconds)
	// assert.Nil(s.T(), workoutSet.Workout.CompletedAt)

	_ = input // avoid unused variable error
}

func (s *IntegrationTestSuite) TestLogWorkoutSet_WithDuration() {
	ctx := s.Context()

	// Create exercise via create_exercise action
	exerciseInput := create_exercise.CreateExerciseInput{
		Name:          "Plank",
		EquipmentType: "bodyweight",
	}
	_, exerciseOutput, err := create_exercise.CreateExercise(ctx, nil, exerciseInput)
	require.NoError(s.T(), err)

	// Call MCP tool log_workout_set with exercise_id, duration_seconds
	input := log_workout_set.LogWorkoutSetInput{
		ExerciseID:      exerciseOutput.ID,
		DurationSeconds: util.Ptr(int64(60)),
	}

	// TODO: implement log_workout_set.LogWorkoutSet action
	// _, output, err := log_workout_set.LogWorkoutSet(ctx, nil, input)
	// require.NoError(s.T(), err)
	// require.NotZero(s.T(), output.SetID)

	// TODO: implement DB.GetLastSet method
	// workoutSet, err := s.Repo().GetLastSet(ctx, s.UserID())
	// require.NoError(s.T(), err)
	// assert.Equal(s.T(), int64(60), *workoutSet.Set.DurationSeconds)
	// assert.Nil(s.T(), workoutSet.Set.Reps)

	_ = input // avoid unused variable error
}

func (s *IntegrationTestSuite) TestLogWorkoutSet_ReusesActiveWorkout() {
	ctx := s.Context()

	// Create exercise via create_exercise action
	exerciseInput := create_exercise.CreateExerciseInput{
		Name:          "Squat",
		EquipmentType: "barbell",
	}
	_, exerciseOutput, err := create_exercise.CreateExercise(ctx, nil, exerciseInput)
	require.NoError(s.T(), err)

	// TODO: implement DB.CreateWorkout method
	// Create active workout
	// workout := domain.Workout{
	// 	UserID:      s.UserID(),
	// 	StartedAt:   time.Now(),
	// 	CompletedAt: nil,
	// }
	// workoutID, err := s.Repo().CreateWorkout(ctx, &workout)
	// require.NoError(s.T(), err)

	// TODO: implement DB.CreateSet method
	// Create a set in the active workout (less than 2 hours ago)
	// set := domain.Set{
	// 	UserID:     s.UserID(),
	// 	WorkoutID:  workoutID,
	// 	ExerciseID: exerciseOutput.ID,
	// 	Reps:       util.Ptr(int64(5)),
	// 	WeightKg:   util.Ptr(100.0),
	// 	CreatedAt:  time.Now().Add(-30 * time.Minute),
	// }
	// err = s.Repo().CreateSet(ctx, &set)
	// require.NoError(s.T(), err)

	// Call MCP tool log_workout_set
	input := log_workout_set.LogWorkoutSetInput{
		ExerciseID: exerciseOutput.ID,
		Reps:       util.Ptr(int64(8)),
		WeightKg:   util.Ptr(100.0),
	}

	// TODO: implement log_workout_set.LogWorkoutSet action
	// _, output, err := log_workout_set.LogWorkoutSet(ctx, nil, input)
	// require.NoError(s.T(), err)
	// assert.False(s.T(), output.IsNewWorkout)
	// assert.Equal(s.T(), workoutID, output.WorkoutID)

	// TODO: implement DB.GetLastSet method
	// workoutSet, err := s.Repo().GetLastSet(ctx, s.UserID())
	// require.NoError(s.T(), err)
	// assert.Equal(s.T(), workoutID, workoutSet.Workout.ID)
	// assert.Equal(s.T(), int64(8), *workoutSet.Set.Reps)

	_ = input // avoid unused variable error
}

func (s *IntegrationTestSuite) TestLogWorkoutSet_ClosesOldWorkoutAndCreatesNew() {
	ctx := s.Context()

	// Create exercise via create_exercise action
	exerciseInput := create_exercise.CreateExerciseInput{
		Name:          "Deadlift",
		EquipmentType: "barbell",
	}
	_, exerciseOutput, err := create_exercise.CreateExercise(ctx, nil, exerciseInput)
	require.NoError(s.T(), err)

	// TODO: implement DB.CreateWorkout method
	// Create active workout started 3 hours ago
	// threeHoursAgo := time.Now().Add(-3 * time.Hour)
	// workout := domain.Workout{
	// 	UserID:      s.UserID(),
	// 	StartedAt:   threeHoursAgo,
	// 	CompletedAt: nil,
	// }
	// oldWorkoutID, err := s.Repo().CreateWorkout(ctx, &workout)
	// require.NoError(s.T(), err)

	// TODO: implement DB.CreateSet method
	// Create set in old workout (more than 2 hours ago)
	// set := domain.Set{
	// 	UserID:     s.UserID(),
	// 	WorkoutID:  oldWorkoutID,
	// 	ExerciseID: exerciseOutput.ID,
	// 	Reps:       util.Ptr(int64(5)),
	// 	WeightKg:   util.Ptr(120.0),
	// 	CreatedAt:  threeHoursAgo,
	// }
	// err = s.Repo().CreateSet(ctx, &set)
	// require.NoError(s.T(), err)

	// Call MCP tool log_workout_set with exercise_id, reps
	input := log_workout_set.LogWorkoutSetInput{
		ExerciseID: exerciseOutput.ID,
		Reps:       util.Ptr(int64(5)),
		WeightKg:   util.Ptr(120.0),
	}

	// TODO: implement log_workout_set.LogWorkoutSet action
	// _, output, err := log_workout_set.LogWorkoutSet(ctx, nil, input)
	// require.NoError(s.T(), err)
	// assert.True(s.T(), output.IsNewWorkout)
	// assert.NotEqual(s.T(), oldWorkoutID, output.WorkoutID)

	// TODO: implement DB.GetLastSet method
	// Verify new set in new workout
	// workoutSet, err := s.Repo().GetLastSet(ctx, s.UserID())
	// require.NoError(s.T(), err)
	// assert.Equal(s.T(), output.WorkoutID, workoutSet.Workout.ID)
	// assert.NotEqual(s.T(), oldWorkoutID, workoutSet.Workout.ID)

	// TODO: implement DB.ListWorkouts method
	// Verify old workout has completed_at set
	// workouts, err := s.Repo().ListWorkouts(ctx, s.UserID())
	// require.NoError(s.T(), err)
	// var oldWorkout *domain.Workout
	// for _, w := range workouts {
	// 	if w.ID == oldWorkoutID {
	// 		oldWorkout = &w
	// 		break
	// 	}
	// }
	// require.NotNil(s.T(), oldWorkout)
	// assert.NotNil(s.T(), oldWorkout.CompletedAt)
	// assert.Equal(s.T(), threeHoursAgo.Unix(), oldWorkout.CompletedAt.Unix())

	_ = input // avoid unused variable error
}

func (s *IntegrationTestSuite) TestLogWorkoutSet_Validation() {
	ctx := s.Context()

	// Create exercise via create_exercise action
	exerciseInput := create_exercise.CreateExerciseInput{
		Name:          "Pull-up",
		EquipmentType: "bodyweight",
	}
	_, exerciseOutput, err := create_exercise.CreateExercise(ctx, nil, exerciseInput)
	require.NoError(s.T(), err)

	// Call MCP tool log_workout_set without reps and duration_seconds
	input := log_workout_set.LogWorkoutSetInput{
		ExerciseID: exerciseOutput.ID,
		// No reps or duration_seconds
	}

	// TODO: implement log_workout_set.LogWorkoutSet action
	// _, _, err = log_workout_set.LogWorkoutSet(ctx, nil, input)
	// require.Error(s.T(), err)
	// assert.Contains(s.T(), err.Error(), "reps")
	// assert.Contains(s.T(), err.Error(), "duration_seconds")

	_ = input // avoid unused variable error
}

// Prevent unused import warnings
var (
	_ = time.Now
	_ = assert.Equal
	_ = domain.Exercise{}
)
