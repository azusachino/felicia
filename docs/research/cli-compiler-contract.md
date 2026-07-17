---
title: "CLI compiler contract"
status: "proposed"
date: "2026-07-17"
---

# CLI compiler contract

This note turns the existing domain and package decisions into an implementation
sequence. The first runnable CLI path now exists; this remains the contract for
its deliberately small v0.1 surface.

## User raw-data workspace

User-owned source files are not repository fixtures and are not written directly
to the public artifact. A local installation uses this boundary:

```text
.felicia/
  inbox/              raw ZIPs, GPX, Timeline exports, and original media
  felicia.sqlite      local canonical working store
  import-reports/     dry-run, conflict, and unresolved-source reports

content/
  journeys/<slug>/    reviewed authored source and selected public media

site/                 generated static output
```

`.felicia/` is gitignored. `content/` is versioned only when the author chooses
to keep reviewed source in the repository. The CLI must never copy raw originals
from `.felicia/inbox` into a public GitHub Pages artifact automatically.

## What already exists

| Concern                          | Current location               | Assessment                                       |
| -------------------------------- | ------------------------------ | ------------------------------------------------ |
| Canonical entities and lifecycle | `core/domain`                  | Reusable starting point                          |
| Storage ports                    | `core/ports`                   | Reusable; publication needs a narrower read port |
| SQLite provider                  | `providers/sqlite`             | First CLI persistence target                     |
| PostgreSQL provider              | `providers/postgres`           | Server/deployment target                         |
| Import joining/no-clobber logic  | `runtime/importer`             | Reusable runtime seam                            |
| Public projection                | `publication`                  | Shared boundary now exists                       |
| Fixture build script             | `scripts/build_static_demo.py` | Demo only; not package compilation               |
| `felicia-cli`                    | `cli/cmd/felicia`              | SQLite import and static compiler entry point    |

## Canonical model

```text
Journal -> Journey -> route geometry + source provenance
                    -> Memento(kind + kind_data + authored fields)
                         -> MementoPhoto(object key + hash + curation)
                    -> TransitLeg (authored route additions)
                    -> ImportRun / SourceObservation (provenance)
```

GPX, Google Timeline exports, Immich/Dawarich records, and local photos are
source material. They normalize into the same route, visit, memento-candidate,
and media structures; they are not alternate canonical models.

Incremental imports preserve stable package and record identities. Existing
records are not renumbered; new mementos require an explicit `seq` or receive a
new sequence after the current maximum. Re-imports merge source-owned fields,
preserve authored fields, and report conflicts instead of silently replacing
content.

## Package-to-static acceptance case

The first end-to-end fixture should contain:

- `manifest.yaml` with checksums and package identity;
- `journey.yaml` with one journey and authored metadata;
- one real `route.gpx` with timestamped track points;
- `timeline.json` containing a timestamped location event;
- `mementos.yaml` containing transit, stamp, receipt, and goods `kind_data`;
- two local JPEGs and one unsupported/private file;
- unknown metadata retained as unresolved data rather than discarded;
- a SQLite database populated by `felicia-cli import --apply`;
- a static artifact containing the route, public mementos, and only safe media.

The verifier must assert GPX provenance, coordinate normalization, media
checksums, kind-data preservation, deterministic `.json` paths, and exclusion
of private/unsupported files. This replaces the current fixture demo check.

## Implementation order

1. Extend the shared publication DTOs and compiler ports in `publication`.
2. Define package DTOs and ZIP validation without a database dependency.
3. Implement GPX and local-media adapters.
4. Implement import plan/apply against SQLite through existing ports.
5. Add `cli/cmd/felicia` with package validation, import, and static compile.
6. Add the end-to-end fixture and compile it into deterministic `.json` paths.
7. Recompose the postponed server and PostgreSQL compiler from the same seams.

## v0.1 commands

The CLI is intentionally local-first and does not contact a server:

```text
felicia-cli package validate <journey.zip>
felicia-cli import [--db .felicia/felicia.sqlite] [--media-root content/media] [--apply] <journey.zip>
felicia-cli static compile --db .felicia/felicia.sqlite --media-root content/media --out site
```

`import` is a dry run unless `--apply` is provided. Static compilation writes
the same `/api/v1/*.json` shape consumed by the public web app and copies only
media referenced by published mementos. The ZIP package remains an intake
format; the SQLite database and selected media are the reviewed working state.
