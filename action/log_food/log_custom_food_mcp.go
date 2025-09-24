package log_food

import (
	"context"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"personal/domain"
	"personal/gateways"
)

var LogCustomFoodMCPDefinition = mcp.Tool{
	Name:        "log_custom_food",
	Description: "Log food consumption with direct nutrient specification",
}

// LogCustomFood is the MCP handler for logging custom food with direct nutrients
func LogCustomFood(ctx context.Context, _ *mcp.CallToolRequest, input LogCustomFoodInput) (*mcp.CallToolResult, ToolResponse, error) {
	// Get database from context
	db := gateways.DBFromContext(ctx)
	if db == nil {
		return nil, ToolResponse{}, fmt.Errorf("database not available in context")
	}

	// 1. Validate input
	if input.ProductName == "" {
		return nil, ToolResponse{Error: "product_name cannot be empty"}, nil
	}
	if input.AmountG <= 0 {
		return nil, ToolResponse{Error: "amount_g must be greater than 0"}, nil
	}
	if input.Calories < 0 || input.ProteinG < 0 || input.TotalFatG < 0 || input.CarbohydratesG < 0 {
		return nil, ToolResponse{Error: "all required nutrients must be >= 0"}, nil
	}

	// 2. Create nutrients from provided values (already for specified amount)
	nutrients := &domain.Nutrients{
		Calories:       &input.Calories,
		ProteinG:       &input.ProteinG,
		TotalFatG:      &input.TotalFatG,
		CarbohydratesG: &input.CarbohydratesG,
	}

	// Add optional nutrients if provided
	if input.CaffeineMg > 0 {
		nutrients.CaffeineMg = &input.CaffeineMg
	}
	if input.EthylAlcoholG > 0 {
		nutrients.EthylAlcoholG = &input.EthylAlcoholG
	}

	// 3. Prepare consumption log
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
		FoodID:     nil, // No food ID for custom food
		FoodName:   input.ProductName,
		AmountG:    input.AmountG,
		MealType:   mealType,
		Note:       note,
		Nutrients:  nutrients,
	}

	// 4. Save consumption log
	if err := db.AddConsumptionLog(ctx, log); err != nil {
		return nil, ToolResponse{Error: fmt.Sprintf("failed to save consumption log: %v", err)}, nil
	}

	return nil, ToolResponse{
		Message: fmt.Sprintf("Successfully logged %.1fg of %s", input.AmountG, input.ProductName),
	}, nil
}
