package add_food

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"personal/domain"
	"personal/gateways"
)

var MCPDefinition = mcp.Tool{
	Name:        "add_food",
	Description: "Add a new food item to the database with nutritional information",
}

type AddFoodInput struct {
	Name            string                   `json:"name" jsonschema:"required,description=Food name"`
	Description     *string                  `json:"description,omitempty" jsonschema:"description=Food description"`
	Barcode         *string                  `json:"barcode,omitempty" jsonschema:"description=Product barcode"`
	FoodType        string                   `json:"food_type" jsonschema:"required,enum=component|product|dish,description=Type of food item"`
	ServingSizeG    *float64                 `json:"serving_size_g,omitempty" jsonschema:"description=Standard serving size in grams"`
	ServingName     *string                  `json:"serving_name,omitempty" jsonschema:"description=Name of serving (e.g. cookie, slice)"`
	Nutrients       *domain.Nutrients        `json:"nutrients,omitempty" jsonschema:"description=Nutritional information per 100g"`
	FoodComposition domain.FoodComponentList `json:"food_composition,omitempty" jsonschema:"description=Recipe composition for dishes"`
}

type AddFoodOutput struct {
	ID      int64  `json:"id" jsonschema:"description=Created food ID"`
	Message string `json:"message" jsonschema:"description=Success message"`
}

func AddFood(ctx context.Context, _ *mcp.CallToolRequest, input AddFoodInput, db gateways.DB) (*mcp.CallToolResult, AddFoodOutput, error) {
	// 1. Validate input
	if err := validateInput(input); err != nil {
		return nil, AddFoodOutput{}, fmt.Errorf("validation error: %w", err)
	}

	// 2. Check for duplicates (simple name check for now)
	// TODO: Implement proper duplicate checking by name and barcode

	// 3. Handle nutrients calculation
	nutrients, err := calculateNutrients(ctx, input, db)
	if err != nil {
		return nil, AddFoodOutput{}, fmt.Errorf("nutrient calculation error: %w", err)
	}

	// 4. Create domain Food object
	food := &domain.Food{
		Name:            input.Name,
		Description:     input.Description,
		Barcode:         input.Barcode,
		FoodType:        input.FoodType,
		IsArchived:      false,
		ServingSizeG:    input.ServingSizeG,
		ServingName:     input.ServingName,
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

	if input.ServingSizeG != nil && *input.ServingSizeG <= 0 {
		return fmt.Errorf("serving_size_g must be positive")
	}

	return nil
}

func calculateNutrients(ctx context.Context, input AddFoodInput, db gateways.DB) (*domain.Nutrients, error) {
	// If nutrients are provided directly, use them
	if input.Nutrients != nil {
		return input.Nutrients, nil
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
		addProportionalNutrients(totalNutrients, componentFood.Nutrients, ratio)
	}

	return totalNutrients, nil
}

func addProportionalNutrients(total *domain.Nutrients, component *domain.Nutrients, ratio float64) {
	// Macronutrients
	if component.Calories != nil {
		if total.Calories == nil {
			val := *component.Calories * ratio
			total.Calories = &val
		} else {
			*total.Calories += *component.Calories * ratio
		}
	}

	if component.ProteinG != nil {
		if total.ProteinG == nil {
			val := *component.ProteinG * ratio
			total.ProteinG = &val
		} else {
			*total.ProteinG += *component.ProteinG * ratio
		}
	}

	if component.TotalFatG != nil {
		if total.TotalFatG == nil {
			val := *component.TotalFatG * ratio
			total.TotalFatG = &val
		} else {
			*total.TotalFatG += *component.TotalFatG * ratio
		}
	}

	if component.CarbohydratesG != nil {
		if total.CarbohydratesG == nil {
			val := *component.CarbohydratesG * ratio
			total.CarbohydratesG = &val
		} else {
			*total.CarbohydratesG += *component.CarbohydratesG * ratio
		}
	}

	// TODO: Add similar calculations for other nutrients as needed
}
