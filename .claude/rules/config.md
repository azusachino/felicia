# Config management & migrations — felicia

## Configuration

- `waypoints` config is TOML (`waypoints.toml`) with env overrides (`WAYPOINTS_*`).
- **Secrets come from the environment only**, never committed: storage credentials,
  `WAYPOINTS_IMMICH_API_KEY`, `WAYPOINTS_DAWARICH_API_KEY`, `ANTHROPIC_API_KEY`,
  `DATABASE_DSN`. Provide a non-secret `.env.example`; never a real `.env`.
- `.env` and `.env.*` are gitignored (except `.env.example`).

## Migrations

- Postgres + PostGIS, managed with **goose** under `migrations/`.
- Apply with `make migrate` (needs `DATABASE_DSN`). Migrations are forward-only in shared
  environments; never edit an applied migration — add a new one.
- The DB is a rebuildable projection: schema changes should be re-runnable from a clean
  database plus a `waypoints import`.

## Privacy invariant

- Public images are resized and **EXIF-stripped** before upload; raw GPS lives only in the
  DB geometry columns, never in a publicly served file (design §2, importer-spec §8).
