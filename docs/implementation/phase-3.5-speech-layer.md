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

Speech providers (ElevenLabs, Azure) are integrated via HTTP. Configure URLs to point to external services or proxy endpoints that wrap vendor SDKs.

## Future Work

- Streaming audio (full duplex)
- Conversation Manager for follow-up context
- Edge-side Audio Gateway for low latency
- Voice personality / emotional synthesis

## Related Documents

- [Speech Layer Architecture](../architecture/speech-layer.md)
- [Cognitive Gateway](../architecture/cognitive-gateway.md)
- [Robot Adapter Contract](../adapters/robot-adapter-contract.md)
