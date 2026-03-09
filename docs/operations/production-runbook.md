# Production Runbook

Deployment, monitoring, alerts, and recovery procedures for SAI AUROSY in production.

## Prerequisites

- Docker and Docker Compose (or Kubernetes)
- PostgreSQL (recommended for production; SQLite for dev only)
- NATS (event broker / telemetry bus)
- Go 1.21+ (if building from source)

## Deployment

### Docker Compose (Production)

Use persistent volumes for database and configure production environment variables:

```yaml
# docker-compose.prod.yml (example)
services:
  nats:
    image: nats:2-alpine
    ports:
      - "4222:4222"
      - "8222:8222"
    command: ["-m", "8222"]
    restart: unless-stopped

  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_USER: saiaurosy
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD}
      POSTGRES_DB: saiaurosy
    volumes:
      - postgres_data:/var/lib/postgresql/data
    restart: unless-stopped

  control-plane:
    build:
      context: .
      dockerfile: Dockerfile.control-plane
    ports:
      - "8080:8080"
    environment:
      NATS_URL: nats://nats:4222
      REGISTRY_DB_DRIVER: postgres
      REGISTRY_DB_DSN: postgres://saiaurosy:${POSTGRES_PASSWORD}@postgres:5432/saiaurosy?sslmode=disable
      JWT_SECRET: ${JWT_SECRET}
      JWT_ISSUER: sai-aurosy
      LOG_FORMAT: json
      # For production: use SECRETS_PROVIDER=vault or aws; see docs/implementation/secrets-management.md
    depends_on:
      - nats
      - postgres
    restart: unless-stopped

  operator-console:
    build:
      context: .
      dockerfile: Dockerfile.console
    ports:
      - "3000:80"
    depends_on:
      - control-plane
    restart: unless-stopped

volumes:
  postgres_data:
```

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `NATS_URL` | NATS connection URL | `nats://localhost:4222` |
| `CONTROL_PLANE_ADDR` | HTTP listen address | `:8080` |
| `REGISTRY_DB_DRIVER` | `sqlite` or `postgres` | (in-memory if unset) |
| `REGISTRY_DB_DSN` | Database connection string | — |
| `JWT_SECRET` | HMAC JWT secret | — |
| `JWT_PUBLIC_KEY` | PEM public key (RS256) | — |
| `JWT_ISSUER` | JWT issuer claim | — |
| `JWT_AUDIENCE` | JWT audience claim | — |
| `CORS_ORIGINS` | Allowed CORS origins (comma-separated) | `*` |
| `RATE_LIMIT_RPS` | Requests per second per IP | 100 |
| `RATE_LIMIT_BURST` | Rate limit burst | 200 |
| `LOG_FORMAT` | `json` for structured logging | — |
| `LOG_LEVEL` | `debug`, `info`, `warn`, `error` | `info` |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | OTLP HTTP endpoint for tracing (e.g. `http://localhost:4318`) | — (disabled) |
| `OTEL_SERVICE_NAME` | Service name for traces | `sai-aurosy-control-plane` |
| `AUTH_REQUIRED` | Require auth for /v1/ | `true` |
| `TELEMETRY_RETENTION_DAYS` | Days to retain telemetry | 30 |
| `SECRETS_PROVIDER` | `env`, `vault`, or `aws` | `env` |
| `SHUTDOWN_TIMEOUT` | HTTP drain timeout in seconds (Control Plane) | 30 |
| `SHUTDOWN_GRACE_SECONDS` | Grace period before cancel (Workforce) | 25 |
| `WORKFORCE_ADDR` | Workforce health server address | `:9090` |

**Production:** Use Vault or AWS Secrets Manager for JWT_SECRET, REGISTRY_DB_DSN. See [Secrets Management](../implementation/secrets-management.md).

See [Phase 2.1 Control Plane](../implementation/phase-2.1-control-plane.md) for full list.

### Audit Log Retention

The `audit_log` table stores entries for robots, commands, tasks, API keys, OAuth clients, and tenants. Consider a retention policy (e.g. 90 days) and periodic cleanup or archival. Audit entries are append-only; no built-in retention job exists.

## Health Checks

| Endpoint | Purpose | Use for |
|----------|---------|---------|
| `GET /health` | Liveness — process is alive | Kubernetes liveness probe |
| `GET /ready` | Readiness — NATS connected, DB reachable | Kubernetes readiness probe |
| `GET /metrics` | Prometheus metrics | Scraping |

### Liveness (`/health`)

Returns `200 OK` with body `ok` if the process is running. No dependencies checked.

### Readiness (`/ready`)

Returns `200 OK` if:

