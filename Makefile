# felicia task runner.
# The repository toolchain comes from the nix flake. These wrappers make the
# targets work both inside `nix develop` and from a normal shell.
NIX_RUN := $(if $(IN_NIX_SHELL),,nix develop --command )
UV_RUN  := $(if $(IN_NIX_SHELL),uv,nix develop --command uv)
GO      ?= $(if $(IN_NIX_SHELL),go,nix develop --command go)
BUN     ?= $(if $(IN_NIX_SHELL),bun,nix develop --command bun)

# Whether any Go sources exist yet. Keep the check so the early repository
# skeleton can still run the non-Go targets cleanly.
GO_FILES := $(shell find . -name '*.go' -not -path './vendor/*' -not -path './.git/*' -not -path '*/node_modules/*' -print -quit 2>/dev/null)

# Every Go module must be checked; the root module alone does not traverse the
# independent workspace modules.
GO_MODULES = apps/felicia-core apps/felicia-runtime apps/felicia-providers apps/felicia-publication apps/felicia-server apps/felicia-cli

DATABASE_DSN ?= postgres://postgres:password@localhost:5432/felicia?sslmode=disable
PORT ?= 8080
CACHE_ADDR ?= localhost:6379

COMPOSE ?= $(shell \
	if command -v podman-compose >/dev/null 2>&1; then echo podman-compose; \
	elif command -v docker >/dev/null 2>&1; then echo docker compose; \
	else echo ''; fi)

.PHONY: help fmt fmt-check vet lint test test-api test-features test-sqlite test-postgres check build cli-build experiment-intake journey-local validate tidy db-up db-down migrate seed admin dev dev-sqlite dev-postgres test-workflow test-workflow-postgres test-admin-e2e sqlc mock-up mock-down web-install web-check web-build web-admin-check web-admin-build site-build site-verify static-build static-validate static-publish pages-workflow-validate fork-smoke pages-preview pages-down docs docs-build share share-down

help: ## List targets
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | \
		awk 'BEGIN{FS=":.*?## "}{printf "  \033[36m%-13s\033[0m %s\n", $$1, $$2}'

fmt: ## Format Go and frontend code
	$(UV_RUN) run python scripts/format.py

fmt-check: ## Check Go and frontend formatting without modifying files
	$(UV_RUN) run python scripts/format.py --check

vet: ## Run go vet
	@if [ -z "$(GO_FILES)" ]; then echo "vet: no Go packages yet, skipping"; else set -e; for module in $(GO_MODULES); do (cd $$module && $(GO) vet $$( $(GO) list ./... | grep -v '/node_modules/' )); done; fi

lint: ## Lint Go (golangci-lint, from nix)
	@if [ -z "$(GO_FILES)" ]; then echo "lint: no Go packages yet, skipping"; else set -e; for module in $(GO_MODULES); do (cd $$module && $(NIX_RUN)golangci-lint run); done; fi

test: ## Run Go tests with race detector + coverage
	@if [ -z "$(GO_FILES)" ]; then echo "test: no Go packages yet, skipping"; else set -e; for module in $(GO_MODULES); do (cd $$module && $(GO) test -race -cover $$( $(GO) list ./... | grep -v '/node_modules/' )); done; fi

check: fmt-check vet lint test test-features ## Pre-commit gate

build: ## Build all binaries
	@if [ -z "$(GO_FILES)" ]; then echo "build: no Go packages yet, skipping"; else set -e; for module in $(GO_MODULES); do (cd $$module && $(GO) build $$( $(GO) list ./... | grep -v '/node_modules/' )); done; fi

cli-build: ## Build the felicia-cli executable into bin/
	@mkdir -p bin
	$(GO) build -o bin/felicia-cli ./apps/felicia-cli/cmd/felicia

experiment-intake: cli-build ## Run the offline intake experiment matrix
	$(UV_RUN) run python scripts/run_intake_experiments.py --out .felicia/experiments/intake/report.json

journey-local: cli-build ## Preprocess raw local sources into an editable journey workspace
	@test -n "$(GPX)" || (echo 'usage: make journey-local GPX=path/to/route.gpx PHOTOS=path/to/photos [SIDECAR=path] [SLUG=name] [TITLE="Trip name"] [JOURNEY=uuid] [JOURNAL=uuid] [WORKSPACE=path]' >&2; exit 1)
	@test -n "$(PHOTOS)" || (echo 'usage: make journey-local GPX=path/to/route.gpx PHOTOS=path/to/photos [SIDECAR=path] [SLUG=name] [TITLE="Trip name"] [JOURNEY=uuid] [JOURNAL=uuid] [WORKSPACE=path]' >&2; exit 1)
	$(UV_RUN) run python scripts/local_journey.py preprocess --gpx "$(GPX)" --photos "$(PHOTOS)" \
		$(if $(SIDECAR),--sidecar "$(SIDECAR)",) \
		$(if $(SLUG),--slug "$(SLUG)",) \
		$(if $(TITLE),--title "$(TITLE)",) \
		$(if $(JOURNEY),--journey "$(JOURNEY)",) \
		$(if $(JOURNAL),--journal "$(JOURNAL)",) \
		$(if $(WORKSPACE),--workspace "$(WORKSPACE)",)

