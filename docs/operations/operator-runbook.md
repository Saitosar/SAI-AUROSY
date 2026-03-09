# Operator Runbook

Step-by-step procedures for common operator scenarios: tenant onboarding, workflow creation, and troubleshooting.

**Prerequisites:** Administrator API key or JWT with `administrator` role. Base URL: `$CONTROL_PLANE_URL` (e.g. `http://localhost:8080`). API paths use `/v1/` prefix.

---

## 1. Tenant Onboarding

Onboard a new tenant: create tenant, API key, and optionally register robots.

### 1.1 Create Tenant

```bash
curl -X POST "$CONTROL_PLANE_URL/v1/tenants" \
  -H "X-API-Key: $ADMIN_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "id": "acme-corp",
    "name": "ACME Corporation",
    "config": {}
  }'
```

**Response:** `201 Created` with tenant object.

**Validation:** `id` must be unique, alphanumeric with hyphens. If tenant exists, you get `409 Conflict`.

### 1.2 Create API Key for Tenant

Create an operator key scoped to the tenant:

```bash
curl -X POST "$CONTROL_PLANE_URL/v1/api-keys" \
  -H "X-API-Key: $ADMIN_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "ACME Operator Key",
    "roles": "operator",
    "tenant_id": "acme-corp"
  }'
```

**Response:** `201 Created` with `key` (raw secret). **Store the key securely; it is shown only once.**

For administrator access across tenants, use `"roles": "administrator"` and omit or set `tenant_id` as needed.

### 1.3 Register Robot for Tenant

```bash
curl -X POST "$CONTROL_PLANE_URL/v1/robots" \
  -H "X-API-Key: $ADMIN_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "id": "acme-r1",
    "vendor": "agibot",
    "model": "X1",
    "tenant_id": "acme-corp",
    "location": "Warehouse A"
  }'
```

**Validation:** `tenant_id` must exist. Robot `id` must be unique.

### 1.4 Verify Onboarding

```bash
# List tenants
curl -H "X-API-Key: $ADMIN_KEY" "$CONTROL_PLANE_URL/v1/tenants"

# List robots for tenant
curl -H "X-API-Key: $ADMIN_KEY" "$CONTROL_PLANE_URL/v1/tenants/acme-corp/robots"

# Test operator key (scoped to acme-corp)
curl -H "X-API-Key: $TENANT_OPERATOR_KEY" "$CONTROL_PLANE_URL/v1/robots"
```

Operator sees only robots in their tenant. Administrator can pass `?tenant_id=acme-corp` to filter.

### Checklist: Tenant Onboarding

| Step | Action | Role |
|------|--------|------|
| 1 | Create tenant via `POST /v1/tenants` | administrator |
| 2 | Create API key with `tenant_id` via `POST /v1/api-keys` | administrator |
| 3 | Register robots with `tenant_id` via `POST /v1/robots` | administrator |
| 4 | Share operator key with tenant (securely) | — |
| 5 | Configure edge agent with `EDGE_API_KEY` (system key) if needed | administrator |

---

## 2. Workflow Creation and Execution

Workflows are multi-robot orchestration definitions. Built-in workflows are defined in the catalog; scenarios can be created via API.

### 2.1 List Available Workflows

```bash
curl -H "X-API-Key: $API_KEY" "$CONTROL_PLANE_URL/v1/workflows"
```

**Built-in example:** `patrol_zones_ABC` — 3 robots patrol zones A, B, C.

### 2.2 Create Scenario (Admin)

Scenarios define task steps (commands, payload, duration). Create custom scenarios for workflows:

```bash
curl -X POST "$CONTROL_PLANE_URL/v1/scenarios" \
  -H "X-API-Key: $ADMIN_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "id": "custom_patrol",
    "name": "Custom Patrol",
    "description": "Patrol with custom duration",
    "steps": [
      {"command": "walk_mode", "payload": null, "duration_sec": 0},
      {"command": "cmd_vel", "payload": {"linear_x": 0.3, "linear_y": 0, "angular_z": 0}, "duration_sec": -1},
      {"command": "cmd_vel", "payload": {"linear_x": 0, "linear_y": 0, "angular_z": 0}, "duration_sec": 0}
    ],
    "required_capabilities": ["walk", "cmd_vel", "patrol"]
  }'
```

**Step format:** `{ "command": string, "payload": object|null, "duration_sec": number }`. Use `duration_sec: -1` for payload-driven duration.

**Built-in scenarios:** `standby`, `patrol`, `navigation`.

### 2.3 Run Workflow

```bash
curl -X POST "$CONTROL_PLANE_URL/v1/workflows/patrol_zones_ABC/run" \
  -H "X-API-Key: $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{}'
```

**Response:** `201 Created` with `WorkflowRun` (id, status, tasks).

**Tenant scope:** Operator runs use only robots from their tenant. Administrator can run for any tenant (tenant inferred from context or request).

### 2.4 Check Workflow Run Status

```bash
# List runs
curl -H "X-API-Key: $API_KEY" "$CONTROL_PLANE_URL/v1/workflow-runs"

# Get specific run
curl -H "X-API-Key: $API_KEY" "$CONTROL_PLANE_URL/v1/workflow-runs/{run_id}"
```

