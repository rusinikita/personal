package log_food

import (
	"context"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"personal/domain"
	"personal/gateways"
)

var MCPDefinition = mcp.Tool{
	Name:        "log_food",
	Description: "Log consumed food items to consumption log with support for 4 scenarios: ID lookup, name search, barcode search, and direct nutrients input",
}

// LogFood is the main MCP handler for logging food consumption
func LogFood(ctx context.Context, _ *mcp.CallToolRequest, input LogFoodInput) (*mcp.CallToolResult, LogFoodOutput, error) {
	// Get database from context
	db := gateways.DBFromContext(ctx)
	if db == nil {
		return nil, LogFoodOutput{}, fmt.Errorf("database not available in context")
	}
	// 1. Validate input
	if err := validateInput(input); err != nil {
		return nil, LogFoodOutput{}, fmt.Errorf("validation error: %w", err)
	}

	var addedItems []AddedConsumptionItem
	var notFoundItems []NotFoundFoodItem
	var errors []error

	// 2. Process each item individually
	for i, item := range input.ConsumedItems {
		result := processItem(ctx, item, db)

		if result.Error != nil {
			errors = append(errors, fmt.Errorf("item %d: %w", i, result.Error))
			continue
		}

		if result.NotFound != nil {
			notFoundItems = append(notFoundItems, *result.NotFound)
		} else if result.Added != nil {
			addedItems = append(addedItems, *result.Added)
		}
	}

	if len(errors) > 0 {
		return nil, LogFoodOutput{}, fmt.Errorf("processing errors: %v", errors)
	}

	// 3. Generate summary and return
	message := generateSummaryMessage(len(addedItems), len(notFoundItems))
	output := LogFoodOutput{
		AddedItems:    addedItems,
		NotFoundItems: notFoundItems,
		Message:       message,
	}

	return nil, output, nil
}

// ItemResult represents the result of processing a single item
type ItemResult struct {
	Added    *AddedConsumptionItem
	NotFound *NotFoundFoodItem
	Error    error
}

// processItem handles a single consumed food item through the entire pipeline
func processItem(ctx context.Context, item ConsumedFoodItem, db gateways.DB) ItemResult {
	// Handle direct nutrients scenario (no search needed)
	if item.DirectNutrients != nil {
		return processDirectNutrients(ctx, item, db)
	}

	// Create search filter based on item type
	filter := createSearchFilter(item)
	if filter == nil {
		return ItemResult{Error: fmt.Errorf("invalid item: no valid search criteria")}
	}

	// Search for food
	foods, err := db.SearchFood(ctx, *filter)
	if err != nil {
		return ItemResult{Error: fmt.Errorf("search failed: %w", err)}
	}

	// Interpret search results
	food, notFound := interpretSearchResult(item, foods)
	if notFound != nil {
		return ItemResult{NotFound: notFound}
	}

	// Create consumption log and save
	consumedAt := getConsumedAt(item)
	consumptionLog, addedItem, err := createConsumptionLogFromFood(DEFAULT_USER_ID, consumedAt, food, item)
	if err != nil {
		return ItemResult{Error: fmt.Errorf("failed to create consumption log: %w", err)}
	}

	if err := db.AddConsumptionLog(ctx, consumptionLog); err != nil {
		return ItemResult{Error: fmt.Errorf("database save failed: %w", err)}
	}

	return ItemResult{Added: &addedItem}
}

// createSearchFilter converts consumed item to search filter based on scenario
func createSearchFilter(item ConsumedFoodItem) *domain.FoodFilter {
	switch {
	case item.FoodID != nil:
		return &domain.FoodFilter{IDs: []int64{*item.FoodID}}
	case item.Name != nil && *item.Name != "":
		return &domain.FoodFilter{Name: item.Name}
	case item.Barcode != nil && *item.Barcode != "":
		return &domain.FoodFilter{Barcode: item.Barcode}
	default:
		return nil
	}
}

