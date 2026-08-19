# felicia public site

Svelte + TypeScript + Tailwind + MapLibre GL decision demo for the felicia front-of-house.

The demo is fixture-only: no backend, no auth, and no live source connectors. It exists to
review the product shape before the roadmap hardens. The four named reader languages share the
same fixture data (`src/data.ts`) and theme/language state:

- **Cartography (default)** — the liuaaron-aligned **map reader**: journey index rail → dark route map
  hero → paper detail. **The map is the front door** (`felicia:decision:map-first-landing`):
  all journeys dim, the selected one bright, mementos as numbered stubs along the route.
  Reaches the collection via **Collection →**.
- **Cabinet (`#cabinet`)** — _memento-first._ A detailed memento "page" (paper stub + facts +
  essay + gallery) with a shelf **carousel** across all journeys. A **depth view** — the seed
  of a future souvenir-shelf landing (per PM feedback 2026-07-07), not the door.
- **Techo (`#techo`)** — _paper journal._ A two-page spread for route and memento context.
- **Atlas (`#atlas`)** — _editorial atlas._ A scroll-driven route and story composition.

The first priority is the artifact moat. If the memento + essay/gallery interaction does not
feel compelling, ingestion and authoring automation should not be first on the roadmap.

## Commands

```sh
bun install
bun run dev
bun run build
bun run check
```
