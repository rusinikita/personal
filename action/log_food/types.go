package log_food

import (
	"time"

	"personal/domain"
)

// Input types for MCP tool
type LogFoodInput struct {
	ConsumedItems []ConsumedFoodItem `json:"consumed_items" jsonschema:"required,description=List of consumed food items"`
	// UserID берется из константы в коде (не передается в запросе)
}

type ConsumedFoodItem struct {
	// Сценарий 1: добавление с известным ID
	FoodID *int64 `json:"food_id,omitempty"` // ID продукта в БД

	// Сценарий 2: добавление с поиском по названию
	Name *string `json:"name,omitempty"` // Название для поиска

	// Сценарий 3: добавление с поиском по баркоду
	Barcode *string `json:"barcode,omitempty"` // Баркод для поиска

	// Сценарий 4: добавление без привязки к БД (прямое указание нутриентов)
	DirectNutrients *DirectNutrients `json:"direct_nutrients,omitempty"` // Макронутриенты напрямую

	// Общие поля для всех сценариев
	AmountG      float64    `json:"amount_g"`                // Количество в граммах
	ServingCount *float64   `json:"serving_count,omitempty"` // Количество порций (альтернатива граммам)
	MealType     *string    `json:"meal_type,omitempty"`     // Тип приема пищи
	ConsumedAt   *time.Time `json:"consumed_at,omitempty"`   // Время потребления
	Note         *string    `json:"note,omitempty"`          // Заметка
}

type DirectNutrients struct {
	// Макронутриенты (обязательные для сценария 4)
	Calories       float64 `json:"calories"`        // ккал на указанное количество
	ProteinG       float64 `json:"protein_g"`       // белки в граммах
	TotalFatG      float64 `json:"total_fat_g"`     // жиры в граммах
	CarbohydratesG float64 `json:"carbohydrates_g"` // углеводы в граммах

	// Специальные вещества (опциональные)
	CaffeineMg    *float64 `json:"caffeine_mg,omitempty"`     // кофеин в мг
	EthylAlcoholG *float64 `json:"ethyl_alcohol_g,omitempty"` // алкоголь в граммах

	// Название продукта (записывается в food_name)
	ProductName string `json:"product_name"` // Название неизвестного продукта
}

// Output types for MCP tool
type LogFoodOutput struct {
	AddedItems    []AddedConsumptionItem `json:"added_items" jsonschema:"description=Successfully logged consumption items"`
	NotFoundItems []NotFoundFoodItem     `json:"not_found_items" jsonschema:"description=Food items that could not be found"`
	Message       string                 `json:"message" jsonschema:"description=Summary message"`
}

type AddedConsumptionItem struct {
	UserID     int64             `json:"user_id"`
	ConsumedAt time.Time         `json:"consumed_at"`
	FoodID     *int64            `json:"food_id,omitempty"` // Nullable для сценария 4
	FoodName   string            `json:"food_name"`         // Название продукта
	Food       *domain.Food      `json:"food,omitempty"`    // Найденный продукт (только для сценариев 1-3)
	AmountG    float64           `json:"amount_g"`
	MealType   *string           `json:"meal_type,omitempty"`
	Note       *string           `json:"note,omitempty"`
	Nutrients  *domain.Nutrients `json:"nutrients"` // Пересчитанные нутриенты
}

type NotFoundFoodItem struct {
	FoodID      *int64      `json:"food_id,omitempty"`
	Name        *string     `json:"name,omitempty"`
	Barcode     *string     `json:"barcode,omitempty"`
	AmountG     float64     `json:"amount_g"`
	Reason      string      `json:"reason"`                // Причина: "id_not_found", "name_not_found", "barcode_not_found", "multiple_matches"
	Suggestions []FoodMatch `json:"suggestions,omitempty"` // Список предложений для "multiple_matches"
}

type FoodMatch struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// Constants
const DEFAULT_USER_ID = int64(1)
