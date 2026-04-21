package tests

import (
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"personal/action/add_transactions"
	"personal/action/compare_periods"
	"personal/action/get_balance"
	"personal/action/get_budget_progress"
	"personal/action/get_spending_by_category"
	"personal/action/get_top_merchants"
	"personal/action/set_budget"
)

// --- get_spending_by_category ------------------------------------------------

func (s *IntegrationTestSuite) TestGetSpendingByCategory_Depth1() {
	ctx := s.Context()

	at := time.Date(2026, 4, 5, 10, 0, 0, 0, time.UTC)
	from := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 4, 30, 23, 59, 59, 0, time.UTC)

	_, _, err := add_transactions.AddTransactions(ctx, nil, add_transactions.AddTransactionsInput{
		Transactions: []add_transactions.TransactionInput{
			{Type: "expense", AmountOriginal: 10, Currency: "EUR", AmountEUR: 10, Account: "Revolut", Category: "food/cafe", Merchant: "Starbucks", TransactedAt: at},
			{Type: "expense", AmountOriginal: 40, Currency: "EUR", AmountEUR: 40, Account: "Revolut", Category: "food/restaurant", Merchant: "Zuma", TransactedAt: at.Add(time.Hour)},
			{Type: "expense", AmountOriginal: 25, Currency: "EUR", AmountEUR: 25, Account: "Revolut", Category: "transport/taxi", Merchant: "Bolt", TransactedAt: at.Add(2 * time.Hour)},
			{Type: "income", AmountOriginal: 3500, Currency: "EUR", AmountEUR: 3500, Account: "Revolut", Category: "salary", Merchant: "Employer", TransactedAt: at.Add(3 * time.Hour)},
		},
	})
	require.NoError(s.T(), err)

	_, out, err := get_spending_by_category.GetSpendingByCategory(ctx, nil,
		get_spending_by_category.GetSpendingByCategoryInput{From: from, To: to, Depth: 1})
	require.NoError(s.T(), err)

	// Income must be excluded; food and transport only
	require.Len(s.T(), out.Categories, 2)
	assert.InDelta(s.T(), 75.00, out.TotalEUR, 0.01)

	// First result is food (highest spend), second is transport
	assert.Equal(s.T(), "food", out.Categories[0].Category)
	assert.InDelta(s.T(), 50.00, out.Categories[0].TotalEUR, 0.01)
	assert.Equal(s.T(), 2, out.Categories[0].Count)

	assert.Equal(s.T(), "transport", out.Categories[1].Category)
	assert.InDelta(s.T(), 25.00, out.Categories[1].TotalEUR, 0.01)
}

func (s *IntegrationTestSuite) TestGetSpendingByCategory_Depth2() {
	ctx := s.Context()

	at := time.Date(2026, 4, 5, 10, 0, 0, 0, time.UTC)
	from := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 4, 30, 23, 59, 59, 0, time.UTC)

	_, _, err := add_transactions.AddTransactions(ctx, nil, add_transactions.AddTransactionsInput{
		Transactions: []add_transactions.TransactionInput{
			{Type: "expense", AmountOriginal: 10, Currency: "EUR", AmountEUR: 10, Account: "Revolut", Category: "food/cafe", Merchant: "Starbucks", TransactedAt: at},
			{Type: "expense", AmountOriginal: 40, Currency: "EUR", AmountEUR: 40, Account: "Revolut", Category: "food/restaurant", Merchant: "Zuma", TransactedAt: at.Add(time.Hour)},
		},
	})
	require.NoError(s.T(), err)

	_, out, err := get_spending_by_category.GetSpendingByCategory(ctx, nil,
		get_spending_by_category.GetSpendingByCategoryInput{From: from, To: to, Depth: 2})
	require.NoError(s.T(), err)

	// Should see food/cafe and food/restaurant as separate rows
	require.Len(s.T(), out.Categories, 2)
	cats := map[string]float64{}
	for _, c := range out.Categories {
		cats[c.Category] = c.TotalEUR
	}
	assert.InDelta(s.T(), 10.00, cats["food/cafe"], 0.01)
	assert.InDelta(s.T(), 40.00, cats["food/restaurant"], 0.01)
}

