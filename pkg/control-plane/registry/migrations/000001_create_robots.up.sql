CREATE TABLE IF NOT EXISTS robots (
    id TEXT PRIMARY KEY,
    vendor TEXT NOT NULL,
    model TEXT NOT NULL,
    adapter_endpoint TEXT NOT NULL,
    tenant_id TEXT NOT NULL DEFAULT 'default',
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);
