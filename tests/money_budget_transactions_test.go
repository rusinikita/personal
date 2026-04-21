package tests

import (
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"personal/action/add_transactions"
	"personal/action/get_transactions"
	"personal/action/set_budget"
)

// --- get_transactions --------------------------------------------------------

func (s *IntegrationTestSuite) TestGetTransactions_NoFilters() {
	ctx := s.Context()

	at := time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)

	addInput := add_transactions.AddTransactionsInput{
		Transactions: []add_transactions.TransactionInput{
			{Type: "expense", AmountOriginal: 10, Currency: "EUR", AmountEUR: 10, Account: "Revolut", Category: "food", Merchant: "Lidl", TransactedAt: at},
			{Type: "expense", AmountOriginal: 20, Currency: "EUR", AmountEUR: 20, Account: "Revolut", Category: "transport", Merchant: "Bolt", TransactedAt: at.Add(time.Hour)},
			{Type: "income", AmountOriginal: 3500, Currency: "EUR", AmountEUR: 3500, Account: "Revolut", Category: "salary", Merchant: "Employer", TransactedAt: at.Add(2 * time.Hour)},
		},
	}
	_, _, err := add_transactions.AddTransactions(ctx, nil, addInput)
	require.NoError(s.T(), err)

	_, out, err := get_transactions.GetTransactions(ctx, nil,
		get_transactions.GetTransactionsInput{Limit: 50})
	require.NoError(s.T(), err)
	assert.Equal(s.T(), 3, out.Total)
	assert.Len(s.T(), out.Transactions, 3)
}

func (s *IntegrationTestSuite) TestGetTransactions_FilterByType() {
	ctx := s.Context()

	at := time.Date(2026, 4, 2, 9, 0, 0, 0, time.UTC)
	expenseType := "expense"

	addInput := add_transactions.AddTransactionsInput{
		Transactions: []add_transactions.TransactionInput{
			{Type: "expense", AmountOriginal: 15, Currency: "EUR", AmountEUR: 15, Account: "Revolut", Category: "food", Merchant: "Lidl", TransactedAt: at},
			{Type: "income", AmountOriginal: 500, Currency: "EUR", AmountEUR: 500, Account: "Revolut", Category: "freelance", Merchant: "Client", TransactedAt: at.Add(time.Hour)},
		},
	}
	_, _, err := add_transactions.AddTransactions(ctx, nil, addInput)
	require.NoError(s.T(), err)

	_, out, err := get_transactions.GetTransactions(ctx, nil,
		get_transactions.GetTransactionsInput{Type: &expenseType, Limit: 50})
	require.NoError(s.T(), err)
	assert.Equal(s.T(), 1, out.Total)
	assert.Equal(s.T(), "expense", out.Transactions[0].Type)
}

func (s *IntegrationTestSuite) TestGetTransactions_FilterByCategory() {
	ctx := s.Context()

	at := time.Date(2026, 4, 3, 10, 0, 0, 0, time.UTC)
	catFilter := "food"

	addInput := add_transactions.AddTransactionsInput{
		Transactions: []add_transactions.TransactionInput{
			{Type: "expense", AmountOriginal: 5, Currency: "EUR", AmountEUR: 5, Account: "Revolut", Category: "food/cafe", Merchant: "Starbucks", TransactedAt: at},
			{Type: "expense", AmountOriginal: 45, Currency: "EUR", AmountEUR: 45, Account: "Revolut", Category: "food/restaurant", Merchant: "Zuma", TransactedAt: at.Add(time.Hour)},
			{Type: "expense", AmountOriginal: 30, Currency: "EUR", AmountEUR: 30, Account: "Revolut", Category: "transport/taxi", Merchant: "Bolt", TransactedAt: at.Add(2 * time.Hour)},
		},
	}
	_, _, err := add_transactions.AddTransactions(ctx, nil, addInput)
	require.NoError(s.T(), err)

	_, out, err := get_transactions.GetTransactions(ctx, nil,
		get_transactions.GetTransactionsInput{Category: &catFilter, Limit: 50})
	require.NoError(s.T(), err)
	assert.Equal(s.T(), 2, out.Total)
	for _, tx := range out.Transactions {
		assert.Contains(s.T(), tx.Category, "food")
	}
}

func (s *IntegrationTestSuite) TestGetTransactions_FilterByDateRange() {
	ctx := s.Context()

	march := time.Date(2026, 3, 15, 12, 0, 0, 0, time.UTC)
	april := time.Date(2026, 4, 15, 12, 0, 0, 0, time.UTC)
	fromFilter := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	toFilter := time.Date(2026, 4, 30, 23, 59, 59, 0, time.UTC)

	addInput := add_transactions.AddTransactionsInput{
		Transactions: []add_transactions.TransactionInput{
			{Type: "expense", AmountOriginal: 10, Currency: "EUR", AmountEUR: 10, Account: "Revolut", Category: "food", Merchant: "Lidl", TransactedAt: march},
			{Type: "expense", AmountOriginal: 20, Currency: "EUR", AmountEUR: 20, Account: "Revolut", Category: "food", Merchant: "Lidl", TransactedAt: april},
		},
	}
	_, _, err := add_transactions.AddTransactions(ctx, nil, addInput)
	require.NoError(s.T(), err)

	_, out, err := get_transactions.GetTransactions(ctx, nil,
		get_transactions.GetTransactionsInput{From: &fromFilter, To: &toFilter, Limit: 50})
	require.NoError(s.T(), err)
	assert.Equal(s.T(), 1, out.Total)
	assert.Equal(s.T(), 20.0, out.Transactions[0].AmountEUR)
}

