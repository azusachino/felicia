# The selected end-to-end user journey

> Decided 2026-07-18. This document records the **target journey** — how a trip
> goes from raw data to a deployed public site — and the current status of each
> stage. It is the primary landing spot for the per-PR docs-sync discipline in
> `AGENTS.md`: when a PR moves a stage forward, update the status here.

## The journey (hybrid: local GUI authoring + static Pages publication)

Per [ADR-0025](../adr/0025-static-and-self-hosted-modes.md), the primary
release workflow is server-mode authoring compiled into a static artifact.
The selected shape:

```
Dawarich / Immich or local GPX + photos          (data stays on the author's machine)
  → felicia local server (SQLite): import + intake planning
  → web-admin GUI: review stop candidates → edit mementos → write essays → publish
  → felicia static compile → dist/ (published-only JSON + safe media + SPA)
  → push / GitHub Actions → GitHub Pages
```

Invariants that must hold at every step:

- **Drafts and originals never leave the machine.** The static artifact contains
  only `published` content, EXIF-stripped public derivatives, and rounded
  geometry ([ADR-0025](../adr/0025-static-and-self-hosted-modes.md),
  [ADR-0026](../adr/0026-local-first-media-and-blob-storage.md)).
- **Re-import never overwrites authored fields** (field-scoped importer,
  [ADR-0022](../adr/0022-unified-intake-and-draft-pipeline.md)).
- **Only the public media boundary ships**: `public` visibility + JPEG/PNG/WebP
  in the v1 package; anything else fails packaging instead of leaking.
- **Contract-first**: fields come from
  [canonical contract v1](../contracts/canonical-v1.md) and its projections,
  never invented per surface.

## Per-stage status

| # | Stage | Status | Where it lives |
|---|-------|--------|----------------|
| 1 | **Data collection** | ✅ Done | Dawarich client (`providers/dawarich/`), Immich client (`providers/immich/`), local GPX + photo/sidecar source (`providers/local/`), mock upstream (`scripts/mock_upstream.py`) |
| 2 | **Import / intake** | ✅ Done (deterministic) | Field-scoped importer (`runtime/importer/`), dwell-cluster intake planner (`runtime/intake/planner.go`), SQLite + PostgreSQL providers behind shared contract tests |
| 3 | **Authoring** | ⚠️ File/CLI path complete; **GUI is the gap** | Local authoring schema v1 (`schemas/local-authoring-v1.schema.json`), CLI `journey plan/apply/review`, full admin API (`server/api/server.go`); `apps/web-admin` is a read-only overview shell |
| 4 | **Publication** | ✅ Done | Published-only static compiler (`publication/compiler.go`), byte-identical to the live `/api/v1` read contract, enforced public media boundary |
| 5 | **Deployment** | ⚠️ Fixture demo only | Pages workflow deploys a checked-in fixture; SQLite-backed publication is epic [FELICIA-PAGES-01](pages-v1-epic.md) (GitHub issues [#40–#50](https://github.com/azusachino/felicia/issues/40)) |

Deliberately deferred (not gaps): AI enrichment
([ADR-0024](../adr/0024-optional-ai-enrichment.md)), R2/S3 object storage
([ADR-0026](../adr/0026-local-first-media-and-blob-storage.md)), the
self-hosted always-on server mode as a release target
([ADR-0025](../adr/0025-static-and-self-hosted-modes.md)), multi-user/auth,
and a translation sidecar.

## Delivery phases

### Phase 1 — static publication on real data (= epic FELICIA-PAGES-01)

Replace the fixture-only Pages demo with the real pipeline: SQLite →
`felicia static compile` → GitHub Pages.

- Compile from SQLite with authored fields (essay/vendor/price) intact.
- Exercise the compiler with a **mixed-state seed dataset** (raw imports,
  drafts, fully authored mementos, missing metadata, broken sidecar paths) —
  not only clean CLI-authored fixtures.
- Assert the **media invariants** in tests: every media URL in the generated
  JSON is relative (no local absolute paths), every file under `dist/` passes
  the public media boundary, zero draft/private leakage.
- Switch the Pages build from fixtures to the compiled artifact; multi-journey
  discovery via the `workspace.json` manifest.
- Prove live-server/static contract parity (epic 01.5) and run the Pages
  workflow end to end on a fork (epic 01.7/01.10).

### Phase 2 — minimal admin GUI (upgrade authoring from CLI to local GUI)

Build the smallest closed authoring loop in `apps/web-admin`, on top of the
existing API boundary (`apps/web-admin/src/api.ts`) and the admin-api/v1
projection:

- Journey list/detail with import triggers (`sync-route` / `visits` / `tray`).
- Intake inbox: review stop candidates (confirm / merge / ignore).
- Memento editor: hardcoded forms for 2–3 baseline kinds (`transit`, `goods`)
  aligned to schema v1 and `core/kinds/*.yaml` — the registry-driven dynamic
  form engine comes after the MVP. Photo curation respects the public media
  boundary; essays; state transitions candidate → draft → authored → published.
- Concurrency: pass the existing revision field through and surface stale-write
  errors; no conflict-resolution UI.
- Publish action: mark published + trigger `static compile` to produce `dist/`.

## Status log

- **2026-07-18 (later)** — Phase 1.1 in progress: found and fixed a draft-leak
  on the live public API (`/api/v1` endpoints did not filter memento state, so
  draft essays and draft-only journey routes were publicly exposed) and closed
  the static-artifact contract gap (missing `essay`/`vendor`/`price_*` and photo
  `taken_at`). Public projection and the published-only gate now live once in
  the `publication` package and are shared by the live API and the static
  compiler; a journey without published mementos has no public projection on
  either side. Verification pending (local toolchain unavailable).
- **2026-07-18** — Journey selected and documented. Stages 1/2/4 complete;
  stage 3 blocked on the admin GUI; stage 5 blocked on epic FELICIA-PAGES-01.
  Baseline includes PR #53 (canonical contract v1, local authoring schema v1,
  read-only admin shell wired to the admin API, enforced media boundary).
