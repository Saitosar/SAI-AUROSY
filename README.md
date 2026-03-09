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

**Phase 2.5** — Deployment и расширяемость: Edge Agent, Cloud edges API, ROS adapter, Adapter SDK. See [Phase 2.5 Deployment Extensibility](docs/implementation/phase-2.5-deployment-extensibility.md).

**Phase 2.9** — Security: Mandatory JWT/API key auth for all endpoints, strict tenant isolation (robots, tasks, audit, scenarios), edge heartbeat auth, secrets management (Vault/AWS), audit for API keys/OAuth/tenants. See [Phase 2.9 Security](docs/implementation/phase-2.9-security.md) and [Secrets Management](docs/implementation/secrets-management.md).

**Phase 2.10** — Scenario Builder MVP: structured step list, capability multi-select, validation. See [Phase 2.10 Scenario Builder MVP](docs/implementation/phase-2.10-scenario-builder-mvp.md).

**Phase 3.1** — Streaming Gateway: SSE extensions (robot_id filter, Last-Event-ID reconnect, backpressure). See [Phase 3.1 Streaming Gateway](docs/implementation/phase-3.1-streaming-gateway.md).

**Phase 3.2** — Cognitive Gateway: AI services (navigation, recognition, planning), mock provider, API /v1/cognitive/*. See [Phase 3.2 Cognitive Gateway](docs/implementation/phase-3.2-cognitive-gateway.md).

**Phase 3.5** — Speech Layer: STT, TTS, intent extraction, Conversation Catalog, multilingual (uz, en, ru, az, ar). See [Phase 3.5 Speech Layer](docs/implementation/phase-3.5-speech-layer.md) and [Speech Layer Architecture](docs/architecture/speech-layer.md).

**Mall Assistant Scenario** — First end-to-end interactive scenario: greet visitors, answer store-location questions, guide to stores, return to standby. See [Mall Assistant Scenario](docs/implementation/mall-assistant-scenario.md).

**Phase 3.3** — Developer Platform: API keys self-service, sandbox tenant, Swagger UI at /api/docs. See [Phase 3.3 Developer Platform](docs/implementation/phase-3.3-developer-platform.md).

**Phase 3.4** — Robot Application Marketplace: catalog of scenarios, categories, ratings, Operator Console section. See [Phase 3.4 Marketplace](docs/implementation/phase-3.4-marketplace.md).

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
- [Control Plane and Workforce split](docs/architecture/control-plane-workforce-split.md)
- [Mall Digital Twin](docs/architecture/mall-digital-twin.md) — mall map, navigation graph, route planning

## Integration

- [Integration Guide](docs/integration/README.md) — API overview, base URL, authentication
- [API Reference](docs/integration/api-reference.md) — Endpoint overview and OpenAPI link
- [API Versioning and Deprecation Policy](docs/integration/api-versioning.md) — When /v2 is introduced, deprecation lifecycle

## Implementation

- [MVP V1 overview](docs/implementation/mvp-v1-overview.md)
- [Status + Safe Stop runbook](docs/implementation/status-safe-stop.md)
- [Phase 2.1 Control Plane](docs/implementation/phase-2.1-control-plane.md)
- [Phase 2.2 Task Engine](docs/implementation/phase-2.2-task-engine.md)
- [Phase 2.3 Multi-Robot](docs/implementation/phase-2.3-multi-robot.md)
- [Phase 2.4 Enterprise Analytics](docs/implementation/phase-2.4-enterprise-analytics.md)
- [Phase 2.5 Deployment Extensibility](docs/implementation/phase-2.5-deployment-extensibility.md)
- [Phase 3.1 Streaming Gateway](docs/implementation/phase-3.1-streaming-gateway.md)
- [Phase 3.2 Cognitive Gateway](docs/implementation/phase-3.2-cognitive-gateway.md)
- [Phase 3.5 Speech Layer](docs/implementation/phase-3.5-speech-layer.md)
- [Phase 3.3 Developer Platform](docs/implementation/phase-3.3-developer-platform.md)
- [Phase 3.4 Marketplace](docs/implementation/phase-3.4-marketplace.md)
- [Phase 2.8 Priority 5 Improvements](docs/implementation/phase-2.8-priority5-improvements.md)
- [Telemetry retention](docs/implementation/telemetry-retention.md)
- [Phase 2.9 Operator Console UX](docs/implementation/phase-2.9-operator-console-ux.md)
- [Phase 2.9 Security](docs/implementation/phase-2.9-security.md)
- [Secrets Management](docs/implementation/secrets-management.md)
- [Phase 2.10 Scenario Builder MVP](docs/implementation/phase-2.10-scenario-builder-mvp.md)
- [Testing and CI](docs/implementation/testing-ci.md)
- [Observability](docs/implementation/observability.md) — OpenTelemetry, structured logging, log-trace correlation
- [Multi-robot architecture](docs/architecture/multi-robot-architecture.md)
- [Adapter layer](docs/architecture/adapter-layer.md)
- [Deployment model (edge/cloud)](docs/architecture/deployment-model.md)

---

## Operations

- [Production runbook](docs/operations/production-runbook.md) — deployment, monitoring, alerts, recovery
- [Operator runbook](docs/operations/operator-runbook.md) — tenant onboarding, workflow creation, troubleshooting

---

## Adapters

- [Adapter development guide](docs/adapters/adapter-development-guide.md) — step-by-step guide with template
- [RobotAdapter contract](docs/adapters/robot-adapter-contract.md) — NATS topics, JSON formats
- [Adapter template](pkg/adapters/template/README.md) — minimal example

---

## Robot Vendors

- [AGIBOT integration notes](docs/vendors/agibot.md)
- [UNITREE integration notes](docs/vendors/unitree.md)
- [ROS integration notes](docs/vendors/ros.md)

---

## Glossary

- [Robotics terminology](docs/glossary/robotics-terms.md)
