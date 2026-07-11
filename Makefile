# felicia task runner.
# Runtimes (go, bun) come from mise (shimmed onto PATH). System tools
# (golangci-lint, goose) come from the nix flake; NIX_RUN prefixes those so
# targets work both inside `nix develop` and outside it.
NIX_RUN := $(if $(IN_NIX_SHELL),,nix develop --command )
GO      ?= go

# Whether any Go sources exist yet. The skeleton has none during the research
# phase, so Go targets no-op cleanly until the first package is written.
GO_FILES := $(shell find . -name '*.go' -not -path './vendor/*' -not -path './.git/*' -not -path '*/node_modules/*' -print -quit 2>/dev/null)

# Real module packages, excluding stray Go files vendored inside web/node_modules.
GO_PKGS = $(shell $(GO) list ./... | grep -v /node_modules/)

DATABASE_DSN ?= postgres://postgres:password@localhost:5432/felicia?sslmode=disable
PORT ?= 8080
CACHE_ADDR ?= localhost:6379

COMPOSE ?= $(shell \
	if command -v podman-compose >/dev/null 2>&1; then echo podman-compose; \
	elif command -v docker >/dev/null 2>&1; then echo docker compose; \
	else echo ''; fi)

.PHONY: help fmt vet lint test check build validate tidy db-up db-down migrate seed dev mock-up mock-down web-install web-check docs docs-build

help: ## List targets
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | \
		awk 'BEGIN{FS=":.*?## "}{printf "  \033[36m%-13s\033[0m %s\n", $$1, $$2}'

fmt: ## Format Go code
	@if [ -z "$(GO_FILES)" ]; then echo "fmt: no Go packages yet, skipping"; else $(GO) fmt ./...; fi

vet: ## Run go vet
	@if [ -z "$(GO_FILES)" ]; then echo "vet: no Go packages yet, skipping"; else $(GO) vet $(GO_PKGS); fi

lint: ## Lint Go (golangci-lint, from nix)
	@if [ -z "$(GO_FILES)" ]; then echo "lint: no Go packages yet, skipping"; else $(NIX_RUN)golangci-lint run; fi

test: ## Run Go tests with race detector + coverage
	@if [ -z "$(GO_FILES)" ]; then echo "test: no Go packages yet, skipping"; else $(GO) test -race -cover $(GO_PKGS); fi

check: fmt vet lint test ## Pre-commit gate

build: ## Build all binaries
	@if [ -z "$(GO_FILES)" ]; then echo "build: no Go packages yet, skipping"; else $(GO) build ./...; fi

# Pre-PR gate. Frontend (web-check) and migration smoke join here once web/ and
# migrations/ have content.
validate: check build ## Pre-PR gate

tidy: ## Tidy go modules
	$(GO) mod tidy

db-up: ## Start local Postgres+PostGIS and Valkey (deploy/compose.yaml)
	@test -n "$(COMPOSE)" || (echo "No container compose command found (install podman-compose or Docker Compose)" >&2; exit 1)
	$(COMPOSE) -f deploy/compose.yaml up -d

db-down: ## Stop the local dev containers (keeps the pgdata volume)
	@test -n "$(COMPOSE)" || (echo "No container compose command found (install podman-compose or Docker Compose)" >&2; exit 1)
	$(COMPOSE) -f deploy/compose.yaml down

migrate: ## Apply DB migrations (goose, from nix) — needs DATABASE_DSN
	$(NIX_RUN)goose -dir migrations postgres "$(DATABASE_DSN)" up

seed: ## Seed the database with sample data (uv run, psycopg) — needs DATABASE_DSN
	uv run --group dev python scripts/seed.py

dev: ## Start the complete local stack, seed mock data, and serve the web app
	@set -e; \
		$(MAKE) db-up; \
		DATABASE_DSN="$(DATABASE_DSN)" $(MAKE) migrate; \
		api_bin="/tmp/felicia-api-$$$$"; \
		go build -o "$$api_bin" ./cmd/api; \
		DATABASE_DSN="$(DATABASE_DSN)" PORT="$(PORT)" CACHE_ADDR="$(CACHE_ADDR)" "$$api_bin" & \
		api_pid=$$!; \
		cleanup() { kill $$api_pid 2>/dev/null || true; rm -f "$$api_bin"; }; \
		trap cleanup EXIT INT TERM; \
		for attempt in $$(seq 1 30); do \
			if curl --fail --silent -X POST -H 'Content-Type: application/json' -d '{"id":"0190cbde-f300-7000-8000-000000000000"}' "http://localhost:$(PORT)/api/admin/journals" >/dev/null; then break; fi; \
			if [ "$$attempt" = 30 ]; then echo "API did not become ready" >&2; exit 1; fi; \
			sleep 1; \
		done; \
		DATABASE_DSN="$(DATABASE_DSN)" SEED_API_BASE="http://localhost:$(PORT)" $(MAKE) seed; \
		if [ ! -d web/node_modules ]; then $(MAKE) web-install; fi; \
		cd web && bun run dev

mock-up: ## Start the mock Dawarich+Immich upstream in the background (:8099)
	nohup uv run python scripts/mock_upstream.py > /tmp/felicia-mock.log 2>&1 & echo "mock up on :8099 (log: /tmp/felicia-mock.log)"

mock-down: ## Stop the mock upstream
	@pkill -f scripts/mock_upstream.py && echo "mock stopped" || echo "no mock running"

test-api: ## Run Python-based E2E API integration tests (requires running server)
	python scripts/test_api.py

web-install: ## Install frontend deps (bun, from mise)
	cd web && bun install

web-dev: ## Run frontend dev server (bun + vite)
	cd web && bun run dev

web-build: ## Build frontend for production (bun + vite)
	cd web && bun run build

web-check: ## Frontend typecheck + lint + format check
	cd web && bun run check

# Docs preview (uv-managed env, isolated from Go/bun). Binds 0.0.0.0 so it is
# reachable over SSH — forward with `ssh -L 8000:localhost:8000 <host>`.
docs: ## Live-preview docs in the browser (uv + mkdocs-material)
	uv run --group docs mkdocs serve -a 0.0.0.0:8000

docs-build: ## Build the static docs site into ./site
	uv run --group docs mkdocs build
