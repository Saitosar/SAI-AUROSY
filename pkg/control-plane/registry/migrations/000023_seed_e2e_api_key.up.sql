-- Seed API keys for E2E tests.
-- e2e-api-key: operator, tenant default. Hash: SHA256("e2e-api-key")
-- e2e-sandbox-key: operator, tenant sandbox. Hash: SHA256("e2e-sandbox-key")
-- e2e-admin-key: administrator. Hash: SHA256("e2e-admin-key")
INSERT INTO api_keys (id, key_hash, name, roles, tenant_id, created_at)
VALUES
  ('e2e-test-key', '4ef78803d8fc2e82a84b91cf3080df3142474ed2864ba0e7b7b890ead8b811c0', 'E2E Test', 'operator', 'default', CURRENT_TIMESTAMP),
  ('e2e-sandbox-key', 'e5182dbbf8580fa7d25b4cfbb6f7007cc4220bca2aa81a33659fbdc86f11b512', 'E2E Sandbox', 'operator', 'sandbox', CURRENT_TIMESTAMP),
  ('e2e-admin-key', '4487c398ecfe2ea2be3ad112057efc64800dfbab1f55a4d5e064fa5268fdb12c', 'E2E Admin', 'administrator', 'default', CURRENT_TIMESTAMP)
ON CONFLICT (id) DO NOTHING;
