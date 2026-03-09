#!/bin/bash
# E2E multi-tenant: verify tenant isolation. Requires docker-compose.e2e.yml (PostgreSQL).
# Keys: e2e-api-key (default), e2e-sandbox-key (sandbox), e2e-admin-key (admin)
set -e

API="${API_URL:-http://localhost:8080}"
API_PREFIX="${API_PREFIX:-/v1}"

echo "=== E2E Multi-Tenant ==="
echo "API: $API$API_PREFIX"

# Operator default: should see x1-001, go2-001 (not sandbox-r1, sandbox-r2)
echo "1. Operator default: GET /v1/robots -> only default tenant robots"
robots=$(curl -s -H "X-API-Key: e2e-api-key" "$API$API_PREFIX/robots")
echo "$robots" | grep -q "x1-001" || { echo "FAIL: default operator should see x1-001"; exit 1; }
echo "$robots" | grep -q "go2-001" || { echo "FAIL: default operator should see go2-001"; exit 1; }
echo "$robots" | grep -q "sandbox-r1" && { echo "FAIL: default operator should NOT see sandbox-r1"; exit 1; }
echo "   OK: default operator sees only default tenant robots"

# Operator sandbox: should see sandbox-r1, sandbox-r2 (not x1-001, go2-001)
echo "2. Operator sandbox: GET /v1/robots -> only sandbox tenant robots"
robots=$(curl -s -H "X-API-Key: e2e-sandbox-key" "$API$API_PREFIX/robots")
echo "$robots" | grep -q "sandbox-r1" || { echo "FAIL: sandbox operator should see sandbox-r1"; exit 1; }
echo "$robots" | grep -q "sandbox-r2" || { echo "FAIL: sandbox operator should see sandbox-r2"; exit 1; }
echo "$robots" | grep -q "x1-001" && { echo "FAIL: sandbox operator should NOT see x1-001"; exit 1; }
echo "   OK: sandbox operator sees only sandbox tenant robots"

# Operator default: POST command to sandbox robot -> 404
echo "3. Operator default: POST command to sandbox-r1 -> 404"
code=$(curl -s -o /dev/null -w "%{http_code}" -X POST -H "X-API-Key: e2e-api-key" -H "Content-Type: application/json" \
  -d '{"command":"safe_stop","operator_id":"e2e"}' "$API$API_PREFIX/robots/sandbox-r1/command")
[ "$code" = "404" ] || { echo "FAIL: expected 404, got $code"; exit 1; }
echo "   OK: 404 for cross-tenant command"

# Operator default: POST task for sandbox robot -> 403
echo "4. Operator default: POST task for sandbox-r1 -> 403"
code=$(curl -s -o /dev/null -w "%{http_code}" -X POST -H "X-API-Key: e2e-api-key" -H "Content-Type: application/json" \
  -d '{"robot_id":"sandbox-r1","scenario_id":"standby","operator_id":"e2e"}' "$API$API_PREFIX/tasks")
[ "$code" = "403" ] || { echo "FAIL: expected 403, got $code"; exit 1; }
echo "   OK: 403 for cross-tenant task"

# Admin: GET /v1/robots?tenant_id=default -> sees default robots
echo "5. Admin: GET /v1/robots?tenant_id=default"
robots=$(curl -s -H "X-API-Key: e2e-admin-key" "$API$API_PREFIX/robots?tenant_id=default")
echo "$robots" | grep -q "x1-001" || { echo "FAIL: admin should see x1-001 for default tenant"; exit 1; }
echo "   OK: admin can filter by tenant_id"

echo "=== E2E Multi-Tenant passed ==="
