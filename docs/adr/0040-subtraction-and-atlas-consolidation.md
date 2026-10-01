---
id: "0040"
title: "Subtraction and Atlas Consolidation"
status: "accepted"
date: "2026-10-01"
related:
  - "0003"
  - "0006"
  - "0007"
  - "0027"
  - "0032"
  - "0034"
  - "0038"
---

# ADR 0040: Subtraction and Atlas Consolidation

## Context

Following Iroha's 2026-09-28 reset ("Grapher only") and Asobi 0.7's subtraction thesis ("subtract first, then re-measure, then add only what still hurts"), Felicia reviewed its architectural surface area.

Although Phase 0–3 foundation repairs successfully resolved content-addressed media identity, safe preview workspaces, atomic journey ingest, and draft preservation, the project carries severe speculative abstraction and multi-variant maintenance churn:

1. **4 reader design languages** (Atlas, Cabinet, Techo, Cartography) plus an incubating 5th (Tabi), requiring repetitive audits and remediation across 33+ interface defects (contrast, hit areas, keyboard traps, reduced motion).
2. **A theme registry and hash routing** (`#cabinet`, `#techo`, `#cartography`) that exists solely to switch among incomplete presentation variants.
3. **3 frontend applications** (`felicia-admin`, `felicia-web`, `felicia-public-site`), where `felicia-web` and `felicia-public-site` are nearly identical wrappers around `@felicia/reader`.
4. **5 monorepo packages** including micro-packages with fewer than 30 lines (`packages/felicia-renderers` [25 LOC], `packages/felicia-runtime` [80 LOC]) created speculatively for 3D map renderers that were never adopted.
5. **A frozen PostgreSQL provider** (`apps/felicia-providers/postgres`, 3,341 LOC) and PostgreSQL-only Goose migrations maintained in the tree despite [ADR-0032](0032-sqlite-first-v1-postgres-follow-up.md) declaring SQLite the sole persistence contract for v1.
6. **5,113 lines of Python scripts and tests** duplicating workflows already implemented by Go's `felicia-cli`.

## Decision

**Subtract speculative complexity and consolidate Felicia into a lean, single-reader, single-persistence studio.**

1. **Atlas is the sole reader.**
   - Retire Cabinet, Techo, Cartography, and incubating Tabi.
   - Delete `packages/felicia-reader/src/theme-ui/{cabinet,techo,cartography,_incubating}/`.
   - Delete `packages/felicia-reader/src/theme-ui/registry.ts` and the theme switcher. Export `Reader.svelte` (Atlas) directly from `@felicia/reader`.
   - Use hash anchors (`/#journey-<id>`, `/#memento-<id>`) for shareable deep links on static hosting.

2. **Consolidate frontend applications to two.**
   - Delete `apps/felicia-web`.
   - Retain `apps/felicia-public-site` as the single reader shell, configurable via `VITE_API_BASE` for local preview or loading static `/journeys/catalog.json` in production.
   - Keep `apps/felicia-admin` as the private authoring studio.

3. **Flatten packages to `@felicia/model` and `@felicia/reader`.**
   - Fold `packages/felicia-renderers`, `packages/felicia-runtime`, and `packages/felicia-components` into `@felicia/reader`.
   - Keep `@felicia/model` as headless, pure TypeScript contracts without Svelte or DOM dependencies.

4. **Hard-delete PostgreSQL provider and Goose migrations.**
   - Delete `apps/felicia-providers/postgres` (3,341 LOC).
   - Delete `apps/felicia-server/migrations/` (Postgres-only migrations). Remove `goose` from `Makefile` and `mise.toml`.
   - SQLite embedded `schema.sql` is the sole persistence schema for Felicia v1.

5. **Canonical Go CLI.**
   - Standardize all package creation, import, and compilation workflows on Go's `felicia-cli`.
   - Deprecate redundant Python authoring scripts (`local_journey_author.py`, `local_journey_package.py`).

## Consequences

### Positive
- Removes ~10,000 lines of unearned code and redundant maintenance surface.
- Eliminates 33+ duplicate UI audit issues across secondary themes.
- Reader bundle and build times are substantially reduced.
- Eliminates external database dependencies, Goose tooling, and dual-provider test burdens.

### Negative / Trade-offs
- Cabinet's "greatest hits" shelf and Cartography's full-map index are no longer switchable in the public reader (preserved in Git history).
- Multi-user remote PostgreSQL deployments are not supported in v1; Felicia is strictly a single-user local studio producing a static publication.
