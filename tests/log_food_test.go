package tests

import (
	"context"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"personal/action/log_food"
	"personal/domain"
	"personal/util"
)

func (s *IntegrationTestSuite) TestLogFood_Scenario1_WithIDs() {
	ctx := context.Background()

	// Setup: Create test foods with nutrients and serving_size_g
	apple := s.createTestFood(ctx, "Apple", "123456", 100.0, &domain.Nutrients{
		Calories:       util.Ptr(52.0),
		ProteinG:       util.Ptr(0.3),
		TotalFatG:      util.Ptr(0.2),
		CarbohydratesG: util.Ptr(14.0),
	})

	bread := s.createTestFood(ctx, "Bread", "789012", 30.0, &domain.Nutrients{
		Calories:       util.Ptr(265.0),
		ProteinG:       util.Ptr(9.0),
		TotalFatG:      util.Ptr(3.2),
		CarbohydratesG: util.Ptr(49.0),
	})

	userID := int64(1)
	now := time.Now().UTC().Truncate(time.Microsecond)
	laterTime := now.Add(time.Second)

	// Call MCP log_food tool handler
	input := log_food.LogFoodInput{
		ConsumedItems: []log_food.ConsumedFoodItem{
			{
				FoodID:     &apple.ID,
				AmountG:    150.0, // 1.5 * 100g serving
				ConsumedAt: &now,
			},
			{
				FoodID:       &bread.ID,
				ServingCount: util.Ptr(2.0), // 2 * 30g servings = 60g
				ConsumedAt:   &laterTime,
			},
		},
	}

	// Call the actual MCP handler
	_, output, err := log_food.LogFood(s.ContextWithDB(ctx), nil, input)
	require.NoError(s.T(), err)

	// Verify the response
	require.Len(s.T(), output.AddedItems, 2)
	require.Len(s.T(), output.NotFoundItems, 0)

	// Verify first item (apple - direct amount_g)
	appleItem := output.AddedItems[0]
	assert.Equal(s.T(), userID, appleItem.UserID)
	assert.Equal(s.T(), &apple.ID, appleItem.FoodID)
	assert.Equal(s.T(), apple.Name, appleItem.FoodName)
	assert.Equal(s.T(), 150.0, appleItem.AmountG)
	assert.InDelta(s.T(), 78.0, *appleItem.Nutrients.Calories, 0.01) // 52.0 * 1.5
	assert.InDelta(s.T(), 0.45, *appleItem.Nutrients.ProteinG, 0.01) // 0.3 * 1.5

	// Verify second item (bread - serving_count conversion)
	breadItem := output.AddedItems[1]
	assert.Equal(s.T(), userID, breadItem.UserID)
	assert.Equal(s.T(), &bread.ID, breadItem.FoodID)
	assert.Equal(s.T(), bread.Name, breadItem.FoodName)
	assert.Equal(s.T(), 60.0, breadItem.AmountG)                      // 2 servings * 30g
	assert.InDelta(s.T(), 159.0, *breadItem.Nutrients.Calories, 0.01) // 265.0 * 0.6
	assert.InDelta(s.T(), 5.4, *breadItem.Nutrients.ProteinG, 0.01)   // 9.0 * 0.6

	// Verify logs were saved to database
	savedAppleLog, err := s.Repo().GetConsumptionLog(ctx, userID, now)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), &apple.ID, savedAppleLog.FoodID)
	assert.Equal(s.T(), apple.Name, savedAppleLog.FoodName)
	assert.Equal(s.T(), 150.0, savedAppleLog.AmountG)
	assert.Equal(s.T(), util.Ptr(78.0), savedAppleLog.Nutrients.Calories)

	savedBreadLog, err := s.Repo().GetConsumptionLog(ctx, userID, laterTime)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), &bread.ID, savedBreadLog.FoodID)
	assert.Equal(s.T(), bread.Name, savedBreadLog.FoodName)
	assert.Equal(s.T(), 60.0, savedBreadLog.AmountG)
	assert.Equal(s.T(), util.Ptr(159.0), savedBreadLog.Nutrients.Calories)

	// Cleanup
	defer s.Repo().DeleteConsumptionLog(ctx, userID, now)
	defer s.Repo().DeleteConsumptionLog(ctx, userID, laterTime)
}

