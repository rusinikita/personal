package tests

import (
	"context"
	"testing"

	"github.com/stretchr/testify/asыsert"
	"github.com/stretchr/testify/require"

	"personal/action/add_food"
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
	// "банан" - найдется в банан (2 раза) и банановый йогурт (1 раз)
	// "яблоко" - найдется только в яблоко красное (1 раз)
	// "красное" - найдется только в яблоко красное (1 раз)
	// TODO: Call MCP resolve_food_id_by_name tool handler
	// input := find_food.ResolveFoodIdByNameInput{
	//     NameVariants: []string{"банан", "банан", "яблоко", "красное"},
	// }
	// _, output, err := find_food.ResolveFoodIdByName(s.ContextWithDB(context.Background()), nil, input)
	// require.NoError(s.T(), err)
	// require.Empty(s.T(), output.Error)
	// require.Len(s.T(), output.Foods, 3)
	//
	// // Ranking by match_count:
	// // 1. "банан" - match_count=2 (found by both "банан" searches)
	// assert.Equal(s.T(), bananaID, output.Foods[0].ID)
	// assert.Equal(s.T(), "банан", output.Foods[0].Name)
	// assert.Equal(s.T(), "штука", output.Foods[0].ServingName)
	// assert.Equal(s.T(), 2, output.Foods[0].MatchCount)
	//
	// // 2. "яблоко красное" - match_count=2 (found by "яблоко" and "красное")
	// assert.Equal(s.T(), appleID, output.Foods[1].ID)
	// assert.Equal(s.T(), "яблоко красное", output.Foods[1].Name)
	// assert.Equal(s.T(), "штука", output.Foods[1].ServingName)
	// assert.Equal(s.T(), 2, output.Foods[1].MatchCount)
	//
	// // 3. "банановый йогурт" - match_count=1 (found only by "банан")
	// assert.Equal(s.T(), bananaYogurtID, output.Foods[2].ID)
	// assert.Equal(s.T(), "банановый йогурт", output.Foods[2].Name)
	// assert.Equal(s.T(), "стакан", output.Foods[2].ServingName)
	// assert.Equal(s.T(), 1, output.Foods[2].MatchCount)

	// Placeholder test - will be updated when find_food package is implemented
	s.T().Skip("TODO: Implement after find_food package is created")
}

func (s *IntegrationTestSuite) TestResolveFoodIdByName_ValidationErrors() {
	ctx := context.Background()

	// Test empty name_variants array
	// TODO: Call MCP resolve_food_id_by_name tool handler
	// input := find_food.ResolveFoodIdByNameInput{
	//     NameVariants: []string{},
	// }
	// _, output, err := find_food.ResolveFoodIdByName(s.ContextWithDB(ctx), nil, input)
	// require.NoError(s.T(), err)
	// require.NotEmpty(s.T(), output.Error)
	// assert.Contains(s.T(), output.Error, "name_variants cannot be empty")

	// Test more than 5 variants
	// input = find_food.ResolveFoodIdByNameInput{
	//     NameVariants: []string{"1", "2", "3", "4", "5", "6"},
	// }
	// _, output, err = find_food.ResolveFoodIdByName(s.ContextWithDB(ctx), nil, input)
	// require.NoError(s.T(), err)
	// require.NotEmpty(s.T(), output.Error)
	// assert.Contains(s.T(), output.Error, "maximum 5 name variants allowed")

	// Test empty string in variants
	// input = find_food.ResolveFoodIdByNameInput{
	//     NameVariants: []string{"банан", "", "яблоко"},
	// }
	// _, output, err = find_food.ResolveFoodIdByName(s.ContextWithDB(ctx), nil, input)
	// require.NoError(s.T(), err)
	// require.NotEmpty(s.T(), output.Error)
	// assert.Contains(s.T(), output.Error, "name variants cannot be empty")

	// Placeholder test - will be updated when find_food package is implemented
	s.T().Skip("TODO: Implement validation after find_food package is created")
}
