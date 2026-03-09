-- Add edge_id to robots (optional: robot is managed by edge node)
ALTER TABLE robots ADD COLUMN edge_id TEXT DEFAULT '';

-- Edges table: edge nodes that sync with cloud
CREATE TABLE IF NOT EXISTS edges (
    id TEXT PRIMARY KEY,
    last_heartbeat TIMESTAMP NOT NULL,
    config_json TEXT,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);

-- Edge commands: queued commands for edge to fetch on heartbeat
CREATE TABLE IF NOT EXISTS edge_commands (
    id TEXT PRIMARY KEY,
    edge_id TEXT NOT NULL,
    robot_id TEXT NOT NULL,
    command_json TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL,
    acked_at TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_edge_commands_edge_pending ON edge_commands(edge_id) WHERE acked_at IS NULL;
