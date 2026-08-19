# Archived status log — user-journey.md

Entries moved verbatim from the "## Status log" section of
[`../roadmap/user-journey.md`](../roadmap/user-journey.md) once they were no
longer the two most recent. Kept for history; not maintained going forward.

## Status log (archived)

- **2026-08-04 (epic ADMIN-02 M2)** — Site identity landed: a journal-scoped
  `tb_site_settings` row (title, description, active design, default
  language/theme, accent) is projected through the shared `publication`
  boundary to `GET /api/v1/site(.json)`, with defaults applied when nothing
  has been configured yet so the endpoint is never a 404. Authored from the
  GUI via `GET/PUT /api/admin/site-settings` (Site & Deploy page gained a
  Site identity section: 4 design cards, title/description, language/theme
  selects, a native color picker for accent, one explicit Save action). The
  public reader now boots into the configured design instead of the old
  hash-based switcher (removed), and seeds its language/theme from the same
  settings; the accent color is wired into all four designs (v1/v2 share
  `--accent-ink`, v4's `--orange`, and v3's `--terracotta` plus its few
  Tailwind-utility-class spots) via a shared `--accent` CSS variable.
  Verified in the containerized toolchain: `make validate`, the extended
  live/static parity check for `site.json`, and `make test-admin-e2e`
  (13/13), which now drives the Site identity flow itself — picks a design,
  saves title/accent, reloads to confirm persistence, builds, and checks
  both the live `/api/v1/site` response and the compiled `site.json` on
  disk.

- **2026-07-22 (issue audit & milestone reconciliation)** — Conducted a full audit of open GitHub issues against implementation status. Verified completion and closed 21 issues across M0 (content & acceptance lock), M1 (canonical storage & public API projections), M2 (declarative templates & projections), M3 (admin authoring GUI MVP), M4 (ingestion connectors & intake planner), and epic FELICIA-PAGES-01 (#40–#50). Added 3 new enhancement issues on GitHub: #57 (automatic journey date bounds derivation), #58 (offline timezone resolution via `tzf`), and #59 (optional local AI agent ticket artwork generator for missing media in admin GUI). Updated roadmap and delivery trackers accordingly.

- **2026-07-19 (memento lifecycle + staged rebuild)** — Formalized the
  memento lifecycle as a binding contract
  ([docs/contracts/memento-lifecycle.md](../contracts/memento-lifecycle.md))
  and enforced it in code: a transition table with provider- and API-level
  guards (illegal jumps → 422 `invalid_transition`), structured state-change
  logging with an event source, delete restricted to non-public states
  (published must be unpublished first → 422 `delete_requires_unpublish`),
  and an omitted state that keeps the current one instead of downgrading.
  Publish/unpublish no longer rebuild eagerly; instead a change to what is
  public is tracked as **pending-build** (compared against the deployed
  artifact, stateless) and shown as highlighted rows plus a `(N)` count on
  the journey-detail and journeys-list Build actions — the author batches
  edits and resolves them with one explicit build. Site & Deploy became the
  output-location chooser with a local directory picker. Verified in the
  containerized toolchain; the closed-loop E2E now drives the staged
  unpublish → pending → build → cleared round trip (10/10).
- **2026-07-19 (epic ADMIN-02 M1)** — Authoring controls from the hands-on
  M0 review: unpublish (published → authored, bidirectional lifecycle),
  memento deletion (new DELETE endpoint across both providers, photos
  cascade, informed two-step confirm), the inbox's discard action named
  and explained, a Build & preview shortcut on journey detail, and topbar
  spacing. The closed-loop E2E now covers an unpublish → re-publish round
  trip (8 steps).
- **2026-07-19 (epic ADMIN-02 M0)** — Offline local deployment landed: the
  GUI's new Site & Deploy page builds the static artifact with one action
  (compile now defaults to the configured `site.out_dir`) and links a
  built-in preview server on a second local port that serves the compiled
  site exactly as a static host would (artifact overlaid on the pre-built
  public SPA). The closed-loop E2E builds through the GUI and asserts the
  preview port serves the compiled manifest. Design pick/style, GitHub
  Pages deploy with URL confirmation, and GUI resource uploads are planned
  in [FELICIA-ADMIN-02](../roadmap/admin-gui-v2-epic.md).
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
  [FELICIA-ADMIN-01](../roadmap/admin-gui-v1-epic.md) designed (adversarially reviewed)
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
