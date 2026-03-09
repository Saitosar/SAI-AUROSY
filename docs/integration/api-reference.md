# API Reference

The SAI AUROSY Control Plane API is a REST API. All endpoints return JSON and require authentication (API key or JWT) unless auth is disabled.

## Versioning and Deprecation

All endpoints are under `/api/v1`. Future breaking changes will introduce `/api/v2`. Deprecated versions include `Deprecation: true` and `Sunset: <date>` response headers. See [API Versioning and Deprecation Policy](api-versioning.md) for details.

## OpenAPI Specification

The full OpenAPI 3.0 specification is available at:

- **JSON:** `GET /api/openapi.json` or `GET /openapi.json`
- **Swagger UI:** `GET /swagger/` (when served)

Use this for code generation, client SDKs, and detailed request/response schemas.

## Idempotency

For `POST /api/v1/robots/{id}/command`, include the `Idempotency-Key` header (UUID or opaque string) to prevent duplicate commands on retry. Keys are valid for 24 hours.

## Endpoint Overview

### Current User

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/me` | Current user info: `{ roles: string[], tenant_id?: string }`. Used by Operator Console for read-only mode. |

### Robots

Robot objects include optional `location` for fleet grouping (e.g. "Warehouse A", "Floor 2"). The Operator Console groups robots by location in the Fleet view.

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/robots` | List robots. Query: `tenant_id` |
| GET | `/api/v1/robots/{id}` | Get robot |
| POST | `/api/v1/robots` | Create robot (admin) |
| PUT | `/api/v1/robots/{id}` | Update robot (admin) |
| DELETE | `/api/v1/robots/{id}` | Delete robot (admin) |
| POST | `/api/v1/robots/{id}/command` | Send command (e.g. safe_stop). Optional `Idempotency-Key` header prevents duplicates on retry. |

### Tasks

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/tasks` | List tasks. Query: `tenant_id`, `robot_id`, `status` |
| GET | `/api/v1/tasks/{id}` | Get task |
| POST | `/api/v1/tasks` | Create task |
| POST | `/api/v1/tasks/{id}/cancel` | Cancel task |

### Scenarios

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/scenarios` | List scenarios |
| GET | `/api/v1/scenarios/{id}` | Get scenario |
| POST | `/api/v1/scenarios` | Create scenario (admin) |
| PUT | `/api/v1/scenarios/{id}` | Update scenario (admin) |
| DELETE | `/api/v1/scenarios/{id}` | Delete scenario (admin) |

### Workflows

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/workflows` | List workflows |
| POST | `/api/v1/workflows/{id}/run` | Run workflow |
| GET | `/api/v1/workflow-runs` | List workflow runs |
| GET | `/api/v1/workflow-runs/{id}` | Get workflow run |

### Zones

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/zones` | List zones |
| GET | `/api/v1/zones/{id}` | Get zone status |

### Tenants

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/tenants` | List tenants |
| GET | `/api/v1/tenants/{id}` | Get tenant |
| GET | `/api/v1/tenants/{id}/robots` | List robots for tenant |
| POST | `/api/v1/tenants` | Create tenant (admin) |
| PUT | `/api/v1/tenants/{id}` | Update tenant (admin) |
| DELETE | `/api/v1/tenants/{id}` | Delete tenant (admin) |

### Webhooks

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/webhooks` | List webhooks (admin) |
| POST | `/api/v1/webhooks` | Create webhook (admin) |
| GET | `/api/v1/webhooks/{id}` | Get webhook (admin) |
| PUT | `/api/v1/webhooks/{id}` | Update webhook (admin) |
| DELETE | `/api/v1/webhooks/{id}` | Delete webhook (admin) |

### Analytics

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/analytics/robots` | List robot analytics summaries. Query: `from`, `to` |
| GET | `/api/v1/analytics/robots/{id}/summary` | Get robot analytics summary |

### Audit

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/audit` | List audit log. Query: `tenant_id`, `robot_id`, `actor`, `action`, `from`, `to`, `limit`, `offset` |

### Edges

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/edges` | List edges |
| GET | `/api/v1/edges/{id}` | Get edge |
| POST | `/api/v1/edges/{id}/heartbeat` | Edge heartbeat (no auth) |

### Telemetry

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/telemetry/stream` | SSE telemetry stream. Query: `robot_id`, `robot_ids`, `tenant_id`. Reconnect: send `Last-Event-ID` header. |

### Events Stream

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/events/stream` | SSE stream of platform events: `robot_online`, `task_completed`, `safe_stop`. Each event: `event: <type>`, `data: { event, timestamp, data }`. |

### Cognitive (AI Services)

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/v1/cognitive/navigate` | Path planning (robot_id, from, to, map_id) |
| POST | `/api/v1/cognitive/recognize` | Object/person recognition (robot_id, sensor_data) |
| POST | `/api/v1/cognitive/plan` | Task planning (task_type, context) |

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
