package log_food

import (
	"fmt"
	"math"
	"reflect"

	"personal/domain"
)

// roundTo3Decimals rounds a float64 value to 3 decimal places
func roundTo3Decimals(value float64) float64 {
	return math.Round(value*1000) / 1000
}

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
	calculatedNutrients := calculateProportionalNutrients(food.Nutrients, ratio)

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

// calculateProportionalNutrients calculates proportional nutrients based on ratio using reflection
func calculateProportionalNutrients(baseNutrients *domain.Nutrients, ratio float64) *domain.Nutrients {
	nutrients := baseNutrients

	baseValue := reflect.ValueOf(nutrients).Elem()
	baseType := baseValue.Type()

	// Iterate through all fields in the struct
	for i := 0; i < baseValue.NumField(); i++ {
		sourceField := baseValue.Field(i)
		fieldType := baseType.Field(i).Type

		if sourceField.IsNil() {
			continue
		}

		// Skip if field is not settable
		if !sourceField.CanSet() {
			continue
		}

		switch fieldType.Elem().Kind() {
		case reflect.Float64:
			// Handle *float64 fields
			sourceVal := sourceField.Elem().Float()
			calculatedVal := roundTo3Decimals(sourceVal * ratio)
			newVal := reflect.New(reflect.TypeOf(float64(0)))
			newVal.Elem().SetFloat(calculatedVal)
			sourceField.Set(newVal)
		case reflect.Int:
			// Handle *int fields (like GlycemicIndex)
			sourceVal := float64(sourceField.Elem().Int())
			calculatedVal := int(math.Round(sourceVal * ratio))
			newVal := reflect.New(reflect.TypeOf(int(0)))
			newVal.Elem().SetInt(int64(calculatedVal))
			sourceField.Set(newVal)
		default:
		}
	}

	return nutrients
}

// createDirectNutrients creates nutrients from DirectNutrients input
func createDirectNutrients(direct DirectNutrients) *domain.Nutrients {
	calories := roundTo3Decimals(direct.Calories)
	protein := roundTo3Decimals(direct.ProteinG)
	totalFat := roundTo3Decimals(direct.TotalFatG)
	carbs := roundTo3Decimals(direct.CarbohydratesG)

	nutrients := &domain.Nutrients{
		Calories:       &calories,
		ProteinG:       &protein,
		TotalFatG:      &totalFat,
		CarbohydratesG: &carbs,
	}

	// Add optional nutrients if provided
	if direct.CaffeineMg != nil {
		caffeine := roundTo3Decimals(*direct.CaffeineMg)
		nutrients.CaffeineMg = &caffeine
	}
	if direct.EthylAlcoholG != nil {
		alcohol := roundTo3Decimals(*direct.EthylAlcoholG)
		nutrients.EthylAlcoholG = &alcohol
	}

	return nutrients
}
