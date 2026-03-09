# Authentication

SAI AUROSY Control Plane API supports API keys and JWT tokens. OAuth 2.0 is planned for third-party integrations.

## API Key

Best for server-to-server integrations, scripts, and automation.

### Headers

```
X-API-Key: <your-api-key>
```

### Creating an API Key

API keys are stored in the `api_keys` table. An administrator creates them by inserting a row:

```sql
-- key_hash = SHA256(raw_key) in hex
-- Example: echo -n "my-secret-key" | sha256sum
INSERT INTO api_keys (id, key_hash, name, roles, tenant_id, created_at)
VALUES (
  'key-integration-1',
  '<sha256_hex_of_your_secret>',
  'CRM Integration',
  'operator',
  'default',
  datetime('now')
);
```

The raw key (e.g. `my-secret-key`) is shown once at creation; store it securely. Use it as `X-API-Key: my-secret-key`.

### Roles

| Role | Scope |
|------|-------|
| `operator` | Read robots, tasks, workflows; create/cancel tasks; send commands. Scoped to `tenant_id`. **Requires `tenant_id` in key.** |
| `administrator` | Full access; can manage robots, tenants, webhooks, scenarios. |
| `system` | For edge agents; heartbeat and command relay. Use `EDGE_API_KEY` in edge agent config. |

### Example

```bash
curl -H "X-API-Key: my-secret-key" \
  https://api.example.com/api/v1/robots
```

---

## JWT (Bearer Token)

Best for user sessions and applications that obtain tokens from an identity provider.

### Headers

```
Authorization: Bearer <jwt-token>
```

### Claims

| Claim | Description |
|-------|-------------|
| `roles` or `role` | `operator`, `administrator`, or `system` |
| `tenant_id` | **Required for operator:** restricts access to this tenant. Operator tokens without `tenant_id` return 403. |
| `iss` | Issuer (if `JWT_ISSUER` is set) |
| `aud` | Audience (if `JWT_AUDIENCE` is set) |

### Example Payload

```json
{
  "sub": "user-123",
  "roles": ["operator"],
  "tenant_id": "default",
  "exp": 1733788800
}
```

### Example

```bash
curl -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIs..." \
  https://api.example.com/api/v1/robots
```

---

## OAuth 2.0

For CRM, ERP, and ticketing integrations, OAuth 2.0 Authorization Code flow is supported (when database is configured):

- **Authorize:** `GET /oauth/authorize?client_id=...&redirect_uri=...&response_type=code&scope=...&state=...`
- **Token:** `POST /oauth/token` with `grant_type=authorization_code`, `code`, `redirect_uri`, `client_id`, `client_secret`
- **Refresh:** `POST /oauth/token` with `grant_type=refresh_token`, `refresh_token`, `client_id`, `client_secret`
- **Revoke:** `POST /oauth/revoke` with `token` and optional `token_type_hint`
- **Scopes:** `robots:read`, `tasks:write`, `webhooks:read`, `analytics:read`
- Tokens are bound to `tenant_id` for multi-tenant isolation

OAuth clients are stored in the `oauth_clients` table. Set `OAUTH_BASE_URL` for the authorize redirect (default: `http://localhost:8080`).

### OAuth Client Admin API (Administrator only)

When database is configured, administrators can manage OAuth clients via REST:

| Method | Path | Description |
|--------|------|-------------|
| GET | `/v1/oauth/clients` | List clients (optional `?tenant_id=`) |
| POST | `/v1/oauth/clients` | Create client (`client_id`, `client_secret`, `redirect_uris`, `scopes`, `tenant_id`) |
| PUT | `/v1/oauth/clients/{client_id}` | Update `redirect_uris`, `scopes`, `tenant_id` |
| DELETE | `/v1/oauth/clients/{client_id}` | Delete client |

Requires `Authorization: Bearer` or `X-API-Key` with role `administrator`.

---

## Edge Agent Authentication

Edge agents send heartbeats to `POST /v1/edges/{id}/heartbeat`. When auth is enabled, they must authenticate using an API key with role `system`.

Set `EDGE_API_KEY` in the edge agent environment:

```bash
export EDGE_API_KEY=sk-...
```

Create a system API key via the database (admin only):

```sql
INSERT INTO api_keys (id, key_hash, name, roles, tenant_id, created_at)
VALUES ('key-edge-1', '<sha256_hex>', 'Edge Agent', 'system', 'default', datetime('now'));
```

---

## Auth Configuration

| Env Var | Description |
|--------|-------------|
| `JWT_SECRET` | HMAC secret for JWT validation |
| `JWT_PUBLIC_KEY` | RSA public key (PEM) for JWT validation |
| `JWT_ISSUER` | Required issuer claim |
| `JWT_AUDIENCE` | Required audience claim |
| `AUTH_REQUIRED` | Default `true`. If auth not configured, fail startup unless `ALLOW_UNSAFE_NO_AUTH=true`. |
| `ALLOW_UNSAFE_NO_AUTH` | Set to `true` for development only. Allows unauthenticated access when no JWT/API key/OAuth is configured. **Do not use in production.** |

When `JWT_SECRET`, `JWT_PUBLIC_KEY`, and API key store are all unset, the Control Plane fails to start unless `ALLOW_UNSAFE_NO_AUTH=true`. With that flag, the API accepts requests without authentication for local development.

---

## Error Responses

| Status | Meaning |
|--------|---------|
| 401 Unauthorized | Missing or invalid token/API key |
| 403 Forbidden | Valid auth but insufficient permissions (e.g. operator accessing another tenant's robot) |
| 404 Not Found | Resource not found or not in tenant scope |
