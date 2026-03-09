CREATE TABLE IF NOT EXISTS command_idempotency (
    idempotency_key TEXT PRIMARY KEY,
    robot_id TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_cmd_idem_created ON command_idempotency(created_at);
