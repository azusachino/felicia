# Setup — felicia

> For collaborators. **Stage: research** — there's no application code yet, so this is the
> toolchain you need to build docs, run the (skipped-until-it-exists) gate, and be ready
> when the first Go package lands. North star: [`direction.md`](direction.md).

## Prerequisites

- **mise** — language runtimes. `mise install` reads `mise.toml` (go 1.26, bun 1.3) and
  shims them onto `PATH`. Use mise for runtimes *only*.
- **nix** (flakes enabled) — system tools. `nix develop` enters a shell with
  golangci-lint, goose, and **Postgres 18 + PostGIS**. You usually don't need to enter it
  manually: `make` wraps nix tools via `NIX_RUN` automatically.
- **uv** — used only for the docs preview (isolated Python env; never touches Go/bun).

## Common commands

```bash
mise install        # runtimes (go, bun)
nix develop         # optional: enter the system-tool shell
make help           # list targets
make docs           # live-preview the docs (see below)
make check          # fmt + vet + lint + test   — before commit (no-ops until Go exists)
make validate       # check + build             — before PR
```

`make check` / `make validate` skip all Go targets cleanly while the tree has no `.go`
files, so the gate is green during the research stage.

## Preview the docs

```bash
make docs           # serves on 0.0.0.0:8000 (MkDocs Material, live-reload)
```

Working over SSH? Forward the port from your machine, then open `http://localhost:8000`:

```bash
ssh -L 8000:localhost:8000 <host>
```

`make docs-build` writes the static site to `./site` (gitignored).

## Database & Ingress Containers (when implementation starts)

- **Linux Dev & Prod Runtime (Podman Compose):**
  We run the database (PostgreSQL 18 + PostGIS), Go API server, and secure ingress tunnel middleware inside containers using **Podman** (and `podman-compose`):
  ```bash
  # Spin up PostgreSQL, the Go API server, and the Cloudflare Tunnel client
  podman-compose -f deploy/docker-compose.yml up -d
  ```
- **macOS Dev Runtime:**
  Leverage the native, lightweight **Bianpai** app ([github.com/bianpai/bianpai](https://github.com/bianpai/bianpai)) to run PostgreSQL 18 + PostGIS natively.
- **Migrations:**
  `make migrate` applies `migrations/` with goose (needs `DATABASE_DSN`).

## Configuration (when implementation starts)

- Copy `.env.example` → `.env` (gitignored). **Secrets via env only**: `DATABASE_DSN`,
  `WAYPOINTS_IMMICH_API_KEY`, `WAYPOINTS_DAWARICH_API_KEY`, `ANTHROPIC_API_KEY`, storage
  credentials.
- Non-secret settings live in `waypoints.toml`.

## Conventions

Conventional commits (`feat:`/`fix:`/`chore:`/`deploy:`, no emojis); 2-space indent for
config files. `make check` before commits, `make validate` before PRs — both hook-enforced,
no `--no-verify`. Full rules: [`AGENTS.md`](https://github.com/) and `.claude/rules/`.
