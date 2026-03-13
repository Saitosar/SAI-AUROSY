# Gemini Adapter

The Gemini Adapter is an HTTP service that implements the SAI AUROSY Cognitive Gateway contract for speech (STT, TTS, Intent) using the Google Gemini API. It enables voice-enabled robots with multilingual support.

## Overview

| Endpoint | SAI AUROSY Contract | Gemini API |
|----------|---------------------|------------|
| `POST /transcribe` | Audio → text | Gemini Audio Understanding |
| `POST /synthesize` | Text → audio | Gemini 2.5 Flash TTS |
| `POST /understand-intent` | Text → intent JSON | Gemini generateContent |

## Prerequisites

- [Google AI Studio API key](https://aistudio.google.com/app/apikey)
- Docker and Docker Compose

## Setup

### 1. Create `.env` file

Copy `.env.example` to `.env` and add your API key:

```bash
cp .env.example .env
# Edit .env and set GEMINI_API_KEY=AIza...
```

Or create `.env` manually (already in `.gitignore`):

```
GEMINI_API_KEY=AIza...
```

Replace `AIza...` with your API key from [Google AI Studio](https://aistudio.google.com/app/apikey).

### 2. Start the stack

```bash
docker compose up -d
```

Docker Compose loads `.env` and passes `GEMINI_API_KEY` to the gemini-adapter service. The Control Plane is configured to use the adapter via `COGNITIVE_PROVIDER=http` and the HTTP URLs.

### 3. Verify

- **Adapter health:** `curl http://localhost:8001/health` — returns `{"status":"ok","configured":true}` when the key is set
- **Operator Console:** Open http://localhost:3000, select a robot with `speech` capability, use the Speech Test (Record → Stop) to test

## Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `GEMINI_API_KEY` | Yes (for speech) | Google AI API key. Alternatively `GOOGLE_API_KEY`. |
| `PORT` | No | HTTP port (default: 8001) |

## API Key Storage

| Method | Location | Use case |
|--------|----------|----------|
| **.env file** | Project root, `GEMINI_API_KEY=...` | Local development (recommended) |
| **Docker Compose** | `environment: GEMINI_API_KEY: ${GEMINI_API_KEY}` | Loads from `.env`; do not hardcode the key in `docker-compose.yml` |
| **Vault** | Add `GEMINI_API_KEY` to Vault secret | Production (adapter would need secrets support) |
| **AWS Secrets Manager** | Same | Production |

**Security:** Never commit `.env`. For production, use Vault, AWS Secrets Manager, or inject via orchestration (e.g. Kubernetes secrets).

## Disabling Gemini

To use the mock provider (no real speech) instead:

1. In `docker-compose.yml`, set `COGNITIVE_PROVIDER: mock` for the control-plane service
2. Remove or comment out `COGNITIVE_HTTP_TRANSCRIBE_URL`, `COGNITIVE_HTTP_SYNTHESIZE_URL`, `COGNITIVE_HTTP_INTENT_URL`
3. Remove `gemini-adapter` from control-plane `depends_on`
4. Optionally stop the adapter: `docker compose stop gemini-adapter`

## Troubleshooting

| Symptom | Cause | Fix |
|---------|-------|-----|
| Speech Test shows no result | Adapter returns 503 | Set `GEMINI_API_KEY` in `.env` |
| "GEMINI_API_KEY not set" | Key not passed to container | Ensure `.env` exists in project root; run `docker compose up -d` |
| Transcription empty | Unsupported audio format | Browser sends webm; adapter converts to mp3. Ensure ffmpeg is present (included in Docker image). |
| Control Plane fails to start | Adapter not ready | Adapter starts without key; if it crashes, check logs: `docker compose logs gemini-adapter` |

## Related

- [Phase 3.5 Speech Layer](../implementation/phase-3.5-speech-layer.md)
- [Speech Layer Architecture](../architecture/speech-layer.md)
- [Secrets Management](../implementation/secrets-management.md)
