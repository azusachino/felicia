---
title: "Epic: SQLite-backed GitHub Pages publication"
status: "shipped"
date: "2026-07-19"
---

# Epic: SQLite-backed GitHub Pages publication

**Epic key:** `FELICIA-PAGES-01`
**Status:** Shipped — merged to `main` in PR #55; the Pages workflow ran end
to end on the merge commit. Delivery status is tracked in
[`../roadmap.md`](../roadmap.md) and
[`user-journey.md`](user-journey.md); this document records scope and
acceptance only.
**Goal:** replace the fixture-only Pages demo with a reproducible SQLite-backed
static publication while preserving the verified public `.json` contract.

## Outcome

Given a clean SQLite database, a local media root, and optional GPX source
files, Felicia produces a portable `dist/` artifact containing the public SPA,
`.json` read projections, route geometry, and safe public media derivatives. The
artifact is previewable locally, deployable by GitHub Actions, and readable
without an API, database, Valkey, or cloud credential.

## Child tasks

All ten child tasks shipped as part of PR #55.

| Task                                          | Issue | Status | Outcome                                                                                                                          |
| --------------------------------------------- | ----- | ------ | -------------------------------------------------------------------------------------------------------------------------------- |
| 01.1 Define the static publication contract   | #48   | done   | Contract test validates both generated `.json` files and Go server responses without design-specific fields.                     |
| 01.2 Compile static output from SQLite        | #43   | done   | Clean SQLite fixture produces a complete artifact with no JSON seed input, no PostgreSQL/Valkey/API dependency.                  |
| 01.3 Preserve GPX route provenance            | #42   | done   | Rebuilding twice from the same DB yields equivalent route JSON and keeps provenance without exposing private source files.       |
| 01.4 Add local filesystem media publication   | #41   | done   | One photo, one unsupported file, one private original → safe derivative plus deterministic media URL.                            |
| 01.5 Prove server/static contract parity      | #44   | done   | Browser can switch between static preview and SQLite server mode without changing read models or hitting `Unexpected token '<'`. |
| 01.6 Measure the 100-journey artifact         | #46   | done   | Measurements recorded; the index stays bounded enough for GitHub Pages.                                                          |
| 01.7 Run and harden GitHub Pages deployment   | #45   | done   | Clean runner publishes a browsable artifact without secrets or local services.                                                   |
| 01.8 Privacy, accessibility, and rebuild gate | #47   | done   | Publication is safe to share and reproducible from a clean checkout.                                                             |
| 01.9 Provide an agent-friendly repository CLI | #49   | done   | Validate, diff, build, and arrange the artifact for a Git commit via documented commands.                                        |
| 01.10 Make forks portable                     | #50   | done   | Fork runs the CLI and workflow without editing source paths or leaking private data.                                             |

## Current gap audit

| Gap                                                                      | Evidence                                                                  | Covered by |
| ------------------------------------------------------------------------ | ------------------------------------------------------------------------- | ---------- |
| Static builder reads checked-in JSON, not SQLite                         | `scripts/build_static_demo.py`                                            | 01.1, 01.2 |
| Existing static compiler is PostgreSQL-specific                          | `apps/felicia-server/cmd/build/main.go`                                   | 01.1, 01.2 |
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
