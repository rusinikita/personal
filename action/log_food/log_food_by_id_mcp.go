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
	Name: "log_food_by_id",
	Annotations: &mcp.ToolAnnotations{
		DestructiveHint: util.Ptr(true),
		Title:           "Log food consumption by database ID",
	},
	Description: `Log food consumption using a known food database ID for precise tracking.

This tool records a consumption entry when you already know the exact food_id from the database. It's the most precise way to log food consumption as it directly references an existing food without any ambiguity.

Required input:
- food_id: The exact database ID of the food to log
- Either amount_g (grams) OR serving_count (number of servings) - never both

Optional input:
- meal_type: breakfast/lunch/dinner/snack categorization
- consumed_at: specific timestamp (defaults to current time)
- note: any additional notes about this consumption

The tool automatically:
- Validates the food exists in the database
- Calculates nutritional values proportionally based on amount
- Converts serving_count to grams using the food's serving_size_g
- Records consumption with calculated nutrients for tracking

Use this tool when:
- You have the exact food_id from resolve_food_id_by_name results
- You want precise logging without name ambiguity
- You're building automated food logging workflows

This creates a permanent consumption log entry in the database.`,
}

// LogFoodById is the MCP handler for logging food consumption by ID
func LogFoodById(ctx context.Context, _ *mcp.CallToolRequest, input LogFoodByIdInput) (*mcp.CallToolResult, ToolResponse, error) {
	// Get database from context
	db := gateways.DBFromContext(ctx)
	if db == nil {
		return nil, ToolResponse{}, fmt.Errorf("database not available in context")
	}

	// Get user ID from context
	userID := gateways.UserIDFromContext(ctx)
	if userID == 0 {
		return nil, ToolResponse{}, fmt.Errorf("user_id not available in context")
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
		UserID:     userID,
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
