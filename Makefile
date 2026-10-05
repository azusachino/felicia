# felicia task runner.
# mise owns the repository toolchain and loads the optional .env file. These
# wrappers keep every target usable without shell activation.
MISE_RUN ?= mise exec --
UV_RUN  := $(MISE_RUN) uv
GO      ?= $(MISE_RUN) go
BUN     ?= $(MISE_RUN) bun
# Markdown formatting is local-only; rumdl is supplied by Nix on PATH.
RUMDL   ?= rumdl

PORT ?= 8080
CACHE_ADDR ?= localhost:6379

COMPOSE ?= $(shell \
	if command -v podman-compose >/dev/null 2>&1; then echo podman-compose; \
	elif command -v docker >/dev/null 2>&1; then echo docker compose; \
	else echo ''; fi)

.PHONY: help fmt fmt-check fmt-docs fmt-docs-check vet lint test test-api test-features layout-check test-sqlite check check-ci build cli-build desktop-assets desktop-build desktop-package desktop experiment-intake journey-local validate deps-check tidy db-up db-down seed admin dev dev-sqlite test-workflow test-admin-e2e mock-up mock-down browser-mock web-install web-check web-build admin-check admin-build site-build site-verify pages-workflow-validate fork-smoke pages-preview pages-down docs docs-build share share-down e2e e2e-install desktop-e2e-build test-reader-fonts-e2e

help: ## List targets
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | \
		awk 'BEGIN{FS=":.*?## "}{printf "  \033[36m%-13s\033[0m %s\n", $$1, $$2}'

fmt: fmt-docs ## Format Go, frontend code and Markdown
	$(UV_RUN) run python scripts/format.py

fmt-check: fmt-docs-check ## Check Go, frontend and Markdown without modifying files
	$(UV_RUN) run python scripts/format.py --check

.PHONY: fmt-go-check local-check
fmt-go-check: ## Check Go formatting only for local iteration
	$(UV_RUN) run python scripts/format.py --check --only go

local-check: ## Cache unchanged local groups (ARGS='--groups scripts' or '--force')
	$(UV_RUN) run python scripts/local_checks.py $(ARGS)

fmt-docs: ## Fix Markdown lint with rumdl
	$(RUMDL) check --config .rumdl.toml --fix .

fmt-docs-check: ## Check Markdown with rumdl
	$(RUMDL) check --config .rumdl.toml .

vet: ## Run go vet
	$(UV_RUN) run python scripts/go_tasks.py vet

lint: ## Lint Go (golangci-lint, from mise)
	$(UV_RUN) run python scripts/go_tasks.py lint

test: ## Run Go tests with race detector + coverage
	$(UV_RUN) run python scripts/go_tasks.py test

check: fmt-check check-ci ## Local pre-commit gate, including formatting

check-ci: vet lint test test-features ## CI checks without local formatting tools

build: ## Build all binaries
	$(UV_RUN) run python scripts/go_tasks.py build

cli-build: ## Build the felicia-cli executable into bin/
	@mkdir -p bin
	$(GO) build -o bin/felicia-cli ./apps/felicia-cli/cmd/felicia

desktop-assets: admin-build web-build ## Prepare built admin and reader assets for embedding
	mkdir -p apps/felicia-desktop/assets/admin apps/felicia-desktop/assets/reader
	cp -R apps/felicia-admin/dist/. apps/felicia-desktop/assets/admin/
	cp -R apps/felicia-public-site/dist/. apps/felicia-desktop/assets/reader/

desktop-e2e-build: desktop-assets ## Build test-only headless desktop composition
	@mkdir -p bin
	$(UV_RUN) run python scripts/go_tasks.py desktop-e2e-build

# Browser verification of desktop composition; does not drive the Wails window.
e2e: desktop-e2e-build ## Run integrated desktop Chromium/WebKit specs on synthetic state
	$(UV_RUN) run python scripts/go_tasks.py desktop-e2e-test
	$(BUN) --bun run --filter @felicia/admin e2e $(ARGS)

