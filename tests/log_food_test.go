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

func (s *IntegrationTestSuite) TestLogFoodById_Success() {
	ctx := s.Context()

	// Setup: Create test food with nutrients and serving_size_g
	apple := s.createTestFood(ctx, "Apple", "123456", 100.0, &domain.Nutrients{
		Calories:       util.Ptr(52.0),
		ProteinG:       util.Ptr(0.3),
		TotalFatG:      util.Ptr(0.2),
		CarbohydratesG: util.Ptr(14.0),
	})

	now := time.Now().UTC().Truncate(time.Microsecond)

	// Call log_food_by_id MCP tool
	input := log_food.LogFoodByIdInput{
		FoodID:     apple.ID,
		AmountG:    150.0, // 1.5 * 100g serving
		ConsumedAt: now,
	}

	_, response, err := log_food.LogFoodById(ctx, nil, input)
	require.NoError(s.T(), err)
	assert.Empty(s.T(), response.Error)
	assert.Contains(s.T(), response.Message, "Successfully logged 150.0g of Apple")

	// Verify log was saved to database
	savedLog, err := s.Repo().GetConsumptionLog(ctx, s.UserID(), now)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), &apple.ID, savedLog.FoodID)
	assert.Equal(s.T(), apple.Name, savedLog.FoodName)
	assert.Equal(s.T(), 150.0, savedLog.AmountG)
	assert.InDelta(s.T(), 78.0, *savedLog.Nutrients.Calories, 0.01) // 52.0 * 1.5
}

func (s *IntegrationTestSuite) TestLogFoodById_WithServingCount() {
	ctx := s.Context()

	// Setup: Create test food with serving size
	bread := s.createTestFood(ctx, "Bread", "789012", 30.0, &domain.Nutrients{
		Calories:       util.Ptr(265.0),
		ProteinG:       util.Ptr(9.0),
		TotalFatG:      util.Ptr(3.2),
		CarbohydratesG: util.Ptr(49.0),
	})

	now := time.Now().UTC().Truncate(time.Microsecond)

	// Call log_food_by_id MCP tool with serving count
	input := log_food.LogFoodByIdInput{
		FoodID:       bread.ID,
		ServingCount: 2.0, // 2 * 30g servings = 60g
		ConsumedAt:   now,
	}

	_, response, err := log_food.LogFoodById(ctx, nil, input)
	require.NoError(s.T(), err)
	assert.Empty(s.T(), response.Error)
	assert.Contains(s.T(), response.Message, "Successfully logged 60.0g of Bread")

	// Verify log was saved with correct calculated amounts
	savedLog, err := s.Repo().GetConsumptionLog(ctx, s.UserID(), now)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), 60.0, savedLog.AmountG)                      // 2 servings * 30g
	assert.InDelta(s.T(), 159.0, *savedLog.Nutrients.Calories, 0.01) // 265.0 * 0.6

}

func (s *IntegrationTestSuite) TestLogFoodById_NotFound() {
	ctx := s.Context()

	// Call log_food_by_id MCP tool with non-existent ID
	input := log_food.LogFoodByIdInput{
		FoodID:  99999, // Non-existent ID
		AmountG: 100.0,
	}

	_, response, err := log_food.LogFoodById(ctx, nil, input)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), "food not found", response.Error)
	assert.Empty(s.T(), response.Message)
}

func (s *IntegrationTestSuite) TestLogFoodByBarcode_Success() {
	ctx := s.Context()

	// Setup: Create food with unique barcode
	banana := s.createTestFood(ctx, "Banana", "999888", 0, &domain.Nutrients{
		Calories:       util.Ptr(89.0),
		ProteinG:       util.Ptr(1.1),
		TotalFatG:      util.Ptr(0.3),
		CarbohydratesG: util.Ptr(23.0),
	})

	now := time.Now().UTC().Truncate(time.Microsecond)

	// Call log_food_by_barcode MCP tool
	input := log_food.LogFoodByBarcodeInput{
		Barcode:    "999888",
		AmountG:    120.0,
		ConsumedAt: now,
	}

	_, response, err := log_food.LogFoodByBarcode(ctx, nil, input)
	require.NoError(s.T(), err)
	assert.Empty(s.T(), response.Error)
	assert.Contains(s.T(), response.Message, "Successfully logged 120.0g of Banana")

	// Verify log was saved to database
	savedLog, err := s.Repo().GetConsumptionLog(ctx, s.UserID(), now)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), &banana.ID, savedLog.FoodID)
	assert.Equal(s.T(), banana.Name, savedLog.FoodName)
	assert.Equal(s.T(), 120.0, savedLog.AmountG)
	assert.Equal(s.T(), 106.8, *savedLog.Nutrients.Calories) // 89.0 * 1.2

}

func (s *IntegrationTestSuite) TestLogFoodByBarcode_NotFound() {
	ctx := s.Context()

	// Call log_food_by_barcode MCP tool with non-existent barcode
	input := log_food.LogFoodByBarcodeInput{
		Barcode: "nonexistent",
		AmountG: 100.0,
	}

	_, response, err := log_food.LogFoodByBarcode(ctx, nil, input)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), "barcode not found", response.Error)
	assert.Empty(s.T(), response.Message)
}

