package log_food

import (
	"fmt"

	"personal/domain"
	"personal/util"
)

// calculateNutrients calculates nutrients based on amount for found food
func calculateNutrients(food *domain.Food, item ConsumedFoodItem) (*domain.Nutrients, float64, error) {
	// Calculate actual amount in grams
	actualAmountG, err := calculateActualAmount(food, item)
	if err != nil {
		return nil, 0, err
	}

	// If food has no nutrients, return nil
	if food.Nutrients == nil {
		return nil, actualAmountG, nil
	}

	// Calculate proportional nutrients (base nutrients are per 100g)
	ratio := actualAmountG / 100.0
	calculatedNutrients := domain.CalculateProportionalNutrients(food.Nutrients, ratio)

	return calculatedNutrients, actualAmountG, nil
}

// calculateActualAmount determines the actual amount in grams
func calculateActualAmount(food *domain.Food, item ConsumedFoodItem) (float64, error) {
	// If amount_g is provided directly
	if item.AmountG > 0 {
		return item.AmountG, nil
	}

	// If serving_count is provided, convert to grams
	if item.ServingCount != nil && *item.ServingCount > 0 {
		if food.ServingSizeG == nil {
			return 0, fmt.Errorf("food '%s' has no serving_size_g defined, cannot use serving_count", food.Name)
		}
		return *item.ServingCount * *food.ServingSizeG, nil
	}

	return 0, fmt.Errorf("no valid amount specified")
}

// createDirectNutrients creates nutrients from DirectNutrients input
func createDirectNutrients(direct DirectNutrients) *domain.Nutrients {
	calories := domain.RoundTo3Decimals(direct.Calories)
	protein := domain.RoundTo3Decimals(direct.ProteinG)
	totalFat := domain.RoundTo3Decimals(direct.TotalFatG)
	carbs := domain.RoundTo3Decimals(direct.CarbohydratesG)

	nutrients := &domain.Nutrients{
		Calories:       util.Ptr(calories),
		ProteinG:       util.Ptr(protein),
		TotalFatG:      util.Ptr(totalFat),
		CarbohydratesG: util.Ptr(carbs),
	}

	// Add optional nutrients if provided
	if direct.CaffeineMg != nil {
		caffeine := domain.RoundTo3Decimals(*direct.CaffeineMg)
		nutrients.CaffeineMg = util.Ptr(caffeine)
	}
	if direct.EthylAlcoholG != nil {
		alcohol := domain.RoundTo3Decimals(*direct.EthylAlcoholG)
		nutrients.EthylAlcoholG = util.Ptr(alcohol)
	}

	return nutrients
}