e2e-install: ## Install browsers for the project's pinned Playwright version
	$(BUN) --bun run --filter @felicia/admin e2e:install $(E2E_INSTALL_FLAGS)

test-reader-fonts-e2e: ## Run standalone built-reader font/static-subpath browser check (isolated build under .tmp; not part of validate)
	$(BUN) --bun run --filter @felicia/admin e2e:reader-fonts

desktop-build: desktop-assets ## Build native desktop binary into bin/felicia-desktop
	@mkdir -p bin
	$(UV_RUN) run python scripts/go_tasks.py desktop-build

desktop-package: desktop-build ## Package the macOS app bundle with the native app icon
	mkdir -p bin/FeliciaStudio.app/Contents/MacOS bin/FeliciaStudio.app/Contents/Resources
	cp bin/felicia-desktop bin/FeliciaStudio.app/Contents/MacOS/felicia-desktop
	cp apps/felicia-desktop/Info.plist bin/FeliciaStudio.app/Contents/Info.plist
	cp apps/felicia-desktop/build/felicia.icns bin/FeliciaStudio.app/Contents/Resources/felicia.icns
	codesign --force --sign - bin/FeliciaStudio.app
	codesign --verify --deep --strict bin/FeliciaStudio.app

desktop: desktop-package ## Run local desktop studio (macOS app)
	./bin/FeliciaStudio.app/Contents/MacOS/felicia-desktop

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

# Pre-PR gate. Deterministic frontend checks belong in this gate.
validate: check build web-check admin-check ## Pre-PR gate

deps-check: ## Check lockfile consistency and report available frontend upgrades
	$(UV_RUN) lock --check
	$(BUN) outdated

tidy: ## Tidy go modules
	$(GO) mod tidy

db-up: ## Start local Postgres+PostGIS and Valkey (ops/compose.yaml)
	@test -n "$(COMPOSE)" || (echo "No container compose command found (install podman-compose or Docker Compose)" >&2; exit 1)
	$(COMPOSE) -f ops/compose.yaml up -d

db-down: ## Stop the local dev containers (keeps the pgdata volume)
	@test -n "$(COMPOSE)" || (echo "No container compose command found (install podman-compose or Docker Compose)" >&2; exit 1)
	$(COMPOSE) -f ops/compose.yaml down

dev: ## Start the local API with SQLite
	$(MAKE) dev-sqlite

dev-sqlite: ## Start the API locally with the default SQLite provider
	$(UV_RUN) run python scripts/dev.py --driver sqlite

mock-up: ## Start the mock Dawarich+Immich upstream in the background (:8099)
	nohup $(UV_RUN) run python scripts/mock_upstream.py > /tmp/felicia-mock.log 2>&1 & echo "mock up on :8099 (log: /tmp/felicia-mock.log)"

mock-down: ## Stop the mock upstream
	@pkill -f scripts/mock_upstream.py && echo "mock stopped" || echo "no mock running"

browser-mock: ## Serve deterministic public-reader data for browser verification
	$(BUN) scripts/browser_preview_mock.ts

test-api: ## Run Python-based E2E API integration tests (requires running server)
	$(UV_RUN) run python scripts/test_api.py

test-workflow: ## Run full journey workflow against disposable SQLite
	$(UV_RUN) run python scripts/test_journey_workflow.py --start-server

test-admin-e2e: ## Run the admin GUI closed-loop E2E pass (disposable server + Bun dev + Playwright/chromium) — ADMIN-01.8, local-only (not part of validate)
	$(UV_RUN) run python scripts/e2e_admin_gui.py

test-sqlite: ## Run all tests with SQLite as the only enabled provider
	$(MAKE) test

test-features: ## Run offline Python feature-contract tests
	$(UV_RUN) run --group dev ruff check --config pyproject.toml scripts tests
	$(UV_RUN) run python -m unittest discover -s tests

