package edit_transactions

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
	Name:        "edit_transactions",
	Description: "Batch edit transactions. All fields except id are optional. Useful for re-categorization or correcting details. All IDs must belong to the current user — partial updates are rejected.",
	Annotations: &mcp.ToolAnnotations{
		DestructiveHint: util.Ptr(true),
		Title:           "Edit transactions",
	},
}

// TransactionUpdate is one patch item.
type TransactionUpdate struct {
	ID                  int64      `json:"id"`
	Type                *string    `json:"type,omitempty"`
	AmountOriginal      *float64   `json:"amount_original,omitempty"`
	Currency            *string    `json:"currency,omitempty"`
	AmountEUR           *float64   `json:"amount_eur,omitempty"`
	Account             *string    `json:"account,omitempty"`
	Category            *string    `json:"category,omitempty"`
	Merchant            *string    `json:"merchant,omitempty"`
	Note                *string    `json:"note,omitempty"`
	OriginalDescription *string    `json:"original_description,omitempty"`
	TransactedAt        *time.Time `json:"transacted_at,omitempty"`
}

// EditTransactionsInput is the MCP tool input.
type EditTransactionsInput struct {
	Updates []TransactionUpdate `json:"updates"`
}

// EditTransactionsOutput is the MCP tool output.
type EditTransactionsOutput struct {
	UpdatedCount int    `json:"updated_count"`
	Error        string `json:"error,omitempty"`
}

func EditTransactions(ctx context.Context, _ *mcp.CallToolRequest, input EditTransactionsInput) (*mcp.CallToolResult, EditTransactionsOutput, error) {
	db := gateways.DBFromContext(ctx)
	if db == nil {
		return nil, EditTransactionsOutput{}, fmt.Errorf("database not available in context")
	}
	userID := gateways.UserIDFromContext(ctx)
	if userID == 0 {
		return nil, EditTransactionsOutput{}, fmt.Errorf("user_id not available in context")
	}

	if len(input.Updates) == 0 {
		return nil, EditTransactionsOutput{Error: "updates list is empty"}, nil
	}

	domainUpdates := make([]domain.TransactionUpdate, len(input.Updates))
	for i, u := range input.Updates {
		if u.ID == 0 {
			return nil, EditTransactionsOutput{Error: fmt.Sprintf("update[%d]: id is required", i)}, nil
		}
		du := domain.TransactionUpdate{ID: u.ID}
		if u.Type != nil {
			t := domain.TransactionType(*u.Type)
			du.Type = &t
		}
		du.AmountOriginal = u.AmountOriginal
		du.Currency = u.Currency
		du.AmountEUR = u.AmountEUR
		du.Account = u.Account
		du.Category = u.Category
		du.Merchant = u.Merchant
		du.Note = u.Note
		du.OriginalDescription = u.OriginalDescription
		du.TransactedAt = u.TransactedAt
		domainUpdates[i] = du
	}

	count, err := db.EditTransactions(ctx, userID, domainUpdates)
	if err != nil {
		return nil, EditTransactionsOutput{Error: err.Error()}, nil
	}

	return nil, EditTransactionsOutput{UpdatedCount: count}, nil
}
