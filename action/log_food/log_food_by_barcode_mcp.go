package log_food

import (
	"context"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"personal/domain"
	"personal/gateways"
)

var LogFoodByBarcodeMCPDefinition = mcp.Tool{
	Name:        "log_food_by_barcode",
	Description: "Find food by barcode and log consumption",
}

// LogFoodByBarcode is the MCP handler for logging food consumption by barcode search
func LogFoodByBarcode(ctx context.Context, _ *mcp.CallToolRequest, input LogFoodByBarcodeInput) (*mcp.CallToolResult, ToolResponse, error) {
	// Get database from context
	db := gateways.DBFromContext(ctx)
	if db == nil {
		return nil, ToolResponse{}, fmt.Errorf("database not available in context")
	}

	// 1. Validate input
	if input.Barcode == "" {
		return nil, ToolResponse{Error: "barcode cannot be empty"}, nil
	}
	if input.AmountG <= 0 && input.ServingCount <= 0 {
		return nil, ToolResponse{Error: "either amount_g or serving_count must be greater than 0"}, nil
	}

	// 2. Search food by barcode
	filter := domain.FoodFilter{Barcode: &input.Barcode}
	foods, err := db.SearchFood(ctx, filter)
	if err != nil {
		return nil, ToolResponse{Error: fmt.Sprintf("search failed: %v", err)}, nil
	}

	// 3. Handle search results
	if len(foods) == 0 {
		return nil, ToolResponse{Error: "barcode not found"}, nil
	}

	// Barcode should be unique, so we take the first result
	food := foods[0]

	// 4. Calculate final amount_g
	finalAmountG := input.AmountG
	if finalAmountG == 0 {
		if food.ServingSizeG == nil {
			return nil, ToolResponse{Error: "food has no serving size, amount_g is required"}, nil
		}
		finalAmountG = input.ServingCount * (*food.ServingSizeG)
	}

	// 5. Calculate nutrients proportionally
	if food.Nutrients == nil {
		return nil, ToolResponse{Error: "food has no nutrients data"}, nil
	}

	nutrients := domain.CalculateProportionalNutrients(food.Nutrients, finalAmountG)

	// 6. Prepare consumption log
	consumedAt := input.ConsumedAt
	if consumedAt.IsZero() {
		consumedAt = time.Now().UTC()
	}

	var mealType *string
	if input.MealType != "" {
		mealType = &input.MealType
	}

	var note *string
	if input.Note != "" {
		note = &input.Note
	}

	log := &domain.ConsumptionLog{
		UserID:     DEFAULT_USER_ID,
		ConsumedAt: consumedAt,
		FoodID:     &food.ID,
		FoodName:   food.Name,
		AmountG:    finalAmountG,
		MealType:   mealType,
		Note:       note,
		Nutrients:  nutrients,
	}

	// 7. Save consumption log
	if err := db.AddConsumptionLog(ctx, log); err != nil {
		return nil, ToolResponse{Error: fmt.Sprintf("failed to save consumption log: %v", err)}, nil
	}

	return nil, ToolResponse{
		Message: fmt.Sprintf("Successfully logged %.1fg of %s", finalAmountG, food.Name),
	}, nil
}
