#!/bin/bash
# Load test using Vegeta. Requires: vegeta (brew install vegeta)
# Usage: API_URL=http://localhost:8080 [E2E_API_KEY=e2e-api-key] bash scripts/load/vegeta.sh
set -e

API="${API_URL:-http://localhost:8080}"
API_PREFIX="${API_PREFIX:-/v1}"
E2E_API_KEY="${E2E_API_KEY:-}"

RATE="${RATE:-50}"
DURATION="${DURATION:-10s}"

if ! command -v vegeta &>/dev/null; then
  echo "vegeta not found. Install: brew install vegeta"
  exit 1
fi

echo "=== Load Test: $API$API_PREFIX ==="
echo "Rate: ${RATE} RPS, Duration: ${DURATION}"

# GET /v1/robots
EXTRA_HEADERS=()
[ -n "$E2E_API_KEY" ] && EXTRA_HEADERS=(-header "X-API-Key: $E2E_API_KEY")

echo "GET /v1/robots"
echo "GET $API$API_PREFIX/robots" | vegeta attack "${EXTRA_HEADERS[@]}" -rate="$RATE" -duration="$DURATION" | vegeta report

echo ""
echo "Latency distribution (GET /v1/robots):"
echo "GET $API$API_PREFIX/robots" | vegeta attack "${EXTRA_HEADERS[@]}" -rate="$RATE" -duration="$DURATION" 2>/dev/null | vegeta report -type="hist[0,10ms,25ms,50ms,100ms,250ms,500ms,1s,2.5s,5s,10s]"

echo ""
echo "=== Load test complete ==="
