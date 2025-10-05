package tests

import (
	"context"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"personal/action/create_exercise"
)

func (s *IntegrationTestSuite) TestCreateExercise_Success() {
	ctx := context.Background()

	// Prepare test input with valid exercise data
	input := create_exercise.CreateExerciseInput{
		Name:          "Bench Press",
		EquipmentType: "barbell",
	}

	// Call MCP create_exercise handler
	_, output, err := create_exercise.CreateExercise(s.ContextWithDB(ctx), nil, input)
	require.NoError(s.T(), err)
	require.NotZero(s.T(), output.ID)
	assert.Equal(s.T(), "Bench Press", output.Name)
	assert.Equal(s.T(), "barbell", output.EquipmentType)
	assert.NotEmpty(s.T(), output.CreatedAt)
	assert.Nil(s.T(), output.LastUsedAt)

	// Verify data was saved correctly by retrieving it from repository
	exercises, err := s.Repo().ListWithLastUsed(ctx, output.UserID)
	require.NoError(s.T(), err)
	require.NotEmpty(s.T(), exercises)

	var found bool
	for _, ex := range exercises {
		if ex.ID == output.ID {
			found = true
			assert.Equal(s.T(), "Bench Press", ex.Name)
			assert.Equal(s.T(), string("barbell"), string(ex.EquipmentType))
			assert.False(s.T(), ex.CreatedAt.IsZero())
			assert.Nil(s.T(), ex.LastUsedAt)
			break
		}
	}
	assert.True(s.T(), found, "Created exercise not found in repository")
}

func (s *IntegrationTestSuite) TestCreateExercise_InvalidEquipmentType() {
	ctx := context.Background()

	// Prepare test input with invalid equipment type
	input := create_exercise.CreateExerciseInput{
		Name:          "Leg Press",
		EquipmentType: "invalid_type",
	}

	// Call MCP create_exercise handler
	_, _, err := create_exercise.CreateExercise(s.ContextWithDB(ctx), nil, input)

	// Verify validation error occurred
	require.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "equipment_type")
}
