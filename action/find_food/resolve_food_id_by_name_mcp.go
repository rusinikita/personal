package find_food

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"personal/gateways"
)

var ResolveFoodIdByNameMCPDefinition = mcp.Tool{
	Name:        "resolve_food_id_by_name",
	Description: "Search for foods by 1-5 name variants and return ranked results with match counts",
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
