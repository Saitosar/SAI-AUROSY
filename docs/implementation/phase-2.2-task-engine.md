# Phase 2.2 — Task Engine и сценарии

Phase 2.2 добавляет Task/Scenario Engine: модель задач, каталог сценариев, выполнение через Command Arbiter и UI в Operator Console.

## Обзор изменений

| Область | Изменения |
|---------|-----------|
| **Task Engine** | Модель Task (id, robot_id, type, payload, status); TaskStore (Memory + SQL) |
| **Scenario Catalog** | Предопределённые сценарии: patrol, standby, navigation |
| **Task Runner** | Выполнение задач через SafetyAllow + PublishCommand; отмена |
| **API** | GET /v1/scenarios, POST /v1/tasks, GET /v1/tasks, GET /v1/tasks/:id, POST /v1/tasks/:id/cancel |
| **Operator Console** | Панель задач, создание по сценарию, отмена |

## API

### Сценарии

| Метод | Путь | Описание |
|-------|------|----------|
| GET | /v1/scenarios | Список предопределённых сценариев |

### Задачи

| Метод | Путь | Описание |
|-------|------|----------|
| GET | /v1/tasks | Список задач (query: robot_id, status) |
| POST | /v1/tasks | Создать задачу |
| GET | /v1/tasks/:id | Задача по ID |
| POST | /v1/tasks/:id/cancel | Отменить задачу (pending/running) |

### Создание задачи

```json
POST /v1/tasks
{
  "robot_id": "x1-001",
  "scenario_id": "patrol",
  "payload": { "duration_sec": 30 },
  "operator_id": "console"
}
```

### Сценарии

| id | name | description | payload |
|----|------|-------------|---------|
| standby | Ожидание | Стоячая поза | — |
| patrol | Патруль | walk_mode + cmd_vel N сек | `{"duration_sec": 30}` |
| navigation | Навигация | walk_mode + движение | `{"linear_x": 0.2, "linear_y": 0, "angular_z": 0, "duration_sec": 10}` |

## Миграции

- `000003_create_tasks.up.sql` — таблица tasks (id, robot_id, type, scenario_id, payload, status, created_at, updated_at, completed_at, operator_id)

## Важные детали

- **Один робот — одна активная задача**: при создании проверяется отсутствие running-задачи у робота
- **Безопасность**: Task Runner использует `arbiter.SafetyAllow` перед каждой командой
- **Отмена**: при отмене Task Runner отправляет safe_stop роботу

## Ссылки

- [Phase 2.1 Control Plane](phase-2.1-control-plane.md)
- [MVP V1 Overview](mvp-v1-overview.md)
- [Roadmap](../product/roadmap.md)
