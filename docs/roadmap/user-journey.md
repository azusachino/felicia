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

| #   | Stage               | Status                  | Where it lives                                                                                                                                                                                                                                                                                                                                              |
| --- | ------------------- | ----------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | **Data collection** | ✅ Done                 | Dawarich client (`apps/felicia-providers/dawarich/`), Immich client (`apps/felicia-providers/immich/`), local GPX + photo/sidecar source (`apps/felicia-providers/local/`), mock upstream (`scripts/mock_upstream.py`)                                                                                                                                      |
| 2   | **Import / intake** | ✅ Done (deterministic) | Field-scoped importer (`apps/felicia-runtime/importer/`), dwell-cluster intake planner (`apps/felicia-runtime/intake/planner.go`), SQLite + PostgreSQL providers behind shared contract tests                                                                                                                                                               |
| 3   | **Authoring**       | ✅ Done (GUI MVP)       | Schema v1 + CLI path complete; `apps/felicia-web` closes the authoring loop — journey shell with import/preview triggers, intake inbox, memento editor with revision-conflict handling — proven end to end in a real browser (`make test-admin-e2e`, epic [FELICIA-ADMIN-01](admin-gui-v1-epic.md)); the registry-driven dynamic form engine stays deferred |
| 4   | **Publication**     | ✅ Done                 | Published-only static compiler (`apps/felicia-publication/compiler.go`); live/static content parity is enforced by a shared projection layer (`apps/felicia-publication/public.go`) and a workflow parity check                                                                                                                                             |
| 5   | **Deployment**      | ✅ Done                 | Pages workflow builds and deploys the real compiled artifact (artifact-based `upload-pages-artifact` → `deploy-pages`, nothing committed); first remote run succeeded on `main` after PR #55 merged — epic [FELICIA-PAGES-01](pages-v1-epic.md)                                                                                                             |

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
- Prove live-apps/felicia-server/static contract parity (epic 01.5) and run the Pages
  workflow end to end on a fork (epic 01.7/01.10).

### Phase 2 — minimal admin GUI (upgrade authoring from CLI to local GUI)

Build the smallest closed authoring loop in `apps/felicia-web`, on top of the
existing API boundary (`apps/felicia-web/src/api.ts`) and the admin-api/v1
projection:

- Journey list/detail with import triggers (`sync-route` / `visits` / `tray`).
- Intake inbox: review stop candidates (confirm / merge / ignore).
- Memento editor: hardcoded forms for 2–3 baseline kinds (`transit`, `goods`)
  aligned to schema v1 and `apps/felicia-core/kinds/*.yaml` — the registry-driven dynamic
  form engine comes after the MVP. Photo curation respects the public media
  boundary; essays; state transitions candidate → draft → authored → published.
- Concurrency: pass the existing revision field through and surface stale-write
  errors; no conflict-resolution UI.
- Publish action: mark published + trigger `static compile` to produce `dist/`.

## Status log

- **2026-08-16 (four P0 data-integrity defects closed)** — Auditing the
  publish flow found that four of the invariants this document asserts were
  documented but never implemented, so every one of them was silently false
  on the real-trip path. (1) `make journey-local` hard-coded one journey
  UUID, slug, and workspace, so importing a second trip overwrote the first
  (#72). (2) The importer recorded its ingest mask and then ignored the
  destination's authored mask, so a re-import destroyed authored titles,
  places, and `kind_data` — and reset the journey's authored mask outright,
  disabling the one guard that did work (#73). (3) The publication boundary
  emitted every coordinate at full float64 precision, so the artifact
  shipped the author's raw GPS trace (#74). (4) Public media keys were the
  source file's basename, so two trips' `IMG_0001.JPG` overwrote each other
  (#75). All four are now enforced rather than asserted: journeys gained the
  ingest/authoring split port the memento layer already had, so an import
  cannot reach the authoring write at compile time
  ([ADR-0033](../adr/0033-authored-field-protection-and-the-journey-ingest-seam.md));
  the authored-field rule lives in one shared `apps/felicia-core/domain` helper and is
  asserted for both providers in `apps/felicia-providers/contract`, per the second
  development-flow constraint in `AGENTS.md`; coordinates round to 4 decimals
  (~11 m, the precision already documented in `docs/archive/spec-gaps.md`
  D2) inside the single projection shared by the static compiler and the live
  API; and media keys are content-addressed from the digest that was already
  being computed. Fixing #73 surfaced a second, opposite defect: PostgreSQL
  guarded authored columns inside `UpsertJourney`, which is also the
  authoring path, so a journey field could be claimed once and then never
  edited again. Stage status unchanged — these restore guarantees the stages
  already claimed.

- **2026-08-16 (raw-input intake unblocked)** — Walking the documented
  entry path from a real GPX file and a photo folder found it broken at
  every step: eight consecutive blockers between `make journey-local` and a
  packaged journey. Four were contract drift against
  `schemas/local-authoring-v1.schema.json` (the planner's `date_start`/
  `date_end`; a candidate's deliberately-unset `kind`; and the whole
  `stop_candidate` definition, still spelled in Go PascalCase after
  `domain.StopCandidate` gained snake_case tags), two were nil maps and
  slices marshalling to JSON `null`, one was derived stops carrying blank
  provenance despite their evidence naming a source (ADR-0010), and the
  last silently dropped every photo — `local_journey.py` read `uri`/`title`
  from a media asset that serialises as `URI`/`Title`, so packaging
  reported success with `media=0`. None of it was caught because every
  local-journey fixture is a hand-written workspace and the one test that
  validated a plan hand-wrote that too, in the same stale casing as the
  schema. `tests/test_local_journey_raw_intake.py` now runs the real
  compiled planner over a checked-in two-dwell track and asserts its output
  against the repository's own schema, that dwell yields stops, and that
  sidecar-timed photos reach both workspace and package. Stage status
  unchanged: this is the CLI entry path the GUI does not yet cover.

- **2026-08-15 (publish path documented)** — The journey's last two stages had
  no user-facing instructions: `docs/release/github-pages-v0.1.md` covered only
  the CI/fork route, `setup.md` stopped at local authoring, and the
  local-authoring deploy (build with the target base path → push the artifact →
  enable Pages) was written down nowhere. Added
  [`docs/publish.md`](../publish.md) covering both routes, the base-path table,
  and the failure modes, linked from the README and the docs index. A person
  taking the deployable artifact from their own journal previously had to hand-
  assemble it (`make static-build` compiles the fixture demo, not an authored
  journal), so `make site-build` and `make site-verify` now name that surface
  per the third development-flow constraint in `AGENTS.md`. No change to the
  pipeline itself; stage status unchanged.

Earlier entries: [archived status log](../archive/user-journey-status-log.md).
