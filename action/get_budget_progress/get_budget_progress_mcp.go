package get_budget_progress

import (
	"context"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"personal/gateways"
)

var MCPDefinition = mcp.Tool{
	Name:        "get_budget_progress",
	Description: "Active budgets with spent and remaining EUR amounts as of a given date. Only expense transactions matching the budget category prefix are counted.",
}

// GetBudgetProgressInput is the MCP tool input.
type GetBudgetProgressInput struct {
	At time.Time `json:"at"`
}

// BudgetProgressRow is one budget with its spending progress.
type BudgetProgressRow struct {
	ID           int64     `json:"id"`
	Name         string    `json:"name"`
	Category     string    `json:"category"`
	AmountEUR    float64   `json:"amount_eur"`
	SpentEUR     float64   `json:"spent_eur"`
	RemainingEUR float64   `json:"remaining_eur"`
	StartsAt     time.Time `json:"starts_at"`
	EndsAt       time.Time `json:"ends_at"`
}

// GetBudgetProgressOutput is the MCP tool output.
type GetBudgetProgressOutput struct {
	Budgets []BudgetProgressRow `json:"budgets"`
}

func GetBudgetProgress(ctx context.Context, _ *mcp.CallToolRequest, input GetBudgetProgressInput) (*mcp.CallToolResult, GetBudgetProgressOutput, error) {
	db := gateways.DBFromContext(ctx)
	if db == nil {
		return nil, GetBudgetProgressOutput{}, fmt.Errorf("database not available in context")
	}
	userID := gateways.UserIDFromContext(ctx)
	if userID == 0 {
		return nil, GetBudgetProgressOutput{}, fmt.Errorf("user_id not available in context")
	}

	at := input.At
	if at.IsZero() {
		at = time.Now().UTC()
	}

	rows, err := db.GetBudgetProgress(ctx, userID, at)
	if err != nil {
		return nil, GetBudgetProgressOutput{}, fmt.Errorf("database error: %w", err)
	}

	budgets := make([]BudgetProgressRow, len(rows))
	for i, r := range rows {
		budgets[i] = BudgetProgressRow{
			ID:           r.ID,
			Name:         r.Name,
			Category:     r.Category,
			AmountEUR:    r.AmountEUR,
			SpentEUR:     r.SpentEUR,
			RemainingEUR: r.RemainingEUR,
			StartsAt:     r.StartsAt,
			EndsAt:       r.EndsAt,
		}
	}

	return nil, GetBudgetProgressOutput{Budgets: budgets}, nil
}
