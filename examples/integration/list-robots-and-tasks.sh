#!/bin/bash
# List robots and tasks using the SAI AUROSY API
# Usage: ./list-robots-and-tasks.sh [BASE_URL] [API_KEY]
# Example: ./list-robots-and-tasks.sh http://localhost:8080/api/v1 sk-integration-abc123

BASE_URL="${1:-http://localhost:8080/api/v1}"
API_KEY="${2:-}"

if [ -z "$API_KEY" ]; then
  echo "Usage: $0 [BASE_URL] [API_KEY]"
  echo "Example: $0 http://localhost:8080/api/v1 your-api-key"
  exit 1
fi

echo "=== Listing robots ==="
curl -s -H "X-API-Key: $API_KEY" "$BASE_URL/robots" | jq .

echo ""
echo "=== Listing tasks ==="
curl -s -H "X-API-Key: $API_KEY" "$BASE_URL/tasks" | jq .
