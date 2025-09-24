package add_food

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"personal/domain"
	"personal/gateways"
	"personal/util"
)

var MCPDefinition = mcp.Tool{
	Name:        "add_food",
	Description: "Add a new food item to the database with nutritional information",
}

type AddFoodInput struct {
	Name            string                   `json:"name" jsonschema:"Food name"`
	Description     string                   `json:"description,omitempty" jsonschema:"Food one sentence summary description"`
	Barcode         string                   `json:"barcode,omitempty" jsonschema:"Product barcode"`
	FoodType        string                   `json:"food_type" jsonschema:"Type of food item (component|product|dish)"`
	ServingSizeG    float64                  `json:"serving_size_g,omitempty" jsonschema:"Standard serving size in grams"`
	ServingName     string                   `json:"serving_name,omitempty" jsonschema:"Name of serving (e.g. cookie, slice)"`
	Nutrients       *domain.BasicNutrients   `json:"nutrients,omitempty" jsonschema:"Nutritional information per 100g"`
	FoodComposition domain.FoodComponentList `json:"food_composition,omitempty" jsonschema:"Recipe composition for dishes"`
}

type AddFoodOutput struct {
	ID      int64  `json:"id" jsonschema:"Created food ID"`
	Message string `json:"message" jsonschema:"Success message"`
}

func AddFood(ctx context.Context, _ *mcp.CallToolRequest, input AddFoodInput) (*mcp.CallToolResult, AddFoodOutput, error) {
	// Get database from context
	db := gateways.DBFromContext(ctx)
	if db == nil {
		return nil, AddFoodOutput{}, fmt.Errorf("database not available in context")
	}
	// 1. Validate input
	if err := validateInput(input); err != nil {
		return nil, AddFoodOutput{}, fmt.Errorf("validation error: %w", err)
	}

	// 2. Check for duplicates
	duplicateMsg, err := checkForDuplicates(ctx, input, db)
	if err != nil {
		return nil, AddFoodOutput{}, fmt.Errorf("duplicate check error: %w", err)
	}
	if duplicateMsg != "" {
		return nil, AddFoodOutput{}, fmt.Errorf("duplicate food found: %s", duplicateMsg)
	}

	// 3. Handle nutrients calculation
	nutrients, err := calculateNutrients(ctx, input, db)
	if err != nil {
		return nil, AddFoodOutput{}, fmt.Errorf("nutrient calculation error: %w", err)
	}

	// 4. Create domain Food object
	food := &domain.Food{
		Name:            input.Name,
		Description:     util.PtrIfNotEmpty(input.Description),
		Barcode:         util.PtrIfNotEmpty(input.Barcode),
		FoodType:        input.FoodType,
		IsArchived:      false,
		ServingSizeG:    util.PtrIfNotZero(input.ServingSizeG),
		ServingName:     util.PtrIfNotEmpty(input.ServingName),
		Nutrients:       nutrients,
		FoodComposition: input.FoodComposition,
	}

	// 5. Save to database
	id, err := db.AddFood(ctx, food)
	if err != nil {
		return nil, AddFoodOutput{}, fmt.Errorf("database error: %w", err)
	}

	// 6. Return success response
	return nil, AddFoodOutput{
		ID:      id,
		Message: fmt.Sprintf("Food '%s' added successfully with ID %d", input.Name, id),
	}, nil
}

func validateInput(input AddFoodInput) error {
	if strings.TrimSpace(input.Name) == "" {
		return fmt.Errorf("name is required")
	}

	validTypes := map[string]bool{
		"component": true,
		"product":   true,
		"dish":      true,
	}
	if !validTypes[input.FoodType] {
		return fmt.Errorf("food_type must be one of: component, product, dish")
	}

	if input.ServingSizeG != 0 && input.ServingSizeG <= 0 {
		return fmt.Errorf("serving_size_g must be positive")
	}

	return nil
}

func calculateNutrients(ctx context.Context, input AddFoodInput, db gateways.DB) (*domain.Nutrients, error) {
	// If nutrients are provided directly, use them
	if input.Nutrients != nil {
		return input.Nutrients.ToFull(), nil
	}

	// If only composition is provided, calculate nutrients from components
	if len(input.FoodComposition) > 0 {
		return calculateNutrientsFromComposition(ctx, input.FoodComposition, db)
	}

	// No nutrients provided - return nil (optional field)
	return nil, nil
}

func calculateNutrientsFromComposition(ctx context.Context, composition domain.FoodComponentList, db gateways.DB) (*domain.Nutrients, error) {
	totalNutrients := &domain.Nutrients{}

	for _, component := range composition {
		// Get component food data
		componentFood, err := db.GetFood(ctx, component.FoodID)
		if err != nil {
			return nil, fmt.Errorf("failed to get component food %d: %w", component.FoodID, err)
		}

		if componentFood.Nutrients == nil {
			continue // Skip components without nutrition data
		}

		// Calculate proportional nutrients (component.AmountG / 100g * nutrient value)
		ratio := component.AmountG / 100.0
		domain.AddProportionalNutrients(totalNutrients, componentFood.Nutrients, ratio)
	}

	return totalNutrients, nil
}

// checkForDuplicates searches for duplicate foods by name and barcode
// Returns error message if duplicate found, empty string if no duplicates
func checkForDuplicates(ctx context.Context, input AddFoodInput, db gateways.DB) (string, error) {
	// Check for name duplicates first (exact name match)
	nameFilter := &domain.FoodFilter{Name: &input.Name}
	nameMatches, err := db.SearchFood(ctx, *nameFilter)
	if err != nil {
		return "", fmt.Errorf("name search failed: %w", err)
	}

	if len(nameMatches) > 0 {
		food := nameMatches[0]
		return fmt.Sprintf("food with name '%s' already exists (ID: %d)", food.Name, food.ID), nil
	}

	// Check for barcode duplicates (if barcode is provided)
	if input.Barcode != "" {
		barcodeFilter := &domain.FoodFilter{Barcode: &input.Barcode}
		barcodeMatches, err := db.SearchFood(ctx, *barcodeFilter)
		if err != nil {
			return "", fmt.Errorf("barcode search failed: %w", err)
		}

		if len(barcodeMatches) > 0 {
			food := barcodeMatches[0]
			return fmt.Sprintf("food with barcode '%s' already exists: '%s' (ID: %d)", input.Barcode, food.Name, food.ID), nil
		}
	}

	return "", nil
}
