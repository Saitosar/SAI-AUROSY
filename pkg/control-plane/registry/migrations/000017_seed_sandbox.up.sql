-- Sandbox tenant for developer API testing
INSERT INTO tenants (id, name, config) VALUES ('sandbox', 'Sandbox', NULL)
ON CONFLICT (id) DO NOTHING;

-- Demo robots for sandbox tenant (SQLite and PostgreSQL compatible)
INSERT INTO robots (id, vendor, model, adapter_endpoint, tenant_id, edge_id, capabilities, created_at, updated_at)
VALUES
  ('sandbox-r1', 'agibot', 'X1', 'nats://localhost:4222', 'sandbox', '', '["walk","stand","cmd_vel","patrol","navigation"]', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
  ('sandbox-r2', 'unitree', 'Go2', 'nats://localhost:4222', 'sandbox', '', '["walk","stand","cmd_vel","patrol","navigation"]', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON CONFLICT (id) DO UPDATE SET
  vendor = EXCLUDED.vendor,
  model = EXCLUDED.model,
  adapter_endpoint = EXCLUDED.adapter_endpoint,
  tenant_id = EXCLUDED.tenant_id,
  edge_id = EXCLUDED.edge_id,
  capabilities = EXCLUDED.capabilities,
  updated_at = EXCLUDED.updated_at;
