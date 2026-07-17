# Setup — felicia

> For collaborators. The repository contains the working Go workspace, SQLite-first workflow,
> optional PostgreSQL provider, and frontend checks. North star: [`direction.md`](direction.md).

## Prerequisites

- **nix** (flakes enabled) — the complete repository toolchain. `nix develop` enters a
  shell with Go 1.26, Bun, uv, Prettier, golangci-lint, goose, sqlc, and **PostgreSQL 18 +
  PostGIS**. You usually don't need to enter it manually: `make` wraps the shell automatically.

## Common commands

```bash
nix develop         # optional: enter the repository toolchain shell
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

## Database providers

SQLite is the default for local development. See [Database development](development/database.md)
for configuration precedence and the PostgreSQL test-database safety rules.

## PostgreSQL Container

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

## Configuration

- Copy `.env.example` → `.env` (gitignored). **Secrets via env only**: `DATABASE_DSN`,
  `WAYPOINTS_IMMICH_API_KEY`, `WAYPOINTS_DAWARICH_API_KEY`, `ANTHROPIC_API_KEY`, storage
  credentials.
- Non-secret settings live in optional `felicia.toml`; environment variables override file values.

## Conventions

Conventional commits (`feat:`/`fix:`/`chore:`/`deploy:`, no emojis); 2-space indent for
config files. `make check` before commits, `make validate` before PRs — both hook-enforced,
no `--no-verify`. Full rules: [`AGENTS.md`](https://github.com/) and `.claude/rules/`.
