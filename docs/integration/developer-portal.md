# Developer Portal

SAI AUROSY provides a REST API for fleet management, task execution, and event notifications. This portal links to all resources you need to integrate with the platform.

## Quick Links

| Resource | Description |
|----------|-------------|
| [Quick Start](quickstart.md) | Minimal integration: list robots, create tasks, webhooks |
| [Authentication](authentication.md) | API keys, JWT, OAuth 2.0 |
| [API Reference](api-reference.md) | Endpoint overview and OpenAPI |
| [Webhooks](webhooks.md) | Events, payload schema, HMAC, retry policy |

## Interactive API Docs

- **Swagger UI:** `https://<control-plane-host>/api/docs` or `/swagger/`
- **OpenAPI spec:** `https://<control-plane-host>/openapi.json` or `/api/openapi.json`

Use Swagger UI to explore endpoints and try requests with your API key.

## Sandbox

For testing without real robots, use the **sandbox** tenant. It includes demo robots `sandbox-r1` and `sandbox-r2`. See [Quick Start — Sandbox](quickstart.md#sandbox).

## API Keys Self-Service

Create and manage API keys via the API (no database access required):

- `POST /api/v1/api-keys` — Create key (returns raw key once)
- `GET /api/v1/api-keys` — List keys
- `DELETE /api/v1/api-keys/{id}` — Revoke key

Requires an existing admin or operator key. See [Authentication](authentication.md).

## SDK and Examples

- **Go SDK:** `sdk/go/` — Client, robots, tasks, webhooks
- **Examples:** `examples/integration/` — Python, Go, shell, webhook receiver, telemetry stream

## Base URL

- **Production:** `https://<control-plane-host>/api/v1`
- **Local:** `http://localhost:8080/api/v1`
