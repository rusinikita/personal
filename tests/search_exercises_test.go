package tests

import (
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"personal/action/create_exercise"
	"personal/action/search_exercises"
	"personal/domain"
)

func (s *IntegrationTestSuite) TestSearchExercises_ReturnsMatchingExercises() {
	ctx := s.Context()

	_, bench, err := create_exercise.CreateExercise(ctx, nil, create_exercise.CreateExerciseInput{
		Name: "Bench Press", EquipmentType: "barbell",
	})
	require.NoError(s.T(), err)

	_, incline, err := create_exercise.CreateExercise(ctx, nil, create_exercise.CreateExerciseInput{
		Name: "Incline Bench Press", EquipmentType: "barbell",
	})
	require.NoError(s.T(), err)

	_, _, err = create_exercise.CreateExercise(ctx, nil, create_exercise.CreateExerciseInput{
		Name: "Squat", EquipmentType: "barbell",
	})
	require.NoError(s.T(), err)

	_, output, err := search_exercises.SearchExercises(ctx, nil, search_exercises.SearchExercisesInput{
		NameVariants: []string{"bench"},
	})
	require.NoError(s.T(), err)
	require.Len(s.T(), output.Exercises, 2)

	ids := []int64{output.Exercises[0].ExerciseID, output.Exercises[1].ExerciseID}
	assert.Contains(s.T(), ids, bench.ID)
	assert.Contains(s.T(), ids, incline.ID)
}

func (s *IntegrationTestSuite) TestSearchExercises_MultipleVariantsIncreaseMatchCount() {
	ctx := s.Context()

	_, _, err := create_exercise.CreateExercise(ctx, nil, create_exercise.CreateExerciseInput{
		Name: "Overhead Press", EquipmentType: "barbell",
	})
	require.NoError(s.T(), err)

	_, output, err := search_exercises.SearchExercises(ctx, nil, search_exercises.SearchExercisesInput{
		NameVariants: []string{"overhead", "press"},
	})
	require.NoError(s.T(), err)
	require.Len(s.T(), output.Exercises, 1)
	assert.Equal(s.T(), 2, output.Exercises[0].MatchCount)
}

func (s *IntegrationTestSuite) TestSearchExercises_CaseInsensitive() {
	ctx := s.Context()

	_, _, err := create_exercise.CreateExercise(ctx, nil, create_exercise.CreateExerciseInput{
		Name: "Deadlift", EquipmentType: "barbell",
	})
	require.NoError(s.T(), err)

	_, output, err := search_exercises.SearchExercises(ctx, nil, search_exercises.SearchExercisesInput{
		NameVariants: []string{"DEADLIFT"},
	})
	require.NoError(s.T(), err)
	require.Len(s.T(), output.Exercises, 1)
	assert.Equal(s.T(), "Deadlift", output.Exercises[0].Name)
}

func (s *IntegrationTestSuite) TestSearchExercises_ReturnsEmptyWhenNoMatches() {
	ctx := s.Context()

	_, output, err := search_exercises.SearchExercises(ctx, nil, search_exercises.SearchExercisesInput{
		NameVariants: []string{"zzznomatch"},
	})
	require.NoError(s.T(), err)
	assert.Empty(s.T(), output.Exercises)
	assert.Empty(s.T(), output.Error)
}

func (s *IntegrationTestSuite) TestSearchExercises_ReturnsLastUsedAt() {
	ctx := s.Context()

	_, ex, err := create_exercise.CreateExercise(ctx, nil, create_exercise.CreateExerciseInput{
		Name: "Pull-up", EquipmentType: "bodyweight",
	})
	require.NoError(s.T(), err)

	wID, err := s.Repo().CreateWorkout(ctx, &domain.Workout{
		UserID: s.UserID(), StartedAt: time.Now(),
	})
	require.NoError(s.T(), err)
	_, err = s.Repo().CreateSet(ctx, &domain.Set{
		UserID: s.UserID(), WorkoutID: wID, ExerciseID: ex.ID,
		Reps: 10, CreatedAt: time.Now(),
	})
	require.NoError(s.T(), err)

	_, output, err := search_exercises.SearchExercises(ctx, nil, search_exercises.SearchExercisesInput{
		NameVariants: []string{"pull"},
	})
	require.NoError(s.T(), err)
	require.Len(s.T(), output.Exercises, 1)
	assert.NotNil(s.T(), output.Exercises[0].LastUsedAt)
}

func (s *IntegrationTestSuite) TestSearchExercises_ValidationErrorForEmptyVariants() {
	ctx := s.Context()

	_, output, err := search_exercises.SearchExercises(ctx, nil, search_exercises.SearchExercisesInput{
		NameVariants: []string{},
	})
	require.NoError(s.T(), err)
	assert.NotEmpty(s.T(), output.Error)
}