- NATS is connected
- Database (if configured) responds to `Ping()`

Returns `503 Service Unavailable` with body `nats disconnected` or `database unavailable` if checks fail.

### Kubernetes Probes

**Control Plane** (port 8080):

```yaml
livenessProbe:
  httpGet:
    path: /health
    port: 8080
  initialDelaySeconds: 5
  periodSeconds: 10

readinessProbe:
  httpGet:
    path: /ready
    port: 8080
  initialDelaySeconds: 5
  periodSeconds: 5
```

**Workforce** (port 9090, when running in split mode):

```yaml
livenessProbe:
  httpGet:
    path: /health
    port: 9090
  initialDelaySeconds: 5
  periodSeconds: 10

readinessProbe:
  httpGet:
    path: /ready
    port: 9090
  initialDelaySeconds: 5
  periodSeconds: 5
```

## Graceful Shutdown

Both Control Plane and Workforce handle SIGTERM for clean shutdown. See [Graceful Shutdown](graceful-shutdown.md) for details.

- **Control Plane**: Drains in-flight HTTP requests within `SHUTDOWN_TIMEOUT` (default 30s), then exits.
- **Workforce**: Waits `SHUTDOWN_GRACE_SECONDS` (default 25s) to allow task runner to cancel active tasks (safe_stop, release zones, update status). In-flight webhook deliveries may be interrupted during shutdown.

## Monitoring

### Prometheus Scrape

Configure Prometheus to scrape `http://<control-plane>:8080/metrics`.

### Key Metrics

| Metric | Type | Description |
|--------|------|-------------|
| `http_requests_total` | Counter | Total HTTP requests by method, path, status |
| `http_requests_errors_total` | Counter | Error responses (4xx, 5xx) by method, path, status |
| `http_request_duration_seconds` | Histogram | Request latency by method, path |
| `streaming_gateway_dropped_events_total` | Counter | Telemetry events dropped due to backpressure |

### Prometheus Config Example

```yaml
scrape_configs:
  - job_name: 'control-plane'
    static_configs:
      - targets: ['control-plane:8080']
    metrics_path: /metrics
```

## Alerting

### Prometheus Rules

Full rule definitions are in [`deploy/prometheus/rules/control-plane.yml`](../../deploy/prometheus/rules/control-plane.yml). Include in your Prometheus config:

```yaml
rule_files:
  - /path/to/deploy/prometheus/rules/control-plane.yml
```

### Alerts

| Alert | Condition | Severity |
|-------|-----------|----------|
| ControlPlaneDown | `up{job="control-plane"} == 0` for 1m | Critical |
| ControlPlaneNotReady | `probe_success{job="control-plane-ready"} == 0` for 1m | Critical |
| HighErrorRate | Error rate > 5% over 5m | Warning |
| HighLatency | p95 latency > 5s over 5m | Warning |
| StreamingGatewayBackpressure | `streaming_gateway_dropped_events_total` increase > 100 in 5m | Warning |

### Runbook Procedures

#### ControlPlaneDown

**Symptoms:** Control Plane process is not running; `/health` and `/ready` unreachable.

**Checks:**
1. `kubectl get pods` or `docker ps` — is the control-plane container running?
2. Check pod/container logs for crash or OOM.
3. Verify resource limits (CPU, memory).

