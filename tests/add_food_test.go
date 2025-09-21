package tests

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"personal/action/add_food"
	"personal/domain"
)

func (s *IntegrationTestSuite) TestAddFood_Success() {
	ctx := context.Background()

	// Prepare test input with valid food data
	input := add_food.AddFoodInput{
		Name:        "Test Apple",
		Description: stringPtr("Fresh red apple"),
		FoodType:    "product",
		Nutrients: &domain.Nutrients{
			Calories:       floatPtr(52.0),
			ProteinG:       floatPtr(0.3),
			TotalFatG:      floatPtr(0.2),
			CarbohydratesG: floatPtr(14.0),
		},
	}

	// Call MCP add_food handler
	_, output, err := add_food.AddFood(ctx, nil, input, s.Repo())
	require.NoError(s.T(), err)
	require.NotZero(s.T(), output.ID)
	assert.Contains(s.T(), output.Message, "Test Apple")
	assert.Contains(s.T(), output.Message, "added successfully")

	// Verify data was saved correctly by retrieving it
	savedFood, err := s.Repo().GetFood(ctx, output.ID)
	require.NoError(s.T(), err)
	require.NotNil(s.T(), savedFood)

	// Verify all fields match input
	assert.Equal(s.T(), input.Name, savedFood.Name)
	assert.Equal(s.T(), input.Description, savedFood.Description)
	assert.Equal(s.T(), input.FoodType, savedFood.FoodType)
	assert.False(s.T(), savedFood.IsArchived)

	// Verify nutrients were saved correctly
	require.NotNil(s.T(), savedFood.Nutrients)
	assert.Equal(s.T(), input.Nutrients.Calories, savedFood.Nutrients.Calories)
	assert.Equal(s.T(), input.Nutrients.ProteinG, savedFood.Nutrients.ProteinG)
	assert.Equal(s.T(), input.Nutrients.TotalFatG, savedFood.Nutrients.TotalFatG)
	assert.Equal(s.T(), input.Nutrients.CarbohydratesG, savedFood.Nutrients.CarbohydratesG)

	// Verify timestamps were set
	assert.False(s.T(), savedFood.CreatedAt.IsZero())
	assert.False(s.T(), savedFood.UpdatedAt.IsZero())
}

func (s *IntegrationTestSuite) TestAddFood_ValidationErrors() {
	ctx := context.Background()

	testCases := []struct {
		name          string
		input         add_food.AddFoodInput
		expectedError string
	}{
		{
			name: "empty name",
			input: add_food.AddFoodInput{
				Name:     "",
				FoodType: "product",
			},
			expectedError: "name is required",
		},
		{
			name: "invalid food type",
			input: add_food.AddFoodInput{
				Name:     "Test Food",
				FoodType: "invalid_type",
			},
			expectedError: "food_type must be one of: component, product, dish",
		},
		{
			name: "negative serving size",
			input: add_food.AddFoodInput{
				Name:         "Test Food",
				FoodType:     "product",
				ServingSizeG: floatPtr(-10.0),
			},
			expectedError: "serving_size_g must be positive",
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			// Call MCP add_food handler with invalid data
			_, _, err := add_food.AddFood(ctx, nil, tc.input, s.Repo())

			// Verify validation error occurred
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.expectedError)
		})
	}
}

// Helper functions
func stringPtr(s string) *string {
	return &s
}

func floatPtr(f float64) *float64 {
	return &f
}
