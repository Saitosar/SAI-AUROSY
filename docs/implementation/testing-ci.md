# Testing and CI

This document describes the testing strategy, how to run tests, and the CI pipeline for SAI AUROSY.

## Test Types

### Unit Tests

Unit tests cover individual packages in isolation. They do not require external services.

**Examples**: `pkg/control-plane/auth/rbac_test.go`, `pkg/control-plane/coordinator/coordinator_test.go`, `pkg/control-plane/orchestration/runner_test.go`, `pkg/control-plane/registry/*_test.go`, `pkg/control-plane/tenants/*_test.go`.

### Integration Tests

Integration tests exercise components with real or mocked dependencies. Some require NATS:

- **Task Runner** (`pkg/control-plane/tasks/runner_test.go`): Requires NATS for command publishing. Skips if NATS is unavailable.
- **Edge Agent** (`pkg/edge/agent_test.go`): Requires NATS for command relay. Skips if NATS is unavailable.
- **Coordinator**, **Workflow Runner**: No external dependencies.

### E2E Tests

End-to-end tests verify the full stack via HTTP. They require:

- Docker Compose (NATS, Control Plane, adapters, webhook receiver)
- All services running and reachable

**Script**: `scripts/e2e.sh`

**Scenarios**:

1. List robots, send safe_stop commands
2. Create task (POST /v1/tasks)
3. Run workflow (POST /v1/workflows/patrol_zones_ABC/run)
4. Register webhook, trigger safe_stop, verify delivery

**E2E with auth** (when `E2E_API_KEY` is set):

0. GET /v1/robots without auth → 401
1. GET /v1/robots with invalid key → 401
2. GET /v1/robots with valid key → 200
3. Then run scenarios 1–4 with API key

## Running Tests

### Unit and Integration Tests

```bash
# All tests (NATS required for Task Runner and Edge Agent)
make test

# Or directly
go test -v ./...
```

**NATS for integration tests**: Start NATS locally or use Docker:

```bash
docker run -d -p 4222:4222 --name nats-test nats:2-alpine
go test -v ./...
```

Tests that need NATS will skip with a message if it is unavailable.

### E2E Tests

```bash
# Build, start stack, run E2E script (no auth)
make e2e

# E2E with auth (PostgreSQL + API key e2e-api-key)
make e2e-auth

# E2E multi-tenant (run after e2e-auth; requires stack with auth)
make e2e-tenant
```

Or manually:

```bash
# No auth (ALLOW_UNSAFE_NO_AUTH=true)
docker compose up -d
sleep 5
bash scripts/e2e.sh

# With auth (PostgreSQL, seeded API key)
docker compose -f docker-compose.yml -f docker-compose.e2e.yml up -d
sleep 15
E2E_API_KEY=e2e-api-key bash scripts/e2e.sh
```

**Environment variables** (optional):

| Variable       | Description                          | Default                    |
|----------------|--------------------------------------|----------------------------|
| `API_URL`      | Control Plane base URL               | `http://localhost:8080`    |
| `API_PREFIX`   | API path prefix                      | `/v1`                      |
| `E2E_API_KEY`  | API key for authenticated requests   | (none)                     |
| `WEBHOOK_URL`  | Webhook receiver URL (for Control Plane) | `http://webhook-receiver:5000/webhooks/sai-aurosy` |
| `LAST_EVENT_URL` | URL to verify webhook delivery    | `http://localhost:5000/last-event` |

## CI Pipeline

GitHub Actions workflow (`.github/workflows/ci.yml`):

1. **test**: Runs `go test ./...` with NATS as a service container.
2. **e2e**: Builds and starts Docker Compose, runs `scripts/e2e.sh` (no auth).
3. **e2e-auth**: Builds and starts with `docker-compose.e2e.yml` (PostgreSQL + auth), runs E2E with `E2E_API_KEY=e2e-api-key`, then runs multi-tenant isolation tests.

Triggers: push and pull requests to `main` or `master`.

## Load Tests

Load tests use [Vegeta](https://github.com/tsenart/vegeta). Install: `brew install vegeta`.

```bash
# With auth (stack from make e2e-auth)
make load-test

# Without auth (stack from make docker)
make load-test-no-auth

# Custom rate and duration
RATE=100 DURATION=30s make load-test
```

**Scenarios:** `GET /v1/robots` at configurable RPS. Targets: p95 < 1s (see [Production Runbook](../operations/production-runbook.md) for SLOs).

## Test Coverage

| Component        | Location                               | Notes                          |
|------------------|----------------------------------------|--------------------------------|
| Coordinator      | `pkg/control-plane/coordinator/`       | Zone acquire/release           |
| Workflow Runner  | `pkg/control-plane/orchestration/`     | Run, robot selector, tenant    |
| Task Runner      | `pkg/control-plane/tasks/`             | Scenario execution, zone, cancel|
| Edge Agent       | `pkg/edge/`                            | Sync, command relay            |
| API (tenant)     | `pkg/control-plane/api/`               | Tenant enforcement             |
| Auth (RBAC)      | `pkg/control-plane/auth/`              | Role checks                    |
