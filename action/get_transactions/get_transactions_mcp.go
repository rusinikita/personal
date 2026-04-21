package get_transactions

import (
	"context"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"personal/action/add_transactions"
	"personal/domain"
	"personal/gateways"
)

var MCPDefinition = mcp.Tool{
	Name:        "get_transactions",
	Description: "List transactions with optional filters: date range, account, category (prefix match), type, merchant. Default limit 50, max 200.",
}

// GetTransactionsInput is the MCP tool input.
type GetTransactionsInput struct {
	From     *time.Time `json:"from,omitempty"`
	To       *time.Time `json:"to,omitempty"`
	Account  *string    `json:"account,omitempty"`
	Category *string    `json:"category,omitempty"`
	Type     *string    `json:"type,omitempty"`
	Merchant *string    `json:"merchant,omitempty"`
	Limit    int        `json:"limit,omitempty"`
	Offset   int        `json:"offset,omitempty"`
}

// GetTransactionsOutput is the MCP tool output.
type GetTransactionsOutput struct {
	Transactions []add_transactions.TransactionOutput `json:"transactions"`
	Total        int                                  `json:"total"`
}

func GetTransactions(ctx context.Context, _ *mcp.CallToolRequest, input GetTransactionsInput) (*mcp.CallToolResult, GetTransactionsOutput, error) {
	db := gateways.DBFromContext(ctx)
	if db == nil {
		return nil, GetTransactionsOutput{}, fmt.Errorf("database not available in context")
	}
	userID := gateways.UserIDFromContext(ctx)
	if userID == 0 {
		return nil, GetTransactionsOutput{}, fmt.Errorf("user_id not available in context")
	}

	filter := domain.TransactionFilter{
		UserID:  userID,
		From:    input.From,
		To:      input.To,
		Account: input.Account,
		Limit:   input.Limit,
		Offset:  input.Offset,
	}
	if input.Category != nil {
		filter.Category = input.Category
	}
	if input.Type != nil {
		t := domain.TransactionType(*input.Type)
		filter.Type = &t
	}
	if input.Merchant != nil {
		filter.Merchant = input.Merchant
	}

	txs, total, err := db.GetTransactions(ctx, filter)
	if err != nil {
		return nil, GetTransactionsOutput{}, fmt.Errorf("database error: %w", err)
	}

	out := make([]add_transactions.TransactionOutput, len(txs))
	for i, tx := range txs {
		out[i] = add_transactions.TransactionOutput{
			ID:                  tx.ID,
			Type:                string(tx.Type),
			AmountOriginal:      tx.AmountOriginal,
			Currency:            tx.Currency,
			AmountEUR:           tx.AmountEUR,
			Account:             tx.Account,
			Category:            tx.Category,
			Merchant:            tx.Merchant,
			Note:                tx.Note,
			OriginalDescription: tx.OriginalDescription,
			TransactedAt:        tx.TransactedAt,
		}
	}

	return nil, GetTransactionsOutput{Transactions: out, Total: total}, nil
}
