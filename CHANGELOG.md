# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- **Live Photo companion linking at intake** ([#173](https://github.com/azusachino/felicia/pull/173)):
  - `journey ingest` pairs image + `.MOV`/`.MP4` companions by file stem, installs them into the content-addressed blob store, and links them on the photo record. Companions are stored, not published (see Removed).
- **Journey enrichment: transport leg modes and day weather** ([#173](https://github.com/azusachino/felicia/pull/173), [#174](https://github.com/azusachino/felicia/pull/174)):
  - Typed leg-mode enum (`car`, `ferry`, `train`, `plane`, `foot`) and weather records flow from trip-folder metadata and portable packages through SQLite to the public contract; unknown modes, malformed weather, and mode/segment count mismatches fail intake loudly.
  - Canonical v1 schema extended with optional `leg_modes`/`weather`.
  - Optional automatic Open-Meteo archive weather lookup at ingest, using the departure-area point for all days and omitting null/incomplete entries ([#174](https://github.com/azusachino/felicia/pull/174)).
- **Photo re-import protection** ([#174](https://github.com/azusachino/felicia/pull/174)):
  - Package re-imports preserve authored photo captions, ordering, and memento assignment; only source-owned metadata (hash, companion link, timestamps) updates.

### Changed

- **Generation-atomic publication** ([#174](https://github.com/azusachino/felicia/pull/174)):
  - `felicia-cli static compile` and the server compile endpoint build into a staging clone and swap atomically; a failed rebuild leaves the previously published generation byte-for-byte intact.
- **Explicit SQLite schema upgrades** ([#174](https://github.com/azusachino/felicia/pull/174)):
  - Missing enrichment columns are added in one controlled transaction at startup; upgrade and serialization failures surface as errors instead of silently empty data.
- **Honest playback** ([#174](https://github.com/azusachino/felicia/pull/174)):
  - The reader no longer invents city names from coordinates or placeholder departure/arrival times; absent facts stay absent.

### Removed

- **Live Photo video publishing** ([#175](https://github.com/azusachino/felicia/pull/175)):
  - Stills only: companion videos stay stored at intake but are never published; the ffmpeg transcode path and its CI dependency are gone ([ADR 0009](docs/adr/0009-retire-live-photo-video-publishing.md)). Already-published JSON containing `video_key` is ignored by the reader.

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

- **CLI Modularization**:
  - Split `apps/felicia-cli/cmd/felicia/main.go` into dedicated single-responsibility command modules (`journey.go`, `package.go`, `import.go`, `static.go`, `helpers.go`, and minimal `main.go`).
- **CLI Framework Migration to `spf13/cobra`**:
  - Replaced standard library `flag` in `apps/felicia-cli` with `github.com/spf13/cobra` and `pflag`.
  - Structured command hierarchy around `felicia-cli` with `journey`, `package`, `import`, and `static` subcommands, preserving POSIX flag compatibility, help generation, and JSON/JSONL output contracts.
- **Desktop Studio Featured in Core Documentation**:
  - Restructured `README.md` to highlight **Felicia Desktop Studio** as the primary human visual authoring environment alongside `felicia-cli`, detailing the asymmetric workflow, interactive MapLibre track inspection, stop candidate triage, memento essay authoring, and in-app publication preview.
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
