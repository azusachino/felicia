# felicia

Felicia（フェリシア）— 琉璃雏菊（蓝费莉菊）

A map-based travel journal — a personal app where each journey is drawn on a dark world map
as an orange route line, with collected **memento stubs** as the stories along the way. Click
a memento and it animates open into an essay and a gallery of photos.

Modeled on [liuaaron.com](https://liuaaron.com/) ("Aaron's Waypoints").

## Direction

See [`docs/direction.md`](docs/direction.md) for the current north star. Exploration notes:
[`docs/research/`](docs/research/). Earlier design/spec drafts are parked in
[`docs/archive/`](docs/archive/).

**MVP stack:** Svelte + TypeScript + Tailwind web app, with MapLibre GL for the dark map and
orange route. Backend/data leanings remain Go + Postgres/PostGIS + S3-compatible object
storage (R2) when the app graduates past the spike.

**Languages:** Japanese-first, with English and Chinese also supported.

Status: **research stage** — implementation follows research → spec → TDD once the moat
spike is settled.
