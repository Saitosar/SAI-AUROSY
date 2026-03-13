# Gemini Live Adapter

WebSocket proxy for [Gemini Live API](https://ai.google.dev/gemini-api/docs/live-api). Enables real-time voice streaming in SAI AUROSY Operator Console.

## Usage

Set `GEMINI_API_KEY` in `.env` and run:

```bash
docker compose up -d gemini-live-adapter operator-console
```

In Operator Console, open a robot with speech capability, switch to **Stream** mode in Speech Test, and click **Start**.

## Protocol

- **Input:** Raw PCM 16-bit, 16 kHz, little-endian (binary WebSocket messages)
- **Output:** Raw PCM 16-bit, 24 kHz (binary) + JSON events `{type, text}` for transcriptions

## Configuration

| Env | Default |
|-----|---------|
| `GEMINI_API_KEY` | Required |
| `PORT` | 8002 |
| `MODEL` | gemini-2.5-flash-native-audio-preview-12-2025 |
