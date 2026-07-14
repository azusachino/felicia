# Research — source connectors (Dawarich, Immich, others)

> 2026-06-19. Question raised against the MVP: rather than hand-write a Dawarich and an Immich
> client, can a user _configure_ their own platforms and have felicia assemble a `memento` from
> them — Dawarich-style, leaning on Immich's OpenAPI? Yes, in spirit — but split into three so
> the "generic config-driven query → memento" trap is avoided. Outcome ADR:
> `felicia:decision:source-connectors`. Research-stage — architecture vocabulary, not a spec.
> Sits beside [`notion-to-stack.md`](notion-to-stack.md),
> [`mementos-not-tickets.md`](mementos-not-tickets.md), [`saas-dataflow.md`](saas-dataflow.md).

## The proposal, restated

Treat felicia like Dawarich treats location apps: let the user point it at _their_ services
(Dawarich, Immich, and others later), and assemble mementos from whatever those services
return — using Immich's published OpenAPI so we don't hand-write the client.

The instinct is right and it sharpens what the design already says (_"every external source is
an interface impl"_, AGENTS.md). But "config-driven generic query → memento" hides one good
idea inside two costly ones. Split them.

## Borrow the _actual_ Dawarich strategy: normalize at the edge

Dawarich's trick isn't "be generic." It accepts OwnTracks / Overland / GPSLogger / etc., but
translates each into **one** internal point shape; everything downstream is generic over that
shape, not over the sources. Copy exactly this — normalize at the edge, keep the core narrow:

```
internal/domain (pure, no I/O) — the normalized shapes everything joins on
  TrackPoint  { At; Lat; Lon }
  PhotoAsset  { ID; At; Lat?; Lon?; Kind }   // Lat/Lon nil → fill from track by timestamp
```

Two small **typed roles** at the seam — not N generic sources:

```
TrackSource  { Track(from, to) -> []TrackPoint }     // Dawarich impls this
PhotoSource  { Assets(Query)   -> []PhotoAsset  }     // Immich impls this
```

"Configure other platforms" = register a new `kind` that implements one of these two
interfaces. A power user with a different photo library writes a ~100-line `PhotoSource`; the
assembly logic never moves.

## Update (2026-07-09): Dawarich has a semantic layer — take it, don't just drain points

The shapes above (`TrackPoint {At,Lat,Lon}`) treat Dawarich as a raw-point hose. But Dawarich's
data model has grown the full pipeline **`points → tracks → visits @ places → trips`** — and its
`visits`/`places` are exactly the _place_ concept felicia was missing
(`felicia:decision:place-as-derived-visit`, data-model §Places). So the `TrackSource` role yields
more than points:

```
TrackSource {
  Track(from, to)  -> []TrackPoint    // the polyline → journeys.gps_route
  Visits(from, to) -> []Visit         // stays: {coord, label, arrive, depart} → derived places
}
```

- **Consume Dawarich's visits** as the derived-place layer instead of re-clustering; a memento
  snaps to the nearest _visit_, not the nearest track vertex. For a plain GPX import (no Dawarich)
  a dwell-time clustering fallback produces the same `Visit` shape **at the edge** — the core
  stays generic over the normalized shape, exactly per the strategy above.
- **Google Maps Timeline is not a first-class connector — it enters through Dawarich.** Google's
  export (`placeVisit`/`activitySegment` Takeout, or the on-device `Timeline.json`
  `semanticSegments`) is a shifting target, and post-2024 Google keeps only ~90 days on-device.
  Dawarich already imports these formats; point a friend's export at Dawarich and felicia reads one
  stable API. We do **not** hand-write a Google parser.
- **Dawarich + Immich are foundational**, not "sources among many" — a pre-history decision. The
  rule-of-three extensibility (a new `TrackSource`/`PhotoSource` impl) still stands for the
  _unusual_ user, but the assumed path is **Dawarich (track + visits) ⋈ Immich (photos)** joined on
  timestamp.

## OpenAPI: yes — for client _generation_, not runtime config

The right use of Immich's spec is **build-time codegen** (`oapi-codegen` over its
`openapi.json`) → a typed Go client committed under `internal/immich/`, tracking their API for
free. Dawarich's API is simpler: generate it too if it ships a spec, hand-write otherwise.

The distinction that matters:

|                                                                                   | Verdict     |
| --------------------------------------------------------------------------------- | ----------- |
| OpenAPI → generated typed client at **build time**                                | ✅ do this  |
| OpenAPI → a **runtime engine** where users author field-mapping queries in config | ❌ DSL trap |

