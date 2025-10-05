package log_food

import (
	"context"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"personal/action/find_food"
	"personal/domain"
	"personal/gateways"
	"personal/util"
)

var LogFoodByNameMCPDefinition = mcp.Tool{
	Name: "log_food_by_name",
	Annotations: &mcp.ToolAnnotations{
		DestructiveHint: util.Ptr(true),
		Title:           "Search and log food consumption by name",
	},
	Description: `Search for a food by name and immediately log its consumption in one convenient step.

This tool combines food search and consumption logging into a single operation. You provide a food name, and the tool searches the database, handles the results, and creates a consumption log entry.

Required input:
- name: The food name to search for (e.g., "банан", "chicken breast")
- Either amount_g (grams) OR serving_count (number of servings) - never both

Optional input:
- meal_type: breakfast/lunch/dinner/snack categorization
- consumed_at: specific timestamp (defaults to current time)
- note: any additional notes about this consumption

Smart behavior:
- If exactly 1 food found: automatically logs consumption
- If 0 foods found: returns "food not found" error
- If multiple foods found: returns up to 2 suggestions with IDs and names for you to choose from

The tool automatically calculates nutritional values and records the consumption with all relevant data.

Use this tool when:
- You want to quickly log food without knowing the exact ID
- You're confident about the food name (single expected match)
- You want the convenience of search + log in one step

For ambiguous names, use resolve_food_id_by_name first, then log_food_by_id for precision.`,
}

// LogFoodByName is the MCP handler for logging food consumption by name search
func LogFoodByName(ctx context.Context, _ *mcp.CallToolRequest, input LogFoodByNameInput) (*mcp.CallToolResult, LogFoodByNameOutput, error) {
	// Get database from context
	db := gateways.DBFromContext(ctx)
	if db == nil {
		return nil, LogFoodByNameOutput{}, fmt.Errorf("database not available in context")
	}

	// Get user ID from context
	userID := gateways.UserIDFromContext(ctx)
	if userID == 0 {
		return nil, LogFoodByNameOutput{}, fmt.Errorf("user_id not available in context")
	}

	// 1. Validate input
	if input.Name == "" {
		return nil, LogFoodByNameOutput{Error: "name cannot be empty"}, nil
	}
	if input.AmountG <= 0 && input.ServingCount <= 0 {
		return nil, LogFoodByNameOutput{Error: "either amount_g or serving_count must be greater than 0"}, nil
	}

	// 2. Search foods by name using shared search function
	foods, err := find_food.SearchFoodsByName(ctx, db, input.Name)
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
		return logFoodConsumption(ctx, db, userID, food, input.AmountG, input.ServingCount, input.MealType, input.ConsumedAt, input.Note)

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
func logFoodConsumption(ctx context.Context, db gateways.DB, userID int64, food *domain.Food, amountG, servingCount float64, mealType string, consumedAt time.Time, note string) (*mcp.CallToolResult, LogFoodByNameOutput, error) {
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
		UserID:     userID,
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