func (s *IntegrationTestSuite) TestGetSpendingByCategory_EmptyPeriod() {
	ctx := s.Context()

	_, _, err := add_transactions.AddTransactions(ctx, nil, add_transactions.AddTransactionsInput{
		Transactions: []add_transactions.TransactionInput{
			{Type: "expense", AmountOriginal: 10, Currency: "EUR", AmountEUR: 10, Account: "Revolut", Category: "food", Merchant: "Lidl",
				TransactedAt: time.Date(2026, 3, 15, 12, 0, 0, 0, time.UTC)},
		},
	})
	require.NoError(s.T(), err)

	from := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 4, 30, 23, 59, 59, 0, time.UTC)

	_, out, err := get_spending_by_category.GetSpendingByCategory(ctx, nil,
		get_spending_by_category.GetSpendingByCategoryInput{From: from, To: to, Depth: 1})
	require.NoError(s.T(), err)
	assert.Empty(s.T(), out.Categories)
	assert.Equal(s.T(), 0.0, out.TotalEUR)
}

// --- get_top_merchants -------------------------------------------------------

func (s *IntegrationTestSuite) TestGetTopMerchants_Success() {
	ctx := s.Context()

	at := time.Date(2026, 4, 5, 10, 0, 0, 0, time.UTC)
	from := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 4, 30, 23, 59, 59, 0, time.UTC)

	_, _, err := add_transactions.AddTransactions(ctx, nil, add_transactions.AddTransactionsInput{
		Transactions: []add_transactions.TransactionInput{
			{Type: "expense", AmountOriginal: 50, Currency: "EUR", AmountEUR: 50, Account: "Revolut", Category: "groceries", Merchant: "Lidl", TransactedAt: at},
			{Type: "expense", AmountOriginal: 30, Currency: "EUR", AmountEUR: 30, Account: "Revolut", Category: "groceries", Merchant: "Lidl", TransactedAt: at.Add(time.Hour)},
			{Type: "expense", AmountOriginal: 45, Currency: "EUR", AmountEUR: 45, Account: "Revolut", Category: "food/cafe", Merchant: "Costa Coffee", TransactedAt: at.Add(2 * time.Hour)},
			{Type: "expense", AmountOriginal: 10, Currency: "EUR", AmountEUR: 10, Account: "Revolut", Category: "transport", Merchant: "Bolt", TransactedAt: at.Add(3 * time.Hour)},
			{Type: "income", AmountOriginal: 3500, Currency: "EUR", AmountEUR: 3500, Account: "Revolut", Category: "salary", Merchant: "Employer", TransactedAt: at.Add(4 * time.Hour)},
		},
	})
	require.NoError(s.T(), err)

	_, out, err := get_top_merchants.GetTopMerchants(ctx, nil,
		get_top_merchants.GetTopMerchantsInput{From: from, To: to, Limit: 10})
	require.NoError(s.T(), err)

	// Income merchant excluded; top: Lidl 80, Costa Coffee 45, Bolt 10
	require.Len(s.T(), out.Merchants, 3)
	assert.Equal(s.T(), "Lidl", out.Merchants[0].Merchant)
	assert.InDelta(s.T(), 80.00, out.Merchants[0].TotalEUR, 0.01)
	assert.Equal(s.T(), 2, out.Merchants[0].Count)

	assert.Equal(s.T(), "Costa Coffee", out.Merchants[1].Merchant)
	assert.InDelta(s.T(), 45.00, out.Merchants[1].TotalEUR, 0.01)
}

func (s *IntegrationTestSuite) TestGetTopMerchants_LimitRespected() {
	ctx := s.Context()

	at := time.Date(2026, 4, 5, 10, 0, 0, 0, time.UTC)
	from := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 4, 30, 23, 59, 59, 0, time.UTC)

	txs := []add_transactions.TransactionInput{}
	for i, m := range []string{"M1", "M2", "M3", "M4", "M5"} {
		txs = append(txs, add_transactions.TransactionInput{
			Type: "expense", AmountOriginal: float64(10 + i), Currency: "EUR",
			AmountEUR: float64(10 + i), Account: "Revolut", Category: "misc", Merchant: m,
			TransactedAt: at.Add(time.Duration(i) * time.Hour),
		})
	}
	_, _, err := add_transactions.AddTransactions(ctx, nil,
		add_transactions.AddTransactionsInput{Transactions: txs})
	require.NoError(s.T(), err)

	_, out, err := get_top_merchants.GetTopMerchants(ctx, nil,
		get_top_merchants.GetTopMerchantsInput{From: from, To: to, Limit: 3})
	require.NoError(s.T(), err)
	assert.Len(s.T(), out.Merchants, 3)
}

