package log_food

import (
	"fmt"
)

// validateInput validates the LogFoodInput and returns error if validation fails
func validateInput(input LogFoodInput) error {
	if len(input.ConsumedItems) == 0 {
		return fmt.Errorf("consumed_items cannot be empty")
	}

	for i, item := range input.ConsumedItems {
		if err := validateConsumedFoodItem(item, i); err != nil {
			return fmt.Errorf("item %d: %w", i, err)
		}
	}

	return nil
}

// validateConsumedFoodItem validates a single consumed food item
func validateConsumedFoodItem(item ConsumedFoodItem, index int) error {
	// Check that exactly one scenario field is provided
	scenarioCount := 0

	if item.FoodID != nil {
		scenarioCount++
	}
	if item.Name != nil && *item.Name != "" {
		scenarioCount++
	}
	if item.Barcode != nil && *item.Barcode != "" {
		scenarioCount++
	}
	if item.DirectNutrients != nil {
		scenarioCount++
	}

	if scenarioCount == 0 {
		return fmt.Errorf("must provide one of: food_id, name, barcode, or direct_nutrients")
	}
	if scenarioCount > 1 {
		return fmt.Errorf("must provide only one of: food_id, name, barcode, or direct_nutrients")
	}

	// Validate amount specification
	hasAmount := item.AmountG > 0
	hasServing := item.ServingCount != nil && *item.ServingCount > 0

	if !hasAmount && !hasServing {
		return fmt.Errorf("must provide either amount_g > 0 or serving_count > 0")
	}

	// Both amount and serving count provided is not allowed
	if hasAmount && hasServing {
		return fmt.Errorf("cannot provide both amount_g and serving_count, use only one")
	}

	// Validate direct nutrients if provided
	if item.DirectNutrients != nil {
		if err := validateDirectNutrients(*item.DirectNutrients); err != nil {
			return fmt.Errorf("direct_nutrients: %w", err)
		}
	}

	return nil
}

// validateDirectNutrients validates DirectNutrients structure
func validateDirectNutrients(nutrients DirectNutrients) error {
	if nutrients.ProductName == "" {
		return fmt.Errorf("product_name is required")
	}

	// All macronutrients must be non-negative
	if nutrients.Calories < 0 {
		return fmt.Errorf("calories must be non-negative")
	}
	if nutrients.ProteinG < 0 {
		return fmt.Errorf("protein_g must be non-negative")
	}
	if nutrients.TotalFatG < 0 {
		return fmt.Errorf("total_fat_g must be non-negative")
	}
	if nutrients.CarbohydratesG < 0 {
		return fmt.Errorf("carbohydrates_g must be non-negative")
	}

	// Optional nutrients must be non-negative if provided
	if nutrients.CaffeineMg != nil && *nutrients.CaffeineMg < 0 {
		return fmt.Errorf("caffeine_mg must be non-negative")
	}
	if nutrients.EthylAlcoholG != nil && *nutrients.EthylAlcoholG < 0 {
		return fmt.Errorf("ethyl_alcohol_g must be non-negative")
	}

	return nil
}
