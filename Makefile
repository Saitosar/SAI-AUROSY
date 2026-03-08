.PHONY: build run docker e2e

build:
	go build -o bin/control-plane ./cmd/control-plane

run: build
	@echo "Start NATS: docker run -d -p 4222:4222 nats:2-alpine"
	@echo "Start AGIBOT adapter: AGIBOT_MOCK=1 NATS_URL=nats://localhost:4222 python pkg/adapters/agibot/adapter.py"
	@echo "Start Unitree adapter: UNITREE_MOCK=1 NATS_URL=nats://localhost:4222 python pkg/adapters/unitree/adapter.py"
	./bin/control-plane

docker:
	docker compose build
	docker compose up -d

e2e:
	@echo "E2E: docker compose up, then open http://localhost:3000"
	@echo "  - Click Safe Stop on x1-001 (AGIBOT) or go2-001 (Unitree)"
