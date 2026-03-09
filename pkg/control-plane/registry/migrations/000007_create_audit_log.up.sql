CREATE TABLE IF NOT EXISTS audit_log (
    id TEXT PRIMARY KEY,
    actor TEXT NOT NULL,
    action TEXT NOT NULL,
    resource TEXT NOT NULL,
    resource_id TEXT,
    timestamp TIMESTAMP NOT NULL,
    details TEXT,
    tenant_id TEXT
);

CREATE INDEX IF NOT EXISTS idx_audit_robot ON audit_log(resource, resource_id);
CREATE INDEX IF NOT EXISTS idx_audit_timestamp ON audit_log(timestamp);
