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

var LogFoodByBarcodeMCPDefinition = mcp.Tool{
	Name: "log_food_by_barcode",
	Annotations: &mcp.ToolAnnotations{
		DestructiveHint: util.Ptr(true),
		Title:           "Log food consumption by barcode scan",
	},
	Description: `Find a food product by its barcode (UPC/EAN) and log its consumption.

This tool is perfect for logging packaged food products that have barcodes. Simply scan or enter the barcode, and the tool will find the exact product in the database and log your consumption.

Required input:
- barcode: The product barcode (UPC, EAN-13, EAN-8, etc.)
- Either amount_g (grams) OR serving_count (number of servings) - never both

Optional input:
- meal_type: breakfast/lunch/dinner/snack categorization
- consumed_at: specific timestamp (defaults to current time)
- note: any additional notes about this consumption

The tool:
- Searches for foods with matching barcode in the database
- Returns "food not found" error if no product with that barcode exists
- Automatically logs consumption with calculated nutritional values
- Uses the product's serving size information for accurate nutrition calculation

Perfect for:
- Logging packaged foods, snacks, drinks with barcodes
- Quick and precise product identification without name ambiguity
- Mobile apps with barcode scanning functionality
- Ensuring exact product match (brand, variant, size specific)

This creates a permanent consumption log entry linked to the specific product.`,
}

// LogFoodByBarcode is the MCP handler for logging food consumption by barcode search
func LogFoodByBarcode(ctx context.Context, _ *mcp.CallToolRequest, input LogFoodByBarcodeInput) (*mcp.CallToolResult, ToolResponse, error) {
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
		UserID:     userID,
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
