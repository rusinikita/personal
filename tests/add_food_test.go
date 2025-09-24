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
		Description: "Fresh red apple",
		FoodType:    "product",
		Nutrients: &domain.BasicNutrients{
			Calories:       52.0,
			ProteinG:       0.3,
			TotalFatG:      0.2,
			CarbohydratesG: 14.0,
		},
	}

	// Call MCP add_food handler
	_, output, err := add_food.AddFood(s.ContextWithDB(ctx), nil, input)
	require.NoError(s.T(), err)
	require.NotZero(s.T(), output.ID)
	assert.Contains(s.T(), output.Message, "Test Apple")
	assert.Contains(s.T(), output.Message, "added successfully")

	// Verify data was saved correctly by retrieving it
	savedFood, err := s.Repo().GetFood(ctx, output.ID)
	require.NoError(s.T(), err)
	require.NotNil(s.T(), savedFood)

	// Verify all fields match input (Description converted from zero value to nil pointer)
	assert.Equal(s.T(), input.Name, savedFood.Name)
	assert.Equal(s.T(), &input.Description, savedFood.Description)
	assert.Equal(s.T(), input.FoodType, savedFood.FoodType)
	assert.False(s.T(), savedFood.IsArchived)

	// Verify nutrients were saved correctly (BasicNutrients converted to Nutrients with pointers)
	require.NotNil(s.T(), savedFood.Nutrients)
	assert.Equal(s.T(), &input.Nutrients.Calories, savedFood.Nutrients.Calories)
	assert.Equal(s.T(), &input.Nutrients.ProteinG, savedFood.Nutrients.ProteinG)
	assert.Equal(s.T(), &input.Nutrients.TotalFatG, savedFood.Nutrients.TotalFatG)
	assert.Equal(s.T(), &input.Nutrients.CarbohydratesG, savedFood.Nutrients.CarbohydratesG)

	// Verify timestamps were set
	assert.False(s.T(), savedFood.CreatedAt.IsZero())
	assert.False(s.T(), savedFood.UpdatedAt.IsZero())
}

func (s *IntegrationTestSuite) TestAddFood_DuplicateChecking() {
	ctx := context.Background()

	s.T().Run("duplicate by name", func(t *testing.T) {
		// First, create a food item
		input1 := add_food.AddFoodInput{
			Name:     "Duplicate Test Food",
			FoodType: "product",
		}
		_, _, err := add_food.AddFood(s.ContextWithDB(ctx), nil, input1)
		require.NoError(t, err)

		// Try to add the same food with same name
		input2 := add_food.AddFoodInput{
			Name:     "Duplicate Test Food", // Same name
			FoodType: "product",
		}
		_, _, err = add_food.AddFood(s.ContextWithDB(ctx), nil, input2)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "duplicate food found")
		assert.Contains(t, err.Error(), "food with name 'Duplicate Test Food' already exists")
	})

	s.T().Run("duplicate by barcode", func(t *testing.T) {
		// First, create a food item with barcode
		input1 := add_food.AddFoodInput{
			Name:     "Barcode Test Food 1",
			Barcode:  "1234567890123",
			FoodType: "product",
		}
		_, _, err := add_food.AddFood(s.ContextWithDB(ctx), nil, input1)
		require.NoError(t, err)

		// Try to add different food with same barcode
		input2 := add_food.AddFoodInput{
			Name:     "Barcode Test Food 2", // Different name
			Barcode:  "1234567890123",       // Same barcode
			FoodType: "product",
		}
		_, _, err = add_food.AddFood(s.ContextWithDB(ctx), nil, input2)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "duplicate food found")
		assert.Contains(t, err.Error(), "food with barcode '1234567890123' already exists")
	})

	s.T().Run("duplicate by both name and barcode", func(t *testing.T) {
		// First, create a food item with both name and barcode
		input1 := add_food.AddFoodInput{
			Name:     "Both Name and Barcode Test",
			Barcode:  "9876543210987",
			FoodType: "product",
		}
		_, _, err := add_food.AddFood(s.ContextWithDB(ctx), nil, input1)
		require.NoError(t, err)

		// Try to add the exact same food
		input2 := add_food.AddFoodInput{
			Name:     "Both Name and Barcode Test", // Same name
			Barcode:  "9876543210987",              // Same barcode
			FoodType: "product",
		}
		_, _, err = add_food.AddFood(s.ContextWithDB(ctx), nil, input2)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "duplicate food found")
		// With simplified logic, name check comes first, so should mention name
		assert.Contains(t, err.Error(), "Both Name and Barcode Test")
		assert.Contains(t, err.Error(), "food with name")
	})

	s.T().Run("no duplicate when both name and barcode are different", func(t *testing.T) {
		// This should succeed - different name and barcode
		input := add_food.AddFoodInput{
			Name:     "Unique Test Food",
			Barcode:  "1111111111111",
			FoodType: "product",
		}
		_, output, err := add_food.AddFood(s.ContextWithDB(ctx), nil, input)
		require.NoError(t, err)
		assert.NotZero(t, output.ID)
		assert.Contains(t, output.Message, "added successfully")
	})
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
				ServingSizeG: -10.0,
			},
			expectedError: "serving_size_g must be positive",
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			// Call MCP add_food handler with invalid data
			_, _, err := add_food.AddFood(s.ContextWithDB(ctx), nil, tc.input)

			// Verify validation error occurred
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.expectedError)
		})
	}
}
