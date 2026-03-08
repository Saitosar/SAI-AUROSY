#!/bin/bash
# E2E test: verify Control Plane API and safe_stop flow
set -e

API="${API_URL:-http://localhost:8080}"

echo "=== E2E: Status + Safe Stop ==="
echo "API: $API"

echo "1. GET /robots"
robots=$(curl -s "$API/robots")
echo "$robots" | grep -q x1-001 || { echo "FAIL: x1-001 not in fleet"; exit 1; }
echo "   OK: x1-001 in fleet"
echo "$robots" | grep -q go2-001 || { echo "FAIL: go2-001 not in fleet"; exit 1; }
echo "   OK: go2-001 in fleet"

echo "2. POST /robots/x1-001/command (safe_stop)"
res=$(curl -s -w "%{http_code}" -X POST "$API/robots/x1-001/command" \
  -H "Content-Type: application/json" \
  -d '{"command":"safe_stop","operator_id":"e2e"}')
code="${res: -3}"
[ "$code" = "202" ] || { echo "FAIL: expected 202, got $code"; exit 1; }
echo "   OK: safe_stop accepted for x1-001 (202)"

echo "3. POST /robots/go2-001/command (safe_stop)"
res=$(curl -s -w "%{http_code}" -X POST "$API/robots/go2-001/command" \
  -H "Content-Type: application/json" \
  -d '{"command":"safe_stop","operator_id":"e2e"}')
code="${res: -3}"
[ "$code" = "202" ] || { echo "FAIL: expected 202, got $code"; exit 1; }
echo "   OK: safe_stop accepted for go2-001 (202)"

echo "=== E2E passed ==="
