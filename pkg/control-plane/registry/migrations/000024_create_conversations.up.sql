CREATE TABLE IF NOT EXISTS conversations (
    id TEXT PRIMARY KEY,
    intent TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    response_template TEXT NOT NULL,
    response_provider_url TEXT,
    supported_languages TEXT NOT NULL,
    tenant_id TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_conversations_intent ON conversations(intent);
CREATE INDEX IF NOT EXISTS idx_conversations_tenant ON conversations(tenant_id);
