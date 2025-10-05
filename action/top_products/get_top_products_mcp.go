package top_products

import (
	"context"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"personal/gateways"
)

var GetTopProductsMCPDefinition = mcp.Tool{
	Name: "get_top_products",
	Annotations: &mcp.ToolAnnotations{
		ReadOnlyHint: true,
		Title:        "Get top 30 most frequently logged products",
	},
	Description: `Get the top 30 most frequently logged products in the last 3 months.

This tool provides statistics about which food products have been logged most frequently by the user.

The tool returns:
- Food ID
- Food name
- Serving name (if available)
- Number of times the product was logged (log count)

The tool automatically:
- Uses a 3-month time window (from now - 3 months to now)
- Groups products by food_id
- Sorts by log count (descending) and food_id (ascending) for tie-breaking
- Returns up to 30 products
- Excludes direct nutrient entries (food_id = NULL)

Use this tool when:
- You want to see which products the user logs most often
- You need to understand user's food preferences
- You want to suggest commonly logged items for quick logging

This is a read-only operation and does not modify any data.`,
}

// GetTopProducts is the MCP handler for getting top frequently logged products
func GetTopProducts(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, GetTopProductsOutput, error) {
	// Get database from context
	db := gateways.DBFromContext(ctx)
	if db == nil {
		return nil, GetTopProductsOutput{}, fmt.Errorf("database not available in context")
	}

	// Get user ID from context
	userID := gateways.UserIDFromContext(ctx)
	if userID == 0 {
		return nil, GetTopProductsOutput{}, fmt.Errorf("user_id not available in context")
	}

	// Get current time
	now := time.Now()

	// Calculate time window: last 3 months
	threeMonthsAgo := now.AddDate(0, -3, 0)

	// Call repository to get top 30 products
	topProducts, err := db.GetTopProducts(ctx, userID, threeMonthsAgo, now, 30)
	if err != nil {
		return nil, GetTopProductsOutput{}, fmt.Errorf("failed to get top products: %v", err)
	}

	// Return results
	output := GetTopProductsOutput{
		Products: topProducts,
	}

	return nil, output, nil
}
