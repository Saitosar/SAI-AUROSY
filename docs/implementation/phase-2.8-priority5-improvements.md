# Phase 2.8 — Priority 5: Improvements and Polish

Phase 2.8 addresses polish and cleanup tasks for the v1 API and Operator Console.

## Overview

| Task | Description | Status |
|------|-------------|--------|
| **E2E for v1 API** | `e2e.sh` uses `/v1/` with auth support | Done |
| **Auth in Operator Console** | JWT/API key when calling API | Pending |
| **Remove legacy routes** | Deprecate and remove `/robots`, `/robots/:id` without `/v1/` | Done |
| **Fleet grouping** | Group robots by locations/zones for Fleet Management | Pending |

## 1. E2E for v1 API (Done)

### Current state
- `scripts/e2e.sh` uses `API_PREFIX=/v1` and `$API$API_PREFIX/robots`, `$API$API_PREFIX/robots/:id/command`, etc.
- Supports `E2E_API_KEY` env var; passes `X-API-Key` header when set
- Works with or without auth (ALLOW_UNSAFE_NO_AUTH for dev)

### Implementation (completed)
- `API_URL` default `http://localhost:8080`
- `API_PREFIX` default `/v1`
- `$API$API_PREFIX/robots` for GET, `$API$API_PREFIX/robots/:id/command` for POST
- `X-API-Key: $E2E_API_KEY` when set

## 2. Auth in Operator Console

### Current state
- All `fetch()` calls use `API_BASE = '/api/v1'` without auth headers
- When JWT/API keys are configured, v1 API returns 401

### Changes
- Add auth context: API key or JWT from env or login
- Pass `Authorization: Bearer <token>` or `X-API-Key: <key>` on all API calls
- Support `VITE_API_KEY` env var for development
- Optional: login UI for JWT (future phase)

### Implementation
- Create `apiClient` helper that adds auth headers to fetch
- Read `import.meta.env.VITE_API_KEY` when set
- Use `credentials: 'include'` for cookie-based auth (if added later)

## 3. Remove Legacy Routes (Done)

### Current state
- Legacy routes removed. All API routes are under `/v1` only.
- `server.go` registers routes exclusively via `v1 := r.PathPrefix("/v1").Subrouter()`.
- No legacy routes without `/v1` prefix exist.

### Changes (completed)
- Legacy route registrations removed from `server.go`
- All clients use `/v1/` paths: e2e.sh, Operator Console, examples

## 4. Fleet Grouping

### Current state
- `hal.Robot` has: ID, Vendor, Model, TenantID, EdgeID, Capabilities
- No `location` or `zone` for grouping
- Coordinator zones (A, B, C) are task locks, not robot locations

### Changes
- Add optional `location` field to Robot for display grouping (e.g. "Warehouse A", "Floor 2")
- API: `GET /v1/robots?group_by=location` or client-side grouping
- Operator Console: group robots by location in Fleet panel; show "Location" when present

### Implementation
- Add `Location string` to `hal.Robot` (optional, JSON: `location`)
- Migration: `ALTER TABLE robots ADD COLUMN location` (nullable)
- API: include location in response; optional `?group_by=location` or client groups
- Console: group by `robot.location || 'Unassigned'` in Fleet view

## Environment Variables

| Variable | Description |
|----------|-------------|
| `VITE_API_KEY` | API key for Operator Console (when auth is required) |
| `E2E_API_KEY` | API key for e2e.sh (when auth is required) |
| `API_PREFIX` | API path prefix for e2e.sh (default: `/v1`) |

## References

- [Phase 2.1 Control Plane](phase-2.1-control-plane.md) — JWT, API keys, rate limiting
- [Phase 2.3 Multi-Robot](phase-2.3-multi-robot.md) — Zones, Coordinator
- [API Reference](../integration/api-reference.md)
- [Authentication](../integration/authentication.md)