// --- compare_periods ---------------------------------------------------------

func (s *IntegrationTestSuite) TestComparePeriods_Success() {
	ctx := s.Context()

	_, _, err := add_transactions.AddTransactions(ctx, nil, add_transactions.AddTransactionsInput{
		Transactions: []add_transactions.TransactionInput{
			{Type: "expense", AmountOriginal: 100, Currency: "EUR", AmountEUR: 100, Account: "Revolut", Category: "food", Merchant: "Lidl", TransactedAt: time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC)},
			{Type: "expense", AmountOriginal: 50, Currency: "EUR", AmountEUR: 50, Account: "Revolut", Category: "transport", Merchant: "Bolt", TransactedAt: time.Date(2026, 3, 15, 12, 0, 0, 0, time.UTC)},
		},
	})
	require.NoError(s.T(), err)

	_, _, err = add_transactions.AddTransactions(ctx, nil, add_transactions.AddTransactionsInput{
		Transactions: []add_transactions.TransactionInput{
			{Type: "expense", AmountOriginal: 120, Currency: "EUR", AmountEUR: 120, Account: "Revolut", Category: "food", Merchant: "Lidl", TransactedAt: time.Date(2026, 4, 5, 12, 0, 0, 0, time.UTC)},
			{Type: "expense", AmountOriginal: 30, Currency: "EUR", AmountEUR: 30, Account: "Revolut", Category: "transport", Merchant: "Bolt", TransactedAt: time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)},
		},
	})
	require.NoError(s.T(), err)

	_, out, err := compare_periods.ComparePeriods(ctx, nil, compare_periods.ComparePeriodsInput{
		PeriodAFrom: time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		PeriodATo:   time.Date(2026, 3, 31, 23, 59, 59, 0, time.UTC),
		PeriodBFrom: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
		PeriodBTo:   time.Date(2026, 4, 30, 23, 59, 59, 0, time.UTC),
	})
	require.NoError(s.T(), err)

	assert.InDelta(s.T(), 150.00, out.PeriodA.TotalEUR, 0.01)
	assert.InDelta(s.T(), 150.00, out.PeriodB.TotalEUR, 0.01)

	diffByCategory := map[string]compare_periods.CategoryDiff{}
	for _, d := range out.Diff {
		diffByCategory[d.Category] = d
	}

	foodDiff := diffByCategory["food"]
	assert.InDelta(s.T(), 100.00, foodDiff.PeriodAEUR, 0.01)
	assert.InDelta(s.T(), 120.00, foodDiff.PeriodBEUR, 0.01)
	assert.InDelta(s.T(), 20.00, foodDiff.DiffEUR, 0.01)
	assert.InDelta(s.T(), 20.00, foodDiff.DiffPct, 0.5)

	transportDiff := diffByCategory["transport"]
	assert.InDelta(s.T(), 50.00, transportDiff.PeriodAEUR, 0.01)
	assert.InDelta(s.T(), 30.00, transportDiff.PeriodBEUR, 0.01)
	assert.InDelta(s.T(), -20.00, transportDiff.DiffEUR, 0.01)
}

func (s *IntegrationTestSuite) TestComparePeriods_NewCategoryInPeriodB() {
	ctx := s.Context()

	_, _, err := add_transactions.AddTransactions(ctx, nil, add_transactions.AddTransactionsInput{
		Transactions: []add_transactions.TransactionInput{
			{Type: "expense", AmountOriginal: 50, Currency: "EUR", AmountEUR: 50, Account: "Revolut", Category: "food", Merchant: "Lidl", TransactedAt: time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC)},
			{Type: "expense", AmountOriginal: 200, Currency: "EUR", AmountEUR: 200, Account: "Revolut", Category: "electronics", Merchant: "Apple", TransactedAt: time.Date(2026, 4, 5, 12, 0, 0, 0, time.UTC)},
		},
	})
	require.NoError(s.T(), err)

	_, out, err := compare_periods.ComparePeriods(ctx, nil, compare_periods.ComparePeriodsInput{
		PeriodAFrom: time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		PeriodATo:   time.Date(2026, 3, 31, 23, 59, 59, 0, time.UTC),
		PeriodBFrom: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
		PeriodBTo:   time.Date(2026, 4, 30, 23, 59, 59, 0, time.UTC),
	})
	require.NoError(s.T(), err)

	diffByCategory := map[string]compare_periods.CategoryDiff{}
	for _, d := range out.Diff {
		diffByCategory[d.Category] = d
	}

	electronics := diffByCategory["electronics"]
	assert.InDelta(s.T(), 0.00, electronics.PeriodAEUR, 0.01)
	assert.InDelta(s.T(), 200.00, electronics.PeriodBEUR, 0.01)
}