// interpretSearchResult interprets search results and returns found food or not-found item
func interpretSearchResult(item ConsumedFoodItem, foods []*domain.Food) (*domain.Food, *NotFoundFoodItem) {
	switch len(foods) {
	case 0:
		// No matches found
		notFound := NotFoundFoodItem{
			AmountG: item.AmountG,
		}

		// Set the appropriate field and reason based on search type
		if item.FoodID != nil {
			notFound.FoodID = item.FoodID
			notFound.Reason = "id_not_found"
		} else if item.Name != nil {
			notFound.Name = item.Name
			notFound.Reason = "name_not_found"
		} else if item.Barcode != nil {
			notFound.Barcode = item.Barcode
			notFound.Reason = "barcode_not_found"
		}

		return nil, &notFound

	case 1:
		// Exact match found
		return foods[0], nil

	default:
		// Multiple matches - only applicable for name search
		suggestions := make([]FoodMatch, 0, 2)
		for i, food := range foods {
			if i >= 2 {
				break
			}
			suggestions = append(suggestions, FoodMatch{
				ID:   food.ID,
				Name: food.Name,
			})
		}

		notFound := NotFoundFoodItem{
			Name:        item.Name,
			AmountG:     item.AmountG,
			Reason:      "multiple_matches",
			Suggestions: suggestions,
		}

		return nil, &notFound
	}
}

// processDirectNutrients handles the direct nutrients scenario
func processDirectNutrients(ctx context.Context, item ConsumedFoodItem, db gateways.DB) ItemResult {
	consumedAt := getConsumedAt(item)
	consumptionLog, addedItem := createConsumptionLogFromDirectNutrients(DEFAULT_USER_ID, consumedAt, item)

	if err := db.AddConsumptionLog(ctx, consumptionLog); err != nil {
		return ItemResult{Error: fmt.Errorf("database save failed: %w", err)}
	}

	return ItemResult{Added: &addedItem}
}

// getConsumedAt returns the consumed time or current time as default
func getConsumedAt(item ConsumedFoodItem) time.Time {
	if item.ConsumedAt != nil {
		return *item.ConsumedAt
	}
	return time.Now().UTC()
}

// createConsumptionLogFromFood creates consumption log from found food
func createConsumptionLogFromFood(userID int64, consumedAt time.Time, food *domain.Food, item ConsumedFoodItem) (*domain.ConsumptionLog, AddedConsumptionItem, error) {
	// Calculate nutrients and actual amount
	nutrients, actualAmountG, err := calculateNutrients(food, item)
	if err != nil {
		return nil, AddedConsumptionItem{}, err
	}

	// Create consumption log
	log := &domain.ConsumptionLog{
		UserID:     userID,
		ConsumedAt: consumedAt,
		FoodID:     &food.ID,
		FoodName:   food.Name,
		AmountG:    actualAmountG,
		MealType:   item.MealType,
		Note:       item.Note,
		Nutrients:  nutrients,
	}

	// Create response item
	addedItem := AddedConsumptionItem{
		UserID:     userID,
		ConsumedAt: consumedAt,
		FoodID:     &food.ID,
		FoodName:   food.Name,
		Food:       food,
		AmountG:    actualAmountG,
		MealType:   item.MealType,
		Note:       item.Note,
		Nutrients:  nutrients,
	}

	return log, addedItem, nil
}

// createConsumptionLogFromDirectNutrients creates consumption log from direct nutrients
func createConsumptionLogFromDirectNutrients(userID int64, consumedAt time.Time, item ConsumedFoodItem) (*domain.ConsumptionLog, AddedConsumptionItem) {
	nutrients := createDirectNutrients(*item.DirectNutrients)
	actualAmountG := item.AmountG

	// Create consumption log
	log := &domain.ConsumptionLog{
		UserID:     userID,
		ConsumedAt: consumedAt,
		FoodID:     nil, // No food ID for direct nutrients
		FoodName:   item.DirectNutrients.ProductName,
		AmountG:    actualAmountG,
		MealType:   item.MealType,
		Note:       item.Note,
		Nutrients:  nutrients,
	}

	// Create response item
	addedItem := AddedConsumptionItem{
		UserID:     userID,
		ConsumedAt: consumedAt,
		FoodID:     nil,
		FoodName:   item.DirectNutrients.ProductName,
		Food:       nil, // No food for direct nutrients
		AmountG:    actualAmountG,
		MealType:   item.MealType,
		Note:       item.Note,
		Nutrients:  nutrients,
	}

	return log, addedItem
}

// generateSummaryMessage creates a summary message for the response
func generateSummaryMessage(addedCount, notFoundCount int) string {
	if notFoundCount == 0 {
		return fmt.Sprintf("Successfully logged %d food consumption item(s)", addedCount)
	}
	return fmt.Sprintf("Logged %d item(s), %d item(s) not found and require clarification", addedCount, notFoundCount)
}