func (s *IntegrationTestSuite) TestLogFood_Scenario2_WithNames() {
	ctx := context.Background()

	// Setup: Create foods with known names
	orange := s.createTestFood(ctx, "Orange", "111111", 0, &domain.Nutrients{
		Calories:       util.Ptr(47.0),
		ProteinG:       util.Ptr(0.9),
		TotalFatG:      util.Ptr(0.1),
		CarbohydratesG: util.Ptr(12.0),
	})

	userID := int64(1)
	now := time.Now().UTC().Truncate(time.Microsecond)

	// Call MCP log_food tool handler with name search
	input := log_food.LogFoodInput{
		ConsumedItems: []log_food.ConsumedFoodItem{
			{
				Name:       util.Ptr("Orange"),
				AmountG:    200.0,
				ConsumedAt: &now,
			},
		},
	}

	// Call the actual MCP handler
	_, output, err := log_food.LogFood(s.ContextWithDB(ctx), nil, input)
	require.NoError(s.T(), err)

	// Verify the response
	require.Len(s.T(), output.AddedItems, 1)
	require.Len(s.T(), output.NotFoundItems, 0)

	addedItem := output.AddedItems[0]
	assert.Equal(s.T(), userID, addedItem.UserID)
	assert.Equal(s.T(), &orange.ID, addedItem.FoodID)
	assert.Equal(s.T(), orange.Name, addedItem.FoodName)
	assert.Equal(s.T(), 200.0, addedItem.AmountG)
	assert.Equal(s.T(), 94.0, *addedItem.Nutrients.Calories) // 47.0 * 2.0
	assert.Equal(s.T(), 1.8, *addedItem.Nutrients.ProteinG)  // 0.9 * 2.0

	// Verify log was saved to database
	savedLog, err := s.Repo().GetConsumptionLog(ctx, userID, now)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), &orange.ID, savedLog.FoodID)
	assert.Equal(s.T(), orange.Name, savedLog.FoodName)
	assert.Equal(s.T(), 200.0, savedLog.AmountG)
	assert.Equal(s.T(), util.Ptr(94.0), savedLog.Nutrients.Calories)

	// Cleanup
	defer s.Repo().DeleteConsumptionLog(ctx, userID, now)
}

func (s *IntegrationTestSuite) TestLogFood_Scenario3_WithBarcodes() {
	ctx := context.Background()

	// Setup: Create foods with unique barcodes
	banana := s.createTestFood(ctx, "Banana", "999888", 0, &domain.Nutrients{
		Calories:       util.Ptr(89.0),
		ProteinG:       util.Ptr(1.1),
		TotalFatG:      util.Ptr(0.3),
		CarbohydratesG: util.Ptr(23.0),
	})

	userID := int64(1)
	now := time.Now().UTC().Truncate(time.Microsecond)

	// Call MCP log_food tool handler with barcode search
	input := log_food.LogFoodInput{
		ConsumedItems: []log_food.ConsumedFoodItem{
			{
				Barcode:    util.Ptr("999888"),
				AmountG:    120.0,
				ConsumedAt: &now,
			},
		},
	}

	// Call the actual MCP handler
	_, output, err := log_food.LogFood(s.ContextWithDB(ctx), nil, input)
	require.NoError(s.T(), err)

	// Verify the response
	require.Len(s.T(), output.AddedItems, 1)
	require.Len(s.T(), output.NotFoundItems, 0)

	addedItem := output.AddedItems[0]
	assert.Equal(s.T(), userID, addedItem.UserID)
	assert.Equal(s.T(), &banana.ID, addedItem.FoodID)
	assert.Equal(s.T(), banana.Name, addedItem.FoodName)
	assert.Equal(s.T(), 120.0, addedItem.AmountG)
	assert.Equal(s.T(), 106.8, *addedItem.Nutrients.Calories) // 89.0 * 1.2
	assert.Equal(s.T(), 1.32, *addedItem.Nutrients.ProteinG)  // 1.1 * 1.2

	// Verify log was saved to database
	savedLog, err := s.Repo().GetConsumptionLog(ctx, userID, now)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), &banana.ID, savedLog.FoodID)
	assert.Equal(s.T(), banana.Name, savedLog.FoodName)
	assert.Equal(s.T(), 120.0, savedLog.AmountG)
	assert.Equal(s.T(), util.Ptr(106.8), savedLog.Nutrients.Calories)

	// Cleanup
	defer s.Repo().DeleteConsumptionLog(ctx, userID, now)
}

