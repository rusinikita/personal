package find_food

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"personal/gateways"
)

var ResolveFoodIdByNameMCPDefinition = mcp.Tool{
	Name: "resolve_food_id_by_name",
	Annotations: &mcp.ToolAnnotations{
		ReadOnlyHint:   true,
		IdempotentHint: true,
		Title:          "Search foods by name variants",
	},
	Description: `Search for existing foods in the database using multiple name variations to find the best matches.

This tool allows you to provide 1-5 different name variants (e.g., "банан", "желтый банан", "banana") and returns all matching foods ranked by relevance. Each result includes the food ID, exact name, serving information, and match count.

Key features:
- Accepts 1-5 name variants for flexible searching
- Returns results ranked by match count (how many variants found each food)
- Includes food ID, exact database name, serving_name, and match_count for each result
- Performs fuzzy matching - finds foods containing any of the search terms
- Deduplicates results automatically while tracking match frequency
- Returns empty list (not error) when no foods are found

Perfect for:
- Finding existing foods before logging consumption
- Resolving ambiguous food names with multiple variants
- Getting food IDs for use in other tools like log_food_by_id
- Checking what foods exist in the database

This is a read-only search tool that doesn't modify any data.`,
}

// ResolveFoodIdByName is the MCP handler for finding foods by name variants
func ResolveFoodIdByName(ctx context.Context, _ *mcp.CallToolRequest, input ResolveFoodIdByNameInput) (*mcp.CallToolResult, ResolveFoodIdByNameOutput, error) {
	// Get database from context
	db := gateways.DBFromContext(ctx)
	if db == nil {
		return nil, ResolveFoodIdByNameOutput{}, fmt.Errorf("database not available in context")
	}

	// Validate input
	if len(input.NameVariants) == 0 {
		return nil, ResolveFoodIdByNameOutput{Error: "name_variants cannot be empty"}, nil
	}
	if len(input.NameVariants) > 5 {
		return nil, ResolveFoodIdByNameOutput{Error: "maximum 5 name variants allowed"}, nil
	}

	// Check for empty strings in variants
	for _, variant := range input.NameVariants {
		if variant == "" {
			return nil, ResolveFoodIdByNameOutput{Error: "name variants cannot be empty"}, nil
		}
	}

	// Search foods using name variants
	foods, err := ResolveFoodsByNameVariants(ctx, db, input.NameVariants)
	if err != nil {
		return nil, ResolveFoodIdByNameOutput{Error: fmt.Sprintf("search failed: %v", err)}, nil
	}

	return nil, ResolveFoodIdByNameOutput{Foods: foods}, nil
}
