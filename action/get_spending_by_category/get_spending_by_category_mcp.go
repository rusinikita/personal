package get_spending_by_category

import (
	"context"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"personal/gateways"
)

var MCPDefinition = mcp.Tool{
	Name:        "get_spending_by_category",
	Description: "Aggregated expense spending per category for a date range. depth=1 groups by top-level (e.g. 'food'), depth=2 by subcategory (e.g. 'food/cafe'). Income and transfers excluded.",
}

// GetSpendingByCategoryInput is the MCP tool input.
type GetSpendingByCategoryInput struct {
	From  time.Time `json:"from"`
	To    time.Time `json:"to"`
	Depth int       `json:"depth,omitempty"`
}

// CategoryRow is one aggregated row in the output.
type CategoryRow struct {
	Category string  `json:"category"`
	TotalEUR float64 `json:"total_eur"`
	Count    int     `json:"count"`
}

// GetSpendingByCategoryOutput is the MCP tool output.
type GetSpendingByCategoryOutput struct {
	From       time.Time     `json:"from"`
	To         time.Time     `json:"to"`
	Categories []CategoryRow `json:"categories"`
	TotalEUR   float64       `json:"total_eur"`
}

func GetSpendingByCategory(ctx context.Context, _ *mcp.CallToolRequest, input GetSpendingByCategoryInput) (*mcp.CallToolResult, GetSpendingByCategoryOutput, error) {
	db := gateways.DBFromContext(ctx)
	if db == nil {
		return nil, GetSpendingByCategoryOutput{}, fmt.Errorf("database not available in context")
	}
	userID := gateways.UserIDFromContext(ctx)
	if userID == 0 {
		return nil, GetSpendingByCategoryOutput{}, fmt.Errorf("user_id not available in context")
	}

	depth := input.Depth
	if depth < 1 {
		depth = 1
	}

	rows, err := db.GetSpendingByCategory(ctx, userID, input.From, input.To, depth)
	if err != nil {
		return nil, GetSpendingByCategoryOutput{}, fmt.Errorf("database error: %w", err)
	}

	cats := make([]CategoryRow, len(rows))
	var total float64
	for i, r := range rows {
		cats[i] = CategoryRow{Category: r.Category, TotalEUR: r.TotalEUR, Count: r.Count}
		total += r.TotalEUR
	}

	return nil, GetSpendingByCategoryOutput{
		From:       input.From,
		To:         input.To,
		Categories: cats,
		TotalEUR:   total,
	}, nil
}