func (s *IntegrationTestSuite) TestLogFood_Scenario4_DirectNutrients() {
	ctx := context.Background()

	userID := int64(1)
	now := time.Now().UTC().Truncate(time.Microsecond)

	// Call MCP log_food tool handler with direct nutrients
	input := log_food.LogFoodInput{
		ConsumedItems: []log_food.ConsumedFoodItem{
			{
				DirectNutrients: &log_food.DirectNutrients{
					Calories:       250.0,
					ProteinG:       12.0,
					TotalFatG:      8.0,
					CarbohydratesG: 35.0,
					ProductName:    "Homemade Sandwich",
				},
				AmountG:    180.0, // This amount is already accounted for in the nutrients
				ConsumedAt: &now,
			},
		},
	}

	// Call the actual MCP handler
	_, output, err := log_food.LogFood(s.ContextWithDB(ctx), nil, input)
	require.NoError(s.T(), err)

	// Verify the response
	require.Len(s.T(), output.AddedItems, 1)
	require.Len(s.T(), output.NotFoundItems, 0)

	addedItem := output.AddedItems[0]
	assert.Equal(s.T(), userID, addedItem.UserID)
	assert.Nil(s.T(), addedItem.FoodID) // Should be null for direct nutrients
	assert.Equal(s.T(), "Homemade Sandwich", addedItem.FoodName)
	assert.Equal(s.T(), 180.0, addedItem.AmountG)
	assert.Equal(s.T(), 250.0, *addedItem.Nutrients.Calories)
	assert.Equal(s.T(), 12.0, *addedItem.Nutrients.ProteinG)
	assert.Equal(s.T(), 8.0, *addedItem.Nutrients.TotalFatG)
	assert.Equal(s.T(), 35.0, *addedItem.Nutrients.CarbohydratesG)

	// Verify log was saved to database
	savedLog, err := s.Repo().GetConsumptionLog(ctx, userID, now)
	require.NoError(s.T(), err)
	assert.Nil(s.T(), savedLog.FoodID) // Should be null
	assert.Equal(s.T(), "Homemade Sandwich", savedLog.FoodName)
	assert.Equal(s.T(), 180.0, savedLog.AmountG)
	assert.Equal(s.T(), util.Ptr(250.0), savedLog.Nutrients.Calories)
	assert.Equal(s.T(), util.Ptr(12.0), savedLog.Nutrients.ProteinG)

	// Cleanup
	defer s.Repo().DeleteConsumptionLog(ctx, userID, now)
}

func (s *IntegrationTestSuite) TestLogFood_AmbiguousNameSearch() {
	ctx := context.Background()

	// Setup: Create foods with similar names
	apple1 := s.createTestFood(ctx, "Apple", "", 0, nil)
	apple2 := s.createTestFood(ctx, "Apple Juice", "", 0, nil)
	apple3 := s.createTestFood(ctx, "Apple Pie", "", 0, nil)

	userID := int64(1)
	now := time.Now().UTC().Truncate(time.Microsecond)

	// Call MCP log_food tool handler with ambiguous name
	input := log_food.LogFoodInput{
		ConsumedItems: []log_food.ConsumedFoodItem{
			{
				Name:       util.Ptr("apple"), // Should match multiple foods
				AmountG:    150.0,
				ConsumedAt: &now,
			},
		},
	}

	// Call the actual MCP handler
	_, output, err := log_food.LogFood(s.ContextWithDB(ctx), nil, input)
	require.NoError(s.T(), err)

	// Verify the response - should have no added items and one not found item
	require.Len(s.T(), output.AddedItems, 0)
	require.Len(s.T(), output.NotFoundItems, 1)

	notFoundItem := output.NotFoundItems[0]
	assert.Equal(s.T(), util.Ptr("apple"), notFoundItem.Name)
	assert.Equal(s.T(), 150.0, notFoundItem.AmountG)
	assert.Equal(s.T(), "multiple_matches", notFoundItem.Reason)
	require.Len(s.T(), notFoundItem.Suggestions, 2) // Should return first 2 alphabetically

	// Verify suggestions are sorted alphabetically
	assert.Equal(s.T(), "Apple", notFoundItem.Suggestions[0].Name)
	assert.Equal(s.T(), apple1.ID, notFoundItem.Suggestions[0].ID)
	assert.Equal(s.T(), "Apple Juice", notFoundItem.Suggestions[1].Name)
	assert.Equal(s.T(), apple2.ID, notFoundItem.Suggestions[1].ID)

	// Verify no logs were saved to database
	_, err = s.Repo().GetConsumptionLog(ctx, userID, now)
	assert.Error(s.T(), err) // Should be no rows found

	_ = apple3 // Used for setup but not in first 2 suggestions
}