func (s *IntegrationTestSuite) TestGetTransactions_FilterByMerchant() {
	ctx := s.Context()

	at := time.Date(2026, 4, 5, 10, 0, 0, 0, time.UTC)
	merchant := "Lidl"

	addInput := add_transactions.AddTransactionsInput{
		Transactions: []add_transactions.TransactionInput{
			{Type: "expense", AmountOriginal: 25, Currency: "EUR", AmountEUR: 25, Account: "Revolut", Category: "groceries", Merchant: "Lidl", TransactedAt: at},
			{Type: "expense", AmountOriginal: 15, Currency: "EUR", AmountEUR: 15, Account: "Revolut", Category: "groceries", Merchant: "Carrefour", TransactedAt: at.Add(time.Hour)},
		},
	}
	_, _, err := add_transactions.AddTransactions(ctx, nil, addInput)
	require.NoError(s.T(), err)

	_, out, err := get_transactions.GetTransactions(ctx, nil,
		get_transactions.GetTransactionsInput{Merchant: &merchant, Limit: 50})
	require.NoError(s.T(), err)
	assert.Equal(s.T(), 1, out.Total)
	assert.Equal(s.T(), "Lidl", out.Transactions[0].Merchant)
}

func (s *IntegrationTestSuite) TestGetTransactions_LimitAndOffset() {
	ctx := s.Context()

	at := time.Date(2026, 4, 1, 9, 0, 0, 0, time.UTC)
	txs := make([]add_transactions.TransactionInput, 5)
	for i := range txs {
		txs[i] = add_transactions.TransactionInput{
			Type: "expense", AmountOriginal: float64(i + 1), Currency: "EUR",
			AmountEUR: float64(i + 1), Account: "Revolut", Category: "misc", Merchant: "Shop",
			TransactedAt: at.Add(time.Duration(i) * time.Hour),
		}
	}
	_, _, err := add_transactions.AddTransactions(ctx, nil,
		add_transactions.AddTransactionsInput{Transactions: txs})
	require.NoError(s.T(), err)

	// Page 1: first 2
	_, page1, err := get_transactions.GetTransactions(ctx, nil,
		get_transactions.GetTransactionsInput{Limit: 2, Offset: 0})
	require.NoError(s.T(), err)
	assert.Equal(s.T(), 5, page1.Total)
	assert.Len(s.T(), page1.Transactions, 2)

	// Page 2: next 2
	_, page2, err := get_transactions.GetTransactions(ctx, nil,
		get_transactions.GetTransactionsInput{Limit: 2, Offset: 2})
	require.NoError(s.T(), err)
	assert.Len(s.T(), page2.Transactions, 2)

	// Ensure no overlap
	ids1 := map[int64]bool{page1.Transactions[0].ID: true, page1.Transactions[1].ID: true}
	for _, tx := range page2.Transactions {
		assert.False(s.T(), ids1[tx.ID], "duplicate transaction on page 2")
	}
}

// --- set_budget --------------------------------------------------------------

func (s *IntegrationTestSuite) TestSetBudget_Success() {
	ctx := s.Context()

	input := set_budget.SetBudgetInput{
		Name:      "Food - April 2026",
		Category:  "food",
		AmountEUR: 500.00,
		StartsAt:  time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
		EndsAt:    time.Date(2026, 4, 30, 23, 59, 59, 0, time.UTC),
	}

	_, out, err := set_budget.SetBudget(ctx, nil, input)
	require.NoError(s.T(), err)
	assert.NotZero(s.T(), out.ID)
	assert.Equal(s.T(), "Food - April 2026", out.Budget.Name)
	assert.Equal(s.T(), "food", out.Budget.Category)
	assert.Equal(s.T(), 500.00, out.Budget.AmountEUR)
}

func (s *IntegrationTestSuite) TestSetBudget_Upsert() {
	ctx := s.Context()

	base := set_budget.SetBudgetInput{
		Name:      "Transport - April 2026",
		Category:  "transport",
		AmountEUR: 100.00,
		StartsAt:  time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
		EndsAt:    time.Date(2026, 4, 30, 23, 59, 59, 0, time.UTC),
	}

	_, first, err := set_budget.SetBudget(ctx, nil, base)
	require.NoError(s.T(), err)

	base.AmountEUR = 150.00
	_, second, err := set_budget.SetBudget(ctx, nil, base)
	require.NoError(s.T(), err)

	assert.Equal(s.T(), first.ID, second.ID)
	assert.Equal(s.T(), 150.00, second.Budget.AmountEUR)
}

func (s *IntegrationTestSuite) TestSetBudget_ValidationError_NegativeAmount() {
	ctx := s.Context()

	input := set_budget.SetBudgetInput{
		Name:      "Bad Budget",
		Category:  "food",
		AmountEUR: -50.00,
		StartsAt:  time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
		EndsAt:    time.Date(2026, 4, 30, 23, 59, 59, 0, time.UTC),
	}

	_, out, err := set_budget.SetBudget(ctx, nil, input)
	require.NoError(s.T(), err)
	assert.NotEmpty(s.T(), out.Error)
}

func (s *IntegrationTestSuite) TestSetBudget_ValidationError_EndBeforeStart() {
	ctx := s.Context()

	input := set_budget.SetBudgetInput{
		Name:      "Wrong Dates",
		Category:  "food",
		AmountEUR: 200.00,
		StartsAt:  time.Date(2026, 4, 30, 0, 0, 0, 0, time.UTC),
		EndsAt:    time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
	}

	_, out, err := set_budget.SetBudget(ctx, nil, input)
	require.NoError(s.T(), err)
	assert.NotEmpty(s.T(), out.Error)
}
