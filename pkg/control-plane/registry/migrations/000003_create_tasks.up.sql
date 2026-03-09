CREATE TABLE IF NOT EXISTS tasks (
    id TEXT PRIMARY KEY,
    robot_id TEXT NOT NULL,
    type TEXT NOT NULL,
    scenario_id TEXT,
    payload TEXT,
    status TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    completed_at TIMESTAMP,
    operator_id TEXT
);
