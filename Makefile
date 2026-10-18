COMPOSE := $(shell if command -v docker > /dev/null 2>&1 && docker compose version > /dev/null 2>&1; then echo "docker compose"; elif command -v docker-compose > /dev/null 2>&1; then echo "docker-compose"; elif command -v podman-compose > /dev/null 2>&1; then echo "podman-compose"; else echo "podman-compose"; fi)

COMPOSE_DEV := $(COMPOSE) -f docker-compose.dev.yml
COMPOSE_RELEASE := $(COMPOSE) -f docker-compose.yml

CONTAINER_RT := $(shell if command -v docker > /dev/null 2>&1; then echo "docker"; else echo "podman"; fi)

.PHONY: dev build up up-release down reset logs restart migrate migrate-down frontend-dev test test-e2e db-shell publish help

up: ## Start all services (dev stack)
	PECULIUM_COMMIT=$$(git rev-parse --short HEAD 2>/dev/null) PECULIUM_BUILT_AT=$$(date -u +%Y-%m-%dT%H:%M:%SZ) $(COMPOSE_DEV) up --build -d

up-release: ## Start release stack, force-pulling images from GHCR
	$(COMPOSE_RELEASE) pull
	$(COMPOSE_RELEASE) up -d --force-recreate

down: ## Stop all services (dev stack)
	$(COMPOSE_DEV) down

reset: ## Stop all services and delete data volumes (fresh start, dev stack)
	$(COMPOSE_DEV) down -v

logs: ## Follow logs (dev stack)
	$(COMPOSE_DEV) logs -f

restart: ## Restart all services (dev stack)
	$(COMPOSE_DEV) restart

migrate: ## Run DB migrations (dev stack)
	$(COMPOSE_DEV) exec backend /server migrate

migrate-down: ## Rollback DB migrations
	migrate -path backend/migrations -database "postgres://peculium:peculium@localhost:5432/peculium?sslmode=disable" down 1

frontend-dev: ## Run frontend in dev mode
	cd frontend && npm run dev

build: ## Build all images (dev stack)
	$(COMPOSE_DEV) build

GO_TEST_CMDS := go build ./... && go vet ./... && go test ./...

test: ## Run backend tests (build, vet, test; host Go if available, else golang:1.27-alpine)
	@if command -v go > /dev/null 2>&1; then \
	  cd backend && $(GO_TEST_CMDS); \
	else \
	  $(CONTAINER_RT) run --rm -v "$(CURDIR)/backend:/app" -w /app golang:1.27-alpine sh -c '$(GO_TEST_CMDS)'; \
	fi

test-e2e: ## Run end-to-end API tests on an isolated stack (EPIC A, portfolio import/export, EPIC K filters/drill-down, EPIC B allocations, FX history)
	$(COMPOSE) -p peculium-test -f docker-compose.test.yml up -d --build
	@echo "Attendo il backend su http://localhost:8081..."
	@until curl -s -o /dev/null http://localhost:8081/api/v1/health/prices; do sleep 1; done
	@rc=0; \
	./tests/test-epic-a.sh http://localhost:8081 && \
	./tests/test-portfolio-io.sh http://localhost:8081 && \
	./tests/test-epic-k.sh http://localhost:8081 && \
	./tests/test-epic-b.sh http://localhost:8081 && \
	./tests/test-fx-history.sh http://localhost:8081 || rc=$$?; \
	$(COMPOSE) -p peculium-test -f docker-compose.test.yml down -v; \
	exit $$rc

db-shell: ## Connect to postgres (dev stack)
	$(COMPOSE_DEV) exec postgres psql -U peculium peculium

RC_REF ?= develop
RC_VERSION ?=

publish: ## Dispatch an image build from a branch (RC_REF=develop, RC_VERSION=extra tag)
	gh workflow run publish-images.yml --ref $(RC_REF) -f version=$(RC_VERSION)
	@echo "Dispatched from $(RC_REF). :$(RC_REF) is published; run it with:"
	@echo "  PECULIUM_VERSION=$(RC_REF) docker compose pull && PECULIUM_VERSION=$(RC_REF) docker compose up -d"
	@if [ -n "$(RC_VERSION)" ]; then echo "  (also tagged $(RC_VERSION))"; fi

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'
