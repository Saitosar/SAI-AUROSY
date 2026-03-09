CREATE TABLE IF NOT EXISTS telemetry_aggregates (
    id TEXT PRIMARY KEY,
    robot_id TEXT NOT NULL,
    bucket_start TIMESTAMP NOT NULL,
    bucket_type TEXT NOT NULL,
    online_count INTEGER NOT NULL,
    error_count INTEGER NOT NULL,
    tenant_id TEXT
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_agg_robot_bucket ON telemetry_aggregates(robot_id, bucket_start, bucket_type);
CREATE INDEX IF NOT EXISTS idx_agg_bucket_start ON telemetry_aggregates(bucket_start);