// --- get_budget_progress -----------------------------------------------------

func (s *IntegrationTestSuite) TestGetBudgetProgress_ActiveBudget() {
	ctx := s.Context()

	_, _, err := set_budget.SetBudget(ctx, nil, set_budget.SetBudgetInput{
		Name:      "Food - April 2026",
		Category:  "food",
		AmountEUR: 500.00,
		StartsAt:  time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
		EndsAt:    time.Date(2026, 4, 30, 23, 59, 59, 0, time.UTC),
	})
	require.NoError(s.T(), err)

	_, _, err = add_transactions.AddTransactions(ctx, nil, add_transactions.AddTransactionsInput{
		Transactions: []add_transactions.TransactionInput{
			{Type: "expense", AmountOriginal: 150, Currency: "EUR", AmountEUR: 150, Account: "Revolut", Category: "food/cafe", Merchant: "Starbucks", TransactedAt: time.Date(2026, 4, 5, 10, 0, 0, 0, time.UTC)},
			{Type: "expense", AmountOriginal: 170, Currency: "EUR", AmountEUR: 170, Account: "Revolut", Category: "food/restaurant", Merchant: "Zuma", TransactedAt: time.Date(2026, 4, 10, 20, 0, 0, 0, time.UTC)},
		},
	})
	require.NoError(s.T(), err)

	at := time.Date(2026, 4, 15, 0, 0, 0, 0, time.UTC)
	_, out, err := get_budget_progress.GetBudgetProgress(ctx, nil,
		get_budget_progress.GetBudgetProgressInput{At: at})
	require.NoError(s.T(), err)

	require.Len(s.T(), out.Budgets, 1)
	b := out.Budgets[0]
	assert.Equal(s.T(), "Food - April 2026", b.Name)
	assert.InDelta(s.T(), 500.00, b.AmountEUR, 0.01)
	assert.InDelta(s.T(), 320.00, b.SpentEUR, 0.01)
	assert.InDelta(s.T(), 180.00, b.RemainingEUR, 0.01)
}

func (s *IntegrationTestSuite) TestGetBudgetProgress_TransfersExcluded() {
	ctx := s.Context()

	_, _, err := set_budget.SetBudget(ctx, nil, set_budget.SetBudgetInput{
		Name:      "Food - April 2026",
		Category:  "food",
		AmountEUR: 300.00,
		StartsAt:  time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
		EndsAt:    time.Date(2026, 4, 30, 23, 59, 59, 0, time.UTC),
	})
	require.NoError(s.T(), err)

	_, _, err = add_transactions.AddTransactions(ctx, nil, add_transactions.AddTransactionsInput{
		Transactions: []add_transactions.TransactionInput{
			{Type: "expense", AmountOriginal: 50, Currency: "EUR", AmountEUR: 50, Account: "Revolut", Category: "food", Merchant: "Lidl", TransactedAt: time.Date(2026, 4, 5, 10, 0, 0, 0, time.UTC)},
			{Type: "transfer", AmountOriginal: 200, Currency: "EUR", AmountEUR: 200, Account: "Revolut", Category: "food", Merchant: "Savings", TransactedAt: time.Date(2026, 4, 6, 10, 0, 0, 0, time.UTC)},
		},
	})
	require.NoError(s.T(), err)

	at := time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC)
	_, out, err := get_budget_progress.GetBudgetProgress(ctx, nil,
		get_budget_progress.GetBudgetProgressInput{At: at})
	require.NoError(s.T(), err)

	require.Len(s.T(), out.Budgets, 1)
	assert.InDelta(s.T(), 50.00, out.Budgets[0].SpentEUR, 0.01)
	assert.InDelta(s.T(), 250.00, out.Budgets[0].RemainingEUR, 0.01)
}

