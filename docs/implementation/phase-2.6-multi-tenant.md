# Phase 2.6 — Multi-Tenant (приоритет 6)

Phase 2.6 реализует Tenant Management и multi-tenant UI в Operator Console.

## Обзор изменений

| Область | Изменения |
|---------|-----------|
| **Tenant Model** | Таблица tenants (id, name, config), модель Tenant |
| **Robot Filtering** | Robot.TenantID уже есть; ListByTenant, GET /robots?tenant_id= |
| **Task Filtering** | tenant_id в tasks, GET /tasks?tenant_id=, createTask устанавливает tenant из робота |
| **API** | GET /v1/tenants, GET /v1/tenants/:id, GET /v1/tenants/:id/robots |
| **Operator Console** | Селектор tenant, фильтрация роботов и задач по tenant |

## Access Control

- **operator** — видит только данные своего tenant (из `api_keys.tenant_id` или JWT claim `tenant_id`). Query-параметр `tenant_id` игнорируется.
- **administrator** — доступ ко всем tenants, может фильтровать по `?tenant_id=`.

Tenant enforcement points:

- **sendCommand** — проверяет `robot.TenantID`; при несовпадении возвращает 404.
- **cancelTask** — проверяет `task.TenantID`; при несовпадении возвращает 404.
- **createTask** — проверяет `robot.TenantID`; при несовпадении возвращает 403.
- **telemetry stream** — фильтрует события по `ListByTenant`; operator получает только телеметрию роботов своего tenant.
- **analytics** — `listRobotAnalyticsSummaries` использует `ListByTenant`; `getRobotAnalyticsSummary` проверяет tenant робота, при несовпадении — 404.
- **runWorkflow** — выбирает роботов только из tenant; run получает TenantID.
- **listWorkflowRuns** / **getWorkflowRun** — фильтрация/проверка по tenant.
- **listEdges** / **getEdge** — фильтрация по edges, обслуживающим роботов tenant.

## Tenant Management

### Модель Tenant

```go
type Tenant struct {
    ID     string          `json:"id"`
    Name   string          `json:"name"`
    Config json.RawMessage `json:"config,omitempty"`
}
```

### API

| Метод | Путь | Описание |
|-------|------|----------|
| GET | /v1/tenants | Список tenants |
| POST | /v1/tenants | Создать tenant (admin) |
| GET | /v1/tenants/:id | Tenant по ID |
| PUT | /v1/tenants/:id | Обновить tenant (admin) |
| DELETE | /v1/tenants/:id | Удалить tenant (admin, только если нет роботов) |
| GET | /v1/tenants/:id/robots | Роботы tenant |
| GET | /v1/robots?tenant_id= | Роботы с фильтром по tenant |
| GET | /v1/tasks?tenant_id= | Задачи с фильтром по tenant |
| GET | /v1/audit?tenant_id= | Audit log с фильтром по tenant |

### Валидация

- При `createRobot` и `updateRobot`: `tenant_id` проверяется — tenant должен существовать.
- `GET /v1/tenants/:id/robots` возвращает 404, если tenant не найден.

## Operator Console

- Селектор tenant в header (dropdown: «Все» | tenant1 | tenant2)
- При выборе tenant — robots и tasks запрашиваются с `?tenant_id=...`
- Выбранный tenant сохраняется в localStorage
- При выборе «Все» — в карточке робота отображается tenant_id

## Миграции

- `000010_create_tenants` — таблица tenants, seed default
- `000011_add_tenant_to_tasks` — колонка tenant_id в tasks
- `000012_add_tenant_to_workflow_runs` — колонка tenant_id в workflow_runs

## Ссылки

- [Platform Architecture](../architecture/platform-architecture.md)
- [Phase 2.1 Control Plane](phase-2.1-control-plane.md)
- [Roadmap](../product/roadmap.md)
