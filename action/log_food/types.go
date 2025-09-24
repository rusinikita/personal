package log_food

import (
	"time"
)

// Tool 1: log_food_by_id
type LogFoodByIdInput struct {
	FoodID       int64     `json:"food_id" jsonschema:"required"`
	AmountG      float64   `json:"amount_g"`      // 0 means use serving_count
	ServingCount float64   `json:"serving_count"` // 0 means use amount_g
	MealType     string    `json:"meal_type"`     // empty string for no meal type
	ConsumedAt   time.Time `json:"consumed_at"`   // zero time means current time
	Note         string    `json:"note"`          // empty string for no note
}

// Tool 2: log_food_by_name
type LogFoodByNameInput struct {
	Name         string    `json:"name" jsonschema:"required"`
	AmountG      float64   `json:"amount_g"`      // 0 means use serving_count
	ServingCount float64   `json:"serving_count"` // 0 means use amount_g
	MealType     string    `json:"meal_type"`     // empty string for no meal type
	ConsumedAt   time.Time `json:"consumed_at"`   // zero time means current time
	Note         string    `json:"note"`          // empty string for no note
}

type LogFoodByNameOutput struct {
	Error       string      `json:"error,omitempty"`       // if error occurred
	Suggestions []FoodMatch `json:"suggestions,omitempty"` // if multiple matches found
	Message     string      `json:"message,omitempty"`     // success message
}

// Tool 3: log_food_by_barcode
type LogFoodByBarcodeInput struct {
	Barcode      string    `json:"barcode" jsonschema:"required"`
	AmountG      float64   `json:"amount_g"`      // 0 means use serving_count
	ServingCount float64   `json:"serving_count"` // 0 means use amount_g
	MealType     string    `json:"meal_type"`     // empty string for no meal type
	ConsumedAt   time.Time `json:"consumed_at"`   // zero time means current time
	Note         string    `json:"note"`          // empty string for no note
}

// Tool 4: log_custom_food
type LogCustomFoodInput struct {
	ProductName    string    `json:"product_name" jsonschema:"required"`
	AmountG        float64   `json:"amount_g" jsonschema:"required"`
	Calories       float64   `json:"calories" jsonschema:"required"`
	ProteinG       float64   `json:"protein_g" jsonschema:"required"`
	TotalFatG      float64   `json:"total_fat_g" jsonschema:"required"`
	CarbohydratesG float64   `json:"carbohydrates_g" jsonschema:"required"`
	CaffeineMg     float64   `json:"caffeine_mg"`     // 0 for no caffeine
	EthylAlcoholG  float64   `json:"ethyl_alcohol_g"` // 0 for no alcohol
	MealType       string    `json:"meal_type"`       // empty string for no meal type
	ConsumedAt     time.Time `json:"consumed_at"`     // zero time means current time
	Note           string    `json:"note"`            // empty string for no note
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

// Constants
const DEFAULT_USER_ID = int64(1)
