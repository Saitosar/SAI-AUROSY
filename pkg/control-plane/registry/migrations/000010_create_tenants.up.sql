CREATE TABLE IF NOT EXISTS tenants (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    config TEXT
);

INSERT INTO tenants (id, name, config) VALUES ('default', 'Default', NULL) ON CONFLICT (id) DO NOTHING;
