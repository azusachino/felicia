# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- **Single-step trip folder intake** (`felicia-cli journey ingest --dir <path>`):
  - Ingests self-contained trip directories containing `route.gpx`, Google Timeline exports (`timeline.json`), photos (`photos/` or top-level), and `photos.jsonl` sidecars ([#166](https://github.com/azusachino/felicia/issues/166)).
  - Provisions sole journals, dedupes journey slugs, resolves GPS/timestamp metadata, content-addresses media into blob storage, and transactions draft records into SQLite.
- **Unified workspace resolution** (`apps/felicia-runtime/workspace`):
  - Shared resolution contract prioritizing explicit flags, `$FELICIA_WORKSPACE`, local `./.felicia` fallback, and user home `~/.felicia` ([#166](https://github.com/azusachino/felicia/issues/166)).
  - Standardized across `felicia-cli`, `felicia-desktop`, and runtime services.
- **Trip folder contract & agent workflow specifications**:
  - `docs/contracts/trip-folder-contract.md`: Formal specification of trip directory contracts, sidecar formats, and CLI flags.
  - `docs/research/agent-and-desktop-workflow.md`: Specification of the asymmetric workflow (AI agents as headless intake engines, Desktop Studio as the human visual authoring studio).
- **Authorship protection and re-ingest invariants**:
  - Verified and tested that repeated ingestion preserves human-curated titles, candidate states, author labels, and memento essays without clobbering (`authored_fields`).

### Changed

- **Decoupled `felicia-cli` from Makefile**:
  - `felicia-cli` is positioned as a standalone CLI tool (`go install ./apps/felicia-cli/cmd/felicia` or `go build`).
  - Removed internal `cli-build` make target and makefile prerequisites; `make site-build` now executes via `go run`.
- **Server Default Workspace Alignment**:
  - `apps/felicia-server` now defaults its database and media root paths to `workspace.Resolve("")`, guaranteeing that headless server authoring opens the exact same SQLite store as CLI and Desktop Studio.
- **Documentation & Agent Boundary**:
  - Updated `README.md`, `AGENTS.md`, and `docs/publish.md` to establish `felicia-cli` as the sole intake tool for agents.
  - Marked legacy manual YAML and Postgres references in `docs/research/ingestion-workflows.md` as superseded by SQLite and the trip folder intake contract.

### Removed

- **Dead Provider Spike**:
  - Removed unused `apps/felicia-providers/manualyaml` package.
- **Zombie PostgreSQL Scaffolding**:
  - Purged dead PostgreSQL provisioning and database lifecycle code from `scripts/test_journey_workflow.py`.
- **Legacy Python Wrappers and Redundant Targets**:
  - Removed `scripts/local_journey_author.py` (legacy terminal authoring prompt wizard superseded by Desktop Studio).
  - Removed `make journey-local` target from `Makefile` and obsolete wrapper documentation from `docs/publish.md`.
  - Pruned redundant `make dev-sqlite` and `make test-sqlite` aliases from `Makefile`, and corrected `make db-up` descriptions.