func (s *IntegrationTestSuite) TestLogCustomFood_Success() {
	ctx := s.Context()

	now := time.Now().UTC().Truncate(time.Microsecond)

	// Call log_custom_food MCP tool
	input := log_food.LogCustomFoodInput{
		ProductName:    "Homemade Sandwich",
		AmountG:        180.0,
		Calories:       250.0,
		ProteinG:       12.0,
		TotalFatG:      8.0,
		CarbohydratesG: 35.0,
		CaffeineMg:     0.0,
		EthylAlcoholG:  0.0,
		ConsumedAt:     now,
	}

	_, response, err := log_food.LogCustomFood(ctx, nil, input)
	require.NoError(s.T(), err)
	assert.Empty(s.T(), response.Error)
	assert.Contains(s.T(), response.Message, "Successfully logged 180.0g of Homemade Sandwich")

	// Verify log was saved to database
	savedLog, err := s.Repo().GetConsumptionLog(ctx, s.UserID(), now)
	require.NoError(s.T(), err)
	assert.Nil(s.T(), savedLog.FoodID) // Should be null for custom food
	assert.Equal(s.T(), "Homemade Sandwich", savedLog.FoodName)
	assert.Equal(s.T(), 180.0, savedLog.AmountG)
	assert.Equal(s.T(), util.Ptr(250.0), savedLog.Nutrients.Calories)
	assert.Equal(s.T(), util.Ptr(12.0), savedLog.Nutrients.ProteinG)
	assert.Equal(s.T(), util.Ptr(8.0), savedLog.Nutrients.TotalFatG)
	assert.Equal(s.T(), util.Ptr(35.0), savedLog.Nutrients.CarbohydratesG)

}

func (s *IntegrationTestSuite) TestLogCustomFood_WithOptionalNutrients() {
	ctx := s.Context()

	now := time.Now().UTC().Truncate(time.Microsecond)

	// Call log_custom_food MCP tool with optional nutrients
	input := log_food.LogCustomFoodInput{
		ProductName:    "Energy Drink",
		AmountG:        250.0,
		Calories:       110.0,
		ProteinG:       0.0,
		TotalFatG:      0.0,
		CarbohydratesG: 28.0,
		CaffeineMg:     80.0, // Optional caffeine
		EthylAlcoholG:  0.0,
		ConsumedAt:     now,
	}

	_, response, err := log_food.LogCustomFood(ctx, nil, input)
	require.NoError(s.T(), err)
	assert.Empty(s.T(), response.Error)
	assert.Contains(s.T(), response.Message, "Successfully logged 250.0g of Energy Drink")

	// Verify log was saved with optional nutrients
	savedLog, err := s.Repo().GetConsumptionLog(ctx, s.UserID(), now)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), util.Ptr(80.0), savedLog.Nutrients.CaffeineMg)

}

func (s *IntegrationTestSuite) TestValidationErrors() {
	ctx := s.Context()

	// Test log_food_by_id with invalid food_id
	input1 := log_food.LogFoodByIdInput{
		FoodID:  0, // Invalid
		AmountG: 100.0,
	}
	_, response1, err := log_food.LogFoodById(ctx, nil, input1)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), "food_id must be greater than 0", response1.Error)

	// Test log_food_by_barcode with empty barcode
	input3 := log_food.LogFoodByBarcodeInput{
		Barcode: "", // Invalid
		AmountG: 100.0,
	}
	_, response3, err := log_food.LogFoodByBarcode(ctx, nil, input3)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), "barcode cannot be empty", response3.Error)

	// Test log_custom_food with empty product name
	input4 := log_food.LogCustomFoodInput{
		ProductName:    "", // Invalid
		AmountG:        100.0,
		Calories:       100.0,
		ProteinG:       5.0,
		TotalFatG:      3.0,
		CarbohydratesG: 15.0,
	}
	_, response4, err := log_food.LogCustomFood(ctx, nil, input4)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), "product_name cannot be empty", response4.Error)

	// Test invalid amounts (no amount_g or serving_count)
	input5 := log_food.LogFoodByIdInput{
		FoodID:       1,
		AmountG:      0,
		ServingCount: 0, // Both zero - invalid
	}
	_, response5, err := log_food.LogFoodById(ctx, nil, input5)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), "either amount_g or serving_count must be greater than 0", response5.Error)
}

// Test helper to create food items for testing (reused from old test)
func (s *IntegrationTestSuite) createTestFood(
	ctx context.Context,
	name, barcode string,
	servingSizeG float64,
	nutrients *domain.Nutrients,
) *domain.Food {
	food := &domain.Food{
		Name:      name,
		UserID:    s.UserID(3),
		FoodType:  "product",
		Nutrients: nutrients,
	}

	if barcode != "" {
		food.Barcode = &barcode
	}

	if servingSizeG > 0 {
		food.ServingSizeG = &servingSizeG
	}

	id, err := s.Repo().CreateFood(ctx, food)
	require.NoError(s.T(), err)
	food.ID = id

	return food
}
