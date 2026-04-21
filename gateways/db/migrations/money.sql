CREATE TABLE IF NOT EXISTS transactions (
    id               BIGSERIAL PRIMARY KEY,
    user_id          BIGINT NOT NULL,
    type             VARCHAR(10) NOT NULL,
    amount_original  DECIMAL(12,2) NOT NULL,
    currency         CHAR(3) NOT NULL,
    amount_eur       DECIMAL(12,2) NOT NULL,
    account          VARCHAR(100) NOT NULL,
    category         VARCHAR(255) NOT NULL DEFAULT '',
    merchant         VARCHAR(255) NOT NULL DEFAULT '',
    note             TEXT,
    original_description TEXT,
    transacted_at    TIMESTAMPTZ NOT NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT check_tx_type CHECK (type IN ('expense', 'income', 'transfer')),
    CONSTRAINT check_tx_amount_original CHECK (amount_original > 0),
    CONSTRAINT check_tx_amount_eur CHECK (amount_eur > 0),
    CONSTRAINT check_tx_currency_length CHECK (char_length(currency) = 3)
);

CREATE INDEX IF NOT EXISTS idx_transactions_user_date ON transactions(user_id, transacted_at DESC);

CREATE TABLE IF NOT EXISTS budgets (
    id          BIGSERIAL PRIMARY KEY,
    user_id     BIGINT NOT NULL,
    name        VARCHAR(255) NOT NULL,
    category    VARCHAR(255) NOT NULL,
    amount_eur  DECIMAL(12,2) NOT NULL,
    starts_at   TIMESTAMPTZ NOT NULL,
    ends_at     TIMESTAMPTZ NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT check_budget_amount CHECK (amount_eur > 0),
    CONSTRAINT check_budget_period CHECK (ends_at > starts_at),
    CONSTRAINT uq_budgets_user_name UNIQUE (user_id, name)
);

CREATE INDEX IF NOT EXISTS idx_budgets_user_period ON budgets(user_id, starts_at, ends_at);