**Statuses:** `pending`, `running`, `completed`, `failed`, `cancelled`.

### 2.5 Create Single Task (Alternative to Workflow)

For a single robot:

```bash
curl -X POST "$CONTROL_PLANE_URL/v1/tasks" \
  -H "X-API-Key: $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "robot_id": "acme-r1",
    "scenario_id": "patrol",
    "payload": {"duration_sec": 60}
  }'
```

### Checklist: Workflow Creation

| Step | Action | Role |
|------|--------|------|
| 1 | List workflows: `GET /v1/workflows` | operator |
| 2 | (Optional) Create scenario: `POST /v1/scenarios` | administrator |
| 3 | Ensure robots exist with required capabilities | administrator |
| 4 | Run workflow: `POST /v1/workflows/{id}/run` | operator |
| 5 | Monitor: `GET /v1/workflow-runs`, `GET /v1/workflow-runs/{id}` | operator |

**Note:** Adding new workflow definitions (beyond built-in catalog) requires code changes in `pkg/control-plane/orchestration/catalog.go`. Scenarios are created via API.

---

## 3. Troubleshooting

### 3.1 Authentication and Authorization

| Symptom | Cause | Resolution |
|---------|-------|-------------|
| `401 Unauthorized` | Missing or invalid API key/JWT | Check `X-API-Key` or `Authorization: Bearer` header; verify key exists and is correct |
| `403 Forbidden` | Valid auth but insufficient permissions | Operator accessing another tenant's resource; use correct `tenant_id` in key or admin key |
| `404 Not Found` | Resource not in tenant scope | Robot/task belongs to different tenant; operator cannot access |
| Startup fails: "Auth required but..." | Auth required but not configured | Set `JWT_SECRET` or `JWT_PUBLIC_KEY`; or `ALLOW_UNSAFE_NO_AUTH=true` (dev only) |

### 3.2 Tenant and Robot Issues

| Symptom | Cause | Resolution |
|---------|-------|-------------|
| `createRobot` returns 404 for tenant | Tenant does not exist | Create tenant first via `POST /v1/tenants` |
| Operator sees no robots | Wrong tenant_id in API key | Ensure API key has correct `tenant_id`; list with `GET /v1/tenants/{id}/robots` |
| `createTask` returns 403 | Robot not in operator's tenant | Verify `robot.tenant_id` matches API key `tenant_id` |
| `sendCommand` returns 404 | Robot not found or wrong tenant | Check robot exists and tenant scope |

### 3.3 Task and Workflow Issues

| Symptom | Cause | Resolution |
|---------|-------|-------------|
| Task stuck in `pending` | No edge agent connected for robot | Ensure edge agent is running, `EDGE_API_KEY` set, robot online |
| Workflow run fails | Not enough robots with required capabilities | Check `required_capabilities` of scenario; ensure robots have matching capabilities |
| `runWorkflow` returns error | Workflow ID invalid or no matching robots | List workflows; verify tenant has robots for workflow steps (zone, capabilities) |
| Scenario not found | Wrong scenario_id or typo | List scenarios: `GET /v1/scenarios` |

### 3.4 Infrastructure

| Symptom | Cause | Resolution |
|---------|-------|-------------|
| `/ready` returns 503 "nats disconnected" | NATS unreachable | Check `NATS_URL`, ensure NATS is running, verify network |
| `/ready` returns 503 "database unavailable" | DB unreachable | Check `REGISTRY_DB_DSN`, credentials, DB connectivity |
| Migration errors on startup | Schema mismatch | Check migrations in `pkg/control-plane/registry/migrations/`; restore DB backup if needed |
| High `streaming_gateway_dropped_events_total` | SSE clients slow | Normal under load; consider tuning buffer or client reconnect logic |

### 3.5 Idempotency (Commands)

For `POST /v1/robots/{id}/command`, include `Idempotency-Key` header (UUID or opaque string) to prevent duplicate commands on retry. Keys are valid for 24 hours.

```bash
curl -X POST "$CONTROL_PLANE_URL/v1/robots/acme-r1/command" \
  -H "X-API-Key: $API_KEY" \
  -H "Idempotency-Key: $(uuidgen)" \
  -H "Content-Type: application/json" \
  -d '{"command": "safe_stop"}'
```

### 3.6 Audit and Debugging

```bash
# Audit log (filter by tenant, robot, action)
curl -H "X-API-Key: $ADMIN_KEY" \
  "$CONTROL_PLANE_URL/v1/audit?tenant_id=acme-corp&limit=50"

# Current user/roles (for debugging auth)
curl -H "X-API-Key: $API_KEY" "$CONTROL_PLANE_URL/v1/me"
```

---

## Related Documents

- [Production Runbook](production-runbook.md) — Deployment, monitoring, recovery
- [API Reference](../integration/api-reference.md) — Full endpoint list
- [Authentication](../integration/authentication.md) — API keys, JWT, OAuth
- [Quick Start](../integration/quickstart.md) — Minimal integration guide
- [Phase 2.6 Multi-Tenant](../implementation/phase-2.6-multi-tenant.md) — Tenant model and access control
