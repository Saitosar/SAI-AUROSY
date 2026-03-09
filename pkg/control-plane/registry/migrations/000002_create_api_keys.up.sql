CREATE TABLE IF NOT EXISTS api_keys (
    id TEXT PRIMARY KEY,
    key_hash TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    roles TEXT NOT NULL,
    tenant_id TEXT NOT NULL DEFAULT 'default',
    created_at TIMESTAMP NOT NULL,
    expires_at TIMESTAMP
);
