# Phase 3.1 — Streaming Gateway

## Overview

Extensions to the SSE telemetry stream for filtering, reconnect, and backpressure.

## Implemented Features

### 1. Filter by robot_id

Query parameters:

- `robot_id` — single robot (e.g. `?robot_id=x1-001`)
- `robot_ids` — multiple robots (e.g. `?robot_ids=x1-001,x1-002`)

When specified, only telemetry for those robots is streamed. Tenant isolation applies: operator must have access to the robot.

### 2. Reconnect (Last-Event-ID)

SSE events include an `id` field (RFC3339Nano timestamp). On reconnect, the client sends `Last-Event-ID: <id>` in the request header. The server replays buffered events newer than that ID, then continues live streaming.

- Buffer: in-memory ring buffer (1000 events)
- Buffer is populated by a background goroutine subscribing to all telemetry

### 3. Backpressure

Per-stream channel capacity: 1000 events. When full, oldest events are dropped. Metric: `streaming_gateway_dropped_events_total`.

## API

```
GET /v1/telemetry/stream?robot_id=x1-001
GET /v1/telemetry/stream?robot_ids=x1-001,x1-002
GET /v1/telemetry/stream?tenant_id=default
```

Headers (client):

- `Last-Event-ID` — optional; for reconnect

## Implementation

- `pkg/telemetry/bus.go` — `SubscribeTelemetryMultiple`
- `pkg/control-plane/streaming/buffer.go` — ring buffer, Prometheus counter
- `pkg/control-plane/api/server.go` — `telemetryStream` handler
- `cmd/control-plane/main.go` — stream buffer creation and population

## Related Documents

- [Platform Architecture](../architecture/platform-architecture.md)
- [Phase 3.2 Cognitive Gateway](phase-3.2-cognitive-gateway.md)
