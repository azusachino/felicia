# felicia

Felicia（フェリシア）— 琉璃雏菊（蓝费莉菊）

A map-based travel journal — a personal site where each journey is drawn on a dark world map
as an orange route line, with collected **ticket stubs** as the stories along the way. Click
a ticket and it animates open into an essay and a gallery of photos.

Modeled on [liuaaron.com](https://liuaaron.com/) ("Aaron's Waypoints").

## Design

See [`docs/design.md`](docs/design.md) for the current design (architecture, data model,
ingestion loop) and [`docs/importer-spec.md`](docs/importer-spec.md) for the `waypoints`
importer spec. Exploration notes: [`docs/research/`](docs/research/).

**Stack:** Go API + Postgres/PostGIS, Vite + Mapbox SPA, S3-compatible object storage (R2),
self-hosted on a Raspberry Pi behind a Cloudflare Tunnel. Ingestion pulls from self-hosted
Immich (photos) + Dawarich (GPS track).

Status: **design/spec phase** — implementation follows a design → spec → TDD flow.