# Pre-PR gate. Database migration smoke remains separate because it needs a
# disposable service; deterministic frontend checks belong in this gate.
validate: check build web-check web-admin-check ## Pre-PR gate

tidy: ## Tidy go modules
	$(GO) mod tidy

db-up: ## Start local Postgres+PostGIS and Valkey (ops/compose.yaml)
	@test -n "$(COMPOSE)" || (echo "No container compose command found (install podman-compose or Docker Compose)" >&2; exit 1)
	$(COMPOSE) -f ops/compose.yaml up -d

db-down: ## Stop the local dev containers (keeps the pgdata volume)
	@test -n "$(COMPOSE)" || (echo "No container compose command found (install podman-compose or Docker Compose)" >&2; exit 1)
	$(COMPOSE) -f ops/compose.yaml down

migrate: ## Apply DB migrations (goose, from nix) — needs DATABASE_DSN
	$(NIX_RUN)goose -dir apps/felicia-server/migrations postgres "$(DATABASE_DSN)" up

seed: ## Seed the database with sample data (uv run, psycopg) — needs DATABASE_DSN
	$(UV_RUN) run --group dev python scripts/seed.py

admin: ## Start the local admin GUI (authoring API + web-admin on 127.0.0.1)
	$(UV_RUN) run python scripts/admin.py

dev: ## Start the local API with SQLite
	$(MAKE) dev-sqlite

dev-sqlite: ## Start the API locally with the default SQLite provider
	$(UV_RUN) run python scripts/dev.py --driver sqlite

dev-postgres: ## Start the PostgreSQL-backed local stack, seed data, and serve the web app
	$(UV_RUN) run python scripts/dev.py --driver postgres --web

mock-up: ## Start the mock Dawarich+Immich upstream in the background (:8099)
	nohup $(UV_RUN) run python scripts/mock_upstream.py > /tmp/felicia-mock.log 2>&1 & echo "mock up on :8099 (log: /tmp/felicia-mock.log)"

mock-down: ## Stop the mock upstream
	@pkill -f scripts/mock_upstream.py && echo "mock stopped" || echo "no mock running"

test-api: ## Run Python-based E2E API integration tests (requires running server)
	$(UV_RUN) run python scripts/test_api.py

test-workflow: ## Run full journey workflow against disposable SQLite
	$(UV_RUN) run python scripts/test_journey_workflow.py --start-server

test-workflow-postgres: ## Run full journey workflow against disposable PostgreSQL
	@test -n "$(FELICIA_TEST_DATABASE_DSN)$(FELICIA_TEST_POSTGRES_ADMIN_DSN)" || (echo "FELICIA_TEST_DATABASE_DSN or FELICIA_TEST_POSTGRES_ADMIN_DSN is required" >&2; exit 1)
	@if test -n "$(FELICIA_TEST_POSTGRES_ADMIN_DSN)"; then \
		FELICIA_TEST_POSTGRES_ADMIN_DSN="$(FELICIA_TEST_POSTGRES_ADMIN_DSN)" $(UV_RUN) run python scripts/test_journey_workflow.py --start-server --database-driver postgres; \
	else \
		DATABASE_DSN="$(FELICIA_TEST_DATABASE_DSN)" $(MAKE) migrate; \
		FELICIA_TEST_DATABASE_DSN="$(FELICIA_TEST_DATABASE_DSN)" $(UV_RUN) run python scripts/test_journey_workflow.py --start-server --database-driver postgres; \
	fi

test-admin-e2e: ## Run the admin GUI closed-loop E2E pass (disposable server + bun run dev + Playwright/chromium) — ADMIN-01.8, local-only (not part of validate)
	$(UV_RUN) run python scripts/e2e_admin_gui.py

sqlc: ## Regenerate the postgres query bindings (apps/felicia-providers/postgres/db) from sqlc.yaml
	sqlc generate

test-sqlite: ## Run all tests with SQLite as the only enabled provider
	DATABASE_DSN= FELICIA_TEST_DATABASE_DSN= $(MAKE) test

