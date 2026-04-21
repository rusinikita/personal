package tests

import (
	"context"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"personal/action/add_transactions"
	"personal/action/delete_transaction"
	"personal/action/edit_transactions"
	"personal/action/get_transactions"
)

// insertTestTransaction inserts a single expense via add_transactions and
// returns the assigned ID. ctx must be the same context used in the test
// to ensure consistent user_id isolation.
func (s *IntegrationTestSuite) insertTestTransaction(
	ctx context.Context,
	category, merchant, account string,
	amountEUR float64,
	at time.Time,
) int64 {
	input := add_transactions.AddTransactionsInput{
		Transactions: []add_transactions.TransactionInput{
			{
				Type:           "expense",
				AmountOriginal: amountEUR,
				Currency:       "EUR",
				AmountEUR:      amountEUR,
				Account:        account,
				Category:       category,
				Merchant:       merchant,
				TransactedAt:   at,
			},
		},
	}
	_, out, err := add_transactions.AddTransactions(ctx, nil, input)
	s.Require().NoError(err)
	s.Require().Len(out.Transactions, 1)
	return out.Transactions[0].ID
}

// --- edit_transactions -------------------------------------------------------

func (s *IntegrationTestSuite) TestEditTransactions_Category() {
	ctx := s.Context()

	id := s.insertTestTransaction(ctx, "food", "Lidl", "Revolut", 32.00,
		time.Date(2026, 4, 2, 11, 0, 0, 0, time.UTC))

	newCategory := "groceries/supermarket"
	editInput := edit_transactions.EditTransactionsInput{
		Updates: []edit_transactions.TransactionUpdate{
			{ID: id, Category: &newCategory},
		},
	}

	_, editOut, err := edit_transactions.EditTransactions(ctx, nil, editInput)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), 1, editOut.UpdatedCount)

	_, listOut, err := get_transactions.GetTransactions(ctx, nil,
		get_transactions.GetTransactionsInput{Limit: 10})
	require.NoError(s.T(), err)
	require.Len(s.T(), listOut.Transactions, 1)
	assert.Equal(s.T(), newCategory, listOut.Transactions[0].Category)
}

func (s *IntegrationTestSuite) TestEditTransactions_MultipleFields() {
	ctx := s.Context()

	id := s.insertTestTransaction(ctx, "food/restaurant", "Unknown", "Revolut", 75.00,
		time.Date(2026, 4, 3, 20, 0, 0, 0, time.UTC))

	newMerchant := "Zuma"
	newNote := "anniversary dinner"
	editInput := edit_transactions.EditTransactionsInput{
		Updates: []edit_transactions.TransactionUpdate{
			{ID: id, Merchant: &newMerchant, Note: &newNote},
		},
	}

	_, editOut, err := edit_transactions.EditTransactions(ctx, nil, editInput)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), 1, editOut.UpdatedCount)

	_, listOut, err := get_transactions.GetTransactions(ctx, nil,
		get_transactions.GetTransactionsInput{Limit: 10})
	require.NoError(s.T(), err)
	require.Len(s.T(), listOut.Transactions, 1)
	assert.Equal(s.T(), newMerchant, listOut.Transactions[0].Merchant)
	assert.Equal(s.T(), &newNote, listOut.Transactions[0].Note)
}

func (s *IntegrationTestSuite) TestEditTransactions_Bulk() {
	ctx := s.Context()

	at := time.Date(2026, 4, 4, 9, 0, 0, 0, time.UTC)
	id1 := s.insertTestTransaction(ctx, "misc", "Shop A", "Revolut", 10.00, at)
	id2 := s.insertTestTransaction(ctx, "misc", "Shop B", "Revolut", 20.00, at.Add(time.Hour))
	id3 := s.insertTestTransaction(ctx, "misc", "Shop C", "Revolut", 30.00, at.Add(2*time.Hour))

	newCat := "shopping/clothes"
	editInput := edit_transactions.EditTransactionsInput{
		Updates: []edit_transactions.TransactionUpdate{
			{ID: id1, Category: &newCat},
			{ID: id2, Category: &newCat},
			{ID: id3, Category: &newCat},
		},
	}

	_, editOut, err := edit_transactions.EditTransactions(ctx, nil, editInput)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), 3, editOut.UpdatedCount)

	_, listOut, err := get_transactions.GetTransactions(ctx, nil,
		get_transactions.GetTransactionsInput{Limit: 10})
	require.NoError(s.T(), err)
	for _, tx := range listOut.Transactions {
		assert.Equal(s.T(), newCat, tx.Category)
	}
}

