# Graceful Shutdown

How Control Plane and Workforce handle SIGTERM/SIGINT for clean shutdown in Kubernetes, Docker Swarm, and other orchestrators.

## Overview

Both services listen for SIGTERM and SIGINT. When received, they drain in-flight work before exiting so that:

- HTTP requests complete or timeout gracefully (Control Plane)
- Active robot tasks are cancelled cleanly with safe_stop and zone release (Workforce)

## Control Plane

### Shutdown Flow

1. SIGTERM/SIGINT received
2. `http.Server.Shutdown(ctx)` is called with a timeout
3. Server stops accepting new connections
4. In-flight HTTP requests complete (or timeout)
5. Background goroutines (task runner, telemetry, etc.) are cancelled
6. Process exits

### Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `SHUTDOWN_TIMEOUT` | Maximum seconds to wait for HTTP drain | 30 |

If the timeout is exceeded, `Shutdown` returns an error and the process exits. Remaining in-flight requests are aborted.

## Workforce

### Shutdown Flow

1. SIGTERM/SIGINT received
2. Grace period starts (configurable seconds)
3. Task runner continues; when it detects `ctx.Done()` at the next checkpoint (between steps or in the step sleep loop), it:
   - Updates task status to `Cancelled`
   - Sends `safe_stop` to the robot
   - Releases zone (if acquired)
   - Emits `task_completed` webhook
4. After grace period, context is cancelled
5. All goroutines (task runner, telemetry consumer, retention, etc.) exit
6. Process exits

### Task Runner Checkpoints

The task runner checks for context cancellation at:

- Start of each scenario step (before sending the next command)
- Every 500ms during step duration waits (e.g. while waiting for a patrol step to complete)

Tasks interrupted by shutdown are marked `Cancelled` and the robot receives `safe_stop`.

### Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `SHUTDOWN_GRACE_SECONDS` | Seconds to wait before cancelling context | 25 |

Set this to at least the typical step duration so in-flight tasks can reach a checkpoint. For long-running steps (e.g. 60s patrol), increase accordingly.

### Webhook Dispatcher

Webhook delivery is fire-and-forget. During shutdown, in-flight webhook HTTP calls may be interrupted. Consider this best-effort; critical events should be retried by the webhook receiver or reconciled from the database.

## Orchestrator Configuration

### Kubernetes

Ensure `terminationGracePeriodSeconds` is at least the sum of:

- Control Plane: `SHUTDOWN_TIMEOUT` (default 30)
- Workforce: `SHUTDOWN_GRACE_SECONDS` (default 25)

Example:

```yaml
spec:
  terminationGracePeriodSeconds: 60
```

### Docker Compose

Docker sends SIGTERM and waits 10 seconds by default. For Workforce, set `stop_grace_period`:

```yaml
workforce:
  stop_grace_period: 30s
```

## Related

- [Production Runbook](production-runbook.md)
- [Control Plane and Workforce Split](../architecture/control-plane-workforce-split.md)
