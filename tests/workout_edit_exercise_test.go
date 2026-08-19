package tests

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"personal/action/workout"
)

func (s *IntegrationTestSuite) TestEditExercise_UpdatesName() {
	ctx := s.Context()

	_, ex, err := workout.CreateExercise(ctx, nil, workout.CreateExerciseInput{
		Name: "Old Name", EquipmentType: "barbell",
	})
	require.NoError(s.T(), err)

	_, output, err := workout.EditExercise(ctx, nil, workout.EditExerciseInput{
		ExerciseID: ex.ID,
		Name:       "New Name",
	})
	require.NoError(s.T(), err)
	assert.Equal(s.T(), "New Name", output.Name)
	assert.Equal(s.T(), "barbell", output.EquipmentType)

	// Verify persisted
	_, found, err := workout.SearchExercises(ctx, nil, workout.SearchExercisesInput{
		NameVariants: []string{"New Name"},
	})
	require.NoError(s.T(), err)
	require.Len(s.T(), found.Exercises, 1)
	assert.Equal(s.T(), "New Name", found.Exercises[0].Name)
}

func (s *IntegrationTestSuite) TestEditExercise_UpdatesEquipmentType() {
	ctx := s.Context()

	_, ex, err := workout.CreateExercise(ctx, nil, workout.CreateExerciseInput{
		Name: "Leg Press", EquipmentType: "barbell",
	})
	require.NoError(s.T(), err)

	_, output, err := workout.EditExercise(ctx, nil, workout.EditExerciseInput{
		ExerciseID:    ex.ID,
		EquipmentType: "machine",
	})
	require.NoError(s.T(), err)
	assert.Equal(s.T(), "machine", output.EquipmentType)
	assert.Equal(s.T(), "Leg Press", output.Name)
}

func (s *IntegrationTestSuite) TestEditExercise_UpdatesBothFields() {
	ctx := s.Context()

	_, ex, err := workout.CreateExercise(ctx, nil, workout.CreateExerciseInput{
		Name: "Wrong Name", EquipmentType: "barbell",
	})
	require.NoError(s.T(), err)

	_, output, err := workout.EditExercise(ctx, nil, workout.EditExerciseInput{
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

	_, _, err := workout.EditExercise(ctx, nil, workout.EditExerciseInput{
		ExerciseID: 999999999,
		Name:       "Anything",
	})
	require.Error(s.T(), err)
}

func (s *IntegrationTestSuite) TestEditExercise_ValidationNoFields() {
	ctx := s.Context()

	_, ex, err := workout.CreateExercise(ctx, nil, workout.CreateExerciseInput{
		Name: "Some Exercise", EquipmentType: "barbell",
	})
	require.NoError(s.T(), err)

	_, _, err = workout.EditExercise(ctx, nil, workout.EditExerciseInput{
		ExerciseID: ex.ID,
	})
	require.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "name")
	assert.Contains(s.T(), err.Error(), "equipment_type")
}

func (s *IntegrationTestSuite) TestEditExercise_ValidationInvalidEquipmentType() {
	ctx := s.Context()

	_, ex, err := workout.CreateExercise(ctx, nil, workout.CreateExerciseInput{
		Name: "Cable Row", EquipmentType: "machine",
	})
	require.NoError(s.T(), err)

	_, _, err = workout.EditExercise(ctx, nil, workout.EditExerciseInput{
		ExerciseID:    ex.ID,
		EquipmentType: "invalid",
	})
	require.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "equipment_type")
}
