package log_food

import (
	"context"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"personal/domain"
	"personal/gateways"
	"personal/util"
)

var LogFoodByIdMCPDefinition = mcp.Tool{
	Name:        "log_food_by_id",
	Description: "Log food consumption by known food ID",
	Annotations: &mcp.ToolAnnotations{
		DestructiveHint: util.Ptr(true),
		Title:           "Add consumed food by id",
	},
}

// LogFoodById is the MCP handler for logging food consumption by ID
func LogFoodById(ctx context.Context, _ *mcp.CallToolRequest, input LogFoodByIdInput) (*mcp.CallToolResult, ToolResponse, error) {
	// Get database from context
	db := gateways.DBFromContext(ctx)
	if db == nil {
		return nil, ToolResponse{}, fmt.Errorf("database not available in context")
	}

	// 1. Validate input
	if input.FoodID <= 0 {
		return nil, ToolResponse{Error: "food_id must be greater than 0"}, nil
	}
	if input.AmountG <= 0 && input.ServingCount <= 0 {
		return nil, ToolResponse{Error: "either amount_g or serving_count must be greater than 0"}, nil
	}

	// 2. Get food from database
	food, err := db.GetFood(ctx, input.FoodID)
	if err != nil {
		return nil, ToolResponse{Error: "food not found"}, nil
	}

	// 3. Calculate final amount_g
	finalAmountG := input.AmountG
	if finalAmountG == 0 {
		if food.ServingSizeG == nil {
			return nil, ToolResponse{Error: "food has no serving size, amount_g is required"}, nil
		}
		finalAmountG = input.ServingCount * (*food.ServingSizeG)
	}

	// 4. Calculate nutrients proportionally
	if food.Nutrients == nil {
		return nil, ToolResponse{Error: "food has no nutrients data"}, nil
	}

	nutrients := domain.CalculateProportionalNutrients(food.Nutrients, finalAmountG)

	// 5. Prepare consumption log
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

	// 6. Save consumption log
	if err := db.AddConsumptionLog(ctx, log); err != nil {
		return nil, ToolResponse{Error: fmt.Sprintf("failed to save consumption log: %v", err)}, nil
	}

	return nil, ToolResponse{
		Message: fmt.Sprintf("Successfully logged %.1fg of %s", finalAmountG, food.Name),
	}, nil
}
