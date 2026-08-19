# TODO — felicia

Durable task state lives in asobi (`asobi` tasks). This is a lightweight mirror.

## In progress

- **M0 spec freeze** (plan.md): review PROPOSED items in `spec-gaps.md` → execute
  fold-in checklist → module re-init + package skeleton.

## Blocked

- Frontend & migration quality gates — wired into `make validate` once `web/` and
  `migrations/` have content.

## Done

- Reverse-engineered the reference concept (liuaaron.com) from screenshots.
- Locked decisions: architecture (Go API + SPA, A+E), hosting (Pi + Cloudflare Tunnel),
  ingestion (Immich + Dawarich), source-of-truth (DB canonical, field-scoped importer),
  content model (Journey → Ticket → essay/photos/animation), storage (S3-compatible, R2).
- Wrote `docs/design.md`, `docs/importer-spec.md`, `docs/research/ingestion-workflows.md`.
- Purged old Go/AWS content; initialized layout + mise/Makefile + agent infra (no code).

## Next (when TDD phase begins)

- First failing tests per `plan.md` M1 order (gpx → simplify → gap-split → EXIF → tz →
  cluster → snap-to-track → anchoring → OCR map → photo-trail → three-class no-clobber →
  zero-diff idempotency).
