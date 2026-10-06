<div align="center">

# 🌼 felicia

**フェリシア** — 琉璃雏菊（蓝费莉菊）

*A map-based travel journal. The map is the index; the mementos are the stories.*

[![status](https://img.shields.io/badge/status-implementation%20stage-e8a33d)](docs/roadmap.md)
[![backend](https://img.shields.io/badge/backend-Go%20%C2%B7%20SQLite-00add8)](docs/development/layout.md)
[![desktop](https://img.shields.io/badge/desktop-Wails%20v3%20%C2%B7%20macOS-333333)](apps/felicia-desktop/)
[![web](https://img.shields.io/badge/web-Svelte%205%20%C2%B7%20Vite-ff3e00)](packages/felicia-reader/)
[![license](https://img.shields.io/badge/license-AGPL--3.0-3da639)](LICENSE)

</div>

---

## What it is

Felicia is a personal travel journal. Journeys appear on a map; visits give places to memories; mementos open into essays and photo galleries. Import creates private, reviewable data. You author and explicitly publish what belongs on the public site.

- **Local-first:** SQLite database, media originals, drafts, and authoring state stay on your machine.
- **Asymmetric collaboration:** AI agents perform headless intake and spatial clustering via `felicia-cli`; humans visually curate routes, review candidates, and craft personal essays in **Felicia Desktop Studio**.
- **Authorship-safe:** Re-ingestion updates source-owned GPS tracks and photo timestamps without clobbering human-authored titles, candidate curation, or essays (`authored_fields`).
- **Private by default:** Only explicitly published content and stripped media derivatives enter the public static site.

## Architecture and components

```text
Trip folder / GPX / Timeline ─┐
Photos / sidecar (photos.jsonl) ┴─> felicia-cli journey ingest
                                         │
                                         ▼
                             Unified Workspace (~/.felicia)
                                   ┌─────┴─────┐
                                   ▼           ▼
                           Desktop Studio    Admin Web
                           (Visual author)   (API/Web)
                                   └─────┬─────┘
                                         ▼
                             felicia-cli static compile
                                         │
                                         ▼
                               Static Reader Site
```

- **`apps/felicia-desktop`**: Native desktop visual authoring studio (Wails v3 + SvelteKit) with interactive MapLibre map track inspection, stop candidate curation, rich memento essay writing, and in-app publication preview.
- **`apps/felicia-cli`**: Standalone Go CLI (`spf13/cobra`) providing single-step trip folder ingestion (`journey ingest`), draft planning, review patching, and static compilation.
- **`apps/felicia-admin`**: Headless web authoring studio alternative sharing components with Desktop Studio.
- **`apps/felicia-runtime` & `apps/felicia-core`**: Core domain logic, unified workspace resolver (`~/.felicia`), spatial visit clustering, content-addressed media blob store, and SQLite persistence.
- **`packages/felicia-reader` & `apps/felicia-public-site`**: Read-only, public Atlas reader.

## Quick start

Install the checked-in toolchain with `mise install`.

### 1. Ingest a trip folder with `felicia-cli`

Install or build the standalone CLI:

```sh
go install ./apps/felicia-cli/cmd/felicia
# or build into bin/:
go build -o bin/felicia-cli ./apps/felicia-cli/cmd/felicia
```

Ingest a trip folder directly into your unified workspace (`~/.felicia` or `$FELICIA_WORKSPACE`):

```sh
felicia-cli journey ingest --dir /path/to/trip-folder
```

A trip folder contains `route.gpx` (or `timeline.json`), a `photos/` folder, and an optional `photos.jsonl` sidecar. The CLI parses GPS tracks, clusters dwell times into candidate stops, installs media into the content-addressed blob store, and stages draft records directly into SQLite with authorship protection.

### 2. Visually curate in Felicia Desktop Studio

Launch Desktop Studio to visually author the ingested journey:

```sh
make desktop
```

In Desktop Studio, you can:

- **Inspect routes**: Explore GPS tracks and elevation on the interactive MapLibre map.
- **Curate stop candidates**: Review detected dwell-time clusters and mark them as *kept*, *merged*, or *ignored*.
- **Write mementos**: Craft personal travel essays, configure dates and locations, and assign curated photos.
- **Preview & Publish**: Inspect the compiled static site with reader fonts bundled directly in-app.

*(For headless or remote browser setups, `make admin` provides the web authoring interface.)*

### 3. Build and publish

Compile the public static publication from published records:

```sh
make site-build
# or using the CLI directly:
felicia-cli static compile --out dist
```

Follow [docs/publish.md](docs/publish.md) for the full publication pipeline, CI workflows, and privacy boundaries.

## Development and quality gates

```sh
make help        # view all available targets
make local-check # fast iterative check runner across changed groups
make check       # pre-commit gate (formatters, linters, Go vet, Python tests)
make validate    # pre-PR gate (full verification across Go, TypeScript, and Python)
```

The native desktop build targets macOS 26.0 or newer. See [the local-check workflow](docs/development/local-checks.md) for incremental validation.

## Status and documentation

Felicia is in active implementation. Detailed documentation:

- **[Agent & Desktop Collaborative Workflow](docs/research/agent-and-desktop-workflow.md)**: Architecture of the asymmetric intake and authoring loop.
- **[Trip Folder Contract](docs/contracts/trip-folder-contract.md)**: Normative specification for trip folder layout, sidecars, and CLI options.
- **[User Journey](docs/roadmap/user-journey.md)** & **[Roadmap](docs/roadmap.md)**: Current development milestones and remaining scope.
- **[Architecture Decisions (ADRs)](docs/adr/)**: Architectural records and boundary definitions.

## Acknowledgements

[liuaaron.com](https://liuaaron.com/) inspired the project. Felicia can connect to [Dawarich](https://github.com/Freika/dawarich) and [Immich](https://github.com/immich-app/immich); map rendering uses [MapLibre GL JS](https://github.com/maplibre/maplibre-gl-js) and [OpenStreetMap](https://www.openstreetmap.org/).

## License

[GNU AGPL-3.0](LICENSE).
