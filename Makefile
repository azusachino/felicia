# felicia task runner.
# Runtimes (go, bun) come from mise (shimmed onto PATH). System tools
# (golangci-lint, goose) come from the nix flake; NIX_RUN prefixes those so
# targets work both inside `nix develop` and outside it.
NIX_RUN := $(if $(IN_NIX_SHELL),,nix develop --command )
GO      ?= go

# Whether any Go sources exist yet. The skeleton has none during the research
# phase, so Go targets no-op cleanly until the first package is written.
GO_FILES := $(shell find . -name '*.go' -not -path './vendor/*' -not -path './.git/*' -not -path '*/node_modules/*' -print -quit 2>/dev/null)

# Every Go module must be checked; the root module alone does not traverse the
# independent workspace modules.
GO_MODULES = . apps/core apps/runtime apps/providers apps/apiserver

DATABASE_DSN ?= postgres://postgres:password@localhost:5432/felicia?sslmode=disable
PORT ?= 8080
CACHE_ADDR ?= localhost:6379

COMPOSE ?= $(shell \
	if command -v podman-compose >/dev/null 2>&1; then echo podman-compose; \
	elif command -v docker >/dev/null 2>&1; then echo docker compose; \
	else echo ''; fi)

.PHONY: help fmt vet lint test check build validate tidy db-up db-down migrate seed dev test-workflow mock-up mock-down web-install web-check docs docs-build share share-down

help: ## List targets
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | \
		awk 'BEGIN{FS=":.*?## "}{printf "  \033[36m%-13s\033[0m %s\n", $$1, $$2}'

fmt: ## Format Go code
	@if [ -z "$(GO_FILES)" ]; then echo "fmt: no Go packages yet, skipping"; else set -e; for module in $(GO_MODULES); do (cd $$module && $(GO) fmt $$( $(GO) list ./... | grep -v '/node_modules/' )); done; fi

vet: ## Run go vet
	@if [ -z "$(GO_FILES)" ]; then echo "vet: no Go packages yet, skipping"; else set -e; for module in $(GO_MODULES); do (cd $$module && $(GO) vet $$( $(GO) list ./... | grep -v '/node_modules/' )); done; fi

lint: ## Lint Go (golangci-lint, from nix)
	@if [ -z "$(GO_FILES)" ]; then echo "lint: no Go packages yet, skipping"; else set -e; for module in $(GO_MODULES); do (cd $$module && $(NIX_RUN)golangci-lint run); done; fi

test: ## Run Go tests with race detector + coverage
	@if [ -z "$(GO_FILES)" ]; then echo "test: no Go packages yet, skipping"; else set -e; for module in $(GO_MODULES); do (cd $$module && $(GO) test -race -cover $$( $(GO) list ./... | grep -v '/node_modules/' )); done; fi

check: fmt vet lint test test-features ## Pre-commit gate

build: ## Build all binaries
	@if [ -z "$(GO_FILES)" ]; then echo "build: no Go packages yet, skipping"; else set -e; for module in $(GO_MODULES); do (cd $$module && $(GO) build $$( $(GO) list ./... | grep -v '/node_modules/' )); done; fi

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
		DATABASE_DRIVER=postgres DATABASE_DSN="$(DATABASE_DSN)" PORT="$(PORT)" CACHE_ADDR="$(CACHE_ADDR)" "$$api_bin" & \
		api_pid=$$!; \
		cleanup() { kill $$api_pid 2>/dev/null || true; rm -f "$$api_bin"; }; \
		trap cleanup EXIT INT TERM; \
		for attempt in $$(seq 1 30); do \
			if curl --fail --silent -X POST -H 'Content-Type: application/json' -d '{"id":"0190cbde-f300-7000-8000-000000000000"}' "http://localhost:$(PORT)/api/admin/journals" >/dev/null; then break; fi; \
			if [ "$$attempt" = 30 ]; then echo "API did not become ready" >&2; exit 1; fi; \
			sleep 1; \
		done; \
		DATABASE_DSN="$(DATABASE_DSN)" SEED_API_BASE="http://localhost:$(PORT)" $(MAKE) seed; \
		if [ ! -d apps/web-public/node_modules ]; then $(MAKE) web-install; fi; \
		cd apps/web-public && bun run dev

mock-up: ## Start the mock Dawarich+Immich upstream in the background (:8099)
	nohup uv run python scripts/mock_upstream.py > /tmp/felicia-mock.log 2>&1 & echo "mock up on :8099 (log: /tmp/felicia-mock.log)"

