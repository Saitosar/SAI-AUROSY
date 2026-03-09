# Phase 3.3 — Developer Platform

Phase 3.3 implements the Developer Platform: API keys self-service, sandbox tenant, and developer documentation.

## Overview

| Area | Changes |
|------|---------|
| **API Keys** | REST API: POST/GET/DELETE `/v1/api-keys` — create, list, revoke keys |
| **Sandbox** | Tenant `sandbox` with demo robots `sandbox-r1`, `sandbox-r2` |
| **Docs** | Swagger UI at `/api/docs`, developer-portal.md, quickstart update |

## API Keys Self-Service

### Endpoints

| Method | Path | Description |
|--------|------|-------------|
| POST | `/v1/api-keys` | Create key; returns `{id, key, name, roles, tenant_id}` — raw `key` shown once |
| GET | `/v1/api-keys` | List keys (admin: all; operator: own tenant) |
| DELETE | `/v1/api-keys/{id}` | Revoke key |

### RBAC

- **Administrator:** Create/list/delete any key, any tenant
- **Operator:** Create/list/delete only keys for own tenant

### Key Format

Keys are generated as `sk-` + 32 hex chars. Stored as SHA256 hash. Raw key returned only in POST response.

## Sandbox

Migration `000017_seed_sandbox.up.sql` creates:

- Tenant `sandbox`
- Robots `sandbox-r1` (agibot X1), `sandbox-r2` (unitree Go2)

Use for API testing without real hardware. See [Quick Start — Sandbox](../integration/quickstart.md#sandbox).

## Developer Documentation

- **Swagger UI:** `/api/docs` or `/swagger/`
- **OpenAPI spec:** `/openapi.json` or `/api/openapi.json`
- **Developer portal:** [docs/integration/developer-portal.md](../integration/developer-portal.md)

## Links

- [Phase 2.7 Enterprise Integration](phase-2.7-enterprise-integration.md)
- [Phase 3.4 Marketplace](phase-3.4-marketplace.md)
- [Roadmap](../product/roadmap.md)