func (s *IntegrationTestSuite) TestEditTransactions_NotFound() {
	ctx := s.Context()

	newCat := "food"
	editInput := edit_transactions.EditTransactionsInput{
		Updates: []edit_transactions.TransactionUpdate{
			{ID: 999999, Category: &newCat},
		},
	}

	_, out, err := edit_transactions.EditTransactions(ctx, nil, editInput)
	require.NoError(s.T(), err)
	assert.NotEmpty(s.T(), out.Error)
}

func (s *IntegrationTestSuite) TestEditTransactions_PartialNotOwned() {
	ctx := s.Context()

	id := s.insertTestTransaction(ctx, "food", "Lidl", "Revolut", 15.00,
		time.Date(2026, 4, 5, 10, 0, 0, 0, time.UTC))

	newCat := "groceries"
	editInput := edit_transactions.EditTransactionsInput{
		Updates: []edit_transactions.TransactionUpdate{
			{ID: id, Category: &newCat},
			{ID: 999999, Category: &newCat},
		},
	}

	_, out, err := edit_transactions.EditTransactions(ctx, nil, editInput)
	require.NoError(s.T(), err)
	assert.NotEmpty(s.T(), out.Error)

	// Original transaction must remain unchanged
	_, listOut, err := get_transactions.GetTransactions(ctx, nil,
		get_transactions.GetTransactionsInput{Limit: 10})
	require.NoError(s.T(), err)
	require.Len(s.T(), listOut.Transactions, 1)
	assert.Equal(s.T(), "food", listOut.Transactions[0].Category)
}

// --- delete_transaction ------------------------------------------------------

func (s *IntegrationTestSuite) TestDeleteTransaction_Success() {
	ctx := s.Context()

	id := s.insertTestTransaction(ctx, "food/cafe", "Costa Coffee", "Revolut", 4.50,
		time.Date(2026, 4, 4, 8, 30, 0, 0, time.UTC))

	_, delOut, err := delete_transaction.DeleteTransaction(ctx, nil,
		delete_transaction.DeleteTransactionInput{ID: id})
	require.NoError(s.T(), err)
	assert.True(s.T(), delOut.Deleted)

	_, listOut, err := get_transactions.GetTransactions(ctx, nil,
		get_transactions.GetTransactionsInput{Limit: 10})
	require.NoError(s.T(), err)
	assert.Equal(s.T(), 0, listOut.Total)
}

func (s *IntegrationTestSuite) TestDeleteTransaction_NotFound() {
	ctx := s.Context()

	_, out, err := delete_transaction.DeleteTransaction(ctx, nil,
		delete_transaction.DeleteTransactionInput{ID: 999999})
	require.NoError(s.T(), err)
	assert.NotEmpty(s.T(), out.Error)
}

func (s *IntegrationTestSuite) TestDeleteTransaction_OtherUserIsolation() {
	ctx := s.Context()

	id := s.insertTestTransaction(ctx, "food", "Lidl", "Revolut", 10.00,
		time.Date(2026, 4, 5, 9, 0, 0, 0, time.UTC))

	_, out, err := delete_transaction.DeleteTransaction(ctx, nil,
		delete_transaction.DeleteTransactionInput{ID: id + 1000000})
	require.NoError(s.T(), err)
	assert.NotEmpty(s.T(), out.Error)
}
