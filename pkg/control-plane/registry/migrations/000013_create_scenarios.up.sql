CREATE TABLE IF NOT EXISTS scenarios (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    steps TEXT NOT NULL,
    required_capabilities TEXT NOT NULL,
    tenant_id TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

-- Seed built-in scenarios
INSERT INTO scenarios (id, name, description, steps, required_capabilities, tenant_id, created_at, updated_at) VALUES
('standby', 'Ожидание', 'Стоячая поза', '[{"command":"stand_mode","payload":null,"duration_sec":0}]', '["stand"]', NULL, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('patrol', 'Патруль', 'walk_mode + cmd_vel N сек', '[{"command":"walk_mode","payload":null,"duration_sec":0},{"command":"cmd_vel","payload":{"linear_x":0.3,"linear_y":0,"angular_z":0},"duration_sec":-1},{"command":"cmd_vel","payload":{"linear_x":0,"linear_y":0,"angular_z":0},"duration_sec":0}]', '["walk","cmd_vel","patrol"]', NULL, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('navigation', 'Навигация', 'walk_mode + движение по параметрам', '[{"command":"walk_mode","payload":null,"duration_sec":0},{"command":"cmd_vel","payload":null,"duration_sec":-1}]', '["walk","cmd_vel","navigation"]', NULL, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON CONFLICT (id) DO NOTHING;
