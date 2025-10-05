package log_food

import (
	"time"
)

// Tool 1: log_food_by_id
type LogFoodByIdInput struct {
	FoodID       int64     `json:"food_id" jsonschema:"Food database ID to log"`
	AmountG      float64   `json:"amount_g,omitempty" jsonschema:"Amount in grams (use this OR serving_count, not both). Do not send if using serving_count instead"`
	ServingCount float64   `json:"serving_count,omitempty" jsonschema:"Number of servings (use this OR amount_g, not both). Do not send if using amount_g instead"`
	MealType     string    `json:"meal_type,omitempty" jsonschema:"Meal category (breakfast/lunch/dinner/snack). Do not send if no meal type specified"`
	ConsumedAt   time.Time `json:"consumed_at,omitempty" jsonschema:"When the food was consumed in RFC3339 format (e.g. 2024-01-15T14:30:00Z). Do not send if not specified by user, server time will be used"`
	Note         string    `json:"note,omitempty" jsonschema:"Optional note about this food log entry. Do not send if not specified by user"`
}

// Tool 3: log_food_by_barcode
type LogFoodByBarcodeInput struct {
	Barcode      string    `json:"barcode" jsonschema:"Product barcode to scan and log"`
	AmountG      float64   `json:"amount_g,omitempty" jsonschema:"Amount in grams (use this OR serving_count, not both). Do not send if using serving_count instead"`
	ServingCount float64   `json:"serving_count,omitempty" jsonschema:"Number of servings (use this OR amount_g, not both). Do not send if using amount_g instead"`
	MealType     string    `json:"meal_type,omitempty" jsonschema:"Meal category (breakfast/lunch/dinner/snack). Do not send if no meal type specified"`
	ConsumedAt   time.Time `json:"consumed_at,omitempty" jsonschema:"Optional time when the food was consumed in RFC3339 format (e.g. 2024-01-15T14:30:00Z). Do not send if not specified by user, server time will be used"`
	Note         string    `json:"note,omitempty" jsonschema:"Optional note about this food log entry. Do not send if not specified by user"`
}

// Tool 4: log_custom_food
type LogCustomFoodInput struct {
	ProductName    string    `json:"product_name" jsonschema:"Name of the custom food product"`
	AmountG        float64   `json:"amount_g" jsonschema:"Amount consumed in grams (must be positive)"`
	Calories       float64   `json:"calories" jsonschema:"Total calories per 100g (must be non-negative)"`
	ProteinG       float64   `json:"protein_g" jsonschema:"Protein content per 100g in grams (must be non-negative)"`
	TotalFatG      float64   `json:"total_fat_g" jsonschema:"Total fat content per 100g in grams (must be non-negative)"`
	CarbohydratesG float64   `json:"carbohydrates_g" jsonschema:"Total carbohydrates per 100g in grams (must be non-negative)"`
	CaffeineMg     float64   `json:"caffeine_mg,omitempty" jsonschema:"Caffeine content per 100g in milligrams. Do not send if no caffeine"`
	EthylAlcoholG  float64   `json:"ethyl_alcohol_g,omitempty" jsonschema:"Alcohol content per 100g in grams. Do not send if no alcohol"`
	MealType       string    `json:"meal_type,omitempty" jsonschema:"Meal category (breakfast/lunch/dinner/snack). Do not send if no meal type specified"`
	ConsumedAt     time.Time `json:"consumed_at,omitempty" jsonschema:"Optional time when the food was consumed in RFC3339 format (e.g. 2024-01-15T14:30:00Z). Do not send if not specified by user, server time will be used"`
	Note           string    `json:"note,omitempty" jsonschema:"Optional note about this food log entry. Do not send if not specified by user"`
}

// Shared response structures

type FoodMatch struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// Standard tool response format (for tools 1, 3, 4)
type ToolResponse struct {
	Error   string `json:"error,omitempty"`
	Message string `json:"message,omitempty"`
}
