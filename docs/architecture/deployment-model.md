## Deployment Model

SAI AUROSY follows a hybrid architecture:

### Edge Layer

Runs on the customer site and provides low-latency robot control.

Includes:
- robot adapters
- safety supervisor
- local event broker
- teleoperation relay

### Cloud Control Plane

Runs in the cloud and provides platform management.

Includes:
- fleet management
- orchestration
- analytics
- enterprise integrations

### Control Plane and Workforce Split

For scaling, Control Plane and Workforce can run as separate services:

- **Control Plane**: API gateway, auth, registry, streaming, cognitive gateway, edges. Handles all HTTP API traffic.
- **Workforce**: Task engine, orchestration execution, analytics consumer, webhook delivery, telemetry retention.

Both share the same database and NATS. Set `WORKFORCE_REMOTE=true` on Control Plane and run `cmd/workforce` separately. See [Control Plane and Workforce Split](control-plane-workforce-split.md).

**Scaling scenarios**: For when to use monolith vs split mode, Control Plane horizontal scaling, and current limitations for multiple Workforce instances, see [Scaling Scenarios](control-plane-workforce-split.md#scaling-scenarios) in the Control Plane and Workforce Split document.

### Network Resilience

This architecture allows robots to operate even in unstable network environments.