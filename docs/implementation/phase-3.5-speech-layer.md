# Phase 3.5 — Speech Layer

## Overview

The Speech Layer extends the Cognitive Gateway with speech-to-text (STT), text-to-speech (TTS), and intent understanding. It enables voice-enabled robots with multilingual support (uz, en, ru, az, ar).

## Implemented Features

### 1. Cognitive Gateway Extensions

- `Transcribe(ctx, req) (*TranscribeResult, error)` — STT
- `Synthesize(ctx, req) (*SynthesizeResult, error)` — TTS
- `UnderstandIntent(ctx, req) (*IntentResult, error)` — LLM intent extraction

### 2. Providers

- **Mock** — Returns empty/placeholder results (default)
- **HTTP** — Calls external services via REST

### 3. Configuration

| Env var | Purpose |
|---------|---------|
| `COGNITIVE_HTTP_TRANSCRIBE_URL` | STT service URL |
| `COGNITIVE_HTTP_SYNTHESIZE_URL` | TTS service URL |
| `COGNITIVE_HTTP_INTENT_URL` | Intent extraction service URL |

Config file (when `COGNITIVE_CONFIG_PATH` is set) supports `transcribe_url`, `synthesize_url`, `understand_intent_url`.

### 4. Conversation Catalog

Separate from motion scenarios. API:

- `GET /v1/conversations` — List (tenant-filtered)
- `POST /v1/conversations` — Create (admin)
- `GET /v1/conversations/{id}` — Get
- `PUT /v1/conversations/{id}` — Update (admin)
- `DELETE /v1/conversations/{id}` — Delete (admin)

Model: `id`, `intent`, `name`, `description`, `response_template`, `response_provider_url`, `supported_languages`, `tenant_id`.

### 5. REST API — Cognitive Speech

- `POST /v1/cognitive/transcribe` — Body: `robot_id`, `audio_base64`, `language` (optional)
- `POST /v1/cognitive/synthesize` — Body: `robot_id`, `text`, `language`
- `POST /v1/cognitive/understand-intent` — Body: `robot_id`, `text`, `language` (optional), `context` (optional)

All require authentication and tenant isolation (robot must belong to tenant).

### 6. NATS Topics

- `audio.robots.{id}.input` / `audio.robots.{id}.output`
- `speech.robots.{id}.transcript`, `speech.robots.{id}.intent`, `speech.robots.{id}.response`

### 7. Speech Pipeline

`pkg/control-plane/speech/pipeline.go` — Full flow: Transcribe → UnderstandIntent → ConversationCatalog.GetByIntent → ResolveResponse → Synthesize → Publish.

#### Speech Pipeline Integration

The pipeline is wired into the Control Plane with two entry points:

1. **REST API** — `POST /v1/cognitive/process-audio` accepts `robot_id` and `audio_base64`. Runs the full pipeline and returns `transcript`, `language`, `intent`, `parameters`, `response`, `audio_base64`. Used by the Operator Console Speech Test UI for browser-based testing.

2. **NATS subscriber** — Subscribes to `audio.robots.*.input`. When a robot adapter publishes raw audio, the pipeline runs automatically. TTS output is published to `audio.robots.{id}.output` for the adapter to play on the robot speaker.

**Configuration for real STT/TTS:** Use `COGNITIVE_PROVIDER=http` with `COGNITIVE_HTTP_TRANSCRIBE_URL`, `COGNITIVE_HTTP_SYNTHESIZE_URL`, and `COGNITIVE_HTTP_INTENT_URL`. With the mock provider, Transcribe and Synthesize return empty; the pipeline flow can still be verified.

**Default conversations:** Migration `000025_seed_conversations` and in-memory seed add `find_store`, `greeting`, `goodbye` with shared templates.

### 8. Language Support

`pkg/control-plane/speech/language.go` — Supported: uz, en, ru, az, ar. Tier 1: uz, en, ru. Tier 2: az, ar. Low-confidence threshold for "ask repeat" flow.

### 9. Capability

`CapSpeech = "speech"` in `pkg/hal/capabilities.go`. Robots with mic + speaker declare this.

## Implementation

- `pkg/control-plane/cognitive/` — gateway, types, config, providers (extended)
- `pkg/control-plane/conversations/` — store, catalog, memorystore, sqlstore
- `pkg/control-plane/speech/` — pipeline, language
- `pkg/telemetry/bus.go` — audio/speech topic helpers
- `pkg/control-plane/registry/migrations/000024_create_conversations.*.sql`

## External Providers

Speech providers (ElevenLabs, Azure, Google Gemini) are integrated via HTTP. Configure URLs to point to external services or proxy endpoints that wrap vendor SDKs.

### Gemini Adapter

The [Gemini Adapter](../integration/gemini-adapter.md) implements the Cognitive Gateway contract using Google Gemini API:

- **STT** — Gemini Audio Understanding (`gemini-2.0-flash`)
- **TTS** — Gemini 2.5 Flash TTS (`gemini-2.5-flash-preview-tts`)
- **Intent** — Gemini generateContent with structured JSON

Set `GEMINI_API_KEY` in `.env` and run `docker compose up -d`. The default `docker-compose.yml` wires the adapter when `COGNITIVE_PROVIDER=http`.

## Future Work

- Streaming audio (full duplex)
- Conversation Manager for follow-up context
- Edge-side Audio Gateway for low latency
- Voice personality / emotional synthesis

## Related Documents

- [Speech Layer Architecture](../architecture/speech-layer.md)
- [Cognitive Gateway](../architecture/cognitive-gateway.md)
- [Gemini Adapter](../integration/gemini-adapter.md)
- [Robot Adapter Contract](../adapters/robot-adapter-contract.md)
