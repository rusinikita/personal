package tests

import (
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"personal/action/create_exercise"
	"personal/action/get_exercise_history"
	"personal/action/merge_exercises"
	"personal/action/search_exercises"
	"personal/domain"
)

func (s *IntegrationTestSuite) TestMergeExercises_MergesSetsAndDeletesSource() {
	ctx := s.Context()

	_, src, err := create_exercise.CreateExercise(ctx, nil, create_exercise.CreateExerciseInput{
		Name: "Bench Press Duplicate", EquipmentType: "barbell",
	})
	require.NoError(s.T(), err)

	_, target, err := create_exercise.CreateExercise(ctx, nil, create_exercise.CreateExerciseInput{
		Name: "Bench Press", EquipmentType: "barbell",
	})
	require.NoError(s.T(), err)

	wID, err := s.Repo().CreateWorkout(ctx, &domain.Workout{
		UserID: s.UserID(), StartedAt: time.Now(),
	})
	require.NoError(s.T(), err)
	for i := range 3 {
		_, err = s.Repo().CreateSet(ctx, &domain.Set{
			UserID: s.UserID(), WorkoutID: wID, ExerciseID: src.ID,
			Reps: 5, WeightKg: 80, CreatedAt: time.Now().Add(time.Duration(i) * time.Minute),
		})
		require.NoError(s.T(), err)
	}

	_, output, err := merge_exercises.MergeExercises(ctx, nil, merge_exercises.MergeExercisesInput{
		SourceExerciseID: src.ID,
		TargetExerciseID: target.ID,
	})
	require.NoError(s.T(), err)
	assert.Equal(s.T(), int64(3), output.SetsMoved)
	assert.Equal(s.T(), "Bench Press Duplicate", output.DeletedExerciseName)

	// Source no longer searchable
	_, found, err := search_exercises.SearchExercises(ctx, nil, search_exercises.SearchExercisesInput{
		NameVariants: []string{"Bench Press Duplicate"},
	})
	require.NoError(s.T(), err)
	assert.Empty(s.T(), found.Exercises)

	// Sets now belong to target
	_, history, err := get_exercise_history.GetExerciseHistory(ctx, nil, get_exercise_history.GetExerciseHistoryInput{
		ExerciseID: target.ID, Limit: 10,
	})
	require.NoError(s.T(), err)
	require.Len(s.T(), history.Sessions, 1)
	assert.Len(s.T(), history.Sessions[0].Sets, 3)
}

func (s *IntegrationTestSuite) TestMergeExercises_WorksWithZeroSets() {
	ctx := s.Context()

	_, src, err := create_exercise.CreateExercise(ctx, nil, create_exercise.CreateExerciseInput{
		Name: "Empty Source", EquipmentType: "barbell",
	})
	require.NoError(s.T(), err)

	_, target, err := create_exercise.CreateExercise(ctx, nil, create_exercise.CreateExerciseInput{
		Name: "Empty Target", EquipmentType: "barbell",
	})
	require.NoError(s.T(), err)

	_, output, err := merge_exercises.MergeExercises(ctx, nil, merge_exercises.MergeExercisesInput{
		SourceExerciseID: src.ID,
		TargetExerciseID: target.ID,
	})
	require.NoError(s.T(), err)
	assert.Equal(s.T(), int64(0), output.SetsMoved)

	_, found, err := search_exercises.SearchExercises(ctx, nil, search_exercises.SearchExercisesInput{
		NameVariants: []string{"Empty Source"},
	})
	require.NoError(s.T(), err)
	assert.Empty(s.T(), found.Exercises)
}

func (s *IntegrationTestSuite) TestMergeExercises_ErrorWhenSourceNotFound() {
	ctx := s.Context()

	_, target, err := create_exercise.CreateExercise(ctx, nil, create_exercise.CreateExerciseInput{
		Name: "Real Exercise", EquipmentType: "barbell",
	})
	require.NoError(s.T(), err)

	_, _, err = merge_exercises.MergeExercises(ctx, nil, merge_exercises.MergeExercisesInput{
		SourceExerciseID: 999999999,
		TargetExerciseID: target.ID,
	})
	require.Error(s.T(), err)
}

func (s *IntegrationTestSuite) TestMergeExercises_ErrorWhenSameIDs() {
	ctx := s.Context()

	_, ex, err := create_exercise.CreateExercise(ctx, nil, create_exercise.CreateExerciseInput{
		Name: "Same Exercise", EquipmentType: "barbell",
	})
	require.NoError(s.T(), err)

	_, _, err = merge_exercises.MergeExercises(ctx, nil, merge_exercises.MergeExercisesInput{
		SourceExerciseID: ex.ID,
		TargetExerciseID: ex.ID,
	})
	require.Error(s.T(), err)
}
