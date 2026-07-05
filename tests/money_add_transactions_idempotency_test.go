package tests

import (
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"personal/domain"
	"personal/util"
)

func (s *IntegrationTestSuite) TestAddTransactions_Idempotency_SkipsDuplicateKey() {
	ctx := s.Context()
	userID := s.UserID()

	at := time.Date(2026, 4, 6, 9, 0, 0, 0, time.UTC)

	first := &domain.Transaction{
		UserID:         userID,
		Type:           domain.TransactionTypeExpense,
		AmountOriginal: 5.00,
		Currency:       "EUR",
		AmountEUR:      5.00,
		Account:        "Revolut",
		Merchant:       "Starbucks",
		IdempotencyKey: util.Ptr("revolut:2026-04-06:starbucks:5.00"),
		TransactedAt:   at,
	}

	saved, err := s.Repo().AddTransactions(ctx, []*domain.Transaction{first})
	require.NoError(s.T(), err)
	require.Len(s.T(), saved, 1)
	firstID := saved[0].ID

	// Re-importing the same row (same idempotency key) alongside a genuinely new one
	// must skip the duplicate and only insert the new transaction.
	duplicate := &domain.Transaction{
		UserID:         userID,
		Type:           domain.TransactionTypeExpense,
		AmountOriginal: 5.00,
		Currency:       "EUR",
		AmountEUR:      5.00,
		Account:        "Revolut",
		Merchant:       "Starbucks",
		IdempotencyKey: util.Ptr("revolut:2026-04-06:starbucks:5.00"),
		TransactedAt:   at,
	}
	fresh := &domain.Transaction{
		UserID:         userID,
		Type:           domain.TransactionTypeExpense,
		AmountOriginal: 45.00,
		Currency:       "EUR",
		AmountEUR:      45.00,
		Account:        "Revolut",
		Merchant:       "Bolt",
		IdempotencyKey: util.Ptr("revolut:2026-04-06:bolt:45.00"),
		TransactedAt:   at.Add(time.Hour),
	}

	saved, err = s.Repo().AddTransactions(ctx, []*domain.Transaction{duplicate, fresh})
	require.NoError(s.T(), err)
	require.Len(s.T(), saved, 1, "duplicate idempotency key must be skipped, only the new transaction inserted")
	assert.Equal(s.T(), "Bolt", saved[0].Merchant)
	assert.NotEqual(s.T(), firstID, saved[0].ID)
}

func (s *IntegrationTestSuite) TestAddTransactions_Idempotency_SameKeyDifferentAccountDoesNotConflict() {
	ctx := s.Context()
	userID := s.UserID()

	at := time.Date(2026, 4, 6, 9, 0, 0, 0, time.UTC)

	// The unique index is scoped to (user_id, account, idempotency_key), so the
	// same key value in two different accounts must not be treated as a duplicate.
	revolutTx := &domain.Transaction{
		UserID:         userID,
		Type:           domain.TransactionTypeExpense,
		AmountOriginal: 5.00,
		Currency:       "EUR",
		AmountEUR:      5.00,
		Account:        "Revolut",
		Merchant:       "Starbucks",
		IdempotencyKey: util.Ptr("2026-04-06T09:00:00Z:5.00"),
		TransactedAt:   at,
	}
	bocTx := &domain.Transaction{
		UserID:         userID,
		Type:           domain.TransactionTypeExpense,
		AmountOriginal: 5.00,
		Currency:       "EUR",
		AmountEUR:      5.00,
		Account:        "Bank of Cyprus",
		Merchant:       "Starbucks",
		IdempotencyKey: util.Ptr("2026-04-06T09:00:00Z:5.00"),
		TransactedAt:   at,
	}

	saved, err := s.Repo().AddTransactions(ctx, []*domain.Transaction{revolutTx, bocTx})
	require.NoError(s.T(), err)
	require.Len(s.T(), saved, 2, "same idempotency key in different accounts must not conflict")
}

func (s *IntegrationTestSuite) TestAddTransactions_Idempotency_NilKeysDoNotConflict() {
	ctx := s.Context()
	userID := s.UserID()

	at := time.Date(2026, 4, 7, 9, 0, 0, 0, time.UTC)

	// Transactions without an idempotency key (e.g. manually added ones) must never
	// be treated as duplicates of one another.
	a := &domain.Transaction{
		UserID:         userID,
		Type:           domain.TransactionTypeExpense,
		AmountOriginal: 10.00,
		Currency:       "EUR",
		AmountEUR:      10.00,
		Account:        "Revolut",
		Merchant:       "Zara",
		TransactedAt:   at,
	}
	b := &domain.Transaction{
		UserID:         userID,
		Type:           domain.TransactionTypeExpense,
		AmountOriginal: 10.00,
		Currency:       "EUR",
		AmountEUR:      10.00,
		Account:        "Revolut",
		Merchant:       "Zara",
		TransactedAt:   at,
	}

	saved, err := s.Repo().AddTransactions(ctx, []*domain.Transaction{a, b})
	require.NoError(s.T(), err)
	require.Len(s.T(), saved, 2)
}
