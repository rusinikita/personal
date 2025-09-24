package log_food

import (
	"context"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"personal/domain"
	"personal/gateways"
)

var LogFoodByNameMCPDefinition = mcp.Tool{
	Name:        "log_food_by_name",
	Description: "Search food by name and log consumption",
}

// LogFoodByName is the MCP handler for logging food consumption by name search
func LogFoodByName(ctx context.Context, _ *mcp.CallToolRequest, input LogFoodByNameInput) (*mcp.CallToolResult, LogFoodByNameOutput, error) {
	// Get database from context
	db := gateways.DBFromContext(ctx)
	if db == nil {
		return nil, LogFoodByNameOutput{}, fmt.Errorf("database not available in context")
	}

	// 1. Validate input
	if input.Name == "" {
		return nil, LogFoodByNameOutput{Error: "name cannot be empty"}, nil
	}
	if input.AmountG <= 0 && input.ServingCount <= 0 {
		return nil, LogFoodByNameOutput{Error: "either amount_g or serving_count must be greater than 0"}, nil
	}

	// 2. Search foods by name
	filter := domain.FoodFilter{Name: &input.Name}
	foods, err := db.SearchFood(ctx, filter)
	if err != nil {
		return nil, LogFoodByNameOutput{Error: fmt.Sprintf("search failed: %v", err)}, nil
	}

	// 3. Handle search results
	switch len(foods) {
	case 0:
		return nil, LogFoodByNameOutput{Error: "food not found"}, nil

	case 1:
		// Exact match - proceed with logging
		food := foods[0]
		return logFoodConsumption(ctx, db, food, input.AmountG, input.ServingCount, input.MealType, input.ConsumedAt, input.Note)

	default:
		// Multiple matches - return suggestions
		suggestions := make([]FoodMatch, 0, 2)
		for i, food := range foods {
			if i >= 2 {
				break
			}
			suggestions = append(suggestions, FoodMatch{
				ID:   food.ID,
				Name: food.Name,
			})
		}
		return nil, LogFoodByNameOutput{
			Error:       "multiple matches found",
			Suggestions: suggestions,
		}, nil
	}
}

// logFoodConsumption is a helper function to log food consumption (shared logic)
func logFoodConsumption(ctx context.Context, db gateways.DB, food *domain.Food, amountG, servingCount float64, mealType string, consumedAt time.Time, note string) (*mcp.CallToolResult, LogFoodByNameOutput, error) {
	// Calculate final amount_g
	finalAmountG := amountG
	if finalAmountG == 0 {
		if food.ServingSizeG == nil {
			return nil, LogFoodByNameOutput{Error: "food has no serving size, amount_g is required"}, nil
		}
		finalAmountG = servingCount * (*food.ServingSizeG)
	}

	// Calculate nutrients proportionally
	if food.Nutrients == nil {
		return nil, LogFoodByNameOutput{Error: "food has no nutrients data"}, nil
	}

	nutrients := domain.CalculateProportionalNutrients(food.Nutrients, finalAmountG)

	// Prepare consumption log
	finalConsumedAt := consumedAt
	if finalConsumedAt.IsZero() {
		finalConsumedAt = time.Now().UTC()
	}

	var mealTypePtr *string
	if mealType != "" {
		mealTypePtr = &mealType
	}

	var notePtr *string
	if note != "" {
		notePtr = &note
	}

	log := &domain.ConsumptionLog{
		UserID:     DEFAULT_USER_ID,
		ConsumedAt: finalConsumedAt,
		FoodID:     &food.ID,
		FoodName:   food.Name,
		AmountG:    finalAmountG,
		MealType:   mealTypePtr,
		Note:       notePtr,
		Nutrients:  nutrients,
	}

	// Save consumption log
	if err := db.AddConsumptionLog(ctx, log); err != nil {
		return nil, LogFoodByNameOutput{Error: fmt.Sprintf("failed to save consumption log: %v", err)}, nil
	}

	return nil, LogFoodByNameOutput{
		Message: fmt.Sprintf("Successfully logged %.1fg of %s", finalAmountG, food.Name),
	}, nil
}
