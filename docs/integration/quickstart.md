# Quick Start

This guide walks through a minimal integration: list robots, create a task, and receive webhook events.

## Prerequisites

- SAI AUROSY Control Plane running (e.g. `http://localhost:8080`)
- API key with `operator` role (or administrator key)
- Database configured (SQLite or PostgreSQL) for tasks and webhooks

## Step 1: Create an API Key

**Option A — Via API** (when you already have an admin key):

```bash
curl -X POST http://localhost:8080/api/v1/api-keys \
  -H "X-API-Key: <your-admin-key>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Quick Start Integration",
    "roles": "operator",
    "tenant_id": "default"
  }'
```

The response includes the raw `key` — store it securely; it is shown only once.

**Option B — Initial setup** (create first admin key in database):

```sql
-- Generate a key: e.g. "sk-integration-abc123"
-- Hash it: echo -n "sk-integration-abc123" | sha256sum
INSERT INTO api_keys (id, key_hash, name, roles, tenant_id, created_at)
VALUES (
  'key-1',
  '<sha256_hex>',
  'Quick Start Integration',
  'administrator',
  'default',
  datetime('now')
);
```

## Sandbox

For API testing without real robots, use the **sandbox** tenant. It includes demo robots `sandbox-r1` and `sandbox-r2`:

```bash
# Create a sandbox API key (admin)
curl -X POST http://localhost:8080/api/v1/api-keys \
  -H "X-API-Key: <admin-key>" \
  -H "Content-Type: application/json" \
  -d '{"name": "Sandbox Test", "roles": "operator", "tenant_id": "sandbox"}'

# List sandbox robots
curl -H "X-API-Key: <sandbox-key>" \
  "http://localhost:8080/api/v1/robots?tenant_id=sandbox"
```

## Step 2: List Robots

```bash
curl -H "X-API-Key: sk-integration-abc123" \
  http://localhost:8080/api/v1/robots
```

Response:

```json
[
  {
    "id": "r1",
    "vendor": "agibot",
    "model": "X1",
    "status": "online",
    "tenant_id": "default"
  }
]
```

## Step 3: Create a Task

```bash
curl -X POST http://localhost:8080/api/v1/tasks \
  -H "X-API-Key: sk-integration-abc123" \
  -H "Content-Type: application/json" \
  -d '{
    "robot_id": "r1",
    "scenario_id": "patrol",
    "payload": {}
  }'
```

Response:

```json
{
  "id": "task-xyz789",
  "robot_id": "r1",
  "scenario_id": "patrol",
  "status": "pending",
  "created_at": "2025-03-09T12:00:00Z"
}
```

## Step 4: Subscribe to Webhooks (Administrator)

To receive `task_completed` events, create a webhook (requires administrator key):

```bash
curl -X POST http://localhost:8080/api/v1/webhooks \
  -H "X-API-Key: <admin-key>" \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://your-server.com/webhooks/sai-aurosy",
    "events": ["task_completed", "task_started"]
  }'
```

When a task completes, your endpoint receives a POST with the payload described in [Webhooks](webhooks.md).

## Step 5: Stream Telemetry (Optional)

For real-time robot telemetry:

```bash
curl -H "X-API-Key: sk-integration-abc123" \
  -H "Accept: text/event-stream" \
  http://localhost:8080/api/v1/telemetry/stream
```

This returns Server-Sent Events (SSE) with telemetry samples.

## Summary

| Action | Endpoint | Method |
|--------|----------|--------|
| Create API key | `/api/v1/api-keys` | POST |
| List API keys | `/api/v1/api-keys` | GET |
| List robots | `/api/v1/robots` | GET |
| Get robot | `/api/v1/robots/:id` | GET |
| Create task | `/api/v1/tasks` | POST |
| Cancel task | `/api/v1/tasks/:id/cancel` | POST |
| List tasks | `/api/v1/tasks` | GET |
| Telemetry stream | `/api/v1/telemetry/stream` | GET |
| Create webhook | `/api/v1/webhooks` | POST (admin) |

See [API Reference](api-reference.md) for the full endpoint list.
