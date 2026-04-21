package add_transactions

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
	Name:        "add_transactions",
	Description: "Add one or multiple financial transactions (expense, income, transfer). Returns inserted count and saved records with IDs.",
	Annotations: &mcp.ToolAnnotations{
		DestructiveHint: util.Ptr(true),
		Title:           "Add transactions",
	},
}

// TransactionInput is one item in the add_transactions request.
type TransactionInput struct {
	Type                string    `json:"type"`
	AmountOriginal      float64   `json:"amount_original"`
	Currency            string    `json:"currency"`
	AmountEUR           float64   `json:"amount_eur"`
	Account             string    `json:"account"`
	Category            string    `json:"category,omitempty"`
	Merchant            string    `json:"merchant,omitempty"`
	Note                *string   `json:"note,omitempty"`
	OriginalDescription *string   `json:"original_description,omitempty"`
	TransactedAt        time.Time `json:"transacted_at"`
}

// TransactionOutput mirrors domain.Transaction for JSON serialization.
type TransactionOutput struct {
	ID                  int64     `json:"id"`
	Type                string    `json:"type"`
	AmountOriginal      float64   `json:"amount_original"`
	Currency            string    `json:"currency"`
	AmountEUR           float64   `json:"amount_eur"`
	Account             string    `json:"account"`
	Category            string    `json:"category"`
	Merchant            string    `json:"merchant"`
	Note                *string   `json:"note,omitempty"`
	OriginalDescription *string   `json:"original_description,omitempty"`
	TransactedAt        time.Time `json:"transacted_at"`
}

// AddTransactionsInput is the MCP tool input.
type AddTransactionsInput struct {
	Transactions []TransactionInput `json:"transactions"`
}

// AddTransactionsOutput is the MCP tool output.
type AddTransactionsOutput struct {
	InsertedCount int                 `json:"inserted_count"`
	Transactions  []TransactionOutput `json:"transactions"`
	Error         string              `json:"error,omitempty"`
}

func AddTransactions(ctx context.Context, _ *mcp.CallToolRequest, input AddTransactionsInput) (*mcp.CallToolResult, AddTransactionsOutput, error) {
	db := gateways.DBFromContext(ctx)
	if db == nil {
		return nil, AddTransactionsOutput{}, fmt.Errorf("database not available in context")
	}
	userID := gateways.UserIDFromContext(ctx)
	if userID == 0 {
		return nil, AddTransactionsOutput{}, fmt.Errorf("user_id not available in context")
	}

	if len(input.Transactions) == 0 {
		return nil, AddTransactionsOutput{Error: "transactions list is empty"}, nil
	}

	// Validate all items before inserting.
	for i, t := range input.Transactions {
		if err := validateTransactionInput(t); err != nil {
			return nil, AddTransactionsOutput{Error: fmt.Sprintf("transaction[%d]: %s", i, err.Error())}, nil
		}
	}

	domainTxs := make([]*domain.Transaction, len(input.Transactions))
	for i, t := range input.Transactions {
		domainTxs[i] = &domain.Transaction{
			UserID:              userID,
			Type:                domain.TransactionType(t.Type),
			AmountOriginal:      t.AmountOriginal,
			Currency:            t.Currency,
			AmountEUR:           t.AmountEUR,
			Account:             t.Account,
			Category:            t.Category,
			Merchant:            t.Merchant,
			Note:                t.Note,
			OriginalDescription: t.OriginalDescription,
			TransactedAt:        t.TransactedAt,
		}
	}

	saved, err := db.AddTransactions(ctx, domainTxs)
	if err != nil {
		return nil, AddTransactionsOutput{}, fmt.Errorf("database error: %w", err)
	}

	out := make([]TransactionOutput, len(saved))
	for i, tx := range saved {
		out[i] = toOutput(tx)
	}

	return nil, AddTransactionsOutput{
		InsertedCount: len(saved),
		Transactions:  out,
	}, nil
}

func validateTransactionInput(t TransactionInput) error {
	switch domain.TransactionType(t.Type) {
	case domain.TransactionTypeExpense, domain.TransactionTypeIncome, domain.TransactionTypeTransfer:
	default:
		return fmt.Errorf("type must be one of: expense, income, transfer")
	}
	if t.AmountOriginal <= 0 {
		return fmt.Errorf("amount_original must be greater than 0")
	}
	if t.AmountEUR <= 0 {
		return fmt.Errorf("amount_eur must be greater than 0")
	}
	if len(t.Currency) != 3 {
		return fmt.Errorf("currency must be exactly 3 characters (ISO 4217)")
	}
	if t.Account == "" {
		return fmt.Errorf("account is required")
	}
	return nil
}

func toOutput(tx *domain.Transaction) TransactionOutput {
	return TransactionOutput{
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
