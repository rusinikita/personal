package log_food

import (
	"context"
	"fmt"

	"personal/domain"
	"personal/gateways"
)

// SearchResult represents the result of food search operation
type SearchResult struct {
	Found    []*domain.Food
	NotFound []NotFoundFoodItem
}

// searchFoodItems searches for food items using all 4 scenarios
func searchFoodItems(ctx context.Context, items []ConsumedFoodItem, db gateways.DB) (SearchResult, error) {
	result := SearchResult{
		Found:    []*domain.Food{},
		NotFound: []NotFoundFoodItem{},
	}

	// Group items by scenario for batch processing
	var idItems []idLookupItem
	var nameItems []nameLookupItem
	var barcodeItems []barcodeLookupItem

	for i, item := range items {
		switch {
		case item.FoodID != nil:
			idItems = append(idItems, idLookupItem{
				Index:  i,
				Item:   item,
				FoodID: *item.FoodID,
			})

		case item.Name != nil && *item.Name != "":
			nameItems = append(nameItems, nameLookupItem{
				Index: i,
				Item:  item,
				Name:  *item.Name,
			})

		case item.Barcode != nil && *item.Barcode != "":
			barcodeItems = append(barcodeItems, barcodeLookupItem{
				Index:   i,
				Item:    item,
				Barcode: *item.Barcode,
			})

		case item.DirectNutrients != nil:
			// No search needed for direct nutrients scenario
			continue

		default:
			result.NotFound = append(result.NotFound, NotFoundFoodItem{
				AmountG: item.AmountG,
				Reason:  "invalid_scenario",
			})
		}
	}

	// Scenario 1: Batch lookup by IDs
	if len(idItems) > 0 {
		if err := searchByIDs(ctx, idItems, db, &result); err != nil {
			return result, fmt.Errorf("ID search failed: %w", err)
		}
	}

	// Scenario 2: Search by names
	if len(nameItems) > 0 {
		if err := searchByNames(ctx, nameItems, db, &result); err != nil {
			return result, fmt.Errorf("name search failed: %w", err)
		}
	}

	// Scenario 3: Search by barcodes
	if len(barcodeItems) > 0 {
		if err := searchByBarcodes(ctx, barcodeItems, db, &result); err != nil {
			return result, fmt.Errorf("barcode search failed: %w", err)
		}
	}

	return result, nil
}

// Helper types for batch processing
type idLookupItem struct {
	Index  int
	Item   ConsumedFoodItem
	FoodID int64
}

type nameLookupItem struct {
	Index int
	Item  ConsumedFoodItem
	Name  string
}

type barcodeLookupItem struct {
	Index   int
	Item    ConsumedFoodItem
	Barcode string
}

// searchByIDs performs batch lookup by food IDs
func searchByIDs(ctx context.Context, items []idLookupItem, db gateways.DB, result *SearchResult) error {
	// Extract IDs for batch lookup
	ids := make([]int64, len(items))
	for i, item := range items {
		ids[i] = item.FoodID
	}

	// Batch lookup using unified search method
	foods, err := db.SearchFood(ctx, domain.FoodFilter{IDs: ids})
	if err != nil {
		return err
	}

	// Convert to map for efficient lookup
	foundFoods := make(map[int64]*domain.Food)
	for _, food := range foods {
		foundFoods[food.ID] = food
	}

	// Process results
	for _, item := range items {
		if food, found := foundFoods[item.FoodID]; found {
			result.Found = append(result.Found, food)
		} else {
			result.NotFound = append(result.NotFound, NotFoundFoodItem{
				FoodID:  &item.FoodID,
				AmountG: item.Item.AmountG,
				Reason:  "id_not_found",
			})
		}
	}

	return nil
}

// searchByNames performs search by food names with multiple match handling
func searchByNames(ctx context.Context, items []nameLookupItem, db gateways.DB, result *SearchResult) error {
	for _, item := range items {
		foods, err := db.SearchFood(ctx, domain.FoodFilter{Name: &item.Name})
		if err != nil {
			return err
		}

		switch len(foods) {
		case 0:
			// No matches found
			result.NotFound = append(result.NotFound, NotFoundFoodItem{
				Name:    &item.Name,
				AmountG: item.Item.AmountG,
				Reason:  "name_not_found",
			})

		case 1:
			// Exact match found
			result.Found = append(result.Found, foods[0])

		default:
			// Multiple matches - return first 2 as suggestions
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

			result.NotFound = append(result.NotFound, NotFoundFoodItem{
				Name:        &item.Name,
				AmountG:     item.Item.AmountG,
				Reason:      "multiple_matches",
				Suggestions: suggestions,
			})
		}
	}

	return nil
}

// searchByBarcodes performs search by food barcodes
func searchByBarcodes(ctx context.Context, items []barcodeLookupItem, db gateways.DB, result *SearchResult) error {
	for _, item := range items {
		foods, err := db.SearchFood(ctx, domain.FoodFilter{Barcode: &item.Barcode})
		if err != nil {
			return err
		}

		if len(foods) == 0 {
			// No match found
			result.NotFound = append(result.NotFound, NotFoundFoodItem{
				Barcode: &item.Barcode,
				AmountG: item.Item.AmountG,
				Reason:  "barcode_not_found",
			})
			continue
		}

		// Barcode should be unique, so take first result
		result.Found = append(result.Found, foods[0])
	}

	return nil
}
