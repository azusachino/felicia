# Setup — felicia

> For collaborators. The repository contains the working Go workspace, SQLite workflow, and
> frontend checks. North star: [`direction.md`](direction.md).

## Prerequisites

- **mise** — language runtimes. `mise install` reads `mise.toml` (go 1.26, bun 1.3) and
  shims them onto `PATH`. Use mise for runtimes _only_.
- **nix** (flakes enabled) — system tools. `nix develop` enters a shell with
  golangci-lint, goose, sqlc, uv, and **Postgres 18 + PostGIS**. You usually don't need to enter it
  manually: `make` wraps Nix tools via `NIX_RUN` and `UV_RUN` automatically.

## Common commands

```bash
mise install        # runtimes (go, bun)
nix develop         # optional: enter the system-tool shell
make help           # list targets
make docs           # live-preview the docs (see below)
make check          # formatting + vet + lint + tests — before commit
make validate       # check + build             — before PR
```

`make check` covers every Go workspace module, UV feature-contract tests, and repository
formatting. `make web-check` adds frontend type, lint, and formatting checks.

## Preview the docs

```bash
make docs           # serves on 0.0.0.0:8000 (MkDocs Material, live-reload)
```

Working over SSH? Forward the port from your machine, then open `http://localhost:8000`:

```bash
ssh -L 8000:localhost:8000 <host>
```

`make docs-build` writes the static site to `./site` (gitignored).

## Database Container (when implementation starts)

- **Linux Dev Runtime (Podman Compose):**
  The database (PostgreSQL 18 + PostGIS) runs inside a container using **Podman** and `podman-compose`. The Go application itself runs **locally** on the host.
  ```bash
  # Spin up the database container
  podman-compose -f deploy/compose.yaml up -d
  ```
- **macOS Dev Runtime:**
  Leverage the native, lightweight **Bianpai** app ([github.com/bianpai/bianpai](https://github.com/bianpai/bianpai)) to run PostgreSQL 18 + PostGIS natively on the host.
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
