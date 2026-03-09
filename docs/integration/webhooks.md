# Webhooks

Webhooks deliver real-time events from SAI AUROSY to your HTTP endpoint. Use them to integrate with CRM, ERP, ticketing systems, or internal dashboards.

## Events

| Event | When |
|-------|------|
| `robot_online` | Robot sends telemetry after being offline |
| `robot_offline` | *(planned)* Robot stops sending telemetry |
| `task_started` | Task execution begins |
| `task_completed` | Task finishes successfully |
| `task_failed` | *(planned)* Task fails |
| `safe_stop` | Robot receives safe_stop command |
| `zone_acquired` | Robot acquires a zone lock |
| `zone_released` | Robot releases a zone lock |
| `workflow_completed` | *(planned)* Workflow run completes |

## Payload Schema

All webhook requests are `POST` with `Content-Type: application/json`:

```json
{
  "event": "task_completed",
  "timestamp": "2025-03-09T12:00:00Z",
  "data": {
    "task_id": "task-abc123",
    "robot_id": "r1",
    "scenario_id": "patrol",
    "status": "completed",
    "tenant_id": "default"
  }
}
```

### Event-Specific Data

| Event | `data` fields (typical) |
|-------|-------------------------|
| `robot_online` | `robot_id`, `tenant_id` |
| `task_started` | `task_id`, `robot_id`, `scenario_id`, `tenant_id` |
| `task_completed` | `task_id`, `robot_id`, `scenario_id`, `status`, `tenant_id` |
| `safe_stop` | `robot_id`, `tenant_id` |
| `zone_acquired` | `robot_id`, `zone_id`, `tenant_id` |
| `zone_released` | `robot_id`, `zone_id`, `tenant_id` |

## Headers

| Header | Description |
|--------|-------------|
| `Content-Type` | `application/json` |
| `X-Webhook-Event` | Event name (e.g. `task_completed`) |
| `X-Webhook-Signature` | HMAC-SHA256 signature (if secret configured) |

## HMAC Verification

If you set a `secret` when creating a webhook, SAI AUROSY signs each request with HMAC-SHA256:

```
X-Webhook-Signature: sha256=<hex-encoded-hmac>
```

The HMAC is computed over the raw request body (UTF-8 JSON).

### Python Example

```python
import hmac
import hashlib

def verify_webhook(body: bytes, signature: str, secret: str) -> bool:
    if not signature.startswith("sha256="):
        return False
    expected = "sha256=" + hmac.new(
        secret.encode(), body, hashlib.sha256
    ).hexdigest()
    return hmac.compare_digest(expected, signature)
```

### Node.js Example

```javascript
const crypto = require('crypto');

function verifyWebhook(body, signature, secret) {
  if (!signature.startsWith('sha256=')) return false;
  const expected = 'sha256=' + crypto
    .createHmac('sha256', secret)
    .update(body)
    .digest('hex');
  return crypto.timingSafeEqual(
    Buffer.from(expected),
    Buffer.from(signature)
  );
}
```

## Retry Policy

- **Retries:** Up to 3 retries (4 total attempts)
- **Backoff:** Exponential (2s, 4s, 8s between attempts)
- **Success:** HTTP 2xx response
- **Failure:** HTTP 4xx/5xx or network error triggers retry

Your endpoint should respond with 2xx quickly. For long processing, acknowledge with 200 and process asynchronously.

## Circuit Breaker

When a webhook URL repeatedly fails (5 consecutive failures), the circuit opens and delivery is skipped for 60 seconds. This prevents one failing endpoint from blocking others. After the open period, one probe request is sent (half-open state); success closes the circuit, failure reopens it.

## Dead Letter

Failed deliveries after all retries are recorded in the `webhook_delivery_failures` table for monitoring and manual replay. Administrators can inspect failures via the database or future API endpoints.

## Creating Webhooks

Webhooks are created via the API (administrator role):

```bash
curl -X POST https://api.example.com/api/v1/webhooks \
  -H "X-API-Key: <admin-key>" \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://your-app.com/webhooks/sai-aurosy",
    "events": ["task_completed", "safe_stop"],
    "secret": "optional-hmac-secret"
  }'
```

| Field | Required | Description |
|-------|----------|-------------|
| `url` | Yes | HTTPS endpoint (HTTP allowed for dev) |
| `events` | Yes | Array of event names |
| `secret` | No | HMAC signing secret |
| `enabled` | No | Default `true` |

## Best Practices

1. **Verify signatures** — Always validate `X-Webhook-Signature` when using a secret
2. **Idempotency** — Events may be retried; use `task_id` or `timestamp` to deduplicate
3. **Respond quickly** — Return 200 within a few seconds; queue work if needed
4. **Log failures** — Monitor 4xx/5xx to detect integration issues
