---
title: "Epic: SQLite-backed GitHub Pages publication"
status: "proposed"
date: "2026-07-17"
---

# Epic: SQLite-backed GitHub Pages publication

**Epic key:** `FELICIA-PAGES-01`
**Status:** Proposed; not a settled architecture decision
**Goal:** replace the fixture-only Pages demo with a reproducible SQLite-backed
static publication while preserving the verified public `.json` contract.

## Outcome

Given a clean SQLite database, a local media root, and optional GPX source
files, Felicia produces a portable `dist/` artifact containing the public SPA,
`.json` read projections, route geometry, and safe public media derivatives. The
artifact is previewable locally, deployable by GitHub Actions, and readable
without an API, database, Valkey, or cloud credential.

## Child tasks

### FELICIA-PAGES-01.1 — Define the static publication contract

Document the exact input/output contract before replacing the fixture builder.

- Define the required SQLite read ports for journals, journeys, mementos, route
  geometry, and media metadata.
- Define the output paths and JSON schemas:
  `/api/v1/journeys.json`, `/api/v1/journeys/<id>.json`, and
  `/api/v1/journeys/<id>/mementos.json`.
- Decide whether the public geometry is GeoJSON only or includes a Felicia
  extension for timestamp/elevation provenance.
- Add golden fixtures and a contract test shared by static and server readers.

Acceptance: a contract test can validate both generated files and Go server
responses without design-specific fields.

### FELICIA-PAGES-01.2 — Compile static output from SQLite

Replace `scripts/build_static_demo.py` as the production-shaped path; retain it
only as a small fixture/demo helper if useful.

- Open a clean SQLite database through the provider interface.
- Read published journeys and mementos through provider-neutral runtime ports.
- Generate the same `.json` files as the current demo.
- Make the compiler fail closed on unpublished/private records.
- Remove direct PostgreSQL construction from the static path.

Acceptance: a clean SQLite fixture produces a complete artifact with no JSON
seed input and no PostgreSQL/Valkey/API dependency.

### FELICIA-PAGES-01.3 — Preserve GPX route provenance

- Import one checked-in GPX fixture into the canonical SQLite route representation.
- Preserve source filename/checksum and import-run identity.
- Preserve timestamp and elevation when available.
- Generate deterministic public geometry from the canonical route data.
- Add round-trip and malformed-GPX cases.

Acceptance: rebuilding twice from the same SQLite database produces equivalent
route JSON and retains source provenance without exposing private source files.

### FELICIA-PAGES-01.4 — Add local filesystem media publication

- Read originals from a configured local media root.
- Generate size-bounded public derivatives and strip EXIF/GPS metadata.
- Copy only referenced public derivatives into `dist/media/`.
- Keep originals outside the artifact and reject path traversal.
- Use stable content-hash paths and preserve captions/order in JSON.

Acceptance: a fixture with one photo, one unsupported file, and one private
original produces only the safe derivative and a deterministic media URL.

### FELICIA-PAGES-01.5 — Prove server/static contract parity

- Exercise `.json` paths against the Go server backed by SQLite.
- Compare server responses with the static files from the same fixture.
- Keep extensionless server aliases only for backward compatibility and test
  both forms explicitly.
- Verify Caddy/API routing does not turn missing JSON into SPA HTML.

Acceptance: the browser can switch between static preview and SQLite server
mode without changing read models or encountering `Unexpected token '<'`.

### FELICIA-PAGES-01.6 — Measure the 100-journey artifact

- Generate 100 journeys with representative mementos and media metadata.
- Measure SQLite size, compiler duration, artifact size, and index size.
- Keep journey details lazy-loadable rather than embedding all essays in the
  index.
- Add a browser smoke check for the index and one detail page.

Acceptance: measurements are recorded and the index remains bounded enough for
GitHub Pages; any limit becomes an explicit product constraint.

### FELICIA-PAGES-01.7 — Run and harden GitHub Pages deployment

- Run `.github/workflows/pages.yml` on the protected branch or via dispatch.
- Verify repository-site base path, `.json` requests, media URLs, and design
  hash switching on the deployed URL.
- Pin or review action versions and document required Pages repository settings.
- Record the deployed URL and workflow run in the experiment report.

Acceptance: a clean runner publishes a browsable artifact without secrets or
local services.

### FELICIA-PAGES-01.8 — Privacy, accessibility, and rebuild gate

