package set_budget

import (
	"context"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"personal/domain"
	"personal/gateways"
	"personal/util"
)

var MCPDefinition = mcp.Tool{
	Name:        "set_budget",
	Description: "Create or update a spending budget for a category over a time period. Upserts by name — calling again with the same name updates the existing budget.",
	Annotations: &mcp.ToolAnnotations{
		DestructiveHint: util.Ptr(true),
		Title:           "Set budget",
	},
}

// SetBudgetInput is the MCP tool input.
type SetBudgetInput struct {
	Name      string    `json:"name"`
	Category  string    `json:"category"`
	AmountEUR float64   `json:"amount_eur"`
	StartsAt  time.Time `json:"starts_at"`
	EndsAt    time.Time `json:"ends_at"`
}

// BudgetOutput mirrors domain.Budget for JSON serialization.
type BudgetOutput struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Category  string    `json:"category"`
	AmountEUR float64   `json:"amount_eur"`
	StartsAt  time.Time `json:"starts_at"`
	EndsAt    time.Time `json:"ends_at"`
}

// SetBudgetOutput is the MCP tool output.
type SetBudgetOutput struct {
	ID     int64        `json:"id"`
	Budget BudgetOutput `json:"budget"`
	Error  string       `json:"error,omitempty"`
}

func SetBudget(ctx context.Context, _ *mcp.CallToolRequest, input SetBudgetInput) (*mcp.CallToolResult, SetBudgetOutput, error) {
	db := gateways.DBFromContext(ctx)
	if db == nil {
		return nil, SetBudgetOutput{}, fmt.Errorf("database not available in context")
	}
	userID := gateways.UserIDFromContext(ctx)
	if userID == 0 {
		return nil, SetBudgetOutput{}, fmt.Errorf("user_id not available in context")
	}

	if input.Name == "" {
		return nil, SetBudgetOutput{Error: "name is required"}, nil
	}
	if input.AmountEUR <= 0 {
		return nil, SetBudgetOutput{Error: "amount_eur must be greater than 0"}, nil
	}
	if !input.EndsAt.After(input.StartsAt) {
		return nil, SetBudgetOutput{Error: "ends_at must be after starts_at"}, nil
	}

	b := &domain.Budget{
		UserID:    userID,
		Name:      input.Name,
		Category:  input.Category,
		AmountEUR: input.AmountEUR,
		StartsAt:  input.StartsAt,
		EndsAt:    input.EndsAt,
	}

	id, err := db.SetBudget(ctx, b)
	if err != nil {
		return nil, SetBudgetOutput{}, fmt.Errorf("database error: %w", err)
	}
	b.ID = id

	return nil, SetBudgetOutput{
		ID: id,
		Budget: BudgetOutput{
			ID:        id,
			Name:      b.Name,
			Category:  b.Category,
			AmountEUR: b.AmountEUR,
			StartsAt:  b.StartsAt,
			EndsAt:    b.EndsAt,
		},
	}, nil
}