func (s *IntegrationTestSuite) TestLogFood_ValidationErrors() {
	ctx := context.Background()

	// Test empty consumed_items list
	emptyInput := log_food.LogFoodInput{
		ConsumedItems: []log_food.ConsumedFoodItem{},
	}
	_, _, err := log_food.LogFood(s.ContextWithDB(ctx), nil, emptyInput)
	assert.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "validation error")

	// Test item with no valid scenario (no food_id, name, barcode, or direct_nutrients)
	invalidInput := log_food.LogFoodInput{
		ConsumedItems: []log_food.ConsumedFoodItem{
			{
				AmountG: 100.0,
				// No food_id, name, barcode, or direct_nutrients
			},
		},
	}
	_, _, err = log_food.LogFood(s.ContextWithDB(ctx), nil, invalidInput)
	assert.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "validation error")

	// Test negative amount_g
	negativeAmountInput := log_food.LogFoodInput{
		ConsumedItems: []log_food.ConsumedFoodItem{
			{
				Name:    util.Ptr("Apple"),
				AmountG: -50.0, // Invalid negative amount
			},
		},
	}
	_, _, err = log_food.LogFood(s.ContextWithDB(ctx), nil, negativeAmountInput)
	assert.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "validation error")

	// Test negative serving_count
	negativeServingInput := log_food.LogFoodInput{
		ConsumedItems: []log_food.ConsumedFoodItem{
			{
				Name:         util.Ptr("Apple"),
				ServingCount: util.Ptr(-1.5), // Invalid negative serving count
			},
		},
	}
	_, _, err = log_food.LogFood(s.ContextWithDB(ctx), nil, negativeServingInput)
	assert.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "validation error")

	// Test invalid direct_nutrients (missing product_name)
	invalidNutrientsInput := log_food.LogFoodInput{
		ConsumedItems: []log_food.ConsumedFoodItem{
			{
				DirectNutrients: &log_food.DirectNutrients{
					Calories:       100.0,
					ProteinG:       5.0,
					TotalFatG:      3.0,
					CarbohydratesG: 15.0,
					// Missing ProductName - required field
				},
				AmountG: 100.0,
			},
		},
	}
	_, _, err = log_food.LogFood(s.ContextWithDB(ctx), nil, invalidNutrientsInput)
	assert.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "validation error")
}

func (s *IntegrationTestSuite) TestLogFood_NutrientRounding() {
	ctx := context.Background()

	// Create a food with values that would cause floating point precision issues
	testFood := s.createTestFood(ctx, "Test Food", "", 100.0, &domain.Nutrients{
		Calories:       util.Ptr(33.333333333), // Will cause precision issues when multiplied
		ProteinG:       util.Ptr(1.666666667),  // Will cause precision issues when multiplied
		TotalFatG:      util.Ptr(2.222222222),  // Will cause precision issues when multiplied
		CarbohydratesG: util.Ptr(5.555555556),  // Will cause precision issues when multiplied
	})

	userID := int64(1)
	now := time.Now().UTC().Truncate(time.Microsecond)

	// Use amount that will create floating point precision issues (33.3g = 0.333 ratio)
	input := log_food.LogFoodInput{
		ConsumedItems: []log_food.ConsumedFoodItem{
			{
				FoodID:     &testFood.ID,
				AmountG:    33.3, // This creates a 0.333 ratio which causes precision issues
				ConsumedAt: &now,
			},
		},
	}

	// Call the actual MCP handler
	_, output, err := log_food.LogFood(s.ContextWithDB(ctx), nil, input)
	require.NoError(s.T(), err)

	// Verify the response has properly rounded values
	require.Len(s.T(), output.AddedItems, 1)
	addedItem := output.AddedItems[0]

	// Verify values are properly rounded to 3 decimal places
	// 33.333333333 * 0.333 = 11.099999999889 -> rounded to 11.1
	assert.Equal(s.T(), 11.1, *addedItem.Nutrients.Calories)
	// 1.666666667 * 0.333 = 0.5549999999111 -> rounded to 0.555
	assert.Equal(s.T(), 0.555, *addedItem.Nutrients.ProteinG)
	// 2.222222222 * 0.333 = 0.7399999999926 -> rounded to 0.74
	assert.Equal(s.T(), 0.74, *addedItem.Nutrients.TotalFatG)
	// 5.555555556 * 0.333 = 1.8499999999148 -> rounded to 1.85
	assert.Equal(s.T(), 1.85, *addedItem.Nutrients.CarbohydratesG)

	// Cleanup
	defer s.Repo().DeleteConsumptionLog(ctx, userID, now)
}

// Test helper to create food items for testing
func (s *IntegrationTestSuite) createTestFood(ctx context.Context, name, barcode string, servingSizeG float64, nutrients *domain.Nutrients) *domain.Food {
	food := &domain.Food{
		Name:      name,
		FoodType:  "product",
		Nutrients: nutrients,
	}

	if barcode != "" {
		food.Barcode = &barcode
	}

	if servingSizeG > 0 {
		food.ServingSizeG = &servingSizeG
	}

	id, err := s.Repo().AddFood(ctx, food)
	require.NoError(s.T(), err)
	food.ID = id

	return food
}
