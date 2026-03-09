#!/bin/bash
# E2E with auth: uses docker-compose.e2e.yml (PostgreSQL + auth), seeds API key e2e-api-key
set -e

export E2E_API_KEY="${E2E_API_KEY:-e2e-api-key}"
echo "=== E2E Auth: API key=$E2E_API_KEY ==="

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
"$SCRIPT_DIR/e2e.sh"
