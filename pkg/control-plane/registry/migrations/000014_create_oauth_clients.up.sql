CREATE TABLE IF NOT EXISTS oauth_clients (
    id TEXT PRIMARY KEY,
    client_id TEXT NOT NULL UNIQUE,
    client_secret_hash TEXT NOT NULL,
    redirect_uris TEXT NOT NULL,
    scopes TEXT NOT NULL,
    tenant_id TEXT NOT NULL DEFAULT 'default',
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_oauth_clients_client_id ON oauth_clients(client_id);
