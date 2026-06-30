# felicia web

Svelte + TypeScript + Tailwind + MapLibre GL decision demo for the felicia front-of-house.

The demo is fixture-only: no backend, no auth, and no live source connectors. It exists to
review the product shape before the roadmap hardens:

- **Artifact** — public map, route, designed memento stub, essay, and gallery.
- **Review Queue** — the author confirms, merges, or hides proposed Immich/Dawarich anchors.
- **Transit Creator** — manual edge-anchored transit legs can draw part of the journey route.

The first priority is the artifact moat. If the route + memento + essay/gallery interaction
does not feel compelling, ingestion and authoring automation should not be first on the roadmap.

## Commands

```sh
bun install
bun run dev
bun run build
bun run check
```
