package tests

import (
	"context"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"personal/action/add_food"
	"personal/action/find_food"
	"personal/domain"
)

// Helper function to add test foods for search tests
func (s *IntegrationTestSuite) addTestFoods() (bananaID, appleID, bananaYogurtID int64) {
	ctx := context.Background()

	// Add banana
	bananaInput := add_food.AddFoodInput{
		Name:        "банан",
		Description: "Обычный желтый банан",
		FoodType:    "product",
		ServingName: "штука",
		Nutrients: &domain.BasicNutrients{
			Calories:       89.0,
			ProteinG:       1.1,
			TotalFatG:      0.3,
			CarbohydratesG: 23.0,
		},
	}
	_, output, err := add_food.AddFood(s.ContextWithDB(ctx), nil, bananaInput)
	require.NoError(s.T(), err)
	bananaID = output.ID

	// Add apple
	appleInput := add_food.AddFoodInput{
		Name:        "яблоко красное",
		Description: "Свежее красное яблоко",
		FoodType:    "product",
		ServingName: "штука",
		Nutrients: &domain.BasicNutrients{
			Calories:       52.0,
			ProteinG:       0.3,
			TotalFatG:      0.2,
			CarbohydratesG: 14.0,
		},
	}
	_, output, err = add_food.AddFood(s.ContextWithDB(ctx), nil, appleInput)
	require.NoError(s.T(), err)
	appleID = output.ID

	// Add banana yogurt
	yogurtInput := add_food.AddFoodInput{
		Name:        "банановый йогурт",
		Description: "Йогурт со вкусом банана",
		FoodType:    "product",
		ServingName: "стакан",
		Nutrients: &domain.BasicNutrients{
			Calories:       150.0,
			ProteinG:       5.0,
			TotalFatG:      3.0,
			CarbohydratesG: 25.0,
		},
	}
	_, output, err = add_food.AddFood(s.ContextWithDB(ctx), nil, yogurtInput)
	require.NoError(s.T(), err)
	bananaYogurtID = output.ID

	return
}

func (s *IntegrationTestSuite) TestResolveFoodIdByName_Success() {
	// Add test foods
	bananaID, appleID, bananaYogurtID := s.addTestFoods()

	// Multiple variants with different match counts
	// "банан" - найдется в банан (2 раза) и банановый йогурт (2 раза)
	// "яблоко" - найдется только в яблоко красное (1 раз)
	// "красное" - найдется только в яблоко красное (1 раз)
	input := find_food.ResolveFoodIdByNameInput{
		NameVariants: []string{"банан", "банановый", "яблоко", "красное"},
	}
	_, output, err := find_food.ResolveFoodIdByName(s.ContextWithDB(context.Background()), nil, input)
	require.NoError(s.T(), err)
	require.Empty(s.T(), output.Error)
	require.Len(s.T(), output.Foods, 3)

	// Ranking by match_count (results ordered by match_count DESC, then by ID ASC):
	// All items have match_count=2, so ordered by ID (bananaID < appleID < bananaYogurtID)

	// "яблоко красное" - match_count=2 (found by "яблоко" and "красное")
	food1 := output.Foods[0]
	assert.Equal(s.T(), appleID, food1.ID)
	assert.Equal(s.T(), "яблоко красное", food1.Name)
	assert.Equal(s.T(), "штука", food1.ServingName)
	assert.Equal(s.T(), 2, food1.MatchCount)

	// "банановый йогурт" - match_count=2 (found by both "банан" searches)
	food0 := output.Foods[1]
	assert.Equal(s.T(), bananaYogurtID, food0.ID)
	assert.Equal(s.T(), "банановый йогурт", food0.Name)
	assert.Equal(s.T(), "стакан", food0.ServingName)
	assert.Equal(s.T(), 2, food0.MatchCount)

	// "банан" - match_count=1
	food3 := output.Foods[2]
	assert.Equal(s.T(), bananaID, food3.ID)
	assert.Equal(s.T(), "банан", food3.Name)
	assert.Equal(s.T(), "штука", food3.ServingName)
	assert.Equal(s.T(), 1, food3.MatchCount)
}

func (s *IntegrationTestSuite) TestResolveFoodIdByName_ValidationErrors() {
	ctx := context.Background()

	// Test empty name_variants array
	input := find_food.ResolveFoodIdByNameInput{
		NameVariants: []string{},
	}
	_, output, err := find_food.ResolveFoodIdByName(s.ContextWithDB(ctx), nil, input)
	require.NoError(s.T(), err)
	require.NotEmpty(s.T(), output.Error)
	assert.Contains(s.T(), output.Error, "name_variants cannot be empty")

	// Test more than 5 variants
	input = find_food.ResolveFoodIdByNameInput{
		NameVariants: []string{"1", "2", "3", "4", "5", "6"},
	}
	_, output, err = find_food.ResolveFoodIdByName(s.ContextWithDB(ctx), nil, input)
	require.NoError(s.T(), err)
	require.NotEmpty(s.T(), output.Error)
	assert.Contains(s.T(), output.Error, "maximum 5 name variants allowed")

	// Test empty string in variants
	input = find_food.ResolveFoodIdByNameInput{
		NameVariants: []string{"банан", "", "яблоко"},
	}
	_, output, err = find_food.ResolveFoodIdByName(s.ContextWithDB(ctx), nil, input)
	require.NoError(s.T(), err)
	require.NotEmpty(s.T(), output.Error)
	assert.Contains(s.T(), output.Error, "name variants cannot be empty")
}
