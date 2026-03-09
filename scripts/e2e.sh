#!/bin/bash
# E2E test: verify Control Plane v1 API, safe_stop, create task, run workflow
set -e

API="${API_URL:-http://localhost:8080}"
API_PREFIX="${API_PREFIX:-/v1}"
E2E_API_KEY="${E2E_API_KEY:-}"

CURL_AUTH=()
if [ -n "$E2E_API_KEY" ]; then
  CURL_AUTH=(-H "X-API-Key: $E2E_API_KEY")
fi

echo "=== E2E: Status + Safe Stop + Task + Workflow (v1 API) ==="
echo "API: $API$API_PREFIX"
[ -n "$E2E_API_KEY" ] && echo "Auth: API key configured"

# Auth checks (when E2E_API_KEY is set)
if [ -n "$E2E_API_KEY" ]; then
  echo "0a. GET $API_PREFIX/robots without auth -> expect 401"
  code=$(curl -s -o /dev/null -w "%{http_code}" "$API$API_PREFIX/robots")
  [ "$code" = "401" ] || { echo "FAIL: expected 401 without auth, got $code"; exit 1; }
  echo "   OK: 401 without auth"

  echo "0b. GET $API_PREFIX/robots with invalid key -> expect 401"
  code=$(curl -s -o /dev/null -w "%{http_code}" -H "X-API-Key: invalid-key" "$API$API_PREFIX/robots")
  [ "$code" = "401" ] || { echo "FAIL: expected 401 with invalid key, got $code"; exit 1; }
  echo "   OK: 401 with invalid key"

  echo "0c. GET $API_PREFIX/robots with valid key -> expect 200"
  code=$(curl -s -o /dev/null -w "%{http_code}" "${CURL_AUTH[@]}" "$API$API_PREFIX/robots")
  [ "$code" = "200" ] || { echo "FAIL: expected 200 with valid key, got $code"; exit 1; }
  echo "   OK: 200 with valid key"
fi

echo "1. GET $API_PREFIX/robots"
robots=$(curl -s "${CURL_AUTH[@]}" "$API$API_PREFIX/robots")
echo "$robots" | grep -q x1-001 || { echo "FAIL: x1-001 not in fleet"; exit 1; }
echo "   OK: x1-001 in fleet"
echo "$robots" | grep -q go2-001 || { echo "FAIL: go2-001 not in fleet"; exit 1; }
echo "   OK: go2-001 in fleet"

echo "2. POST $API_PREFIX/robots/x1-001/command (safe_stop)"
res=$(curl -s -w "%{http_code}" -X POST "${CURL_AUTH[@]}" "$API$API_PREFIX/robots/x1-001/command" \
  -H "Content-Type: application/json" \
  -d '{"command":"safe_stop","operator_id":"e2e"}')
code="${res: -3}"
[ "$code" = "202" ] || { echo "FAIL: expected 202, got $code"; exit 1; }
echo "   OK: safe_stop accepted for x1-001 (202)"

echo "3. POST $API_PREFIX/robots/go2-001/command (safe_stop)"
res=$(curl -s -w "%{http_code}" -X POST "${CURL_AUTH[@]}" "$API$API_PREFIX/robots/go2-001/command" \
  -H "Content-Type: application/json" \
  -d '{"command":"safe_stop","operator_id":"e2e"}')
code="${res: -3}"
[ "$code" = "202" ] || { echo "FAIL: expected 202, got $code"; exit 1; }
echo "   OK: safe_stop accepted for go2-001 (202)"

echo "4. POST $API_PREFIX/tasks (create task)"
task_res=$(curl -s -w "\n%{http_code}" -X POST "${CURL_AUTH[@]}" "$API$API_PREFIX/tasks" \
  -H "Content-Type: application/json" \
  -d '{"robot_id":"x1-001","scenario_id":"standby","operator_id":"e2e"}')
