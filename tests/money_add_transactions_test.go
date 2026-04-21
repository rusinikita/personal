package tests

import (
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"personal/action/add_transactions"
	"personal/action/get_transactions"
)

func (s *IntegrationTestSuite) TestAddTransactions_Success() {
	ctx := s.Context()

	at := time.Date(2026, 4, 4, 9, 0, 0, 0, time.UTC)

	input := add_transactions.AddTransactionsInput{
		Transactions: []add_transactions.TransactionInput{
			{
				Type:           "expense",
				AmountOriginal: 5.00,
				Currency:       "EUR",
				AmountEUR:      5.00,
				Account:        "Revolut",
				Category:       "food/cafe",
				Merchant:       "Starbucks",
				TransactedAt:   at,
			},
			{
				Type:           "expense",
				AmountOriginal: 45.00,
				Currency:       "EUR",
				AmountEUR:      45.00,
				Account:        "Bank of Cyprus",
				Category:       "transport/taxi",
				Merchant:       "Bolt",
				TransactedAt:   at.Add(11 * time.Hour),
			},
		},
	}

	_, out, err := add_transactions.AddTransactions(ctx, nil, input)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), 2, out.InsertedCount)
	require.Len(s.T(), out.Transactions, 2)
	assert.NotZero(s.T(), out.Transactions[0].ID)
	assert.NotZero(s.T(), out.Transactions[1].ID)

	// Verify via get_transactions
	listInput := get_transactions.GetTransactionsInput{
		From:  &at,
		Limit: 10,
	}
	_, listOut, err := get_transactions.GetTransactions(ctx, nil, listInput)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), 2, listOut.Total)
}

func (s *IntegrationTestSuite) TestAddTransactions_SingleIncome() {
	ctx := s.Context()

	at := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)

	input := add_transactions.AddTransactionsInput{
		Transactions: []add_transactions.TransactionInput{
			{
				Type:           "income",
				AmountOriginal: 3500.00,
				Currency:       "EUR",
				AmountEUR:      3500.00,
				Account:        "Revolut",
				Category:       "salary",
				Merchant:       "Employer",
				TransactedAt:   at,
			},
		},
	}

	_, out, err := add_transactions.AddTransactions(ctx, nil, input)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), 1, out.InsertedCount)
	assert.Equal(s.T(), "income", out.Transactions[0].Type)
	assert.Equal(s.T(), 3500.00, out.Transactions[0].AmountEUR)
}

func (s *IntegrationTestSuite) TestAddTransactions_WithNote() {
	ctx := s.Context()

	at := time.Date(2026, 4, 5, 14, 0, 0, 0, time.UTC)
	note := "team lunch"
	origDesc := "ZUMA RESTAURANT 0012 NICOSIA"

	input := add_transactions.AddTransactionsInput{
		Transactions: []add_transactions.TransactionInput{
			{
				Type:                "expense",
				AmountOriginal:      120.00,
				Currency:            "EUR",
				AmountEUR:           120.00,
				Account:             "Revolut",
				Category:            "food/restaurant",
				Merchant:            "Zuma",
				Note:                &note,
				OriginalDescription: &origDesc,
				TransactedAt:        at,
			},
		},
	}

	_, out, err := add_transactions.AddTransactions(ctx, nil, input)
	require.NoError(s.T(), err)
	require.Len(s.T(), out.Transactions, 1)
	assert.Equal(s.T(), &note, out.Transactions[0].Note)
	assert.Equal(s.T(), &origDesc, out.Transactions[0].OriginalDescription)
}

func (s *IntegrationTestSuite) TestAddTransactions_ValidationError_InvalidType() {
	ctx := s.Context()

	input := add_transactions.AddTransactionsInput{
		Transactions: []add_transactions.TransactionInput{
			{
				Type:           "refund",
				AmountOriginal: 10.00,
				Currency:       "EUR",
				AmountEUR:      10.00,
				Account:        "Revolut",
				TransactedAt:   time.Now(),
			},
		},
	}

	_, out, err := add_transactions.AddTransactions(ctx, nil, input)
	require.NoError(s.T(), err)
	assert.NotEmpty(s.T(), out.Error)
}

func (s *IntegrationTestSuite) TestAddTransactions_ValidationError_ZeroAmount() {
	ctx := s.Context()

	input := add_transactions.AddTransactionsInput{
		Transactions: []add_transactions.TransactionInput{
			{
				Type:           "expense",
				AmountOriginal: 0,
				Currency:       "EUR",
				AmountEUR:      0,
				Account:        "Revolut",
				TransactedAt:   time.Now(),
			},
		},
	}

	_, out, err := add_transactions.AddTransactions(ctx, nil, input)
	require.NoError(s.T(), err)
	assert.NotEmpty(s.T(), out.Error)
}

func (s *IntegrationTestSuite) TestAddTransactions_ValidationError_BadCurrency() {
	ctx := s.Context()

	input := add_transactions.AddTransactionsInput{
		Transactions: []add_transactions.TransactionInput{
			{
				Type:           "expense",
				AmountOriginal: 10.00,
				Currency:       "EU",
				AmountEUR:      10.00,
				Account:        "Revolut",
				TransactedAt:   time.Now(),
			},
		},
	}

	_, out, err := add_transactions.AddTransactions(ctx, nil, input)
	require.NoError(s.T(), err)
	assert.NotEmpty(s.T(), out.Error)
}

func (s *IntegrationTestSuite) TestAddTransactions_MultiCurrency() {
	ctx := s.Context()

	at := time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC)

	input := add_transactions.AddTransactionsInput{
		Transactions: []add_transactions.TransactionInput{
			{
				Type:           "expense",
				AmountOriginal: 10.00,
				Currency:       "USD",
				AmountEUR:      9.20,
				Account:        "Revolut",
				Category:       "shopping",
				Merchant:       "Amazon",
				TransactedAt:   at,
			},
		},
	}

	_, out, err := add_transactions.AddTransactions(ctx, nil, input)
	require.NoError(s.T(), err)
	require.Len(s.T(), out.Transactions, 1)
	assert.Equal(s.T(), "USD", out.Transactions[0].Currency)
	assert.Equal(s.T(), 10.00, out.Transactions[0].AmountOriginal)
	assert.Equal(s.T(), 9.20, out.Transactions[0].AmountEUR)
}
