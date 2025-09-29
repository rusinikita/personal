package find_food

import (
	"context"
	"sort"

	"personal/domain"
	"personal/gateways"
)

// SearchFoodsByName searches for foods by a single name variant
// Extracted from log_food_by_name_mcp.go for reuse
func SearchFoodsByName(ctx context.Context, db gateways.DB, name string) ([]*domain.Food, error) {
	filter := domain.FoodFilter{Name: &name}
	return db.SearchFood(ctx, filter)
}

// ResolveFoodsByNameVariants searches for foods using multiple name variants
// and returns ranked results with match counts
func ResolveFoodsByNameVariants(ctx context.Context, db gateways.DB, nameVariants []string) ([]FoodMatch, error) {
	// Track match counts for each food
	foodMatches := make(map[int64]*FoodMatch)

	// Search for each name variant
	for _, variant := range nameVariants {
		if variant == "" {
			continue // Skip empty strings
		}

		foods, err := SearchFoodsByName(ctx, db, variant)
		if err != nil {
			return nil, err
		}

		// Update match counts
		for _, food := range foods {
			if existing, exists := foodMatches[food.ID]; exists {
				existing.MatchCount++
			} else {
				servingName := ""
				if food.ServingName != nil {
					servingName = *food.ServingName
				}
				foodMatches[food.ID] = &FoodMatch{
					ID:          food.ID,
					Name:        food.Name,
					ServingName: servingName,
					MatchCount:  1,
				}
			}
		}
	}

	// Convert map to slice for sorting
	results := make([]FoodMatch, 0, len(foodMatches))
	for _, match := range foodMatches {
		results = append(results, *match)
	}

	// Sort by match count (descending), then by ID (ascending) for stability
	sort.Slice(results, func(i, j int) bool {
		if results[i].MatchCount == results[j].MatchCount {
			return results[i].ID < results[j].ID
		}
		return results[i].MatchCount > results[j].MatchCount
	})

	return results, nil
}
