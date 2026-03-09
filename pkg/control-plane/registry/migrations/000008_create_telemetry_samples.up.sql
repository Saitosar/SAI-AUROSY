CREATE TABLE IF NOT EXISTS telemetry_samples (
    id TEXT PRIMARY KEY,
    robot_id TEXT NOT NULL,
    timestamp TIMESTAMP NOT NULL,
    online INTEGER NOT NULL,
    actuator_status TEXT,
    current_task TEXT,
    imu_json TEXT,
    joint_states_json TEXT,
    tenant_id TEXT
);

CREATE INDEX IF NOT EXISTS idx_telemetry_robot_time ON telemetry_samples(robot_id, timestamp);
