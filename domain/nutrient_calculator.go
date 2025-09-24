package domain

import (
	"math"
	"reflect"
)

// RoundTo3Decimals rounds a float64 value to 3 decimal places
func RoundTo3Decimals(value float64) float64 {
	return math.Round(value*1000) / 1000
}

// CalculateProportionalNutrients calculates proportional nutrients based on ratio using reflection
// This creates a new Nutrients struct with values proportionally scaled by the given ratio
func CalculateProportionalNutrients(baseNutrients *Nutrients, amountG float64) *Nutrients {
	ratio := amountG / 100.0

	nutrients := &Nutrients{}

	baseValue := reflect.ValueOf(baseNutrients).Elem()
	baseType := baseValue.Type()
	targetValue := reflect.ValueOf(nutrients).Elem()

	// Iterate through all fields in the struct
	for i := 0; i < baseValue.NumField(); i++ {
		sourceField := baseValue.Field(i)

		if sourceField.IsNil() {
			continue
		}

		targetField := targetValue.Field(i)

		// Skip if field is not settable
		if !targetField.CanSet() {
			continue
		}

		fieldType := baseType.Field(i).Type

		switch fieldType.Elem().Kind() {
		case reflect.Float64:
			// Handle *float64 fields
			sourceVal := sourceField.Elem().Float()
			calculatedVal := RoundTo3Decimals(sourceVal * ratio)
			newVal := reflect.New(reflect.TypeOf(float64(0)))
			newVal.Elem().SetFloat(calculatedVal)
			targetField.Set(newVal)
		case reflect.Int:
			// Handle *int fields (like GlycemicIndex)
			sourceVal := float64(sourceField.Elem().Int())
			calculatedVal := int(math.Round(sourceVal * ratio))
			newVal := reflect.New(reflect.TypeOf(int(0)))
			newVal.Elem().SetInt(int64(calculatedVal))
			targetField.Set(newVal)
		default:
		}
	}

	return nutrients
}

// AddProportionalNutrients adds proportional nutrients to the total nutrients
// This modifies the total nutrients in place by adding scaled component nutrients
func AddProportionalNutrients(total *Nutrients, component *Nutrients, amountG float64) {
	// Calculate proportional component nutrients
	proportionalComponent := CalculateProportionalNutrients(component, amountG)

	// Add each field from proportional component to total
	totalValue := reflect.ValueOf(total).Elem()
	componentValue := reflect.ValueOf(proportionalComponent).Elem()
	totalType := totalValue.Type()

	for i := 0; i < totalValue.NumField(); i++ {
		totalField := totalValue.Field(i)
		componentField := componentValue.Field(i)
		fieldType := totalType.Field(i).Type

		if componentField.IsNil() {
			continue
		}

		if !totalField.CanSet() {
			continue
		}

		switch fieldType.Elem().Kind() {
		case reflect.Float64:
			// Handle *float64 fields
			componentVal := componentField.Elem().Float()

			if totalField.IsNil() {
				// Initialize total field with component value
				newVal := reflect.New(reflect.TypeOf(float64(0)))
				newVal.Elem().SetFloat(componentVal)
				totalField.Set(newVal)
			} else {
				// Add to existing total value
				totalVal := totalField.Elem().Float()
				newVal := RoundTo3Decimals(totalVal + componentVal)
				totalField.Elem().SetFloat(newVal)
			}

		case reflect.Int:
			// Handle *int fields (like GlycemicIndex)
			componentVal := int64(componentField.Elem().Int())

			if totalField.IsNil() {
				// Initialize total field with component value
				newVal := reflect.New(reflect.TypeOf(int(0)))
				newVal.Elem().SetInt(componentVal)
				totalField.Set(newVal)
			} else {
				// Add to existing total value
				totalVal := totalField.Elem().Int()
				newVal := totalVal + componentVal
				totalField.Elem().SetInt(newVal)
			}

		default:
		}
	}
}
