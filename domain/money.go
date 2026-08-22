package domain

import "time"

// TransactionType represents the direction of a financial transaction.
type TransactionType string

const (
	TransactionTypeExpense  TransactionType = "expense"
	TransactionTypeIncome   TransactionType = "income"
	TransactionTypeTransfer TransactionType = "transfer"
)

// Transaction is a single financial record.
type Transaction struct {
	ID                  int64           `db:"id"`
	UserID              int64           `db:"user_id"`
	Type                TransactionType `db:"type"`
	AmountOriginal      float64         `db:"amount_original"`
	Currency            string          `db:"currency"`
	AmountEUR           float64         `db:"amount_eur"`
	Account             string          `db:"account"`
	Category            string          `db:"category"`
	Merchant            string          `db:"merchant"`
	Note                *string         `db:"note"`
	OriginalDescription *string         `db:"original_description"`
	IdempotencyKey      *string         `db:"idempotency_key"`
	TransactedAt        time.Time       `db:"transacted_at"`
	CreatedAt           time.Time       `db:"created_at"`
}

// Budget represents a spending limit for a category over a time period.
type Budget struct {
	ID        int64     `db:"id"`
	UserID    int64     `db:"user_id"`
	Name      string    `db:"name"`
	Category  string    `db:"category"`
	AmountEUR float64   `db:"amount_eur"`
	StartsAt  time.Time `db:"starts_at"`
	EndsAt    time.Time `db:"ends_at"`
	CreatedAt time.Time `db:"created_at"`
}

// TransactionFilter defines query parameters for listing transactions.
type TransactionFilter struct {
	UserID   int64
	From     *time.Time
	To       *time.Time
	Account  *string
	Category *string
	Type     *TransactionType
	Merchant *string
	Limit    int
	Offset   int
}

// TransactionUpdate is one item in a bulk edit_transactions call.
// All fields except ID are optional.
type TransactionUpdate struct {
	ID                  int64
	Type                *TransactionType
	AmountOriginal      *float64
	Currency            *string
	AmountEUR           *float64
	Account             *string
	Category            *string
	Merchant            *string
	Note                *string
	OriginalDescription *string
	TransactedAt        *time.Time
}

// SpendingByCategory is an aggregated spending row for one category prefix.
type SpendingByCategory struct {
	Category string
	TotalEUR float64
	Count    int
}

// MerchantSummary is an aggregated row for one merchant.
type MerchantSummary struct {
	Merchant string
	TotalEUR float64
	Count    int
}

// BudgetProgress is a budget enriched with spent and remaining amounts.
type BudgetProgress struct {
	Budget
	SpentEUR     float64
	RemainingEUR float64
}

// BalanceResult is income minus expenses for a period.
type BalanceResult struct {
	From       time.Time
	To         time.Time
	IncomeEUR  float64
	ExpenseEUR float64
	BalanceEUR float64
}

// MoneySummary is the user's transaction history bounds — the web
// dashboard's only source for the "all time" query range and the
// months-span used by every average-monthly figure it shows.
type MoneySummary struct {
	FirstTransactionAt *time.Time // MIN(transacted_at); nil when the user has no transactions yet
	LastSyncedAt       *time.Time // MAX(created_at) — import freshness, not transaction age
}

// DailySummary is one calendar day's activity — powers the transaction
// calendar. Only days with at least one transaction are returned by the
// repository; the handler zero-fills the rest of the displayed month.
type DailySummary struct {
	Date     time.Time // day, truncated to midnight in the display timezone
	Count    int       // all transaction types
	SpendEUR float64   // sum of amount_eur where type = 'expense' only
}
