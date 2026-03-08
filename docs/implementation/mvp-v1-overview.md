# MVP V1 Overview

## Scope

MVP V1 реализует минимальный сценарий **«Status + Safe Stop»** для платформы SAI-AUROSY.

- Робот AGIBOT X1 (или mock) подключается к платформе
- Operator Console отображает статус (online, actuator_status, current_task)
- Оператор может отправить команду **safe_stop**
- Робот переходит в idle (без крутящего момента)

## Компоненты

| Компонент | Описание |
|-----------|----------|
| Fleet Registry | Реестр роботов, API GET /robots, GET /robots/:id |
| Event Broker / Telemetry Bus | NATS, топики telemetry.robots.*, commands.robots.* |
| Command Arbiter | Маршрутизация команд по robot_id |
| Safety Supervisor | Проверка safe_stop, release_control |
| Operator Console | React SPA, список роботов, real-time телеметрия, кнопка Safe Stop |
| AGIBOT Adapter | Python: ROS2 ↔ NATS bridge (или mock) |
| HAL | RobotAdapter interface, Telemetry, Command types |

## Границы MVP V1

**В scope:**
- Один робот (x1-001)
- Телеметрия: online, actuator_status, imu, current_task
- Команда safe_stop
- Mock-режим адаптера (без ROS2)

**Вне scope:**
- Multi-tenant
- Identity/Policy (JWT)
- Orchestrator, Scenario Catalog
- UNITREE adapter
- Observability (Prometheus, traces)

## Ссылки

- [Status + Safe Stop Runbook](status-safe-stop.md)
- [MVP V1 Architecture](../architecture/mvp-v1-architecture.md)
- [Adapter Layer](../architecture/adapter-layer.md)
- [AGIBOT Protocols](../vendors/agibot.md)
