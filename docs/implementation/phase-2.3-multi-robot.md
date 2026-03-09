# Phase 2.3 — Multi-Robot и оркестрация

Phase 2.3 добавляет Multi-Robot Coordination (зоны, блокировки), Orchestration Service (multi-robot workflows) и Capability Model.

## Обзор изменений

| Область | Изменения |
|---------|-----------|
| **Capability Model** | Robot.Capabilities, RequiredCapabilities в Scenario, фильтрация задач |
| **Coordinator** | Зоны A/B/C, AcquireZone/ReleaseZone, эксклюзивный доступ |
| **Orchestration** | Workflow Catalog, WorkflowRun, POST /v1/workflows/:id/run |
| **API** | GET /v1/zones, GET /v1/zones/:id, GET /v1/workflows, GET /v1/workflow-runs |
| **Operator Console** | Панели Zones, Workflows, capabilities в карточке робота |

## Capability Model

### HAL

- `Robot.Capabilities` — массив строк: `walk`, `stand`, `safe_stop`, `release_control`, `cmd_vel`, `zero_mode`, `patrol`, `navigation`
- `hal.HasCapability(robot, required)` — проверка наличия всех требуемых возможностей
- Константы в `pkg/hal/capabilities.go`

### Scenario

- `Scenario.RequiredCapabilities` — список возможностей, необходимых для выполнения сценария
- standby: `[stand]`
- patrol: `[walk, cmd_vel, patrol]`
- navigation: `[walk, cmd_vel, navigation]`

### Валидация

- Task Runner: перед запуском проверяет capabilities робота
- API POST /v1/tasks: возвращает 400 при несоответствии capabilities

## Coordinator

### Зоны

- Предопределённые зоны: A, B, C
- Эксклюзивный доступ: один робот на зону
- FIFO: первый запросивший получает доступ

### Интеграция с Task Runner

- Payload задачи может содержать `zone_id`
- Для patrol/navigation с `zone_id`: AcquireZone перед выполнением, ReleaseZone при завершении/отмене
- Если зона занята — задача переходит в StatusFailed

### API

| Метод | Путь | Описание |
|-------|------|----------|
| GET | /v1/zones | Список зон со статусом (свободна/занята, robot_id) |
| GET | /v1/zones/:id | Статус конкретной зоны |

## Orchestration

### Workflow

- `patrol_zones_ABC` — 3 робота патрулируют зоны A, B, C
- Каждый шаг: scenario_id=patrol, payload с zone_id и duration_sec

### WorkflowRun

- При запуске создаются N задач (по одной на шаг)
- Robot selector: выбирается свободный робот с нужными capabilities
- Статус run: running, completed, failed

### API

| Метод | Путь | Описание |
|-------|------|----------|
| GET | /v1/workflows | Список workflows |
| POST | /v1/workflows/:id/run | Запуск workflow (body: operator_id) |
| GET | /v1/workflow-runs | Список runs |
| GET | /v1/workflow-runs/:id | Детали run |

### Миграции

- `000004_add_robot_capabilities` — колонка capabilities в robots
- `000005_create_workflows` — таблицы workflow_runs, workflow_run_tasks

## Operator Console

- **Zones** — карточки зон, статус (свободна/занята, какой робот)
- **Workflows** — список workflows, кнопка «Запустить», активные runs
- **Robot card** — отображение capabilities

## Ссылки

- [Phase 2.2 Task Engine](phase-2.2-task-engine.md)
- [Adapter Layer](../architecture/adapter-layer.md)
- [Multi-Robot Architecture](../architecture/multi-robot-architecture.md)
