# Research — source connectors (Dawarich, Immich, others)

> 2026-06-19. Question raised against the MVP: rather than hand-write a Dawarich and an Immich
> client, can a user *configure* their own platforms and have felicia assemble a `memento` from
> them — Dawarich-style, leaning on Immich's OpenAPI? Yes, in spirit — but split into three so
> the "generic config-driven query → memento" trap is avoided. Outcome ADR:
> `felicia:decision:source-connectors`. Research-stage — architecture vocabulary, not a spec.
> Sits beside [`notion-to-stack.md`](notion-to-stack.md),
> [`mementos-not-tickets.md`](mementos-not-tickets.md), [`saas-dataflow.md`](saas-dataflow.md).

## The proposal, restated

Treat felicia like Dawarich treats location apps: let the user point it at *their* services
(Dawarich, Immich, and others later), and assemble mementos from whatever those services
return — using Immich's published OpenAPI so we don't hand-write the client.

The instinct is right and it sharpens what the design already says (*"every external source is
an interface impl"*, AGENTS.md). But "config-driven generic query → memento" hides one good
idea inside two costly ones. Split them.

## Borrow the *actual* Dawarich strategy: normalize at the edge

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
`visits`/`places` are exactly the *place* concept felicia was missing
(`felicia:decision:place-as-derived-visit`, data-model §Places). So the `TrackSource` role yields
more than points:

```
TrackSource {
  Track(from, to)  -> []TrackPoint    // the polyline → journeys.gps_route
  Visits(from, to) -> []Visit         // stays: {coord, label, arrive, depart} → derived places
}
```

- **Consume Dawarich's visits** as the derived-place layer instead of re-clustering; a memento
  snaps to the nearest *visit*, not the nearest track vertex. For a plain GPX import (no Dawarich)
  a dwell-time clustering fallback produces the same `Visit` shape **at the edge** — the core
  stays generic over the normalized shape, exactly per the strategy above.
- **Google Maps Timeline is not a first-class connector — it enters through Dawarich.** Google's
  export (`placeVisit`/`activitySegment` Takeout, or the on-device `Timeline.json`
  `semanticSegments`) is a shifting target, and post-2024 Google keeps only ~90 days on-device.
  Dawarich already imports these formats; point a friend's export at Dawarich and felicia reads one
  stable API. We do **not** hand-write a Google parser.
- **Dawarich + Immich are foundational**, not "sources among many" — a pre-history decision. The
  rule-of-three extensibility (a new `TrackSource`/`PhotoSource` impl) still stands for the
  *unusual* user, but the assumed path is **Dawarich (track + visits) ⋈ Immich (photos)** joined on
  timestamp.

## OpenAPI: yes — for client *generation*, not runtime config

The right use of Immich's spec is **build-time codegen** (`oapi-codegen` over its
`openapi.json`) → a typed Go client committed under `internal/immich/`, tracking their API for
free. Dawarich's API is simpler: generate it too if it ships a spec, hand-write otherwise.

The distinction that matters:

| | Verdict |
| --- | --- |
| OpenAPI → generated typed client at **build time** | ✅ do this |
| OpenAPI → a **runtime engine** where users author field-mapping queries in config | ❌ DSL trap |

## Config drives *instances*, not *logic*

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

"Some kind of query to assemble a memento" sounds clean, but the assembly *is* the product,
and none of it is expressible as a generic OpenAPI field mapping:

```
pull track  ─┐
pull assets ─┴─▶ timestamp-join ─▶ cluster (time+space) ─▶ pick stub
                                                          └▶ vision-LLM pre-fill kind/vendor/price
                                                          └▶ write FIELD-SCOPED (never clobber authored — design §5)
```

A generic engine for that is a mini-ETL DSL with **one** user and **two** sources today —
exactly the speculative abstraction the project rules forbid.

- **Now:** the assembly is a *fixed pipeline parameterized by config*
  (`waypoints import --trip japan-2026 --from … --to …`). Each connector hardcodes its own
  mapping into the normalized shape.
- **Later (rule of three):** if a *third* source appears and the mappings genuinely rhyme,
  extract the commonality then — not before.

## Net shape

**Two typed roles · generated OpenAPI clients · config for instances · one fixed assembly
pipeline.** Not a generic config-query engine. This is the auto-ingest (**A**) half of the
A+E model; the manual-authoring (**E**) half — including the transit
[ticket creator](transit-tickets.md) — lands mementos through the *same* creation seam.

The MVP's hardcoded `mementosData` / `routeCoordinates` is exactly what this pipeline emits as
JSON, so the output contract is already proven on screen.

## Open

- Codegen toolchain: `oapi-codegen` config + where the generated client is vendored / how it's
  refreshed when Immich bumps its spec.
- Dawarich API: does it publish an OpenAPI spec, or hand-write the thin `TrackSource`?
- Clustering thresholds (time + distance) — config defaults vs. per-trip overrides.
- Whether `notion` ([notion-to-stack](notion-to-stack.md)) is a third role or folds into
  `PhotoSource` + a metadata source — a near-term test of the rule-of-three line above.
