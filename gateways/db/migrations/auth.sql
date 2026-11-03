CREATE TABLE IF NOT EXISTS oauth_code (
    code_hash      TEXT PRIMARY KEY,
    client_id      TEXT NOT NULL,
    user_id        BIGINT NOT NULL,
    redirect_uri   TEXT NOT NULL,
    code_challenge TEXT NOT NULL,
    resource       TEXT NOT NULL,
    expires_at     TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS oauth_refresh_token (
    token_hash TEXT PRIMARY KEY,
    family_id  UUID NOT NULL,
    client_id  TEXT NOT NULL,
    user_id    BIGINT NOT NULL,
    resource   TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at    TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS oauth_refresh_token_family_idx ON oauth_refresh_token (family_id);