- Assert no raw GPS/source files/private originals enter `dist/`.
- Check missing media and malformed JSON fail with useful diagnostics.
- Add keyboard/reduced-motion and basic mobile browser checks.
- Run two builds from identical inputs and compare manifest/output hashes.
- Add the complete gate to `make validate` only after it is deterministic.

Acceptance: the publication is safe to share and reproducible from a clean
checkout.

### FELICIA-PAGES-01.9 — Provide an agent-friendly repository CLI

- Define a repository-local layout for source packages, SQLite state, generated
  projections, and public media.
- Add deterministic commands for `validate`, `import`, `diff`, `build`, and
  `publish`.
- Make dry-run output reviewable and machine-readable.
- Keep generated files limited to configured public paths and never require
  hand-editing generated JSON.

Acceptance: an agent or user can validate a package, review a diff, build the
public artifact, and arrange it for a Git commit using documented commands.

### FELICIA-PAGES-01.10 — Make forks portable

- Derive repository name and project-site base path from GitHub context.
- Remove hardcoded owner, repository, asset, and local absolute paths.
- Document Pages settings for forks and keep workflow permissions minimal.
- Add a clone-at-another-path smoke fixture.

Acceptance: a fork can run the CLI and workflow without editing source paths or
leaking private data.

## Current gap audit

| Gap                                                                      | Evidence                                                                  | Covered by |
| ------------------------------------------------------------------------ | ------------------------------------------------------------------------- | ---------- |
| Static builder reads checked-in JSON, not SQLite                         | `scripts/build_static_demo.py`                                            | 01.1, 01.2 |
| Existing static compiler is PostgreSQL-specific                          | `server/cmd/build/main.go`                                                | 01.1, 01.2 |
| GPX exists but is not in the Pages build path                            | `scripts/tracks/*.gpx` and provider baseline                              | 01.3       |
| Demo media mapping is hardcoded; no safe local FS pipeline               | `scripts/build_static_demo.py`                                            | 01.4       |
| Server `.json` aliases are newly added but parity is unproven end-to-end | Go route tests pass; browser parity remains untested                      | 01.5       |
| 100-journey behavior is unmeasured                                       | Experiment E6 is open                                                     | 01.6       |
| Pages workflow has not run remotely                                      | `.github/workflows/pages.yml` exists only locally                         | 01.7       |
| Privacy/accessibility/rebuild checks are incomplete                      | Pages experiment open gaps                                                | 01.8       |
| No repository-local CLI arranges source/import/build/publish data        | Existing commands are workflow scripts, not a stable user/agent interface | 01.9       |
| Fork behavior is not tested                                              | Workflow derives the base path, but no fork smoke case exists             | 01.10      |

## Not part of this epic

- choosing the winning visual design;
- replacing SQLite/PostgreSQL with another database;
- implementing OCR/AI enrichment;
- building Immich/Dawarich synchronization;
- adding accounts, teams, or a mobile application.

Those remain separate decisions and should not block proving the portable
publication path.

## GitHub issue wiring

The epic and child issues are wired into project #8 (`felicia v1.0`) with parent
relationships and the existing Status/Phase fields. **GitHub is the single
ledger for issue state** — the original local issue drafts are archived under
[`docs/archive/github-issues/`](../archive/github-issues/pages-01.1.md) and are
no longer kept in sync. Scripted status checks should use a `GITHUB_TOKEN`
environment variable rather than interactive `gh auth login`.

1. [#40 Epic: SQLite-backed GitHub Pages publication](https://github.com/azusachino/felicia/issues/40)
2. [#48 Define static publication contract](https://github.com/azusachino/felicia/issues/48)
3. [#43 Compile static output from SQLite](https://github.com/azusachino/felicia/issues/43)
4. [#42 Preserve GPX route provenance](https://github.com/azusachino/felicia/issues/42)
5. [#41 Add local filesystem media publication](https://github.com/azusachino/felicia/issues/41)
6. [#44 Prove server/static contract parity](https://github.com/azusachino/felicia/issues/44)
7. [#46 Measure the 100-journey artifact](https://github.com/azusachino/felicia/issues/46)
8. [#45 Run and harden GitHub Pages deployment](https://github.com/azusachino/felicia/issues/45)
9. [#47 Add privacy, accessibility, and rebuild gates](https://github.com/azusachino/felicia/issues/47)
10. [#49 Provide an agent-friendly repository CLI](https://github.com/azusachino/felicia/issues/49)
11. [#50 Make forks portable](https://github.com/azusachino/felicia/issues/50)