layout-check: ## Verify application/package layout and dependency boundaries
	$(UV_RUN) run python -m unittest tests.test_layout tests.test_kind_registry_drift

web-install: ## Install locked frontend workspace deps (Bun from mise)
	$(BUN) install --frozen-lockfile

web-dev: ## Run public frontend dev server (Bun + Vite)
	$(BUN) --bun run --filter @felicia/public-site dev

web-build: ## Build public frontend for production (Bun + Vite)
	$(BUN) run web:public:build

admin-build: ## Build static studio frontend (Bun + SvelteKit)
	$(BUN) run web:admin:build

# The deployable site: the public SPA built for the target base path, with the
# author's own journal compiled into the same directory. The compiler only
# removes files its previous manifest listed, so the co-located SPA survives.
# `site-build` below is the compiler-backed publication path, not this.
site-build: cli-build ## Build the deployable site (SPA + your journal) into apps/felicia-public-site/dist
	BASE_PATH="$${BASE_PATH:-/}" $(BUN) run web:public:build
	./bin/felicia-cli static compile \
		--db "$${DATABASE_PATH:-.felicia/felicia.sqlite}" \
		--media-root "$${MEDIA_ROOT:-.felicia/media}" \
		--out "$${SITE_DIST:-apps/felicia-public-site/dist}"

site-verify: ## Verify the deployable site artifact (base path, journeys, media)
	BASE_PATH="$${BASE_PATH:-/}" $(UV_RUN) run python scripts/verify_static_artifact.py

pages-workflow-validate: ## Verify the Pages workflow is fork-safe
	$(UV_RUN) run python scripts/verify_pages_workflow.py

fork-smoke: ## Build a clean checkout from another filesystem path
	$(UV_RUN) run python scripts/verify_fork_smoke.py

pages-preview: ## Build the production publication, compile, and serve on localhost:8082
	BASE_PATH=/ $(UV_RUN) run python scripts/felicia.py publish
	@test -n "$(COMPOSE)" || (echo "No container compose command found (install podman-compose or Docker Compose)" >&2; exit 1)
	$(COMPOSE) -f ops/compose.yaml --profile pages up -d pages-preview
	@echo "Felicia Pages preview: http://localhost:8082"

pages-down: ## Stop the local static Pages preview
	@test -n "$(COMPOSE)" || (echo "No container compose command found (install podman-compose or Docker Compose)" >&2; exit 1)
	$(COMPOSE) -f ops/compose.yaml --profile pages down pages-preview

web-check: ## Frontend typecheck + lint + format check
	$(BUN) run web:public:check

admin-check: ## Admin frontend typecheck + lint + format check
	$(BUN) run web:admin:check

# Docs preview (uv-managed env, isolated from Go/Node). Binds 0.0.0.0 so it is
# reachable over SSH — forward with `ssh -L 8000:localhost:8000 <host>`.
docs: ## Live-preview docs in the browser (uv + mkdocs-material)
	$(UV_RUN) run --group docs mkdocs serve -a 0.0.0.0:8000

docs-build: ## Build the static docs site into ./site
	$(UV_RUN) run --group docs mkdocs build

# Share the running stack to a friend over an ephemeral Cloudflare tunnel.
# Builds the SPA, brings the whole stack up under compose (db+cache+api+web),
# migrates+seeds via the host, then fronts it all with a trycloudflare.com URL.
# No CF account/domain needed; only /api/v1 is exposed (admin stays host-only).
share: ## Build + serve the full stack behind a quick Cloudflare tunnel (share to a friend)
	$(UV_RUN) run python scripts/share.py

share-down: ## Stop the shared stack (api, web, cloudflared); keeps db+cache+data
	@test -n "$(COMPOSE)" || (echo "No container compose command found" >&2; exit 1)
	$(COMPOSE) -f ops/compose.yaml rm -sf api web cloudflared
