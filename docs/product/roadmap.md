# Platform Roadmap

## Phase 1 — MVP

- [x] Unified Robot API (HAL, RobotAdapter interface)
- [x] AGIBOT adapter (Status + Safe Stop)
- [x] UNITREE adapter
- [x] Fleet management (Fleet Registry)
- [x] Operator dashboard (Operator Console)
- [x] Task execution engine (Command Arbiter, Safety Supervisor)

**MVP V1 delivered:** Status + Safe Stop scenario. See [MVP V1 Overview](../implementation/mvp-v1-overview.md).

---

## Phase 2 — Enterprise Platform

- [x] **Phase 2.1** — Control Plane: persistence (SQLite/PostgreSQL), JWT/RBAC, API /v1/, /health, /ready, /metrics, OpenAPI. See [Phase 2.1 Control Plane](../implementation/phase-2.1-control-plane.md).
- [x] **Phase 2.2** — Task Engine: задачи, Scenario Catalog (patrol, standby, navigation), API /v1/tasks, Operator Console — задачи. See [Phase 2.2 Task Engine](../implementation/phase-2.2-task-engine.md).
- [x] **Phase 2.3** — Multi-Robot: Capability Model, Coordinator (зоны, блокировки), Orchestration (workflows), API /v1/zones, /v1/workflows, Operator Console — Zones, Workflows. See [Phase 2.3 Multi-Robot](../implementation/phase-2.3-multi-robot.md).
- [x] **Phase 2.4** — Enterprise и аналитика: Webhooks (robot_online, task_completed, safe_stop), REST API, OpenAPI/Swagger, телеметрия (telemetry_samples), агрегации, API /v1/analytics, audit_log, API /v1/audit, дашборды в Operator Console. See [Phase 2.4 Enterprise Analytics](../implementation/phase-2.4-enterprise-analytics.md).
- [x] **Phase 2.5** — Deployment и расширяемость: Edge Agent, Cloud /v1/edges, heartbeat, ROS adapter, Adapter SDK (contract, template, robot_id prefix). See [Phase 2.5 Deployment Extensibility](../implementation/phase-2.5-deployment-extensibility.md).
- [x] **Phase 2.6** — Multi-Tenant: Tenant model, API /v1/tenants, /v1/tenants/:id/robots, фильтрация robots/tasks по tenant, Operator Console — выбор tenant. See [Phase 2.6 Multi-Tenant](../implementation/phase-2.6-multi-tenant.md).
- [x] **Phase 2.7** — Enterprise Integration: OAuth 2.0 (authorize, token, revoke), docs/integration/, examples/integration/, Go SDK (sdk/go/). See [Phase 2.7 Enterprise Integration](../implementation/phase-2.7-enterprise-integration.md).
- [x] **Phase 2.8** — Priority 5 improvements: E2E for v1 API, Auth in Operator Console, legacy routes removed, Fleet grouping by location. See [Phase 2.8 Priority 5 Improvements](../implementation/phase-2.8-priority5-improvements.md).
- [x] **Phase 2.9** — Operator Console UX: read-only mode (viewer role), command history in robot card, toast notifications (safe_stop, robot_online, task_completed). See [Phase 2.9 Operator Console UX](../implementation/phase-2.9-operator-console-ux.md).
- scenario builder

---

## Phase 3 — Robotics Ecosystem

- [x] Adapter SDK (documentation, template, prefix convention)
- [x] **Phase 3.1** — Streaming Gateway: SSE extensions (robot_id filter, Last-Event-ID reconnect, backpressure). See [Phase 3.1 Streaming Gateway](../implementation/phase-3.1-streaming-gateway.md).
- [x] **Phase 3.2** — Cognitive Gateway: AI services (navigation, recognition, planning), mock provider, API /v1/cognitive/*. See [Phase 3.2 Cognitive Gateway](../implementation/phase-3.2-cognitive-gateway.md).
- [x] **Phase 3.3** — Developer Platform: API keys self-service, sandbox tenant, Swagger UI, developer docs. See [Phase 3.3 Developer Platform](../implementation/phase-3.3-developer-platform.md).
- [x] **Phase 3.4** — Robot Application Marketplace: catalog, categories, ratings, Operator Console section. See [Phase 3.4 Marketplace](../implementation/phase-3.4-marketplace.md).