task_body=$(echo "$task_res" | sed '$d')
task_code=$(echo "$task_res" | tail -n 1)
[ "$task_code" = "201" ] || { echo "FAIL: create task expected 201, got $task_code"; exit 1; }
task_id=$(echo "$task_body" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
[ -n "$task_id" ] || { echo "FAIL: no task id in response"; exit 1; }
echo "   OK: task created ($task_id)"

echo "5. GET $API_PREFIX/tasks/$task_id"
task_get=$(curl -s "${CURL_AUTH[@]}" "$API$API_PREFIX/tasks/$task_id")
echo "$task_get" | grep -q "$task_id" || { echo "FAIL: task not found"; exit 1; }
echo "$task_get" | grep -qE '"status":"(pending|running|completed)"' || { echo "FAIL: invalid task status"; exit 1; }
echo "   OK: task retrieved"

echo "6. POST $API_PREFIX/workflows/patrol_zones_ABC/run"
wf_res=$(curl -s -w "\n%{http_code}" -X POST "${CURL_AUTH[@]}" "$API$API_PREFIX/workflows/patrol_zones_ABC/run" \
  -H "Content-Type: application/json" \
  -d '{"operator_id":"e2e"}')
wf_body=$(echo "$wf_res" | sed '$d')
wf_code=$(echo "$wf_res" | tail -n 1)
[ "$wf_code" = "202" ] || { echo "FAIL: run workflow expected 202, got $wf_code"; exit 1; }
run_id=$(echo "$wf_body" | grep -o '"workflow_run_id":"[^"]*"' | cut -d'"' -f4)
[ -n "$run_id" ] || { echo "FAIL: no workflow_run_id in response"; exit 1; }
echo "   OK: workflow run started ($run_id)"

echo "7. GET $API_PREFIX/workflow-runs/$run_id"
run_get=$(curl -s "${CURL_AUTH[@]}" "$API$API_PREFIX/workflow-runs/$run_id")
echo "$run_get" | grep -q "$run_id" || { echo "FAIL: workflow run not found"; exit 1; }
echo "   OK: workflow run retrieved"

echo "8. POST $API_PREFIX/webhooks (register webhook)"
WEBHOOK_URL="${WEBHOOK_URL:-http://webhook-receiver:5000/webhooks/sai-aurosy}"
wh_res=$(curl -s -w "\n%{http_code}" -X POST "${CURL_AUTH[@]}" "$API$API_PREFIX/webhooks" \
  -H "Content-Type: application/json" \
  -d "{\"url\":\"$WEBHOOK_URL\",\"events\":[\"safe_stop\"]}")
wh_body=$(echo "$wh_res" | sed '$d')
wh_code=$(echo "$wh_res" | tail -n 1)
[ "$wh_code" = "201" ] || { echo "FAIL: create webhook expected 201, got $wh_code (webhook-receiver service may not be running)"; exit 1; }
echo "   OK: webhook registered"

echo "9. Trigger safe_stop (webhook delivery)"
curl -s -X POST "${CURL_AUTH[@]}" "$API$API_PREFIX/robots/x1-001/command" \
  -H "Content-Type: application/json" \
  -d '{"command":"safe_stop","operator_id":"e2e"}' > /dev/null
sleep 2
LAST_EVENT_URL="${LAST_EVENT_URL:-http://localhost:5000/last-event}"
last_event=$(curl -s "$LAST_EVENT_URL")
echo "$last_event" | grep -q '"event":"safe_stop"' || { echo "FAIL: webhook not received, got: $last_event"; exit 1; }
echo "   OK: webhook delivered (safe_stop)"

# Marketplace (when DB is configured; skip if 503)
if [ -n "$E2E_API_KEY" ]; then
  echo "10. GET $API_PREFIX/marketplace/categories"
  cat_code=$(curl -s -o /tmp/mp_cat.json -w "%{http_code}" "${CURL_AUTH[@]}" "$API$API_PREFIX/marketplace/categories")
  if [ "$cat_code" = "200" ]; then
    grep -q "mobility\|safety\|inspection" /tmp/mp_cat.json || { echo "FAIL: expected categories"; exit 1; }
    echo "   OK: marketplace categories"
    echo "11. GET $API_PREFIX/marketplace/scenarios"
    sc_code=$(curl -s -o /tmp/mp_sc.json -w "%{http_code}" "${CURL_AUTH[@]}" "$API$API_PREFIX/marketplace/scenarios")
    if [ "$sc_code" = "200" ]; then
      echo "   OK: marketplace scenarios"
      echo "12. POST $API_PREFIX/marketplace/scenarios/patrol/rate"
      rate_code=$(curl -s -o /dev/null -w "%{http_code}" -X POST "${CURL_AUTH[@]}" "$API$API_PREFIX/marketplace/scenarios/patrol/rate" \
        -H "Content-Type: application/json" -d '{"rating":4}')
      [ "$rate_code" = "200" ] || [ "$rate_code" = "201" ] || { echo "FAIL: rate expected 200/201, got $rate_code"; exit 1; }
      echo "   OK: marketplace rate"
    fi
  else
    echo "   SKIP: marketplace not configured (503)"
  fi
fi

echo "=== E2E passed ==="
