# Spec-gap register — every vague point, resolved

> Audit of 2026-06-12: everywhere two reasonable implementers would build different
> things. Each item: the gap, why it causes rework, the concrete resolution.
> **LOCKED** = decided (user-confirmed or contract-forced). **PROPOSED** = accepted
> defaults, amendable only via a new ADR.
>
> **Status: FROZEN 2026-06-12** — all PROPOSED items accepted at freeze (user call,
> entering TDD). This register is a **normative annex** to `importer-spec.md` and wins
> where they disagree; the fold-in checklist below is now documentation debt, not a
> decision gate. Test files cite items by ID (B2, B4, …).

## A — Source contracts

### A1. Immich ticket marker — LOCKED

"⭐/tag" was ambiguous. **Tag `ticket`** (Immich tags API), not favorite — favorites
are global and would pollute real usage. Curate ritual: album per trip + tag `ticket`
on stub shots. `PhotoSource.Album()` must surface tags per asset.

### A2. Journey identity & slug — PROPOSED

- Journey `source_ref` = `immich-album:<album-uuid>` (survives album rename).
- `slug` = `<yyyy>-<mm>-<slugify(album-name)>` computed **once at first import**,
  then never re-derived (it's in URLs). Match order on upsert: `source_ref`, then
  `slug` (the YAML path has no album id).
- YAML-sourced journeys: `slug` is required in the file and is the identity.

### A3. Trip dates & route pull window — PROPOSED

First import of an album: `date_start/date_end` = min/max asset capture dates
(local). Route pull = `[date_start − pad, date_end + pad]`, `pad` config
`geo.range_pad_hours = 3`. `--since/--until` override the window; they never change
stored journey dates.

### A4. Timezone resolution — PROPOSED

Storage is UTC everywhere (`timestamptz`). Per asset, resolve the EXIF wall-clock to
UTC by first match:

1. Timezone/offset present in Immich `exifInfo` → use it.
2. Asset has GPS → tz lookup at that coordinate (offline index, e.g. `ringsaturn/tzf`).
3. Else → journey default tz = tz at the route's **first point**.

OCR-extracted datetimes (printed local time): resolve with tz at the route-snap
location, else journey default tz. **Fixture must include a cross-timezone trip**
(e.g. flight day KR→JP).

### A5. Image bytes come from Immich previews — PROPOSED

iPhones shoot **HEIC**; decoding it in Go means cgo (libheif/libvips) on a Pi.
Sidestep: `PhotoSource.Download()` fetches **Immich's preview JPEG** (~1440px+,
already transcoded) — not the original. The manual `fsphotos` provider accepts
JPEG/PNG only in v1. Originals stay in Immich, never needed by felicia.

## B — Derivation rules (pure functions, all test targets)

### B1. Field provenance is THREE classes, not two — LOCKED (contract-forced)

The two-class model (ingested/authored) is broken: the spec itself says waypoint
`Name` is "ingested, **editable**" — under two classes, re-import clobbers your edit.
And OCR fields are explicitly "pre-filled **for confirmation**". Classes:

| Class           | Importer                               | Admin UI  | Examples                                                                                      |
| --------------- | -------------------------------------- | --------- | --------------------------------------------------------------------------------------------- |
| **INGESTED**    | always writes                          | read-only | route, stub_image, source_ref, taken_at, location                                             |
| **OVERRIDABLE** | writes **until human edits**           | editable  | ticket type/vendor/price/occurred_at, waypoint name, ticket seq, journey country/region/dates |
| **AUTHORED**    | never writes (absent from Patch types) | owns      | title, essay, summary, animation, captions, photo selection/order                             |

Mechanism: every row carries `authored_fields text[] NOT NULL DEFAULT '{}'`. The
admin API appends the column name when a human edits an OVERRIDABLE field. The
repository filters incoming patches: an OVERRIDABLE field is written only if its name
is **not** in `authored_fields`. AUTHORED fields stay structurally absent from Patch
types. Canonical test extends: import → admin corrects OCR'd `vendor` → re-import
with changed source data → `vendor` keeps the correction, untouched fields refresh.

### B2. Route geometry — LOCKED

`geometry(MultiLineString, 4326)`. Split the raw track into segments where
time gap > `geo.gap_split_min` (default **60**) or point-to-point jump >
`geo.gap_split_km` (default **50**, catches flights); Douglas-Peucker each segment.
Honest gaps; matches the reference's per-day look (nights are gaps).

### B3. Raw track is NOT stored — PROPOSED

Dawarich/GPX retain the raw points; our DB stays a rebuildable projection holding
only the derived MultiLineString. Consequence, stated plainly: **route-snap happens
at import time only** — correcting a ticket's `occurred_at` in the admin UI does not
move its pin until the next `import` run re-snaps it (cheap, safe, no-clobber).

### B4. Route-snap algorithm — LOCKED (follows ticket-time-place decision)

`geo.SnapToTrack(track []TimedPoint, t, maxGap) (Point, ok)`: find raw-track points
bracketing `t`; if bracket span ≤ `geo.snap_max_gap_min` (default **30**) → linear
interpolation; else nearest single point within the same limit; else `ok=false` →
fall back to EXIF location → else waypoint-less, location NULL + stderr warning.

### B5. Ticket → waypoint anchoring — PROPOSED

After clustering: anchor to the waypoint whose dwell window
`[arrived_at, departed_at]` contains `occurred_at`; else the nearest waypoint within
`geo.cluster_radius_m` of the ticket location; else `waypoint_id = NULL` (renders on
the route, not in a stop). Pure function over `([]Waypoint, []Ticket)`.

### B6. Photo-trail synthesis (exact) — PROPOSED

