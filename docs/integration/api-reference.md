# API Reference

The SAI AUROSY Control Plane API is a REST API. All endpoints return JSON and require authentication (API key or JWT) unless auth is disabled.

## Versioning and Deprecation

All endpoints are under `/v1`. When accessing Control Plane directly (e.g. `http://localhost:8080`), use base path `/v1`. When using Operator Console proxy or a reverse proxy that adds `/api`, the full path is `/api/v1` (proxied to `/v1`). Future breaking changes will introduce `/v2`. Deprecated versions include `Deprecation: true` and `Sunset: <date>` response headers. See [API Versioning and Deprecation Policy](api-versioning.md) for details.

## OpenAPI Specification

The full OpenAPI 3.0 specification is available at:

- **JSON:** `GET /api/openapi.json` or `GET /openapi.json`
- **Swagger UI:** `GET /swagger/` (when served)

Use this for code generation, client SDKs, and detailed request/response schemas.

## Idempotency

For `POST /v1/robots/{id}/command`, include the `Idempotency-Key` header (UUID or opaque string) to prevent duplicate commands on retry. Keys are valid for 24 hours.

## Endpoint Overview

### Current User

| Method | Path | Description |
|--------|------|-------------|
| GET | `/v1/me` | Current user info: `{ roles: string[], tenant_id?: string }`. Used by Operator Console for read-only mode. |

### Robots

Robot objects include optional `location` for fleet grouping (e.g. "Warehouse A", "Floor 2"). The Operator Console groups robots by location in the Fleet view.

| Method | Path | Description |
|--------|------|-------------|
| GET | `/v1/robots` | List robots. Query: `tenant_id` |
| GET | `/v1/robots/{id}` | Get robot |
| POST | `/v1/robots` | Create robot (admin) |
| PUT | `/v1/robots/{id}` | Update robot (admin) |
| DELETE | `/v1/robots/{id}` | Delete robot (admin) |
| POST | `/v1/robots/{id}/command` | Send command (e.g. safe_stop). Optional `Idempotency-Key` header prevents duplicates on retry. |

### Tasks

| Method | Path | Description |
|--------|------|-------------|
| GET | `/v1/tasks` | List tasks. Query: `tenant_id`, `robot_id`, `status` |
| GET | `/v1/tasks/{id}` | Get task |
| POST | `/v1/tasks` | Create task |
| POST | `/v1/tasks/{id}/cancel` | Cancel task |

### Scenarios

| Method | Path | Description |
|--------|------|-------------|
| GET | `/v1/scenarios` | List scenarios |
| GET | `/v1/scenarios/{id}` | Get scenario |
| POST | `/v1/scenarios` | Create scenario (admin) |
| PUT | `/v1/scenarios/{id}` | Update scenario (admin) |
| DELETE | `/v1/scenarios/{id}` | Delete scenario (admin) |
| POST | `/v1/scenarios/mall_assistant/start` | Start Mall Assistant scenario |
| POST | `/v1/scenarios/mall_assistant/visitor-request` | Submit visitor request (store name) to Mall Assistant |

### Workflows

| Method | Path | Description |
|--------|------|-------------|
| GET | `/v1/workflows` | List workflows |
| POST | `/v1/workflows/{id}/run` | Run workflow |
| GET | `/v1/workflow-runs` | List workflow runs |
| GET | `/v1/workflow-runs/{id}` | Get workflow run |

### Zones

| Method | Path | Description |
|--------|------|-------------|
| GET | `/v1/zones` | List zones |
| GET | `/v1/zones/{id}` | Get zone status |

### Tenants

| Method | Path | Description |
|--------|------|-------------|
| GET | `/v1/tenants` | List tenants |
| GET | `/v1/tenants/{id}` | Get tenant |
| GET | `/v1/tenants/{id}/robots` | List robots for tenant |
| POST | `/v1/tenants` | Create tenant (admin) |
| PUT | `/v1/tenants/{id}` | Update tenant (admin) |
| DELETE | `/v1/tenants/{id}` | Delete tenant (admin) |

### Webhooks

| Method | Path | Description |
|--------|------|-------------|
| GET | `/v1/webhooks` | List webhooks (admin) |
| POST | `/v1/webhooks` | Create webhook (admin) |
| GET | `/v1/webhooks/{id}` | Get webhook (admin) |
| PUT | `/v1/webhooks/{id}` | Update webhook (admin) |
| DELETE | `/v1/webhooks/{id}` | Delete webhook (admin) |

### Analytics

| Method | Path | Description |
|--------|------|-------------|
| GET | `/v1/analytics/robots` | List robot analytics summaries. Query: `from`, `to` |
| GET | `/v1/analytics/robots/{id}/summary` | Get robot analytics summary |

### Audit

| Method | Path | Description |
|--------|------|-------------|
| GET | `/v1/audit` | List audit log. Query: `tenant_id`, `robot_id`, `actor`, `action`, `from`, `to`, `limit`, `offset` |

### Edges

| Method | Path | Description |
|--------|------|-------------|
| GET | `/v1/edges` | List edges |
| GET | `/v1/edges/{id}` | Get edge |
| POST | `/v1/edges/{id}/heartbeat` | Edge heartbeat (no auth) |

### Telemetry

| Method | Path | Description |
|--------|------|-------------|
| GET | `/v1/telemetry/stream` | SSE telemetry stream. Query: `robot_id`, `robot_ids`, `tenant_id`. Reconnect: send `Last-Event-ID` header. |

### Events Stream

| Method | Path | Description |
|--------|------|-------------|
| GET | `/v1/events/stream` | SSE stream of platform events: `robot_online`, `task_completed`, `safe_stop`. Each event: `event: <type>`, `data: { event, timestamp, data }`. |

### Cognitive (AI Services)

| Method | Path | Description |
|--------|------|-------------|
| POST | `/v1/cognitive/navigate` | Path planning (robot_id, from, to, map_id) |
| POST | `/v1/cognitive/recognize` | Object/person recognition (robot_id, sensor_data) |
| POST | `/v1/cognitive/plan` | Task planning (task_type, context) |
| POST | `/v1/cognitive/transcribe` | Speech-to-text (robot_id, audio_base64, language) |
| POST | `/v1/cognitive/synthesize` | Text-to-speech (robot_id, text, language) |
| POST | `/v1/cognitive/understand-intent` | Intent extraction from user text (robot_id, text, language, context) |

## Health and Metrics

| Path | Description |
|------|-------------|
| GET `/health` | Liveness probe |
| GET `/ready` | Readiness (NATS, DB) |
| GET `/metrics` | Prometheus metrics |

## Roles and Read-Only Mode

- **viewer:** Read-only access. Can list robots, tasks, telemetry, audit, etc. Cannot send commands, create tasks, or modify resources.
- **operator:** Full access within tenant. Can send commands, create tasks, run workflows.
- **administrator:** Full access across all tenants.

The Operator Console fetches `GET /v1/me` to determine if the user has write access. When the user has only the `viewer` role, the console hides all command buttons and write actions.

## Multi-Tenant Filtering

- **Operator:** Automatically filtered by `tenant_id` from API key or JWT. Query param `tenant_id` is ignored.
- **Administrator:** Can pass `?tenant_id=<id>` on list endpoints to filter.
