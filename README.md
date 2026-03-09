# SAI AUROSY

Enterprise Multi-Robot Workforce Platform.

SAI AUROSY is a platform designed to transform robots from different vendors into a unified enterprise workforce.

The platform provides:

- fleet management
- robot task orchestration
- enterprise integrations
- analytics and telemetry
- multi-robot coordination

SAI AUROSY supports multiple robotics ecosystems:

- AGIBOT
- UNITREE
- ROS-based robots
- future robotics vendors

---

# Quick Start (MVP V1)

```bash
docker compose up -d
```

Open [http://localhost:3000](http://localhost:3000) — Operator Console. Robot x1-001 appears with real-time telemetry. Click **Safe Stop** to send the command.

See [Status + Safe Stop Runbook](docs/implementation/status-safe-stop.md) and [MVP V1 Overview](docs/implementation/mvp-v1-overview.md).

**Phase 2.1** — Control Plane: persistence (SQLite/PostgreSQL), JWT/RBAC, API keys, rate limiting, Prometheus, health endpoints. See [Phase 2.1 Control Plane](docs/implementation/phase-2.1-control-plane.md).

**Phase 2.2** — Task Engine: задачи, сценарии (patrol, standby, navigation), API /v1/tasks, /v1/scenarios, Operator Console — задачи. See [Phase 2.2 Task Engine](docs/implementation/phase-2.2-task-engine.md).

**Phase 2.3** — Multi-Robot: Capability Model, Coordinator (зоны, блокировки), Orchestration (workflows), API /v1/zones, /v1/workflows, Operator Console — Zones, Workflows. See [Phase 2.3 Multi-Robot](docs/implementation/phase-2.3-multi-robot.md).

**Phase 2.4** — Enterprise и аналитика: Webhooks, REST API, OpenAPI/Swagger, телеметрия, агрегации, audit_log, дашборды. See [Phase 2.4 Enterprise Analytics](docs/implementation/phase-2.4-enterprise-analytics.md).

---

# Documentation

## Product

- [Product overview](docs/product/platform-overview.md)
- [Problem statement](docs/product/problem-statement.md)
- [Value proposition](docs/product/value-proposition.md)
- [Use cases](docs/product/use-cases.md)
- [Roadmap](docs/product/roadmap.md)

---

## Architecture

- [Platform architecture](docs/architecture/platform-architecture.md)
- [MVP V1 architecture](docs/architecture/mvp-v1-architecture.md)

## Implementation

- [MVP V1 overview](docs/implementation/mvp-v1-overview.md)
- [Status + Safe Stop runbook](docs/implementation/status-safe-stop.md)
- [Phase 2.1 Control Plane](docs/implementation/phase-2.1-control-plane.md)
- [Phase 2.2 Task Engine](docs/implementation/phase-2.2-task-engine.md)
- [Phase 2.3 Multi-Robot](docs/implementation/phase-2.3-multi-robot.md)
- [Phase 2.4 Enterprise Analytics](docs/implementation/phase-2.4-enterprise-analytics.md)
- [Multi-robot architecture](docs/architecture/multi-robot-architecture.md)
- [Adapter layer](docs/architecture/adapter-layer.md)
- [Deployment model (edge/cloud)](docs/architecture/deployment-model.md)

---

## Robot Vendors

- [AGIBOT integration notes](docs/vendors/agibot.md)
- [UNITREE integration notes](docs/vendors/unitree.md)

---

## Glossary

- [Robotics terminology](docs/glossary/robotics-terms.md)
