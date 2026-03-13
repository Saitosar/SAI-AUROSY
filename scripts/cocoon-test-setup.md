# Cocoon Test Setup (fake-ton)

This guide describes how to run Cocoon in test mode (fake-ton) and verify SAI-AUROSY integration.

## Prerequisites

- Linux (Cocoon is built for Linux)
- For Option B: Intel TDX-capable CPU, NVIDIA GPU (H100+), QEMU with TDX support

## Clone and Build Cocoon

```bash
git clone --recursive https://github.com/TelegramMessenger/cocoon.git
cd cocoon
```

`cocoon-launch` auto-builds on first run. For manual build:

```bash
./scripts/cocoon-launch --build-dir build --just-build
```

## Option A: Local-all (No TDX/GPU)

Runs all components locally with fake-TON. Tests HTTP connectivity; uses fake HTTP backend for inference (protocol test only).

```bash
cd cocoon
./scripts/cocoon-launch --local-all
```

- Client listens on port 10000
- No TDX or GPU required
- Suitable for verifying SAI-AUROSY can reach Cocoon client

## Option B: Test + fake-ton (Full Stack)

Runs proxy, worker, and client in TDX VMs with fake TON. Real LLM inference via vLLM.

### Build Test Image (one-time, ~10–30 min)

```bash
cd cocoon
./scripts/build-image test
```

### Start Components (three terminals)

**Terminal 1 – Proxy:**
```bash
cd cocoon
./scripts/cocoon-launch --test --fake-ton scripts/proxy.conf
```

**Terminal 2 – Worker:**
```bash
cd cocoon
export HF_TOKEN=hf_...   # Hugging Face token for model download
./scripts/cocoon-launch --test --fake-ton --gpu 0000:01:00.0 scripts/worker.conf
```

Find GPU PCI address: `lspci | grep -i nvidia`

**Terminal 3 – Client:**
```bash
cd cocoon
./scripts/cocoon-launch --test --fake-ton scripts/client.conf
```

Client exposes HTTP on port 10000.

### Verify Cocoon Client

```bash
curl -X POST http://localhost:10000/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{"model":"Qwen/Qwen3-0.6B","messages":[{"role":"user","content":"hi"}],"max_tokens":10}'
```

Expected: JSON response with `choices[0].message.content`.

## Model for Test

Cocoon test configs (`scripts/client.conf`, `scripts/proxy.conf`) use **Qwen/Qwen3-0.6B**. SAI-AUROSY must use the same model when connecting:

```
COGNITIVE_COCOON_MODEL=Qwen/Qwen3-0.6B
```

## Connect SAI-AUROSY

1. Ensure Cocoon client is running (Option A or B).
2. Set environment:
   ```bash
   export COGNITIVE_PROVIDER=cocoon
   export COGNITIVE_COCOON_CLIENT_URL=http://localhost:10000
   export COGNITIVE_COCOON_MODEL=Qwen/Qwen3-0.6B
   ```
3. Start SAI-AUROSY Control Plane:
   ```bash
   go run ./cmd/control-plane
   ```
   Or with Docker:
   ```bash
   docker compose -f docker-compose.yml -f docker-compose.cocoon-test.yml up -d
   ```

## Verification

Run the verification script:

```bash
./scripts/verify-cocoon-integration.sh
```

See [Cocoon Integration](../docs/architecture/cocoon-integration.md) for more details.
