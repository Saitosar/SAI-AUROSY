# Phase 2.1 — Control Plane Strengthening

Phase 2.1 укрепляет Control Plane: персистентный Fleet Registry, Identity & Policy (JWT, RBAC, API keys), API Gateway (версионирование, rate limiting, CORS), Observability.

## Обзор изменений

| Область | Изменения |
|---------|-----------|
| **Persistence** | SQLite/PostgreSQL вместо in-memory; CRUD API (POST/PUT/DELETE); миграции схемы |
| **Identity** | JWT (HMAC/RS256), RBAC (operator, administrator, system), API keys |
| **API Gateway** | Версионирование /v1/, rate limiting, CORS, security headers |
| **Observability** | Prometheus metrics, structured JSON logging, /health, /ready |

## Переменные окружения

| Переменная | Описание | По умолчанию |
|------------|----------|--------------|
| `NATS_URL` | URL NATS | `nats://localhost:4222` |
| `CONTROL_PLANE_ADDR` | Адрес сервера | `:8080` |
| `REGISTRY_DB_DRIVER` | `sqlite` или `postgres` | — (in-memory) |
| `REGISTRY_DB_DSN` | Строка подключения к БД | для sqlite: `file::memory:?cache=shared` |
| `JWT_SECRET` | Секрет для HMAC JWT | — |
| `JWT_PUBLIC_KEY` | PEM публичный ключ для RS256 | — |
| `JWT_ISSUER` | Проверка claim `iss` | — |
| `JWT_AUDIENCE` | Проверка claim `aud` | — |
| `CORS_ORIGINS` | Разрешённые origins (через запятую) | `*` |
| `RATE_LIMIT_RPS` | Запросов в секунду на IP | 100 |
| `RATE_LIMIT_BURST` | Burst для rate limit | 200 |
| `LOG_FORMAT` | `json` для structured logging | — |

## Запуск

### In-memory (по умолчанию)

```bash
export NATS_URL=nats://localhost:4222
go run ./cmd/control-plane
```

### SQLite

```bash
export NATS_URL=nats://localhost:4222
export REGISTRY_DB_DRIVER=sqlite
export REGISTRY_DB_DSN=file:./registry.db
go run ./cmd/control-plane
```

### PostgreSQL

```bash
export NATS_URL=nats://localhost:4222
export REGISTRY_DB_DRIVER=postgres
export REGISTRY_DB_DSN=postgres://user:pass@localhost:5432/sai_aurosy?sslmode=disable
go run ./cmd/control-plane
```

### С JWT

```bash
export JWT_SECRET=your-secret-key
export JWT_ISSUER=sai-aurosy
go run ./cmd/control-plane
```

Все запросы к /v1/ требуют `Authorization: Bearer <token>` или `X-API-Key: <key>`.

## API

### Версионирование

Все маршруты под префиксом `/v1/`. Legacy-маршруты `/robots`, `/robots/{id}` и т.д. работают с заголовком `Deprecation: true`.

### CRUD роботов

| Метод | Путь | Описание |
|-------|------|----------|
| GET | /v1/robots | Список роботов |
| GET | /v1/robots/:id | Робот по ID |
| POST | /v1/robots | Создать робота |
| PUT | /v1/robots/:id | Обновить робота |
| DELETE | /v1/robots/:id | Удалить робота |
| POST | /v1/robots/:id/command | Отправить команду |

### Health и метрики

| Путь | Описание |
|------|----------|
| GET /health | Liveness |
| GET /ready | Readiness (NATS, DB) |
| GET /metrics | Prometheus metrics |

### Примеры curl

```bash
# Список роботов (без auth, если JWT не настроен)
curl http://localhost:8080/v1/robots

# Создать робота
curl -X POST http://localhost:8080/v1/robots \
  -H "Content-Type: application/json" \
  -d '{"id":"r1","vendor":"agibot","model":"X1","adapter_endpoint":"nats://localhost:4222"}'

# Health
curl http://localhost:8080/health
curl http://localhost:8080/ready
curl http://localhost:8080/metrics
```

## Миграции

Миграции выполняются автоматически при старте Control Plane (при использовании SQL store). Файлы в `pkg/control-plane/registry/migrations/`:

- `000001_create_robots.up.sql` — таблица robots
- `000002_create_api_keys.up.sql` — таблица api_keys

## API Keys

API keys хранятся в таблице `api_keys` (key_hash, name, roles, tenant_id, expires_at). Для создания ключа вставьте запись вручную:

```sql
-- key_hash = SHA256(raw_key) в hex
INSERT INTO api_keys (id, key_hash, name, roles, tenant_id, created_at)
VALUES ('key-1', '<sha256_hex_of_secret>', 'integration', 'operator', 'default', datetime('now'));
```

Использование: заголовок `X-API-Key: <raw_key>`.

## RBAC

| Роль | Доступ |
|------|--------|
| operator | GET robots, POST command, telemetry stream |
| administrator | Всё выше + POST/PUT/DELETE robots |
| system | Внутренние вызовы, API keys |

## Ссылки

- [Status + Safe Stop](status-safe-stop.md)
- [MVP V1 Overview](mvp-v1-overview.md)
- [Roadmap](../product/roadmap.md)
