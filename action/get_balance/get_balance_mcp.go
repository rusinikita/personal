package get_balance

import (
	"context"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"personal/gateways"
)

var MCPDefinition = mcp.Tool{
	Name:        "get_balance",
	Description: "Income minus expenses for a period in EUR. Transfer transactions are excluded from the calculation.",
}

// GetBalanceInput is the MCP tool input.
type GetBalanceInput struct {
	From time.Time `json:"from"`
	To   time.Time `json:"to"`
}

// GetBalanceOutput is the MCP tool output.
type GetBalanceOutput struct {
	From       time.Time `json:"from"`
	To         time.Time `json:"to"`
	IncomeEUR  float64   `json:"income_eur"`
	ExpenseEUR float64   `json:"expense_eur"`
	BalanceEUR float64   `json:"balance_eur"`
}

func GetBalance(ctx context.Context, _ *mcp.CallToolRequest, input GetBalanceInput) (*mcp.CallToolResult, GetBalanceOutput, error) {
	db := gateways.DBFromContext(ctx)
	if db == nil {
		return nil, GetBalanceOutput{}, fmt.Errorf("database not available in context")
	}
	userID := gateways.UserIDFromContext(ctx)
	if userID == 0 {
		return nil, GetBalanceOutput{}, fmt.Errorf("user_id not available in context")
	}

	result, err := db.GetBalance(ctx, userID, input.From, input.To)
	if err != nil {
		return nil, GetBalanceOutput{}, fmt.Errorf("database error: %w", err)
	}

	return nil, GetBalanceOutput{
		From:       result.From,
		To:         result.To,
		IncomeEUR:  result.IncomeEUR,
		ExpenseEUR: result.ExpenseEUR,
		BalanceEUR: result.BalanceEUR,
	}, nil
}
