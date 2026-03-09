.PHONY: build build-workforce run docker test test-integration e2e

build:
	go build -o bin/control-plane ./cmd/control-plane

build-workforce:
	go build -o bin/workforce ./cmd/workforce

run: build
	@echo "Start NATS: docker run -d -p 4222:4222 nats:2-alpine"
	@echo "Start AGIBOT adapter: AGIBOT_MOCK=1 NATS_URL=nats://localhost:4222 python pkg/adapters/agibot/adapter.py"
	@echo "Start Unitree adapter: UNITREE_MOCK=1 NATS_URL=nats://localhost:4222 python pkg/adapters/unitree/adapter.py"
	./bin/control-plane

docker:
	docker compose build
	docker compose up -d

test:
	go test -v ./...

test-integration: test
	@echo "Integration tests (Task Runner, Edge Agent) require NATS at nats://localhost:4222"
	@echo "Run: docker run -d -p 4222:4222 --name nats-e2e nats:2-alpine"
	@echo "Tests will skip if NATS is unavailable"

e2e: docker
	@echo "Waiting for services to be ready..."
	@sleep 5
	bash scripts/e2e.sh

e2e-auth:
	@echo "Building and starting E2E stack with auth (PostgreSQL + API key)..."
	docker compose -f docker-compose.yml -f docker-compose.e2e.yml build
	docker compose -f docker-compose.yml -f docker-compose.e2e.yml up -d
	@echo "Waiting for services to be ready..."
	@sleep 15
	E2E_API_KEY=e2e-api-key bash scripts/e2e.sh

e2e-tenant:
	@echo "Running E2E multi-tenant (requires e2e stack with auth)..."
	@echo "Start with: make e2e-auth (or docker compose -f docker-compose.yml -f docker-compose.e2e.yml up -d)"
	@sleep 2
	bash scripts/e2e-tenant.sh

load-test:
	@echo "Running load test (requires stack with auth: make e2e-auth)"
	@echo "Default: 50 RPS, 10s. Override: RATE=100 DURATION=30s make load-test"
	@sleep 2
	E2E_API_KEY=e2e-api-key bash scripts/load/vegeta.sh

load-test-no-auth:
	@echo "Running load test without auth (requires make docker)"
	bash scripts/load/vegeta.sh
