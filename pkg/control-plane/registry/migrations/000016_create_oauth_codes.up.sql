CREATE TABLE IF NOT EXISTS oauth_codes (
    id TEXT PRIMARY KEY,
    code_hash TEXT NOT NULL UNIQUE,
    client_id TEXT NOT NULL,
    tenant_id TEXT NOT NULL DEFAULT 'default',
    scopes TEXT NOT NULL,
    redirect_uri TEXT NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_oauth_codes_code ON oauth_codes(code_hash);
