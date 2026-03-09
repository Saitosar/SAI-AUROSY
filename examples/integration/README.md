# SAI AUROSY Integration Examples

This directory contains example scripts for integrating with the SAI AUROSY Control Plane API.

## Prerequisites

- SAI AUROSY Control Plane running (default: `http://localhost:8080`)
- API key with `operator` role (or `administrator` for webhooks)

## Examples

### list-robots-and-tasks

Basic GET requests to list robots and tasks.

| File | Usage |
|------|-------|
| `list-robots-and-tasks.sh` | `./list-robots-and-tasks.sh [BASE_URL] [API_KEY]` (requires `jq`) |
| `list-robots-and-tasks.py` | `python list-robots-and-tasks.py [base_url] [api_key]` |
| `list-robots-and-tasks.go` | `go run list-robots-and-tasks.go [base_url] [api_key]` |

### create-task-via-api

Create a task via POST.

```bash
python create-task-via-api.py [base_url] [api_key] [robot_id] [scenario_id]
```

### webhook-receiver

Receive webhook events with HMAC verification.
Expose via ngrok or similar for external access.

**Python (Flask):**
```bash
cd webhook-receiver
pip install -r requirements.txt
WEBHOOK_SECRET=your-secret python app.py
```

**Node.js (Express):**
```bash
cd webhook-receiver
npm install
WEBHOOK_SECRET=your-secret node server.js
```

### telemetry-stream

Stream telemetry via SSE.

```bash
python telemetry-stream.py [base_url] [api_key]
```

## See Also

- [Integration Guide](../../docs/integration/README.md)
- [Quick Start](../../docs/integration/quickstart.md)