test-postgres: ## Run PostgreSQL tests against the disposable test database
	@test -n "$(FELICIA_TEST_DATABASE_DSN)" || (echo "FELICIA_TEST_DATABASE_DSN is required" >&2; exit 1)
	DATABASE_DSN="$(FELICIA_TEST_DATABASE_DSN)" $(MAKE) migrate
	DATABASE_DSN= FELICIA_TEST_DATABASE_DSN="$(FELICIA_TEST_DATABASE_DSN)" $(MAKE) test

test-features: ## Run offline Python feature-contract tests
	$(UV_RUN) run --group dev ruff check scripts tests
	$(UV_RUN) run python -m unittest discover -s tests

web-install: ## Install all frontend workspace deps (bun from nix)
	$(BUN) install

web-dev: ## Run frontend dev server (bun + vite)
	$(BUN) run --cwd apps/felicia-public-site dev

web-build: ## Build public frontend for production (bun + vite)
	$(BUN) run web:public:build

web-admin-build: ## Build admin frontend for production (bun + vite)
	bun run web:admin:build

# The deployable site: the public SPA built for the target base path, with the
# author's own journal compiled into the same directory. The compiler only
# removes files its previous manifest listed, so the co-located SPA survives.
# `static-build` below is the fixture design demo, not this.
site-build: cli-build ## Build the deployable site (SPA + your journal) into apps/felicia-public-site/dist
	BASE_PATH="$${BASE_PATH:-/}" $(BUN) run web:public:build
	./bin/felicia-cli static compile \
		--db "$${DATABASE_PATH:-.felicia/local.sqlite}" \
		--media-root "$${MEDIA_ROOT:-.felicia/media}" \
		--out "$${SITE_DIST:-apps/felicia-public-site/dist}"

site-verify: ## Verify the deployable site artifact (base path, journeys, media)
	BASE_PATH="$${BASE_PATH:-/}" $(UV_RUN) run python scripts/verify_static_artifact.py

static-build: ## Build the v0.1 static artifact
	$(UV_RUN) run python scripts/felicia.py build --base-path "$${BASE_PATH:-/}"

static-validate: ## Validate the generated v0.1 static artifact
	$(UV_RUN) run python scripts/felicia.py validate --base-path "$${BASE_PATH:-/}"

static-publish: ## Build, validate, and print the v0.1 publication manifest
	$(UV_RUN) run python scripts/felicia.py publish --base-path "$${BASE_PATH:-/}"

pages-workflow-validate: ## Verify the Pages workflow is fork-safe
	$(UV_RUN) run python scripts/verify_pages_workflow.py

fork-smoke: ## Build a clean checkout from another filesystem path
	$(UV_RUN) run python scripts/verify_fork_smoke.py

pages-preview: ## Import inbox packages, compile, and serve on localhost:8082
	BASE_PATH=/ $(UV_RUN) run python scripts/felicia.py preview
	@test -n "$(COMPOSE)" || (echo "No container compose command found (install podman-compose or Docker Compose)" >&2; exit 1)
	$(COMPOSE) -f ops/compose.yaml --profile pages up -d pages-preview
	@echo "Felicia Pages preview: http://localhost:8082"

pages-down: ## Stop the local static Pages preview
	@test -n "$(COMPOSE)" || (echo "No container compose command found (install podman-compose or Docker Compose)" >&2; exit 1)
	$(COMPOSE) -f ops/compose.yaml --profile pages down pages-preview

web-check: ## Frontend typecheck + lint + format check
	$(BUN) run web:public:check

web-admin-check: ## Admin frontend typecheck + lint + format check
	$(BUN) run web:admin:check

# Docs preview (uv-managed env, isolated from Go/bun). Binds 0.0.0.0 so it is
# reachable over SSH — forward with `ssh -L 8000:localhost:8000 <host>`.
docs: ## Live-preview docs in the browser (uv + mkdocs-material)
	$(UV_RUN) run --group docs mkdocs serve -a 0.0.0.0:8000

docs-build: ## Build the static docs site into ./site
	$(UV_RUN) run --group docs mkdocs build

# Share the running demo to a friend over an ephemeral Cloudflare tunnel.
# Builds the SPA, brings the whole stack up under compose (db+cache+api+web),
# migrates+seeds via the host, then fronts it all with a trycloudflare.com URL.
# No CF account/domain needed; only /api/v1 is exposed (admin stays host-only).
share: ## Build + serve the full stack behind a quick Cloudflare tunnel (share to a friend)
	$(UV_RUN) run python scripts/share.py

share-down: ## Stop the shared stack (api, web, cloudflared); keeps db+cache+data
	@test -n "$(COMPOSE)" || (echo "No container compose command found" >&2; exit 1)
	$(COMPOSE) -f ops/compose.yaml rm -sf api web cloudflared
