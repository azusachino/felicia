# felicia v1 roadmap

Epics: [SQLite-backed GitHub Pages publication](roadmap/pages-v1-epic.md)
(shipped, PR #55); [admin GUI MVP](roadmap/admin-gui-v1-epic.md) (M1–M4
landed, in review); [GUI site configuration and deployment](roadmap/admin-gui-v2-epic.md)
(M0–M2 landed, M3–M4 planned). Selected end-to-end journey and its
per-stage status: [User journey](roadmap/user-journey.md).

> Drafted 2026-07-16 — a delivery roadmap, not a commitment to build every
> future seam in the current release.

## v1 outcome

Ship one beautiful, public, personal travel journal: one real journey,
imported or authored, mapped, and readable through mementos that open into
essays and photos — curated and published from a private admin surface,
deployed safely with configurable ingress.

## Release boundary

### In v1

- Public reader for journeys, routes, visits, mementos, essays, and galleries.
- One canonical API/data contract used by the public frontend and admin surface.
- SQLite-first local storage; PostgreSQL/PostGIS is deferred to v1.1/v1.2,
  not a v1 compatibility promise.
- Authored content that is never overwritten by ingestion.
- A declarative memento template registry (`core/kinds/*.yaml`): `goods`, `live`, `transit`, `stamp`, `receipt`, and `souvenir`; admin MVP hand-builds forms for `transit`/`goods` first.
- Japanese-first system UI with English and Chinese catalogs; authored content is shown exactly as entered.
- One authoring flow: create or import a journey, curate mementos/photos, preview, and publish.
- One real journey as the acceptance fixture and one complete end-to-end publish flow.
- EXIF-stripped, resized public media stored behind an S3-compatible interface.
- Containerized single-owner deployment on a supported host, with private
  admin access, public-reader ingress, and object storage selected by config.
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

> Numbering convention: roadmap-level milestones are **R0–R5**; each
> epic's own local milestones are **M1–M4** (e.g. "ADMIN-01 M2" is not a
> roadmap milestone). The delivery phases in
> [user-journey.md](roadmap/user-journey.md) (Phase 1 = static
> publication, Phase 2 = admin GUI) cut across R2–R5.

### R0 — Product and content lock

Deliverables:

- Freeze the first reader flow: landing index → journey → memento → essay/gallery.
- Choose the first two visual open interactions and memento forms to polish; the current lean is warm paper plus a shared-element morph.
- Freeze the public read contract and the authoring write contract.
- Select one real journey and define its route, visits, mementos, essays, and photos.
- Write the v1 acceptance scenarios and privacy invariants —
  [`docs/release/v1-acceptance.md`](release/v1-acceptance.md).
- Define the import-package manifest, review states, and agent confirmation boundary.

Exit check: a reviewer can describe the complete author and reader journey without falling back to an archived design draft.

### R1 — Canonical storage and public API

Deliverables:

- Complete the goose schema/migration path for the stable memento-era model.
- Implement the package adapter and canonical intake boundary alongside connected source adapters; both must land as draft import runs.
- Keep the four Go module boundaries: core, runtime, providers, and API server.
- Prove SQLite contract tests and keep PostgreSQL contract tests passing.
- Implement the public journey index, journey detail, memento, route, visit, and media projections under `/api/v1`.
- Add the flat memento read projection only if the chosen v1 landing needs it.
- Keep geometry, ordering, provenance, and locale behavior presentation-agnostic.

Exit check: the public app can run entirely from the API against a clean SQLite database, with no fixture-only behavior in the production path.

### R2 — Public reader v1

Deliverables:

- Make the landing index reachable on first load and useful without knowing the map UI.
- Keep a dark MapLibre route map, journey index, visit/memento markers, and responsive mobile layout.
- Render the first memento templates from structured data rather than per-item markup.
- Implement the signature open interaction into an essay and photo gallery.
- Remove prototype/scaffolding chrome from the public reader.
- Verify Japanese, English, Chinese system UI, keyboard navigation, reduced motion, and readable essay typography.

Exit check: a first-time visitor can discover a journey, open a memento, read its story, view its photos, and return to the index on desktop and mobile.

### R3 — Authoring and publish flow

Deliverables:

- Journey create/edit: title, dates, place, route import/manual fallback, visibility.
- Memento create/edit driven by the declarative template registry.
- Photo upload, ordering, captions, derivative generation, and EXIF stripping.
- Preview the public projection before publishing.
- Review imported candidates and enrichment suggestions before applying authored fields.
- Enforce revision/conflict checks and the rule that imports never clobber authored fields.
- Publish/unpublish as an explicit boundary; draft content must not leak into public reads.

Exit check: the selected real journey can be created or repaired through the admin UI, previewed, published, and then read through the same public API as any other journey.

### R4 — Ingestion and route enrichment

Deliverables:

- Import a GPX/manual route as the dependable fallback.
- Add Dawarich track and visit ingestion behind a provider interface.
- Add Immich photo/stub ingestion and timestamp joining.
- Add the versioned package import path for route/timeline/photos/notes.
- Keep OCR/AI enrichment optional and confirmation-based.
- Implement idempotent, field-scoped upserts with provenance and import-run history.
- Snap point mementos to visits using temporal checks before spatial fallback.
- Compose authored transit legs and passive tracks into the display route.
- Automatically derive default journey date bounds (`date_start`/`date_end`) from media and track timestamp bounds.
- Resolve default memento timezones (`occurred_tz`) from GPS coordinates via offline timezone lookup.

Exit check: re-running an import produces no duplicates, does not overwrite authored fields, and leaves an auditable result when source data is incomplete.

### R5 — Production deployment and v1 acceptance

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

1. R0 product/content lock.
2. R1 API and storage contract.
3. R2 public reader using one real journey.
4. R3 authoring and publish.
5. R4 ingestion automation.
6. R5 deployment and acceptance.