## Config drives _instances_, not _logic_

```toml
[sources.track]
kind     = "dawarich"
base_url = "https://dawarich.lan"   # WAYPOINTS_DAWARICH_API_KEY from env

[sources.photo]
kind     = "immich"
base_url = "https://immich.lan"     # WAYPOINTS_IMMICH_API_KEY from env
album    = "Japan 2026"
```

A user configures their own instances (URLs, keys, album/date filters). Secrets stay in the
env, never the file (config rule). Extensibility is a new interface impl, not a new query
dialect.

## Why the generic version bites: assembly is irreducibly domain logic

"Some kind of query to assemble a memento" sounds clean, but the assembly _is_ the product,
and none of it is expressible as a generic OpenAPI field mapping:

```
pull track  ─┐
pull assets ─┴─▶ timestamp-join ─▶ cluster (time+space) ─▶ pick stub
                                                          └▶ vision-LLM pre-fill kind/vendor/price
                                                          └▶ write FIELD-SCOPED (never clobber authored — design §5)
```

A generic engine for that is a mini-ETL DSL with **one** user and **two** sources today —
exactly the speculative abstraction the project rules forbid.

- **Now:** the assembly is a _fixed pipeline parameterized by config_
  (`waypoints import --trip japan-2026 --from … --to …`). Each connector hardcodes its own
  mapping into the normalized shape.
- **Later (rule of three):** if a _third_ source appears and the mappings genuinely rhyme,
  extract the commonality then — not before.

## Net shape

**Two typed roles · generated OpenAPI clients · config for instances · one fixed assembly
pipeline.** Not a generic config-query engine. This is the auto-ingest (**A**) half of the
A+E model; the manual-authoring (**E**) half — including the transit
[ticket creator](transit-tickets.md) — lands mementos through the _same_ creation seam.

The MVP's hardcoded `mementosData` / `routeCoordinates` is exactly what this pipeline emits as
JSON, so the output contract is already proven on screen.

## Sync operations — pinned API bindings (2026-07-10)

Learned directly from `Freika/dawarich` (routes + controllers) and the `immich-app/immich`
OpenAPI. **Base case = query routes/visits (Dawarich) ⋈ photos (Immich, direct) within a time
range**, joined on timestamp. Raw points are a fallback. All list endpoints paginate — loop until
exhausted.

### Auth & config

- **Dawarich** — `base_url` + `api_key` (`authenticate_active_api_user!`). Pass `?api_key=<key>`
  (query) or `Authorization: Bearer <key>`. Env `WAYPOINTS_DAWARICH_API_KEY`.
- **Immich** — `base_url` + `x-api-key: <key>` header. Env `WAYPOINTS_IMMICH_API_KEY`.

### TrackSource (Dawarich) — the base case

**Routes** — `GET /api/v1/tracks?start_at&end_at&page&per_page=500`

- Time filter is an **overlap**: `end_at >= :start_at AND start_at <= :end_at`.
- Response: GeoJSON **FeatureCollection**; each Feature geometry is a **LineString** (from
  `original_path`); properties `{id, start_at, end_at, distance(m), avg_speed, duration,
dominant_mode, mode_timeline}`.
- Pagination headers: `X-Current-Page`, `X-Total-Pages`, `X-Total-Count` — loop `1..X-Total-Pages`.
- Map → `[]Route{ Line orb.LineString, From/To time.Time, DistanceM, Mode }`.
  `journeys.gps_route = ST_Collect(route lines)` (MultiLineString), RDP-simplified inline
  (`simplify.DouglasPeucker(0.0001)`, ADR 0009).

**Visits** — `GET /api/v1/visits?start_at&end_at&page&per_page=500` (via `Visits::Finder`)

- Response: **plain JSON array** of `{id, started_at, ended_at, duration, name, status,
confidence, confidence_band, place:{id, latitude, longitude}}`; same pagination headers.
- ⚠️ `place` carries **coords + id only — no name/city**. The visit's top-level **`name` is the
  label**; richer place naming is authored (**E**) later, not ingested.
- `status` ∈ suggested|confirmed|declined → take confirmed+suggested, skip declined.
- Map → `[]Visit{ Coord orb.Point{place.longitude, place.latitude}, Label name, Arrive
started_at, Depart ended_at, Confidence, SourceRef "dawarich:visit:{id}" }`. These are the
  derived places (`place-as-derived-visit`, ADR 0005); a memento snaps to the nearest **visit**,
  not the nearest track vertex.