mock-down: ## Stop the mock upstream
	@pkill -f scripts/mock_upstream.py && echo "mock stopped" || echo "no mock running"

test-api: ## Run Python-based E2E API integration tests (requires running server)
	uv run python scripts/test_api.py

test-workflow: ## Run full journey workflow against disposable SQLite
	uv run python scripts/test_journey_workflow.py --start-server

test-features: ## Run offline Python feature-contract tests
	uv run python -m unittest discover -s tests

web-install: ## Install frontend deps (bun, from mise)
	cd apps/web-public && bun install

web-dev: ## Run frontend dev server (bun + vite)
	cd apps/web-public && bun run dev

web-build: ## Build frontend for production (bun + vite)
	cd apps/web-public && bun run build

web-check: ## Frontend typecheck + lint + format check
	cd apps/web-public && bun run check

# Docs preview (uv-managed env, isolated from Go/bun). Binds 0.0.0.0 so it is
# reachable over SSH — forward with `ssh -L 8000:localhost:8000 <host>`.
docs: ## Live-preview docs in the browser (uv + mkdocs-material)
	uv run --group docs mkdocs serve -a 0.0.0.0:8000

docs-build: ## Build the static docs site into ./site
	uv run --group docs mkdocs build

# Share the running demo to a friend over an ephemeral Cloudflare tunnel.
# Builds the SPA, brings the whole stack up under compose (db+cache+api+web),
# migrates+seeds via the host, then fronts it all with a trycloudflare.com URL.
# No CF account/domain needed; only /api/v1 is exposed (admin stays host-only).
share: ## Build + serve the full stack behind a quick Cloudflare tunnel (share to a friend)
	@test -n "$(COMPOSE)" || (echo "No container compose command found (install podman-compose or Docker Compose)" >&2; exit 1)
	@set -e; \
		echo ">> building SPA (web/dist)"; \
		$(MAKE) web-build; \
		echo ">> starting db + cache"; \
		$(COMPOSE) -f deploy/compose.yaml up -d db cache; \
		echo ">> waiting for db"; \
		for i in $$(seq 1 30); do \
			if $(COMPOSE) -f deploy/compose.yaml exec -T db pg_isready -U postgres -d felicia >/dev/null 2>&1; then break; fi; \
			[ "$$i" = 30 ] && { echo "db not ready" >&2; exit 1; }; sleep 1; \
		done; \
		echo ">> migrating"; \
		DATABASE_DSN="$(DATABASE_DSN)" $(MAKE) migrate; \
		echo ">> building + starting api"; \
		$(COMPOSE) -f deploy/compose.yaml up -d --build api; \
		echo ">> waiting for api"; \
		for i in $$(seq 1 60); do \
			if curl --fail --silent -X POST -H 'Content-Type: application/json' -d '{"id":"0190cbde-f300-7000-8000-000000000000"}' "http://localhost:8080/api/admin/journals" >/dev/null 2>&1; then break; fi; \
			[ "$$i" = 60 ] && { echo "api not ready" >&2; exit 1; }; sleep 1; \
		done; \
		echo ">> seeding"; \
		DATABASE_DSN="$(DATABASE_DSN)" SEED_API_BASE="http://localhost:8080" $(MAKE) seed; \
		echo ">> starting web + cloudflared"; \
		$(COMPOSE) -f deploy/compose.yaml up -d web cloudflared; \
		echo ">> waiting for tunnel URL"; \
		url=""; \
		for i in $$(seq 1 30); do \
			url=$$($(COMPOSE) -f deploy/compose.yaml logs cloudflared 2>&1 | grep -oE 'https://[a-z0-9-]+\.trycloudflare\.com' | head -1); \
			[ -n "$$url" ] && break; sleep 1; \
		done; \
		echo ""; \
		if [ -n "$$url" ]; then echo "  share this URL:  $$url"; \
		else echo "  tunnel URL not ready — check: $(COMPOSE) -f deploy/compose.yaml logs cloudflared"; fi; \
		echo "  local preview:   http://localhost:8081"; \
		echo "  stop sharing:    make share-down"

share-down: ## Stop the shared stack (api, web, cloudflared); keeps db+cache+data
	@test -n "$(COMPOSE)" || (echo "No container compose command found" >&2; exit 1)
	$(COMPOSE) -f deploy/compose.yaml rm -sf api web cloudflared
