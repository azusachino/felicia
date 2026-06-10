# Setup — felicia

> Design/spec phase: there is no application code yet. This describes the toolchain so the
> skeleton builds and is ready for the TDD phase.

## Prerequisites

- **mise** — provides runtimes. `mise install` (reads `mise.toml`: go 1.26, bun 1.3).
- **nix** (flakes enabled) — provides system tools. `nix develop` enters a shell with
  golangci-lint, goose, postgresql/postgis. Or rely on `make`, which wraps nix tools via
  `NIX_RUN` automatically.

## Common commands

```bash
mise install        # runtimes
nix develop         # optional: enter the system-tool shell
make help           # list targets
make check          # fmt + vet + lint + test   (before commit)
make validate       # check + build             (before PR)
```

## Configuration (when implementation starts)

- Copy `.env.example` → `.env` (gitignored). Secrets via env only:
  `DATABASE_DSN`, `WAYPOINTS_IMMICH_API_KEY`, `WAYPOINTS_DAWARICH_API_KEY`,
  `ANTHROPIC_API_KEY`, storage credentials.
- Non-secret settings in `waypoints.toml` (see `docs/importer-spec.md` §3).

## Database (when implementation starts)

- Postgres + PostGIS (local via the nix shell, or a container from `deploy/`).
- `make migrate` applies `migrations/` with goose (needs `DATABASE_DSN`).
