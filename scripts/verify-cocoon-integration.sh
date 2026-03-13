#!/usr/bin/env bash
# Verify SAI-AUROSY Cocoon integration.
# Prerequisites: Cocoon client running (see scripts/cocoon-test-setup.md)
# Optional: --start to launch docker compose with cocoon-test override

set -e

COCOON_URL="${COGNITIVE_COCOON_CLIENT_URL:-http://localhost:10000}"
CP_URL="${CONTROL_PLANE_URL:-http://localhost:8080}"
START_STACK=false

for arg in "$@"; do
  case "$arg" in
    --start) START_STACK=true ;;
  esac
done

echo "=== Cocoon Integration Verification ==="
echo "Cocoon client: $COCOON_URL"
echo "Control Plane: $CP_URL"
echo ""

# 1. Check Cocoon client reachable
echo "1. Checking Cocoon client..."
if curl -sf -X POST "$COCOON_URL/v1/chat/completions" \
  -H "Content-Type: application/json" \
  -d '{"model":"Qwen/Qwen3-0.6B","messages":[{"role":"user","content":"hi"}],"max_tokens":5}' >/dev/null 2>&1; then
  echo "   OK: Cocoon client reachable"
else
  echo "   FAIL: Cocoon client not reachable at $COCOON_URL"
  echo "   Start Cocoon first: see scripts/cocoon-test-setup.md"
  exit 1
fi

# 2. Start stack if requested
if [ "$START_STACK" = true ]; then
  echo ""
  echo "2. Starting SAI-AUROSY stack with Cocoon..."
  cd "$(dirname "$0")/.."
  docker compose -f docker-compose.yml -f docker-compose.cocoon-test.yml up -d
  echo "   Waiting for Control Plane..."
  for i in $(seq 1 30); do
    if curl -sf "$CP_URL/health" >/dev/null 2>&1; then
      break
    fi
    sleep 1
  done
  if ! curl -sf "$CP_URL/health" >/dev/null 2>&1; then
    echo "   FAIL: Control Plane did not become ready"
    exit 1
  fi
  echo "   OK: Control Plane ready"
else
  echo ""
  echo "2. Checking Control Plane..."
  if ! curl -sf "$CP_URL/health" >/dev/null 2>&1; then
    echo "   FAIL: Control Plane not reachable at $CP_URL"
    echo "   Start with: docker compose -f docker-compose.yml -f docker-compose.cocoon-test.yml up -d"
    echo "   Or run with --start to launch the stack automatically"
    exit 1
  fi
  echo "   OK: Control Plane reachable"
fi

# 3. Call understand-intent
echo ""
echo "3. Calling POST /v1/cognitive/understand-intent..."
RESP=$(curl -sf -X POST "$CP_URL/v1/cognitive/understand-intent" \
  -H "Content-Type: application/json" \
  -d '{"robot_id":"test","text":"Where is Nike?"}' 2>/dev/null || true)

if [ -z "$RESP" ]; then
  echo "   FAIL: No response (auth may be required; ensure ALLOW_UNSAFE_NO_AUTH=true)"
  exit 1
fi

# 4. Verify response contains intent
if echo "$RESP" | grep -q '"intent"'; then
  echo "   OK: Response contains intent"
  echo "   Response: $RESP"
else
  echo "   WARN: Response may not contain intent (Cocoon may return empty on error)"
  echo "   Response: $RESP"
  echo "   If Cocoon local-all uses fake backend, intent may be empty. Use test+fake-ton for real LLM."
fi

echo ""
echo "=== Verification complete ==="
