CREATE TABLE IF NOT EXISTS webhook_delivery_failures (
    id TEXT PRIMARY KEY,
    webhook_id TEXT NOT NULL,
    event TEXT NOT NULL,
    payload_json TEXT,
    error TEXT,
    created_at TIMESTAMP NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_webhook_failures_created ON webhook_delivery_failures(created_at);
