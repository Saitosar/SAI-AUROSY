# Phase 2.9 — Security: Mandatory Auth and Tenant Isolation

Phase 2.9 enforces mandatory JWT/API key authentication for all API endpoints, strict tenant isolation for robots, tasks, audit, and scenarios, secrets management, and audit for sensitive operations.

## Overview

| Area | Changes |
|------|---------|
| **Auth** | Auth required by default; `ALLOW_UNSAFE_NO_AUTH` for dev only |
| **Secrets** | Vault, AWS Secrets Manager, or env; no JWT/DB credentials in env for production |
| **Audit** | API keys, OAuth clients, tenant changes logged |
| **Edge Heartbeat** | Protected with JWT/API key; `RoleSystem` for edge agents |
| **Tenant Isolation** | Operator must have `tenant_id`; no query override for tenant-scoped data |
| **Scenarios** | Filter by tenant (shared + tenant-specific) |

## Secrets Management

Do not store `JWT_SECRET`, `REGISTRY_DB_DSN`, or other sensitive values in environment variables for production. Use Vault, AWS Secrets Manager, or an equivalent.

| Variable | Description |
|----------|-------------|
| `SECRETS_PROVIDER` | `env` (default), `vault`, or `aws` |
| `VAULT_ADDR` | Vault server URL (when provider=vault) |
| `VAULT_TOKEN` | Vault token |
| `VAULT_SECRET_PATH` | KV v2 path (default: `sai-aurosy`) |
| `AWS_REGION` | Region for Secrets Manager |
| `AWS_SECRET_NAME` | Secret name/ARN (default: `sai-aurosy`) |

See [Secrets Management](secrets-management.md) for setup and migration.

## Audit for Sensitive Operations

The audit log records the following sensitive operations:

| Resource | Actions | Notes |
|----------|---------|-------|
| `api_key` | create, delete | name, roles, tenant_id in details; raw key never logged |
| `tenant` | create, update, delete | Admin only |
| `oauth_client` | create, update, delete | Admin only; client_secret never logged |
| `robot` | create, update, delete | Existing |
| `command` | send | Existing |
| `task` | cancel | Existing |

## Mandatory Authentication

### Startup

- If no auth is configured (`JWT_SECRET`, `JWT_PUBLIC_KEY`, `api_keys` table, OAuth) and `AUTH_REQUIRED` is effective → Control Plane fails to start.
- Set `ALLOW_UNSAFE_NO_AUTH=true` for local development without auth.

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `AUTH_REQUIRED` | `true` | Fail startup when auth not configured |
| `ALLOW_UNSAFE_NO_AUTH` | (unset) | Allow no-auth when auth not configured (dev only) |

### Protected Routes

All `/v1/*` endpoints require authentication (JWT, API key, or OAuth). Exceptions:

- `/health`, `/ready` — public (load balancers)
- `/oauth/*` — public (OAuth flow)

`/v1/edges/{id}/heartbeat` now requires auth with role `operator`, `administrator`, or `system`.

## Edge Agent

Edge agents must send `X-API-Key` or `Authorization: Bearer` when auth is enabled.

Configure:

```bash
export EDGE_API_KEY=sk-...
```

Create API keys with role `system` for edge agents.

## Tenant Isolation

### Operator

- `tenant_id` **required** in JWT or API key. If missing → 403.
- `tenant_id` from claims only; query param `tenant_id` is ignored for operators.

### Administrator

- May filter by `?tenant_id=` or omit for all tenants.

### Enforced Resources

| Resource | Operator | Admin |
|----------|----------|-------|
| Robots | `ListByTenant(claims.TenantID)` | `?tenant_id=` or all |
| Tasks | `TenantID` from claims | `?tenant_id=` or all |
| Audit | `TenantID` from claims only | `?tenant_id=` or all |
| Workflow runs | `TenantID` from claims | `?tenant_id=` or all |
| Edges | Filter by robots in tenant | `?tenant_id=` or all |
| Scenarios | Filter by tenant (shared + tenant-specific) | `?tenant_id=` or all |

### Scenarios

- Scenarios with `tenant_id IS NULL` are shared.
- Scenarios with `tenant_id = X` are tenant-specific.
- Operator sees shared + own tenant scenarios.
- `ListByTenant` and `GetByTenant` in store; `ListForTenant` and `GetForTenant` in Catalog.

## Migration

- **Edge agents**: Must set `EDGE_API_KEY` before deploying auth-required Control Plane.
- **Operator tokens**: Must include `tenant_id` claim.
- **Existing deployments**: Set `JWT_SECRET` or `JWT_PUBLIC_KEY` or use DB with `api_keys`; otherwise set `ALLOW_UNSAFE_NO_AUTH=true` for dev.

## References

- [Secrets Management](secrets-management.md)
- [Authentication](../integration/authentication.md)
- [Phase 2.6 Multi-Tenant](phase-2.6-multi-tenant.md)
- [Platform Architecture](../architecture/platform-architecture.md)
