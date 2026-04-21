package get_top_merchants

import (
	"context"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"personal/gateways"
)

var MCPDefinition = mcp.Tool{
	Name:        "get_top_merchants",
	Description: "Top merchants ranked by total EUR spend for a given period. Only expense transactions are counted. Default limit 10.",
}

// GetTopMerchantsInput is the MCP tool input.
type GetTopMerchantsInput struct {
	From  time.Time `json:"from"`
	To    time.Time `json:"to"`
	Limit int       `json:"limit,omitempty"`
}

// MerchantRow is one aggregated merchant row.
type MerchantRow struct {
	Merchant string  `json:"merchant"`
	TotalEUR float64 `json:"total_eur"`
	Count    int     `json:"count"`
}

// GetTopMerchantsOutput is the MCP tool output.
type GetTopMerchantsOutput struct {
	Merchants []MerchantRow `json:"merchants"`
}

func GetTopMerchants(ctx context.Context, _ *mcp.CallToolRequest, input GetTopMerchantsInput) (*mcp.CallToolResult, GetTopMerchantsOutput, error) {
	db := gateways.DBFromContext(ctx)
	if db == nil {
		return nil, GetTopMerchantsOutput{}, fmt.Errorf("database not available in context")
	}
	userID := gateways.UserIDFromContext(ctx)
	if userID == 0 {
		return nil, GetTopMerchantsOutput{}, fmt.Errorf("user_id not available in context")
	}

	limit := input.Limit
	if limit <= 0 {
		limit = 10
	}

	rows, err := db.GetTopMerchants(ctx, userID, input.From, input.To, limit)
	if err != nil {
		return nil, GetTopMerchantsOutput{}, fmt.Errorf("database error: %w", err)
	}

	merchants := make([]MerchantRow, len(rows))
	for i, r := range rows {
		merchants[i] = MerchantRow{Merchant: r.Merchant, TotalEUR: r.TotalEUR, Count: r.Count}
	}

	return nil, GetTopMerchantsOutput{Merchants: merchants}, nil
}
