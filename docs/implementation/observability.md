# Observability

This document describes the observability stack for SAI AUROSY Control Plane: distributed tracing (OpenTelemetry), structured logging, and log-trace correlation.

## Distributed Tracing (OpenTelemetry)

The Control Plane supports OpenTelemetry for distributed tracing. When configured, traces are exported via OTLP HTTP to a collector (e.g. Jaeger, Tempo, or a vendor backend).

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `OTEL_EXPORTER_OTLP_ENDPOINT` | OTLP HTTP endpoint (e.g. `http://localhost:4318` or `localhost:4318`) | — (tracing disabled) |
| `OTEL_SERVICE_NAME` | Service name for traces | `sai-aurosy-control-plane` |

### Behavior

- If `OTEL_EXPORTER_OTLP_ENDPOINT` is empty, tracing is disabled (no-op). The default no-op tracer is used.
- When enabled, the tracer exports spans to the configured endpoint. W3C Trace Context is propagated in HTTP headers.
- Instrumented paths: HTTP requests (via middleware), `workflow.run`, `task.run`, `webhook.dispatch`.

### Instrumented Spans

| Span Name | Location | Attributes |
|-----------|----------|------------|
| HTTP request | `observability.TracingMiddleware` | `http.method`, `http.route`, `http.status_code` |
| `workflow.run` | `orchestration.Runner.Run` | `workflow_id`, `operator_id`, `tenant_id` |
| `task.run` | `tasks.Runner.runOne` | `task_id`, `robot_id`, `scenario_id` |
| `webhook.dispatch` | `webhooks.Dispatcher.Dispatch` | `event` |

## Structured Logging

Logging uses Go's `log/slog` with optional JSON output for production.

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `LOG_FORMAT` | `json` for structured JSON; otherwise text | text |
| `LOG_LEVEL` | `debug`, `info`, `warn`, `error` | `info` |

### Log-Trace Correlation

When OpenTelemetry tracing is active, request logs include `trace_id` and `span_id` from the active span context. This allows correlating logs with traces in observability platforms.

Example JSON log line:

```json
{"time":"2025-03-09T12:00:00Z","level":"INFO","msg":"request","request_id":"a1b2c3d4","method":"GET","path":"/v1/robots","status":200,"duration_ms":5,"trace_id":"4bf92f3577b34da6a3ce929d0e0e4736","span_id":"00f067aa0ba902b7"}
```

## Related Documents

- [Production Runbook](../operations/production-runbook.md) — Prometheus metrics, alerts, recovery
- [Phase 2.1 Control Plane](phase-2.1-control-plane.md) — LOG_FORMAT, metrics, health
