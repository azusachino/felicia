# TODO — felicia

Durable task state lives in rosemary (`/rosemary tasks`). This is a lightweight mirror.

## In progress

- (research/spec phase wrapping up)

## Blocked

- Frontend & migration quality gates — wired into `make validate` once `web/` and
  `migrations/` have content.

## Done

- Reverse-engineered the reference concept (liuaaron.com) from screenshots.
- Locked decisions: architecture (Go API + SPA, A+E), hosting (Pi + Cloudflare Tunnel),
  ingestion (Immich + Dawarich), source-of-truth (DB canonical, field-scoped importer),
  content model (Journey → Ticket → essay/photos/animation), storage (S3-compatible, R2).
- Wrote `docs/design.md`, `docs/importer-spec.md`, `docs/research/ingestion-workflows.md`.
- Purged old Go/AWS content; initialized layout + mise/nix/Makefile + agent infra (no code).

## Next (when TDD phase begins)

- First failing tests per `importer-spec.md` §11 (gpx/simplify → EXIF → cluster → OCR map →
  no-clobber upsert → Immich client → sync golden).
