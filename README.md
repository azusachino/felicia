<div align="center">

# 🌼 felicia

**フェリシア** — 琉璃雏菊（蓝费莉菊）

_A map-based travel journal. The map is the index; the mementos are the stories._

[![status](https://img.shields.io/badge/status-implementation%20stage-e8a33d)](docs/roadmap.md)
[![backend](https://img.shields.io/badge/backend-Go%20%C2%B7%20SQLite-00add8)](docs/development/layout.md)
[![web](https://img.shields.io/badge/web-Svelte%205%20%C2%B7%20Vite-ff3e00)](packages/felicia-reader/)
[![license](https://img.shields.io/badge/license-AGPL--3.0-3da639)](LICENSE)

</div>

---

## What it is

Felicia is a personal travel journal. Journeys appear on a map; visits give places to memories; mementos open into essays and photo galleries. Import creates private, reviewable data. You author and explicitly publish what belongs on the public site.

- **Local-first:** SQLite, originals, drafts, and authoring data stay on your machine.
- **Authorship-safe:** re-import updates source-owned fields without overwriting authored fields.
- **Private by default:** only published content and safe media derivatives enter a static site.

## Current shape

```text
Trip folder / GPX / Google Timeline ─┐
Photos / sidecar (photos.jsonl) ─────┴─> felicia-cli journey ingest
                                             -> unified workspace (~/.felicia)
                                             -> visual review & authoring in Desktop / Admin
                                             -> publish -> static site
```

Go modules under `apps/` own the domain, runtime, providers, server, CLI, desktop studio, and publication compiler. `packages/felicia-model` holds frontend contracts; `packages/felicia-reader` is the Atlas reader. The two web hosts are `apps/felicia-admin` and `apps/felicia-public-site`. SQLite is the only persistence implementation for v1.

## Quick start

Install the checked-in toolchain with `mise install`.

### 1. Trip intake with `felicia-cli`

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

A trip folder contains `route.gpx` (or `timeline.json`), a `photos/` folder, and an optional `photos.jsonl` sidecar. The CLI stages candidate stops and installs photos into the content-addressed blob store with full authorship protection (subsequent re-ingests never overwrite human-curated titles or essays).

### 2. Studio authoring & verification

Use the Make targets for authoring studios and quality gates:

```sh
make help
make desktop     # native desktop studio (macOS 26+ app)
make admin       # web authoring stack; see network-binding note below
make web-dev     # public reader
make local-check # reuse unchanged local check groups while iterating
make check       # uncached pre-commit gate
make validate    # uncached pre-PR gate
```

`make admin` binds the authoring stack to `0.0.0.0` for tailnet access; the admin API has no authentication. Use only on a trusted host/network, or set `FELICIA_HOST=127.0.0.1` for host-only access.

For small changes, select a check group with
`make local-check ARGS='--groups scripts'` (or `admin`, `public`, `go`, `docs`).
Use `ARGS='--force'` to bypass local results. See
[the local-check workflow](docs/development/local-checks.md); full acceptance gates
never consume this cache.

## Build and publish

The native desktop build targets macOS 26.0 or newer. The Go task runner derives
compilation/link deployment flags from the bundle's minimum system version;
SDK version and deployment minimum are distinct. Native visual acceptance remains
separate from a successful build.

The admin studio runs locally. The public site is compiled from published records; deploying that static output is separate from authoring:

```sh
make admin
make site-build
```

Follow [docs/publish.md](docs/publish.md) for the full local and CI publication flow and privacy boundary.

## Status and documentation

Felicia is in the implementation stage. The selected end-to-end flow and current stage status are in [the user journey](docs/roadmap/user-journey.md); milestones and remaining work are in [the roadmap](docs/roadmap.md). Read [the project instructions](AGENTS.md) before contributing. Architecture records live in [`docs/adr/`](docs/adr/), and the broader design/research trail in [`docs/research/`](docs/research/).

The desktop follow-up provides isolated sample/empty temporary workspaces,
journey and memento authoring, shared library controls and an in-app last-built
preview with reader fonts bundled from pinned Fontsource dependencies. Ordinary
desktop image upload shares web validation/storage; temporary workspaces add
only generated photos and manage output automatically. Internal dirty Return
uses a shared confirmation dialog. The owner reviewed and approved
[PR #160](https://github.com/azusachino/felicia/pull/160), with green backend,
frontend and desktop-composition CI. The rebuilt isolated Sample is basically
working according to owner feedback. Standalone Close has bounded native
acceptance; broader native S1 acceptance, normal file-picker testing and S2
reviewed imports remain open under [#161](https://github.com/azusachino/felicia/issues/161).
See the [desktop execution plan](docs/roadmap/desktop-studio-redesign.md) for the
verified slices and explicit capability limits.

## Acknowledgements

[liuaaron.com](https://liuaaron.com/) inspired the project. Felicia can connect to [Dawarich](https://github.com/Freika/dawarich) and [Immich](https://github.com/immich-app/immich); map rendering uses [MapLibre GL JS](https://github.com/maplibre/maplibre-gl-js) and [OpenStreetMap](https://www.openstreetmap.org/).

## License

[GNU AGPL-3.0](LICENSE).