Only when **no** track source is configured (never a silent fallback, spec §10):
order geotagged assets by resolved UTC time; drop consecutive points < 50 m apart;
result is one segment per `gap_split` rule (B2 applies afterwards).

### B7. OCR contract — PROPOSED

One vision call per ticket image; structured output, temperature 0. Schema:

```json
{
  "type": "receipt|transit|admission|null",
  "vendor": "string|null",
  "price": { "amount": 5000, "currency": "KRW" },
  "occurred_at": "2026-01-10T12:36:54",
  "confidence": { "type": 0.97, "vendor": 0.92, "price": 0.99, "occurred_at": 0.85 },
  "extra": { "venue_name_local": "성산일출봉", "ticket_no": "A3-2025052500185" }
}
```

Rules: `null` over guessing; `price.amount` in minor units; `currency` only if
printed or symbol-unambiguous; `occurred_at` has **no tz** (A4 resolves it); any
field with confidence < **0.6** → null. `extra` is free-form, stored as `jsonb`,
and feeds the stub **templates** (decision: stub-templates). Tests run on recorded
responses (`testdata/ocr/*.json`), never the API.

## C — Persistence & idempotency

### C1. Upsert conflict targets — PROPOSED

Unique indexes: `journeys(slug)`, `journeys(source_ref)`,
`tickets(journey_id, source_ref)`, `ticket_photos(ticket_id, source_ref)`,
`waypoints(journey_id, seq)`. Postgres `ON CONFLICT ... DO UPDATE` with the B1
authored-fields filter.

### C2. Idempotency, measurably — PROPOSED

"Re-run = zero writes" becomes testable: the repository compares the incoming
ingested/overridable tuple with the current row and **skips no-op UPDATEs** (no
`updated_at` churn); `ObjectStore.Put` is skipped on `Exists(hash-key)`. Test
asserts: second import on identical fixtures → repo write count == 0 and store put
count == 0.

### C3. Orphans — PROPOSED

`orphaned_at timestamptz NULL` on tickets/photos. Set when the source album no
longer contains `source_ref`; cleared if it reappears; surfaced in admin UI. Never
auto-deleted (design §5 stands).

### C4. Seq ownership — PROPOSED

`waypoint.seq` INGESTED (chronological). `ticket.seq` OVERRIDABLE (chronological
default, admin may reorder). `ticket_photo.seq` AUTHORED.

### C5. Derivative spec — PROPOSED

Input: Immich preview JPEG (A5). Output: JPEG quality **82**, longest edge
`image.max_edge_px` (2000), all metadata stripped (encode fresh via `image/jpeg` —
pure Go, no cgo). `ContentHash` = SHA-256 of **derivative bytes**; key
`journeys/<slug>/<kind>/<hash>.jpg`. WebP/AVIF is a later swap behind `ImageRef`.

## D — Privacy & public surface

### D1. Privacy zones — PROPOSED

```toml
[[geo.privacy_zone]]
lat = 0.0
lng = 0.0
radius_m = 500
```

Track points inside any zone are dropped **before** gap-split/simplify (B2). Applies
to photo-trail synthesis too. (EXIF strip already covers images.)

### D2. Public coordinate rounding — PROPOSED

The API/export rounds all public geometry to **4 decimals** (~11 m — what the
reference ships). Full precision stays DB-only. One knob: `api.coord_decimals = 4`.

### D3. Public read API — PROPOSED

- `GET /api/journeys` → `[{slug, title, summary, country, date_start, date_end,
bbox, ticket_count}]`, sorted `date_end` desc (SPA's `Newest ⇄ Oldest` flips client-side).
- `GET /api/journeys/{slug}` → journey + route (GeoJSON MultiLineString, rounded) +
  waypoints + tickets (template fields incl. `extra`, image URLs on
  `storage.public_base_url`) + photos.
- No pagination, no auth (public data only); `ETag` + `Cache-Control` from row
  `updated_at`. Admin API is a later milestone.

### D4. Admin auth — LOCKED

**Cloudflare Access** policy on the admin hostname; the Go API verifies the
`Cf-Access-Jwt-Assertion` JWT (Access public keys) on `/api/admin/*`. No app-level
passwords or sessions. Same story at home and on the road.

## E — Formats

### E1. `trip.yaml` schema (the v1 manual path) — PROPOSED

```yaml
journey:
  slug: 2026-01-jeju # required — identity for YAML-sourced trips (A2)
  title: "Jeju" # AUTHORED (sync seeds it from the album name)
  country: KR
  date_start: 2026-01-09
  date_end: 2026-01-12
route:
  source: gpx # gpx | dawarich | photo-trail
  gpx: track/jeju.gpx
tickets:
  - source_ref: "file:tickets/seongsan.jpg" # or "immich:<asset-uuid>"
    type: admission
    vendor: "성산일출봉"
    price: { amount: 5000, currency: KRW }
    occurred_at: 2026-01-10T12:36:54+09:00 # tz explicit in YAML
    photos:
      - source_ref: "file:photos/crater.jpg"
```

`waypoints validate` checks this against a JSON Schema kept in-repo;
`golden/<slug>.yaml` fixtures use the same schema. Export writes the superset
(authored fields included, marked).

## Fold-in checklist (spec freeze)

- [ ] B1 three-class table + `authored_fields` → importer-spec §7 (replaces two-class)
- [ ] B2 MultiLineString → importer-spec §4/§9 + design ER diagram
- [ ] A4/B4/B5/B6/B7 → importer-spec §9 + §11 test list
- [ ] C1–C5, D1–D2 → importer-spec §3 (config keys) + §8
- [ ] E1 → importer-spec appendix
- [ ] D3/D4 → design §6 + future api-spec.md
