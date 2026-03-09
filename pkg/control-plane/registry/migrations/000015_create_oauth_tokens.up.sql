CREATE TABLE IF NOT EXISTS oauth_tokens (
    id TEXT PRIMARY KEY,
    access_token_hash TEXT NOT NULL UNIQUE,
    refresh_token_hash TEXT,
    client_id TEXT NOT NULL,
    tenant_id TEXT NOT NULL DEFAULT 'default',
    scopes TEXT NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_oauth_tokens_access_token ON oauth_tokens(access_token_hash);
CREATE INDEX IF NOT EXISTS idx_oauth_tokens_refresh_token ON oauth_tokens(refresh_token_hash);
