# Phase 3.2 — Cognitive Gateway

## Overview

Cognitive Gateway provides an abstraction for AI services (navigation, recognition, planning) used by scenarios and tasks.

## Implemented Features

### 1. Gateway Interface

`pkg/control-plane/cognitive/gateway.go`:

- `Navigate(ctx, req) (*NavigateResult, error)` — path planning
- `Recognize(ctx, req) (*RecognizeResult, error)` — object/person recognition
- `Plan(ctx, req) (*PlanResult, error)` — task planning

### 2. Mock Provider

`MockGateway` returns empty or mock results. Used by default when no external AI is configured. Does not change existing scenario behavior.

### 3. HTTP Provider

`HTTPGateway` calls external AI services via REST. Configure URLs per capability; add new AI backends without control-plane code changes.

| Env var | Purpose |
|---------|---------|
| `COGNITIVE_HTTP_NAV_URL` | Navigation service base URL |
| `COGNITIVE_HTTP_RECOGNIZE_URL` | Recognition service base URL |
| `COGNITIVE_HTTP_PLAN_URL` | Planning service base URL |
| `COGNITIVE_HTTP_API_KEY` | Optional API key (Bearer header) |

### 4. Cocoon Provider

`CocoonGateway` routes UnderstandIntent and Plan to [Cocoon](https://github.com/TelegramMessenger/cocoon) (TEE-isolated LLM inference). Navigate, Recognize, Transcribe, Synthesize use mock. See [Cocoon Integration](../architecture/cocoon-integration.md).

| Env var | Purpose |
|---------|---------|
| `COGNITIVE_COCOON_CLIENT_URL` | Cocoon client base URL (default `http://localhost:10000`) |
| `COGNITIVE_COCOON_MODEL` | Model name (default `Qwen/Qwen3-32B`) |
| `COGNITIVE_COCOON_TIMEOUT_SEC` | Request timeout (default 30) |
| `COGNITIVE_COCOON_MAX_TOKENS` | Max tokens per request (default 512) |

### 5. Provider Registry

- `pkg/control-plane/cognitive/registry.go` — `Register(name, factory)`, `NewGateway(cfg)`
- Providers register in `init()`. Adding a new built-in provider: implement `Gateway`, add file, call `Register` — no changes to `main.go`

### 6. REST API

- `POST /v1/cognitive/navigate` — request body: `{ "robot_id", "from", "to", "map_id" }`
- `POST /v1/cognitive/recognize` — request body: `{ "robot_id", "sensor_data" }`
- `POST /v1/cognitive/plan` — request body: `{ "task_type", "context" }`

All endpoints require authentication and tenant isolation.

### 7. Integration

- `cmd/control-plane/main.go` — loads config via `LoadConfig()`, creates gateway via `NewGateway(cfg)`
- Task Engine can optionally call `Gateway.Navigate()` for `navigation` scenario (future enhancement)

## Implementation

- `pkg/control-plane/cognitive/` — gateway, types, providers, config, registry, http_provider, cocoon_provider
- `pkg/control-plane/api/server.go` — cognitive handlers
- `cmd/control-plane/main.go` — gateway wiring via LoadConfig + NewGateway

## Future Work

- Scenario integration: `navigation` scenario invokes `Navigate()` when provider is configured
- Per-tenant provider selection

## Related Documents

- [Cognitive Gateway Architecture](../architecture/cognitive-gateway.md)
- [Cocoon Integration](../architecture/cocoon-integration.md)
- [Phase 3.1 Streaming Gateway](phase-3.1-streaming-gateway.md)
