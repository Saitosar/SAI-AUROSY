# Phase 2.7 — Enterprise Integration (Приоритет 2)

Phase 2.7 реализует Enterprise Integration: интеграции CRM/ERP/Ticketing (OAuth, webhooks, коннекторы) и REST API для внешних систем (документация, примеры, SDK).

## Обзор изменений

| Область | Изменения |
|---------|-----------|
| **Документация** | docs/integration/ — README, authentication, webhooks, quickstart, api-reference |
| **Примеры** | examples/integration/ — curl, Python, Go, webhook receiver (Flask/Express), telemetry stream |
| **Go SDK** | sdk/go/ — Client, ListRobots, GetRobot, SendCommand, ListTasks, CreateTask, CancelTask, ListWebhooks, CreateWebhook |
| **OAuth 2.0** | Authorization Code flow, /oauth/authorize, /oauth/token, /oauth/revoke, scopes, tenant_id |

## OAuth 2.0

При наличии БД (REGISTRY_DB_DRIVER) OAuth provider включается автоматически.

### Миграции

- `000014_create_oauth_clients` — таблица oauth_clients
- `000015_create_oauth_tokens` — таблица oauth_tokens
- `000016_create_oauth_codes` — таблица oauth_codes

### Endpoints

| Метод | Путь | Описание |
|-------|------|----------|
| GET | /oauth/authorize | Authorization Code flow — redirect с code |
| POST | /oauth/token | Обмен code на token, refresh_token |
| POST | /oauth/revoke | Отзыв токена |

### Scopes

- `robots:read`
- `tasks:write`
- `webhooks:read`
- `analytics:read`

### Переменные окружения

| Переменная | Описание |
|------------|----------|
| `OAUTH_BASE_URL` | URL для redirect (default: http://localhost:8080) |

### Создание OAuth клиента

OAuth клиенты создаются вручную в таблице `oauth_clients`:

```sql
INSERT INTO oauth_clients (id, client_id, client_secret_hash, redirect_uris, scopes, tenant_id, created_at, updated_at)
VALUES (
  '<uuid>',
  'crm-integration',
  '<sha256_hex_of_client_secret>',
  'https://crm.example.com/oauth/callback',
  'robots:read tasks:write',
  'default',
  datetime('now'),
  datetime('now')
);
```

## Go SDK

```go
client := sdk.New("http://localhost:8080", "api-key")
robots, _ := client.ListRobots(ctx, "")
task, _ := client.CreateTask(ctx, sdk.CreateTaskRequest{
    RobotID: "r1", ScenarioID: "patrol",
})
```

## Документация для интеграторов

- [docs/integration/README.md](../integration/README.md) — обзор
- [docs/integration/authentication.md](../integration/authentication.md) — API key, JWT, OAuth
- [docs/integration/webhooks.md](../integration/webhooks.md) — события, HMAC, retry
- [docs/integration/quickstart.md](../integration/quickstart.md) — quick start
- [docs/integration/api-reference.md](../integration/api-reference.md) — endpoints

## Ссылки

- [Phase 2.4 Enterprise Analytics](phase-2.4-enterprise-analytics.md)
- [Phase 2.6 Multi-Tenant](phase-2.6-multi-tenant.md)
- [platform-overview.md](../product/platform-overview.md)
- [Roadmap](../product/roadmap.md)
