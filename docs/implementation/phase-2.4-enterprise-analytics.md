# Phase 2.4 — Enterprise и аналитика (приоритет 4)

Phase 2.4 добавляет Enterprise Integrations (webhooks, REST API, OpenAPI), Analytics (телеметрия, агрегации, API, дашборды) и Audit (логирование, audit_log, API).

## Обзор изменений

| Область | Изменения |
|---------|-----------|
| **Enterprise** | Webhook-события (robot_online, task_completed, safe_stop), CRUD /v1/webhooks, OpenAPI/Swagger |
| **Analytics** | Хранение телеметрии (telemetry_samples), агрегации (uptime, commands, errors), API /v1/analytics/robots, дашборд в Operator Console |
| **Audit** | audit_log (actor, action, resource, resource_id, timestamp), API GET /v1/audit |

## API

### Webhooks

| Метод | Путь | Описание |
|-------|------|----------|
| GET | /v1/webhooks | Список webhooks |
| POST | /v1/webhooks | Создать webhook |
| GET | /v1/webhooks/:id | Webhook по ID |
| PUT | /v1/webhooks/:id | Обновить webhook |
| DELETE | /v1/webhooks/:id | Удалить webhook |

События: `robot_online`, `task_completed`, `safe_stop`.

### Analytics

| Метод | Путь | Описание |
|-------|------|----------|
| GET | /v1/analytics/robots | Сводка по всем роботам (query: from, to) |
| GET | /v1/analytics/robots/:id/summary | Сводка по роботу (query: from, to) |

Response: `{ robot_id, uptime_sec, commands_count, errors_count, tasks_completed, tasks_failed }`.

### Audit

| Метод | Путь | Описание |
|-------|------|----------|
| GET | /v1/audit | Список записей (query: robot_id, actor, action, from, to, limit, offset) |

### OpenAPI

| Путь | Описание |
|------|----------|
| GET /openapi.json | OpenAPI 3.0 спецификация |
| GET /swagger/ | Swagger UI |

## Миграции

- `000006_create_webhooks` — таблица webhooks
- `000007_create_audit_log` — таблица audit_log
- `000008_create_telemetry_samples` — таблица telemetry_samples

## Переменные окружения

| Переменная | Описание |
|------------|----------|
| `REGISTRY_DB_DRIVER` | sqlite или postgres — при наличии БД включаются audit, webhooks, analytics |
| `REGISTRY_DB_DSN` | Строка подключения к БД |

## Ссылки

- [Phase 2.3 Multi-Robot](phase-2.3-multi-robot.md)
- [Roadmap](../product/roadmap.md)
