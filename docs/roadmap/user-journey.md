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

| #   | Stage               | Status                  | Where it lives                                                                                                                                                                                                                                                                                                                                            |
| --- | ------------------- | ----------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | **Data collection** | ✅ Done                 | Dawarich client (`providers/dawarich/`), Immich client (`providers/immich/`), local GPX + photo/sidecar source (`providers/local/`), mock upstream (`scripts/mock_upstream.py`)                                                                                                                                                                           |
| 2   | **Import / intake** | ✅ Done (deterministic) | Field-scoped importer (`runtime/importer/`), dwell-cluster intake planner (`runtime/intake/planner.go`), SQLite + PostgreSQL providers behind shared contract tests                                                                                                                                                                                       |
| 3   | **Authoring**       | ✅ Done (GUI MVP)       | Schema v1 + CLI path complete; `apps/web-admin` closes the authoring loop — journey shell with import/preview triggers, intake inbox, memento editor with revision-conflict handling — proven end to end in a real browser (`make test-admin-e2e`, epic [FELICIA-ADMIN-01](admin-gui-v1-epic.md)); the registry-driven dynamic form engine stays deferred |
| 4   | **Publication**     | ✅ Done                 | Published-only static compiler (`publication/compiler.go`); live/static content parity is enforced by a shared projection layer (`publication/public.go`) and a workflow parity check                                                                                                                                                                     |
| 5   | **Deployment**      | ✅ Done                 | Pages workflow builds and deploys the real compiled artifact (artifact-based `upload-pages-artifact` → `deploy-pages`, nothing committed); first remote run succeeded on `main` after PR #55 merged — epic [FELICIA-PAGES-01](pages-v1-epic.md)                                                                                                           |

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

- **2026-07-19 (epic ADMIN-02 M0)** — Offline local deployment landed: the
  GUI's new Site & Deploy page builds the static artifact with one action
  (compile now defaults to the configured `site.out_dir`) and links a
  built-in preview server on a second local port that serves the compiled
  site exactly as a static host would (artifact overlaid on the pre-built
  public SPA). The closed-loop E2E builds through the GUI and asserts the
  preview port serves the compiled manifest. Design pick/style, GitHub
  Pages deploy with URL confirmation, and GUI resource uploads are planned
  in [FELICIA-ADMIN-02](admin-gui-v2-epic.md).
- **2026-07-19 (Phase 2 complete — epic ADMIN-01 M4)** — Closed-loop E2E
  verification landed (ADMIN-01.8): `make test-admin-e2e` drives the real
  GUI in Playwright/chromium against the disposable server — plan intake
  from a mock Dawarich upstream, promote a candidate, author, publish,
  compile, and assert the artifact carries the authored essay (asserted in
  the browser and again from the filesystem). The pass immediately caught
  and fixed two real bugs: a SQLite single-connection deadlock in the
  stop-candidate list (evidence queried while the candidate rows were
  still open) and a journey-list crash on Go nil slices encoding as JSON
  `null`. Every epic milestone (M1–M4) is now landed and verified.
- **2026-07-19 (Phase 2 M3)** — Memento editor landed in `apps/web-admin`
  (ADMIN-01.4 + 01.5): common fields, lat/lng inputs with snap-to-route,
  registry-aligned hardcoded `transit`/`goods` kind_data forms (other kinds
  read-only), photo metadata rows backed by a new
  `GET /api/admin/mementos/{id}/photos` endpoint, and explicit
  draft → authored → published actions with inline server validation.
  Every save passes `expected_revision`; stale writes surface a
  reload-and-reapply conflict banner. Remaining for the epic: the
  closed-loop E2E pass (ADMIN-01.8).
- **2026-07-19 (Phase 1 complete)** — PR #55 merged to `main` and the GitHub
  Pages workflow ran end to end on the merge commit: the deployed demo is now
  the real compiled artifact (example workspace → SQLite import →
  `felicia static compile` → artifact-based Pages deployment). The legacy
  `server/cmd/build` binary was also migrated to the shared publication
  compiler (manifest reconciliation included), retiring the last duplicated
  public projection.
- **2026-07-19 (Phase 2 M2)** — Intake inbox landed in `apps/web-admin`
  (ADMIN-01.3b): candidates surface per journey with plan trigger, promote
  (kind picker driven by the template registry), ignore, and
  merge-into-sibling; stale-revision conflicts (409) surface inline with a
  reload hint instead of silently overwriting. Verified in the containerized
  toolchain (`make validate`, `make test-features`, `make test-workflow` —
  including live/static parity and stale-artifact cleanup).
- **2026-07-18 (Phase 2 begins)** — Epic
  [FELICIA-ADMIN-01](admin-gui-v1-epic.md) designed (adversarially reviewed)
  and M1 landed with the parallel-safe server tasks: navigable web-admin
  journey shell with import/preview triggers, intake over HTTP
  (plan + promote endpoints), a compile endpoint sharing the CLI's
  publication path, and kind-contract convergence on six kinds with a
  registry-vs-frontend drift test. PR #55 gained manifest-based
  stale-artifact reconciliation after review (unpublished content cannot
  linger in a reused output directory) — merged into this branch so the
  compile endpoint inherits it. All verified in the containerized toolchain.
- **2026-07-18 (later)** — Phase 1.1 in progress: found and fixed a draft-leak
  on the live public API (`/api/v1` endpoints did not filter memento state, so
  draft essays and draft-only journey routes were publicly exposed) and closed
  the static-artifact contract gap (missing `essay`/`vendor`/`price_*` and photo
  `taken_at`). Public projection and the published-only gate now live once in
  the `publication` package and are shared by the live API and the static
  compiler; a journey without published mementos has no public projection on
  either side. Verified end to end (`make check`, `make test-features`,
  `make test-workflow`): a mixed-state seed fixture
  (`tests/fixtures/local-journey-mixed-state/`) drives the real
  package→import→compile pipeline with media/publish-invariant assertions,
  and the workflow harness now proves live/static JSON parity plus
  unpublished-journey exclusion against the same SQLite database. Remaining
  for Phase 1: run the Pages workflow remotely (epic 01.7/01.10).
- **2026-07-18** — Journey selected and documented. Stages 1/2/4 complete;
  stage 3 blocked on the admin GUI; stage 5 blocked on epic FELICIA-PAGES-01.
  Baseline includes PR #53 (canonical contract v1, local authoring schema v1,
  read-only admin shell wired to the admin API, enforced media boundary).
