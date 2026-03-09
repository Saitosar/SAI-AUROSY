# SAI AUROSY Integration Guide

This guide helps external systems integrate with the SAI AUROSY Control Plane API for fleet management, task execution, and event notifications.

## Overview

SAI AUROSY exposes a REST API for:

- **Fleet management** — list robots, send commands, stream telemetry
- **Task execution** — create and cancel tasks, run workflows
- **Event notifications** — webhooks for robot_online, task_completed, safe_stop, and more
- **Analytics** — robot summaries, uptime, commands count
- **Audit** — audit log for compliance

## Base URL and Versioning

- **Base URL:** `https://<control-plane-host>/api/v1` (or `http://localhost:8080/api/v1` for local development)
- **Versioning:** All endpoints are under `/api/v1`. Future versions will use `/api/v2`, etc. See [API Versioning and Deprecation Policy](api-versioning.md) for when new versions are introduced and how deprecation works.
- **Content-Type:** `application/json` for request and response bodies

## Authentication

The API supports three authentication methods:

| Method | Use case |
|--------|----------|
| **API Key** | Server-to-server integrations, scripts, CI/CD |
| **JWT (Bearer)** | User sessions, Operator Console, OIDC/SAML integrations |
| **OAuth 2.0** | Third-party apps (CRM, ERP, ticketing) — *planned* |

See [Authentication](authentication.md) for details.

## Quick Start

1. Obtain an API key (administrator creates it in the database) or JWT token
2. List robots: `GET /api/v1/robots` with `X-API-Key: <key>` or `Authorization: Bearer <token>`
3. Create a task: `POST /api/v1/tasks` with robot_id, scenario_id, payload
4. Subscribe to webhooks for real-time events

See [Quick Start](quickstart.md) for a step-by-step tutorial.

## Documentation Index

- [Developer Portal](developer-portal.md) — Central hub for all integration resources
- [Quick Start](quickstart.md) — Minimal integration scenario
- [Authentication](authentication.md) — API keys, JWT, OAuth scopes
- [Webhooks](webhooks.md) — Events, payload schema, HMAC verification, retry policy
- [API Reference](api-reference.md) — Endpoint overview and OpenAPI link
- [API Versioning and Deprecation Policy](api-versioning.md) — When /v2 is introduced, deprecation lifecycle

## Multi-Tenant

When using the `operator` role, access is scoped to a single tenant (via API key `tenant_id` or JWT claim `tenant_id`). Administrators can filter by `?tenant_id=` on list endpoints.
