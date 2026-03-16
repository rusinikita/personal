package tests

import (
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"personal/action/create_exercise"
	"personal/action/get_exercise_history"
	"personal/domain"
)

func (s *IntegrationTestSuite) TestGetExerciseHistory_ReturnsMultipleWorkouts() {
	ctx := s.Context()

	_, ex, err := create_exercise.CreateExercise(ctx, nil, create_exercise.CreateExerciseInput{
		Name: "Bench Press", EquipmentType: "barbell",
	})
	require.NoError(s.T(), err)

	// Workout 1 — older, 3 sets
	w1ID, err := s.Repo().CreateWorkout(ctx, &domain.Workout{
		UserID: s.UserID(), StartedAt: time.Now().Add(-48 * time.Hour),
	})
	require.NoError(s.T(), err)
	for i, reps := range []int64{5, 5, 3} {
		_, err = s.Repo().CreateSet(ctx, &domain.Set{
			UserID: s.UserID(), WorkoutID: w1ID, ExerciseID: ex.ID,
			Reps: reps, WeightKg: float64(80 + i*5), CreatedAt: time.Now().Add(-48*time.Hour + time.Duration(i)*time.Minute),
		})
		require.NoError(s.T(), err)
	}

	// Workout 2 — newer, 2 sets
	w2ID, err := s.Repo().CreateWorkout(ctx, &domain.Workout{
		UserID: s.UserID(), StartedAt: time.Now().Add(-24 * time.Hour),
	})
	require.NoError(s.T(), err)
	for i, reps := range []int64{8, 6} {
		_, err = s.Repo().CreateSet(ctx, &domain.Set{
			UserID: s.UserID(), WorkoutID: w2ID, ExerciseID: ex.ID,
			Reps: reps, WeightKg: 85, CreatedAt: time.Now().Add(-24*time.Hour + time.Duration(i)*time.Minute),
		})
		require.NoError(s.T(), err)
	}

	_, output, err := get_exercise_history.GetExerciseHistory(ctx, nil, get_exercise_history.GetExerciseHistoryInput{
		ExerciseID: ex.ID,
		Limit:      10,
	})
	require.NoError(s.T(), err)
	require.Len(s.T(), output.Sessions, 2)
	assert.Equal(s.T(), w2ID, output.Sessions[0].WorkoutID, "newest first")
	assert.Len(s.T(), output.Sessions[0].Sets, 2)
	assert.Equal(s.T(), w1ID, output.Sessions[1].WorkoutID)
	assert.Len(s.T(), output.Sessions[1].Sets, 3)
}

func (s *IntegrationTestSuite) TestGetExerciseHistory_Pagination() {
	ctx := s.Context()

	_, ex, err := create_exercise.CreateExercise(ctx, nil, create_exercise.CreateExerciseInput{
		Name: "Squat", EquipmentType: "barbell",
	})
	require.NoError(s.T(), err)

	for i := 0; i < 3; i++ {
		wID, err := s.Repo().CreateWorkout(ctx, &domain.Workout{
			UserID: s.UserID(), StartedAt: time.Now().Add(-time.Duration(3-i) * 24 * time.Hour),
		})
		require.NoError(s.T(), err)
		_, err = s.Repo().CreateSet(ctx, &domain.Set{
			UserID: s.UserID(), WorkoutID: wID, ExerciseID: ex.ID,
			Reps: 5, WeightKg: 100, CreatedAt: time.Now().Add(-time.Duration(3-i) * 24 * time.Hour),
		})
		require.NoError(s.T(), err)
	}

	_, page1, err := get_exercise_history.GetExerciseHistory(ctx, nil, get_exercise_history.GetExerciseHistoryInput{
		ExerciseID: ex.ID, Limit: 2, Offset: 0,
	})
	require.NoError(s.T(), err)
	assert.Len(s.T(), page1.Sessions, 2)

	_, page2, err := get_exercise_history.GetExerciseHistory(ctx, nil, get_exercise_history.GetExerciseHistoryInput{
		ExerciseID: ex.ID, Limit: 2, Offset: 2,
	})
	require.NoError(s.T(), err)
	assert.Len(s.T(), page2.Sessions, 1)
}

func (s *IntegrationTestSuite) TestGetExerciseHistory_NoHistory() {
	ctx := s.Context()

	_, ex, err := create_exercise.CreateExercise(ctx, nil, create_exercise.CreateExerciseInput{
		Name: "Deadlift", EquipmentType: "barbell",
	})
	require.NoError(s.T(), err)

	_, output, err := get_exercise_history.GetExerciseHistory(ctx, nil, get_exercise_history.GetExerciseHistoryInput{
		ExerciseID: ex.ID,
	})
	require.NoError(s.T(), err)
	assert.Empty(s.T(), output.Sessions)
}
