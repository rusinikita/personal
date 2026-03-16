package tests

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"personal/action/create_exercise"
	"personal/action/edit_exercise"
	"personal/action/search_exercises"
)

func (s *IntegrationTestSuite) TestEditExercise_UpdatesName() {
	ctx := s.Context()

	_, ex, err := create_exercise.CreateExercise(ctx, nil, create_exercise.CreateExerciseInput{
		Name: "Old Name", EquipmentType: "barbell",
	})
	require.NoError(s.T(), err)

	_, output, err := edit_exercise.EditExercise(ctx, nil, edit_exercise.EditExerciseInput{
		ExerciseID: ex.ID,
		Name:       "New Name",
	})
	require.NoError(s.T(), err)
	assert.Equal(s.T(), "New Name", output.Name)
	assert.Equal(s.T(), "barbell", output.EquipmentType)

	// Verify persisted
	_, found, err := search_exercises.SearchExercises(ctx, nil, search_exercises.SearchExercisesInput{
		NameVariants: []string{"New Name"},
	})
	require.NoError(s.T(), err)
	require.Len(s.T(), found.Exercises, 1)
	assert.Equal(s.T(), "New Name", found.Exercises[0].Name)
}

func (s *IntegrationTestSuite) TestEditExercise_UpdatesEquipmentType() {
	ctx := s.Context()

	_, ex, err := create_exercise.CreateExercise(ctx, nil, create_exercise.CreateExerciseInput{
		Name: "Leg Press", EquipmentType: "barbell",
	})
	require.NoError(s.T(), err)

	_, output, err := edit_exercise.EditExercise(ctx, nil, edit_exercise.EditExerciseInput{
		ExerciseID:    ex.ID,
		EquipmentType: "machine",
	})
	require.NoError(s.T(), err)
	assert.Equal(s.T(), "machine", output.EquipmentType)
	assert.Equal(s.T(), "Leg Press", output.Name)
}

func (s *IntegrationTestSuite) TestEditExercise_UpdatesBothFields() {
	ctx := s.Context()

	_, ex, err := create_exercise.CreateExercise(ctx, nil, create_exercise.CreateExerciseInput{
		Name: "Wrong Name", EquipmentType: "barbell",
	})
	require.NoError(s.T(), err)

	_, output, err := edit_exercise.EditExercise(ctx, nil, edit_exercise.EditExerciseInput{
		ExerciseID:    ex.ID,
		Name:          "Correct Name",
		EquipmentType: "dumbbells",
	})
	require.NoError(s.T(), err)
	assert.Equal(s.T(), "Correct Name", output.Name)
	assert.Equal(s.T(), "dumbbells", output.EquipmentType)
}

func (s *IntegrationTestSuite) TestEditExercise_NotFound() {
	ctx := s.Context()

	_, _, err := edit_exercise.EditExercise(ctx, nil, edit_exercise.EditExerciseInput{
		ExerciseID: 999999999,
		Name:       "Anything",
	})
	require.Error(s.T(), err)
}

func (s *IntegrationTestSuite) TestEditExercise_ValidationNoFields() {
	ctx := s.Context()

	_, ex, err := create_exercise.CreateExercise(ctx, nil, create_exercise.CreateExerciseInput{
		Name: "Some Exercise", EquipmentType: "barbell",
	})
	require.NoError(s.T(), err)

	_, _, err = edit_exercise.EditExercise(ctx, nil, edit_exercise.EditExerciseInput{
		ExerciseID: ex.ID,
	})
	require.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "name")
	assert.Contains(s.T(), err.Error(), "equipment_type")
}

func (s *IntegrationTestSuite) TestEditExercise_ValidationInvalidEquipmentType() {
	ctx := s.Context()

	_, ex, err := create_exercise.CreateExercise(ctx, nil, create_exercise.CreateExerciseInput{
		Name: "Cable Row", EquipmentType: "machine",
	})
	require.NoError(s.T(), err)

	_, _, err = edit_exercise.EditExercise(ctx, nil, edit_exercise.EditExerciseInput{
		ExerciseID:    ex.ID,
		EquipmentType: "invalid",
	})
	require.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "equipment_type")
}
