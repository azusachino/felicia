# felicia v1 roadmap

Epics: [SQLite-backed GitHub Pages publication](roadmap/pages-v1-epic.md)
(implementation complete pending PR review; remote workflow run outstanding)
and the active [admin GUI MVP](roadmap/admin-gui-v1-epic.md) (M1 landed;
intake inbox and memento editor in progress).
The selected end-to-end journey and its per-stage status: [User journey](roadmap/user-journey.md).

> Drafted 2026-07-16. This is a delivery roadmap, not a commitment to build every future seam in the current release.

## v1 outcome

Ship one beautiful, public, personal travel journal: one real journey is imported or authored, shown on a map, and readable through designed mementos that open into essays and photos. The author can
curate and publish it from a private admin surface, and the deployed site runs safely on a supported production host with configurable ingress.

The map remains the index. A memento is the story's click target. The first release is personal-first and product-ready at the seams; it is not a multi-user SaaS product.

## Release boundary

### In v1

- Public reader for journeys, routes, visits, mementos, essays, and galleries.
- One canonical API/data contract used by the public frontend and admin surface.
- SQLite-first local storage, with the existing PostgreSQL path kept healthy.
- Authored content that is never overwritten by ingestion.
- A small, declarative memento template set: `goods`, `transit`, `stamp`, and `receipt`.
- Japanese-first system UI with English and Chinese catalogs; authored content is shown exactly as entered.
- One authoring flow: create or import a journey, curate mementos/photos, preview, and publish.
- One real journey as the acceptance fixture and one complete end-to-end publish flow.
- EXIF-stripped, resized public media stored behind an S3-compatible interface.
- Containerized deployment on a supported host, with ingress/tunnel and object storage selected by deployment configuration.
- Two intake modes through one draft pipeline: connected Dawarich/Immich sources and versioned user-provided journey packages.
- Dry-run import, reviewable changes, stable package identities, and agent-friendly validate/import/diff commands.
- Optional OCR/AI suggestions for structured fields; no AI provider is required to import, author, or publish.

### Explicitly out of v1

- Multiple accounts, teams, OAuth, billing, subscriptions, or a companion mobile app.
- Background sync and a general-purpose ingestion platform.
- Public comments, social feeds, likes, or analytics dashboards.
- Automatic translation of authored titles, essays, captions, or memento metadata.
- A full template marketplace or arbitrary user-defined template language.
- Open-ended provider support beyond the seams needed for SQLite, PostgreSQL, R2/S3, Immich, Dawarich, and a manual/GPX fallback.

## Milestones

### M0 — Product and content lock

Turn the research decisions into a small executable spec before expanding the UI.

Deliverables:

- Freeze the first reader flow: landing index → journey → memento → essay/gallery.
- Choose the first two visual open interactions and memento forms to polish; the current lean is warm paper plus a shared-element morph.
- Freeze the public read contract and the authoring write contract.
- Select one real journey and define its route, visits, mementos, essays, and photos.
- Write the v1 acceptance scenarios and privacy invariants.
- Define the import-package manifest, review states, and agent confirmation boundary.

Exit check: a reviewer can describe the complete author and reader journey without falling back to an archived design draft.

### M1 — Canonical storage and public API

Make the existing domain and provider work the source of truth for the reader.

Deliverables:

- Complete the goose schema/migration path for the stable memento-era model.
- Implement the package adapter and canonical intake boundary alongside connected source adapters; both must land as draft import runs.
- Keep the four Go module boundaries: core, runtime, providers, and API server.
- Prove SQLite contract tests and keep PostgreSQL contract tests passing.
- Implement the public journey index, journey detail, memento, route, visit, and media projections under `/api/v1`.
- Add the flat memento read projection only if the chosen v1 landing needs it.
- Keep geometry, ordering, provenance, and locale behavior presentation-agnostic.

Exit check: the public app can run entirely from the API against a clean SQLite database, with no fixture-only behavior in the production path.

### M2 — Public reader v1

Replace the design showcase with the first coherent Felicia reading experience.

Deliverables:

- Make the landing index reachable on first load and useful without knowing the map UI.
- Keep a dark MapLibre route map, journey index, visit/memento markers, and responsive mobile layout.
- Render the first memento templates from structured data rather than per-item markup.
- Implement the signature open interaction into an essay and photo gallery.
- Remove prototype/scaffolding chrome from the public reader.
- Verify Japanese, English, Chinese system UI, keyboard navigation, reduced motion, and readable essay typography.

Exit check: a first-time visitor can discover a journey, open a memento, read its story, view its photos, and return to the index on desktop and mobile.

### M3 — Authoring and publish flow

Give the single author a reliable private surface for producing the public artifact.

Deliverables:

- Journey create/edit: title, dates, place, route import/manual fallback, visibility.
- Memento create/edit driven by the declarative template registry.
- Photo upload, ordering, captions, derivative generation, and EXIF stripping.
- Preview the public projection before publishing.
- Review imported candidates and enrichment suggestions before applying authored fields.
- Enforce revision/conflict checks and the rule that imports never clobber authored fields.
- Publish/unpublish as an explicit boundary; draft content must not leak into public reads.

Exit check: the selected real journey can be created or repaired through the admin UI, previewed, published, and then read through the same public API as any other journey.

### M4 — Ingestion and route enrichment

Automate the parts that are repetitive while preserving author control.

Deliverables:

- Import a GPX/manual route as the dependable fallback.
- Add Dawarich track and visit ingestion behind a provider interface.
- Add Immich photo/stub ingestion and timestamp joining.
- Add the versioned package import path for route/timeline/photos/notes.
- Keep OCR/AI enrichment optional and confirmation-based.
- Implement idempotent, field-scoped upserts with provenance and import-run history.
- Snap point mementos to visits using temporal checks before spatial fallback.
- Compose authored transit legs and passive tracks into the display route.

Exit check: re-running an import produces no duplicates, does not overwrite authored fields, and leaves an auditable result when source data is incomplete.

### M5 — Production deployment and v1 acceptance

Make the result dependable as a personal service.

Deliverables:

- Containerized API, public app, admin app, database, and object-storage configuration.
- Production deployment through Docker Compose or the selected hosting platform.
- Configurable ingress, TLS, and object storage; no hardware or tunnel vendor is part of the product contract.
- Backup/restore procedure for authored database state and media references.
- Health checks, structured logs, and privacy checks for public geometry/media.
- Seed the real journey and run the full reader → author → import → publish workflow.
- Run `make validate`, the browser checks, and a manual mobile/desktop acceptance pass.

Exit check: the owner can publish a new memento without an agent or developer, and a visitor can read the published journal from the public URL.

## Verification gates

Every milestone keeps the smallest relevant gate green:

- Go: `make fmt-check`, `make vet`, focused tests, then `make check`.
- Web: `make web-check` plus browser checks for the public and admin flows.
- Data: clean-database migration, seed, API contract, and re-import idempotency checks.
- Privacy: no raw GPS in public responses; public images are resized and EXIF-stripped.
- Release: `make validate`, deployment smoke test, backup/restore rehearsal, and manual responsive/accessibility review.

## Suggested order of work

1. M0 product/content lock.
2. M1 API and storage contract.
3. M2 public reader using one real journey.
4. M3 authoring and publish.
5. M4 ingestion automation.
6. M5 deployment and acceptance.

The important cut is to reach a complete public reader with one real journey before investing in the full Immich/Dawarich automation. The importer is valuable, but it is not the proof of Felicia's
core experience.
