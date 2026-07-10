# felicia web

Svelte + TypeScript + Tailwind + MapLibre GL decision demo for the felicia front-of-house.

The demo is fixture-only: no backend, no auth, and no live source connectors. It exists to
review the product shape before the roadmap hardens. Two front-door concepts share the same
fixture data (`src/data.ts`) and theme/language state:

- **v1 (default)** — the liuaaron-aligned **map reader**: journey index rail → dark route map
  hero → paper detail. **The map is the front door** (`felicia:decision:map-first-landing`):
  all journeys dim, the selected one bright, mementos as numbered stubs along the route.
  Reaches the collection via **Collection →**.
- **v2 (`#collection`)** — _memento-first._ A detailed memento "page" (paper stub + facts +
  essay + gallery) with a shelf **carousel** across all journeys. A **depth view** — the seed
  of a future souvenir-shelf landing (per PM feedback 2026-07-07), not the door.

The first priority is the artifact moat. If the memento + essay/gallery interaction does not
feel compelling, ingestion and authoring automation should not be first on the roadmap.

## Commands

```sh
bun install
bun run dev
bun run build
bun run check
```
