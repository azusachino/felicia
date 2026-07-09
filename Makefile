# felicia task runner.
# Runtimes (go, bun) come from mise (shimmed onto PATH). System tools
# (golangci-lint, goose) come from the nix flake; NIX_RUN prefixes those so
# targets work both inside `nix develop` and outside it.
NIX_RUN := $(if $(IN_NIX_SHELL),,nix develop --command )
GO      ?= go

# Whether any Go sources exist yet. The skeleton has none during the research
# phase, so Go targets no-op cleanly until the first package is written.
GO_FILES := $(shell find . -name '*.go' -not -path './vendor/*' -not -path './.git/*' -not -path '*/node_modules/*' -print -quit 2>/dev/null)

.PHONY: help fmt vet lint test check build validate tidy migrate web-install web-check docs docs-build

help: ## List targets
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | \
		awk 'BEGIN{FS=":.*?## "}{printf "  \033[36m%-13s\033[0m %s\n", $$1, $$2}'

fmt: ## Format Go code
	@if [ -z "$(GO_FILES)" ]; then echo "fmt: no Go packages yet, skipping"; else $(GO) fmt ./...; fi

vet: ## Run go vet
	@if [ -z "$(GO_FILES)" ]; then echo "vet: no Go packages yet, skipping"; else $(GO) vet ./...; fi

lint: ## Lint Go (golangci-lint, from nix)
	@if [ -z "$(GO_FILES)" ]; then echo "lint: no Go packages yet, skipping"; else $(NIX_RUN)golangci-lint run; fi

test: ## Run Go tests with race detector + coverage
	@if [ -z "$(GO_FILES)" ]; then echo "test: no Go packages yet, skipping"; else $(GO) test -race -cover ./...; fi

check: fmt vet lint test ## Pre-commit gate

build: ## Build all binaries
	@if [ -z "$(GO_FILES)" ]; then echo "build: no Go packages yet, skipping"; else $(GO) build ./...; fi

# Pre-PR gate. Frontend (web-check) and migration smoke join here once web/ and
# migrations/ have content.
validate: check build ## Pre-PR gate

tidy: ## Tidy go modules
	$(GO) mod tidy

migrate: ## Apply DB migrations (goose, from nix) — needs DATABASE_DSN
	$(NIX_RUN)goose -dir migrations postgres "$(DATABASE_DSN)" up

seed: ## Seed the database with sample data (uv run, psycopg) — needs DATABASE_DSN
	uv run --group dev python scripts/seed.py

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