**Points (fallback only)** — `GET /api/v1/points?start_at&end_at&slim=true&per_page=1000&order=asc`
→ plain array `{id, latitude, longitude, timestamp(epoch)}`. Used for a bare GPX import with no
Dawarich track layer; a dwell-time cluster then synthesizes the same `Visit` shape at the edge.

### PhotoSource (Immich, direct)

felicia integrates Immich **directly** (`internal/immich`). Dawarich's own Immich integration is
a **how-to reference**, not a layer to route through — it shows the exact call to make. Direct
gives full control: _all_ photos (not just the geolocated subset Dawarich's `/photos` surfaces
for the map), richer EXIF, and no dependency on a user's Dawarich↔Immich config.

**Assets** — `POST /search/metadata` (`x-api-key`), body (verified against the official OpenAPI)
`{takenAfter, takenBefore, type:"IMAGE", visibility:"timeline", withExif:true, order:"asc",
size:1000, page:1}`

- Enums: `type` ∈ IMAGE|VIDEO|AUDIO|OTHER; `visibility` ∈ archive|timeline|hidden|locked (use
  `timeline` to skip archived/hidden); `order` ∈ asc|desc. Optional filters: `albumIds`,
  `isNotInAlbum`, `city`, `country`, `isFavorite`.
- Response: `SearchResponseDto.assets{ items:[AssetResponseDto], total, count, nextPage }` —
  `nextPage` is a **cursor string**; re-issue with the next `page` until it is null.
- `AssetResponseDto{ id, checksum, type, originalFileName, fileCreatedAt, localDateTime,
visibility, exifInfo{ latitude(number), longitude(number), dateTimeOriginal, city, country,
make, model, timeZone } }`.
- Map → `[]PhotoAsset{ ID, At exifInfo.dateTimeOriginal (fallback localDateTime→fileCreatedAt),
Lat/Lon exifInfo (nil when the photo has no GPS), Checksum, SourceRef "immich:asset:{id}" }`.
  Media: thumb `GET /assets/{id}/thumbnail?size=preview` (size ∈ original|fullsize|preview|
  thumbnail), original `GET /assets/{id}/original`.
- No-GPS photos → `Lat/Lon` nil → fill from the Dawarich track by timestamp, or snap on drop.

_Reference — how Dawarich calls Immich:_ `GET /api/v1/photos?start_date&end_date` →
`Photos::Search` proxies Immich `POST /search/metadata` (`takenAfter`/`takenBefore`, `withExif`),
returns geolocated photos tagged with a `source` field, and Redis-caches both the search response
and per-photo thumbnails (`/api/v1/photos/:id/thumbnail.jpg?source=immich`). We copy the _call
and caching shape_, not the proxy hop — and confirm auth needs Immich key scopes `asset.read` +
`asset.view`.

### Sync ops (importer)

- `SyncRoute(journey)` — `Dawarich.FetchRoutes(range)` → RDP → MultiLineString → **field-scoped**
  `gps_route` upsert (never clobbers authored — design §5).
- `SyncVisits(journey)` — `Dawarich.FetchVisits(range)` → derived places (memento-anchor candidates).
- `SyncPhotoTray(journey)` — `Immich.FetchAssets(range)` (direct) → tray candidates,
  timestamp-joined to the Dawarich visits/route.

### Testing (no network)

Saved fixtures — a tracks `FeatureCollection`, a visits array, an Immich search response — drive
pure unit tests over the mappers; the HTTP call is behind an injected `Doer` (mocked via
`httptest`).

## Open

- **Client generation** — Immich publishes OpenAPI, but felicia touches only `POST
/search/metadata` + two media GETs; a thin hand-written `internal/immich` client beats
  vendoring a large generated one. Revisit `oapi-codegen` if the surface grows. Dawarich (3 REST
  endpoints) is hand-written in `internal/dawarich` — no codegen. _(resolves the prior two Open
  bullets: Dawarich has no OpenAPI but a small stable REST surface.)_
- Timestamp format on Dawarich `start_at`/`end_at` query params — accepts ISO8601; points use
  epoch `timestamp`. Send RFC3339, confirm against a live instance during TDD.
- Clustering thresholds (time + distance) for the GPX-fallback `Visit` synthesis — config
  defaults vs. per-trip overrides.
- Whether `notion` ([notion-to-stack](notion-to-stack.md)) is a third role or folds into
  `PhotoSource` + a metadata source — a near-term test of the rule-of-three line above.
