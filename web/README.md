# felicia web

Svelte + TypeScript + Tailwind + MapLibre GL decision demo for the felicia front-of-house.

The demo is fixture-only: no backend, no auth, and no live source connectors. It exists to
review the product shape before the roadmap hardens. Two front-door concepts share the same
fixture data (`src/data.ts`) and theme/language state:

- **v2 (default)** — *memento-first.* A detailed memento "page" is the centre (paper stub +
  facts + essay + gallery); a preview **carousel** of mementos across all journeys is the
  index. Titled *"…'s True Memories."* Reaches v1 via **Map view**.
- **v1 (`#map`)** — the liuaaron-aligned **map reader**: journey index rail → dark route map
  hero → paper detail. The map is the index. Reached from v2 as the "more" view.

The first priority is the artifact moat. If the memento + essay/gallery interaction does not
feel compelling, ingestion and authoring automation should not be first on the roadmap.

## Commands

```sh
bun install
bun run dev
bun run build
bun run check
```
