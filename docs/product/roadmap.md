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

- [x] **Phase 2.2** — Task Engine: задачи, Scenario Catalog (patrol, standby, navigation), API /v1/tasks, Operator Console — задачи. See [Phase 2.2 Task Engine](../implementation/phase-2.2-task-engine.md).
- [x] **Phase 2.3** — Multi-Robot: Capability Model, Coordinator (зоны, блокировки), Orchestration (workflows), API /v1/zones, /v1/workflows, Operator Console — Zones, Workflows. See [Phase 2.3 Multi-Robot](../implementation/phase-2.3-multi-robot.md).
- [x] **Phase 2.4** — Enterprise и аналитика: Webhooks (robot_online, task_completed, safe_stop), REST API, OpenAPI/Swagger, телеметрия (telemetry_samples), агрегации, API /v1/analytics, audit_log, API /v1/audit, дашборды в Operator Console. See [Phase 2.4 Enterprise Analytics](../implementation/phase-2.4-enterprise-analytics.md).
- [x] **Phase 2.5** — Deployment и расширяемость: Edge Agent, Cloud /v1/edges, heartbeat, ROS adapter, Adapter SDK (contract, template, robot_id prefix). See [Phase 2.5 Deployment Extensibility](../implementation/phase-2.5-deployment-extensibility.md).
- scenario builder

---

## Phase 3 — Robotics Ecosystem

- [x] Adapter SDK (documentation, template, prefix convention)
- developer platform
- robot application marketplace