func (s *IntegrationTestSuite) TestGetBudgetProgress_InactiveBudget() {
	ctx := s.Context()

	_, _, err := set_budget.SetBudget(ctx, nil, set_budget.SetBudgetInput{
		Name:      "March Budget",
		Category:  "food",
		AmountEUR: 400.00,
		StartsAt:  time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		EndsAt:    time.Date(2026, 3, 31, 23, 59, 59, 0, time.UTC),
	})
	require.NoError(s.T(), err)

	at := time.Date(2026, 4, 5, 0, 0, 0, 0, time.UTC)
	_, out, err := get_budget_progress.GetBudgetProgress(ctx, nil,
		get_budget_progress.GetBudgetProgressInput{At: at})
	require.NoError(s.T(), err)
	assert.Empty(s.T(), out.Budgets)
}

// --- get_balance -------------------------------------------------------------

func (s *IntegrationTestSuite) TestGetBalance_Success() {
	ctx := s.Context()

	from := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 4, 30, 23, 59, 59, 0, time.UTC)

	_, _, err := add_transactions.AddTransactions(ctx, nil, add_transactions.AddTransactionsInput{
		Transactions: []add_transactions.TransactionInput{
			{Type: "income", AmountOriginal: 3500, Currency: "EUR", AmountEUR: 3500, Account: "Revolut", Category: "salary", Merchant: "Employer", TransactedAt: time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)},
			{Type: "expense", AmountOriginal: 320, Currency: "EUR", AmountEUR: 320, Account: "Revolut", Category: "food", Merchant: "Lidl", TransactedAt: time.Date(2026, 4, 5, 12, 0, 0, 0, time.UTC)},
			{Type: "expense", AmountOriginal: 700, Currency: "EUR", AmountEUR: 700, Account: "Revolut", Category: "rent", Merchant: "Landlord", TransactedAt: time.Date(2026, 4, 1, 9, 0, 0, 0, time.UTC)},
			{Type: "transfer", AmountOriginal: 500, Currency: "EUR", AmountEUR: 500, Account: "Revolut", Category: "", Merchant: "Savings", TransactedAt: time.Date(2026, 4, 10, 14, 0, 0, 0, time.UTC)},
		},
	})
	require.NoError(s.T(), err)

	_, out, err := get_balance.GetBalance(ctx, nil,
		get_balance.GetBalanceInput{From: from, To: to})
	require.NoError(s.T(), err)

	assert.InDelta(s.T(), 3500.00, out.IncomeEUR, 0.01)
	assert.InDelta(s.T(), 1020.00, out.ExpenseEUR, 0.01)
	assert.InDelta(s.T(), 2480.00, out.BalanceEUR, 0.01)
}

func (s *IntegrationTestSuite) TestGetBalance_OnlyExpenses() {
	ctx := s.Context()

	from := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 4, 30, 23, 59, 59, 0, time.UTC)

	_, _, err := add_transactions.AddTransactions(ctx, nil, add_transactions.AddTransactionsInput{
		Transactions: []add_transactions.TransactionInput{
			{Type: "expense", AmountOriginal: 200, Currency: "EUR", AmountEUR: 200, Account: "Revolut", Category: "food", Merchant: "Lidl", TransactedAt: time.Date(2026, 4, 5, 12, 0, 0, 0, time.UTC)},
		},
	})
	require.NoError(s.T(), err)

	_, out, err := get_balance.GetBalance(ctx, nil,
		get_balance.GetBalanceInput{From: from, To: to})
	require.NoError(s.T(), err)

	assert.InDelta(s.T(), 0.00, out.IncomeEUR, 0.01)
	assert.InDelta(s.T(), 200.00, out.ExpenseEUR, 0.01)
	assert.InDelta(s.T(), -200.00, out.BalanceEUR, 0.01)
}

func (s *IntegrationTestSuite) TestGetBalance_EmptyPeriod() {
	ctx := s.Context()

	from := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 4, 30, 23, 59, 59, 0, time.UTC)

	_, out, err := get_balance.GetBalance(ctx, nil,
		get_balance.GetBalanceInput{From: from, To: to})
	require.NoError(s.T(), err)

	assert.Equal(s.T(), 0.0, out.IncomeEUR)
	assert.Equal(s.T(), 0.0, out.ExpenseEUR)
	assert.Equal(s.T(), 0.0, out.BalanceEUR)
}