**Remediation:**
1. Restart: `kubectl rollout restart deployment/control-plane` or `docker compose restart control-plane`
2. If crash loop: inspect logs, check NATS/DB connectivity, verify env vars (JWT_SECRET, REGISTRY_DB_DSN)
3. See [Restart Procedure](#restart-procedure)

#### ControlPlaneNotReady

**Symptoms:** Readiness probe failing; Control Plane returns 503 on `/ready`.

**Checks:**
1. `curl http://<control-plane>:8080/ready` — response body: `nats disconnected` or `database unavailable`
2. Verify NATS: `curl http://<nats>:8222/varz` (if monitoring enabled)
3. Verify PostgreSQL: `pg_isready -h <host> -U saiaurosy`

**Remediation:**
1. **NATS disconnected:** Ensure NATS is running, check NATS_URL, network connectivity
2. **Database unavailable:** Check DSN, credentials, DB connectivity; restart DB if needed
3. See [NATS Reconnection](#nats-reconnection), [Database Backup and Restore](#database-backup-and-restore)

#### HighErrorRate

**Symptoms:** More than 5% of HTTP requests return 4xx or 5xx.

**Checks:**
1. Query `http_requests_errors_total` by status: `sum by (status) (rate(http_requests_errors_total[5m]))`
2. Check logs for repeated errors (auth failures, validation, DB errors)
3. Review recent deployments or config changes

**Remediation:**
1. Identify error type from status distribution (401/403: auth; 500: backend)
2. For auth: verify JWT/API keys, tenant configuration
3. For 500: check DB, NATS, and component logs; consider rollback

#### HighLatency

**Symptoms:** p95 request duration exceeds 5 seconds.

**Checks:**
1. `histogram_quantile(0.95, sum(rate(http_request_duration_seconds_bucket[5m])) by (le, path))` — identify slow paths
2. Check DB query performance, NATS latency
3. Review CPU/memory usage

**Remediation:**
1. Scale up Control Plane replicas if CPU-bound
2. Optimize slow endpoints; add indexes if DB-bound
3. Check for long-running tasks or workflow runs

#### StreamingGatewayBackpressure

**Symptoms:** Telemetry events dropped; SSE clients may miss updates.

**Checks:**
1. `streaming_gateway_dropped_events_total` — rate of increase
2. Number of active SSE connections to `/v1/telemetry/stream`
3. Client consumption rate (slow or disconnected clients)

**Remediation:**
1. Normal under burst; monitor trend
2. If sustained: increase `streaming.RingBuffer` capacity (code change) or reduce client load
3. Ensure clients use Last-Event-ID for reconnect to avoid full replay

### Webhook Delivery Failures

There is no built-in Prometheus metric for webhook failures. Monitor the `webhook_delivery_failures` table (when using SQL store with dead letter):

```sql
SELECT COUNT(*) FROM webhook_delivery_failures WHERE created_at > NOW() - INTERVAL '1 hour';
```

Consider adding a periodic job or exporter to expose this as a metric. High counts indicate webhook endpoints are down or rejecting requests; check circuit breaker state and endpoint health.

### Alertmanager Example

```yaml
route:
  receiver: default
  group_by: [alertname, severity]
  group_wait: 30s
  group_interval: 5m
  repeat_interval: 4h

receivers:
  - name: default
    # Configure Slack, PagerDuty, or email
```

## Recovery

### Restart Procedure

1. Stop Control Plane: `docker compose stop control-plane` or `kubectl rollout restart deployment/control-plane`
2. Verify NATS and PostgreSQL are running
3. Start Control Plane: `docker compose up -d control-plane` or wait for rollout
4. Check `/ready` returns 200

### Database Backup and Restore

**PostgreSQL:**

```bash
# Backup
pg_dump -h localhost -U saiaurosy saiaurosy > backup_$(date +%Y%m%d).sql

# Restore
psql -h localhost -U saiaurosy saiaurosy < backup_YYYYMMDD.sql
```

**SQLite:**

```bash
cp /path/to/registry.db /path/to/registry.db.backup
```

### NATS Reconnection

Control Plane reconnects to NATS automatically. If NATS is restarted:

1. NATS comes back up
2. Control Plane Telemetry Bus reconnects
3. `/ready` returns 200 once NATS is connected

No manual intervention required for normal NATS restarts.

### Rollback

1. Revert to previous image tag or deployment revision
2. Restart Control Plane
3. If schema changed: run down migrations if supported, or restore DB backup before migration

## Troubleshooting

| Symptom | Cause | Resolution |
|---------|-------|------------|
| `/ready` returns 503 "nats disconnected" | NATS unreachable or down | Check NATS URL, ensure NATS is running, verify network |
| `/ready` returns 503 "database unavailable" | DB unreachable or credentials wrong | Check DSN, credentials, DB connectivity |
| `log.Fatal("Auth required but...")` on startup | Auth required but not configured | Set `JWT_SECRET` or `JWT_PUBLIC_KEY`, or `ALLOW_UNSAFE_NO_AUTH=true` (dev only) |
| Migration errors on startup | Schema mismatch or failed migration | Check migration files in `pkg/control-plane/registry/migrations/`, ensure DB is clean or restore backup |
| JWT validation failures | Invalid token, wrong secret/key | Verify `JWT_SECRET`/`JWT_PUBLIC_KEY`, issuer, audience |
| High `streaming_gateway_dropped_events_total` | SSE clients slow or disconnected | Normal under load; consider increasing buffer or tuning |

## Related Documents

- [Operator Runbook](operator-runbook.md) — Tenant onboarding, workflow creation, troubleshooting
- [Graceful Shutdown](graceful-shutdown.md) — SIGTERM handling, drain behavior, timeouts
- [Phase 2.1 Control Plane](../implementation/phase-2.1-control-plane.md)
- [Deployment Model](../architecture/deployment-model.md)
- [Status + Safe Stop Runbook](../implementation/status-safe-stop.md)
