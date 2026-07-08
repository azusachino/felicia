# Research — backend stack & decisions (memento-era)

> 2026-07-08. A re-audit of the backend against the **memento** model (not the frozen
> ticket-era [`archive/design.md`](../archive/design.md) / [`archive/spec-gaps.md`](../archive/spec-gaps.md)),
> the v2 demo shape ([`web/src/data.ts`](../../web/src/data.ts)), and the
> presentation-agnostic-contract direction. Two halves: **(1)** popular-OSS library
> selections for the Go backend, and **(2)** rewritten recommended decisions + todos.
> Research-stage **leanings**, not locks — they graduate to a spec when ratified.
> Related ADRs: `felicia:decision:{map-first-landing, presentation-agnostic-contract,
> memento-not-ticket, memento-template-registry, transit-ticket-creator,
> place-as-derived-visit, source-connectors, jp-first-i18n}`.
>
> **2026-07-09 revision.** D2/D4 and the todos below are reconciled with Dawarich's
> **semantic layer** (`points → tracks → visits @ places → trips`), per
> [`data-model.md §Places`](data-model.md) and [`source-connectors.md`](source-connectors.md)
> (`felicia:decision:place-as-derived-visit`); Dawarich is consumed for **tracks + visits**,
> not drained as a raw-point hose. New **D9** records the declarative
> memento-template registry (`felicia:decision:memento-template-registry`).

## Framing

The backend is the layer that earns rigor: the frontend is deliberately multi-version
(v1 map / v2 memento / future shelf-drawer), all rendering the **same** canonical data
(`felicia:decision:presentation-agnostic-contract`). So the schema and API contract must
be presentation-agnostic; nothing view-specific in the DB, only **semantic** orderings
(chronological / along-route / by-kind).

Deploy target is a **production-ready** application — containers on a real host (VPS /
cloud), Postgres+PostGIS (managed or self-run), R2 for objects. (The archived "Raspberry Pi
+ Cloudflare Tunnel" was the earlier self-hosted-sovereign framing; it no longer constrains
the design.)

Guiding filters for every library pick — chosen on their **production** merits, not a Pi:
**pure-Go where possible** (lean `scratch`/distroless images, trivial multi-arch builds, no
libvips/GEOS system deps in the image, fewer CVEs, faster CI; and for GEOS/libvips cgo also
brings C-heap memory pressure); **stdlib-compatible** (keeps the pure TDD core and swappable
seams); **heavy geo stays in PostGIS SQL**, Go only does I/O + light geometry.

## 1. OSS stack selections

| Concern | Pick | Why (vs. the alternative) |
| --- | --- | --- |
| HTTP router | **chi** | stdlib-compatible — handlers stay `http.HandlerFunc`, testable with `httptest`; middleware chain gives the Cloudflare Access-JWT verify as one `r.Use` on the `/api/admin/*` subrouter. Gin is more popular (~48%) but brings its own `gin.Context` and opinions; we want the "stay close to net/http" camp. Plain `net/http` 1.22+ routing is the zero-dep fallback. |
| Postgres driver | **pgx v5** | The modern default since lib/pq entered maintenance; native `jsonb`, `timestamptz`, geometry/WKB, `COPY`, `LISTEN/NOTIFY`. lib/pq is legacy. |
| Query layer | **sqlc** (on pgx) | Compile-time-safe codegen from raw SQL — invalid SQL fails at build, zero runtime reflection, and **the actual SQL is visible in PRs** (matters for the field-scoped upsert). GORM's reflection/opacity is the wrong trade for a correctness-critical importer; sqlx is a smaller upgrade but still hand-scans. sqlc interleaves with raw pgx for the geometry queries sqlc can't model. |
| Geometry (Go side) | **paulmach/orb** | Pure Go, and it matches our exact needs: `orb/geojson` (MapLibre output), WKB encode/scan (PostGIS I/O), `orb/simplify` (Douglas–Peucker for route B2), coordinate rounding. `twpayne/go-geom` is an equally-valid pure-Go alternative. Avoid `go-geos` (cgo/GEOS: C-heap pressure, panics-on-error, heavier images/builds) — we don't need GEOS predicates in Go because **PostGIS does the heavy geo in SQL** (`ST_*`). |
| Migrations | **goose** | Already the project choice; pure CLI, numbered SQL files, Go migrations when a data migration needs to query. Atlas (declarative, drift detection, CI diff) is a *future* option but heavier than a research-stage repo needs; goose stays. |
| Object storage | **minio-go v7** | One lightweight, idiomatic client that targets **any** S3-compatible endpoint — exactly the "R2 primary, MinIO/B2 swappable by config" requirement — behind our own `ObjectStore` interface. `aws-sdk-go-v2` is Cloudflare's officially-documented R2 path and the fallback, but heavier and AWS-shaped. |
| Image processing | **kovidgoyal/imaging** (maintained fork of `disintegration/imaging`) | Pure Go, no cgo → lean image, simple build. **Re-encoding via the stdlib naturally strips EXIF** (the privacy invariant); `AutoOrientation` applies the orientation tag first so nothing rotates wrong. The fork adds live EXIF read + WebP/jpegli. `bimg`/`govips` (libvips, 4–8× faster) are deferred — cgo isn't worth it at personal scale. HEIC decode is side-stepped by pulling **Immich preview JPEGs** (archive spec A5), so no libheif. |
| EXIF read | **imaging fork's EXIF, else `dsoprea/go-exif/v3`** | We only need to *read* lat/lng/timestamp. Prefer the metadata read already in the imaging fork to avoid a dep; if insufficient, `dsoprea/go-exif/v3` is actively maintained (read+write, HEIC). `rwcarlsen/goexif` works but is effectively frozen. |
| GPX parse | **tkrajina/gpxgo** | The de-facto Go GPX library (port of gpxpy); GPX 1.0/1.1, tracks→segments→points, analysis utils. |
| Timezone lookup | **ringsaturn/tzf** | Offline coord→tz index (archive spec A4) — no network in the pure core. |
| Config | **go-toml/v2 + explicit env overrides** | The config surface is tiny (`waypoints.toml` + a handful of `WAYPOINTS_*` secrets) — decode into a struct with `go-toml/v2`, apply env overrides explicitly. No config framework earns its keep here (keep-it-simple). If a provider framework is ever wanted, **knadh/koanf** (modular: pick only the TOML+env providers) over **Viper** — see note below. |

> **Not OCR yet.** Vision-LLM pre-fill (official `anthropic-sdk-go` + Structured Outputs to
> enforce the archive-B7 schema) is a **deferred enrichment**, not MVP. The near-term path is
> **manual/authored** — the hand-authored transit creator and back-fillable goods — so no
> ticket to OCR. When it lands it's one source *behind* the memento-creation seam; nothing in
> the model changes.

> **Why not Viper?** It's popular and capable, but for our tiny config it's the wrong shape:
> a large transitive dep tree, a **global singleton** (`viper.Get` from anywhere — hidden
> coupling, harder to test the pure core), and **case-insensitive key folding** that has
> surprised people (keys silently collide/lowercase). koanf is modular and singleton-free if
> we ever need a framework; but for `waypoints.toml` a plain struct decode is simpler than
> either.
| Test diffs | **google/go-cmp** | Struct/record diffs for the fixture-driven importer tests; stdlib `testing` otherwise. |

**The load-bearing consequence:** PostGIS owns the geometry math (`ST_Simplify`,
`ST_AsBinary`, `ST_Collect`, distance), so the Go geometry lib is only WKB↔GeoJSON +
Douglas–Peucker-at-import + rounding. That keeps everything pure-Go and cgo-free.

## 2. Recommended decisions (rewritten, memento-era)

Supersedes the ticket-era `TICKET`/two-class assumptions in `archive/`. Ranked by
schema impact; **★** = reshapes the schema (settle first).

### D1 ★ — Memento is one table + `kind_data jsonb`

A single `mementos` table: common columns (`id, journey_id, kind, occurred_at, geom,
vendor, price_amount, price_currency, source_ref, authored_fields, orphaned_at`) plus a
`kind_data jsonb` for kind-specific fields, validated per-kind in `internal/domain`.
Kinds proliferate (transit/goods/stamp/receipt/souvenir…); a uniform table keeps the
API and importer uniform. Rejected: per-kind sidecar tables (join sprawl), wide nullable
typed columns (sparse).

### D2 ★ — Geometry is mixed; the route is derived

`mementos.geom geometry(Geometry, 4326)` — a **Point** for goods/stamp/ticket, a
**LineString** for a transit leg (edge-anchored, per `transit-tickets.md`). A point memento
snaps to the nearest **Dawarich visit** (not merely the nearest track vertex), so its
`place`/coord inherits a stable visit identity (`place-as-derived-visit`). A journey's
route is **not** one stored raw column: it is the **derived union of authored transit
legs ∪ the passive Dawarich track**, assembled `ST_Collect` into `MultiLineString`.
**Proposed resolution of the open sub-question** (route composition order): interleave
by start-time — each leg/track-segment ordered by its first-point timestamp. The Dawarich
track is consumed already as tracks/segments; a raw GPX dev-fallback is simplified
(Douglas–Peucker) + gap-split (archive B2) before union. Note the **two route
projections** kept distinct (data-model §Places): the organic **display route** (this
union) vs. the stylized **places skeleton** (ordered visit centroids) — Dawarich's
visit-vs-activity split.

### D3 ★ — i18n via a sidecar `translations` table, JP canonical

Every user-facing string is `{ja, en, zh}` in the demo. Store translations in a sidecar
`translations(owner_type, owner_id, lang, field, value, provenance)` — avoids a
column-per-locale explosion and, crucially, gives **per-language provenance**. Rule:
**Japanese is the authored canonical** (`jp-first-i18n`); EN/ZH start as machine drafts
(`provenance = machine`) that become `authored` once hand-corrected — i.e. the three-class
no-clobber rule gains a **language axis** (re-translation never clobbers a hand-corrected
locale). Translatable fields: `title, place, essay, caption, vendor, line, operator,
station-name`. **Not** translatable: `price_amount/currency`, `coords`, instants.

### D4 — Station catalog is a bundled fixture, denormalized onto the memento

Transit needs station→coords. Ship a **bundled JSON catalog** (JR + Tokyo Metro MVP,
`{name_en, name_ja, operator, line, lat, lon}`, from ekidata/OSM, committed as fixture) —
no live geocoding. **Denormalize** `from/to` name+coords into the transit memento's
`kind_data` so the DB renders without a `stations` FK, keeping it a rebuildable projection.
The catalog is the resolver behind the template's `station` **field type** (D9): the transit
creator's autocomplete queries it, and the picked station is written denormalized into
`kind_data`. It is **not** a `places` table — places are derived Dawarich visits (D2/§Places),
a distinct concept from a static station gazetteer.

### D5 — Add the flat cross-journey memento endpoint

The frozen API is journey-scoped only; v2's shelf (`allMementos`) needs a flat collection.
`GET /api/v1/mementos?sort=recent|kind&kind=…` → `{memento, journey-ref}` cards. Blessed
orderings only (chronological / along-route / by-kind); **no** view-specific sort column.
Keep `GET /api/v1/journeys` and `GET /api/v1/journeys/{slug}` (journey + derived route GeoJSON
+ mementos). The **list** `GET /api/v1/journeys` is a lightweight index projection that still
carries per journey a `memento_count` aggregate + a few **representative place dots** (coord +
label, derived from `mementos.geom`/`mementos.place`) — enough to render an index/landing map +
card grid (techo v3, `felicia:decision:techo-paper-v3`) in one call, no N+1 to `/{slug}`. **The API is versioned in the path** (`/api/v1/…`) so a future breaking shape is
an additive `/api/v2`, never a silent break. Public reads rounded to 4 decimals (archive D2),
`ETag`/`Cache-Control` from `updated_at`.

### D6 — Dates & money are structured; the client formats

API returns **ISO instant + resolved tz** and structured money `{amount minor-units,
currency}`. The client formats per-locale (`Intl`). One truth, no format drift — the
demo's pre-formatted `"2026年5月12日"` / `"JPY 210"` are a *view* concern.

### D7 — Waypoints demote to a derived rendering aid

The demo has no waypoints; mementos carry their own coords. Drop `WAYPOINT` from the
canonical contract — clustered "stops" become an optional derived overlay, computed when a
map view wants them, not a first-class nav level. (Push back if stops should stay a nav tier.)

### D8 — Canonical `kind` enum

`ticket | transit | goods | stamp | receipt | souvenir`, with **`transit` top-level**
(its edge-anchoring changes rendering and route assembly). Extensible by migration, but
enum'd because `kind` drives the stub templates. The **set** of valid kinds is the set of
registered templates (D9); the DB `kind` column is a soft enum (`text` + a check or a
lookup table), not a hard Postgres `enum` type, so a new template doesn't force a type
migration.

### D9 ★ — Kinds are declarative templates, not hardcoded per-kind code

`felicia:decision:memento-template-registry`. A memento **kind is a template declared as
data** — one entry per kind describing everything the three surfaces need:

```
template
  kind         "transit"
  anchor       point | edge          # edge => LineString geom (from→to); point => single coord
  fields[]     { name; type; required; widget?; translatable? }
  stub         template ref (frontend component id)
```

The same declaration drives all three consumers, so they can never drift:

1. **Admin authoring form** — the field list + `type`/`widget` generate the creator form
   (the transit creator, `transit-tickets.md`, becomes *one instance* of this).
2. **`kind_data` validation** — a pure `internal/domain` validator checks a submitted
   `kind_data` blob against its template (required present, types match, `anchor` ↔ geom
   consistent). `kind_data` stays `jsonb`; the template is its schema.
3. **Stub render + i18n** — the `stub` ref picks the frontend form; `translatable` fields
   feed the `translations` sidecar keys (D3).

**Field `type` catalog** (small, closed, extend by rule-of-three): `text`, `money`
(→ `price_*` / a `kind_data` money pair), `date`/`datetime`, `station` (resolves via the D4
catalog, writes name+coords), `venue` (a named place + coord — for `live`), `url`, `enum`.
A `type` may imply a resolver (`station`, `venue`) or a widget.

**Two worked kinds** settle the shape: **`transit`** (`anchor: edge`; `from`/`to` = `station`,
`operator`/`line` = `text` translatable, `fare` = `money`) and **`live`** (`anchor: point`;
`artist` = `text`, `venue` = `venue`, `date` = `datetime`, `seat` = `text`) — a concert/event
ticket, the "live ticket" case. Design + examples in
[`memento-templates.md`](memento-templates.md).

**Guardrail (anti-DSL).** The template describes *fields, an anchor, and refs* — nothing
more. It must **not** grow into a query/mapping/ETL language; that trap is already rejected
for source connectors (`source-connectors.md` §"Why the generic version bites"). Assembly
logic stays Go. Justified over hardcoded per-kind structs because kinds genuinely
proliferate (transit, live, goods, stamp, goshuin, omiyage…) and the form/validation/stub
triplet is real triplicated work.

## 3. Todos (rewritten)

Replaces the ticket-era `archive/todo.md` M0 "spec freeze." Flow stays research → spec →
TDD → build; we are finishing **research**.

**R — ratify (finish research)**
- [x] Ratify D1–D9 (or amend) and the §1 stack picks. *(D9 declarative registry ratified
      2026-07-09; D1–D8 still standing.)*
- [ ] Settle the two open sub-questions: route-composition order (D2 timestamp-interleave)
      and translation provenance mechanics (D3 language-axis no-clobber).
- [ ] Decide which `kind` **stub templates** ship first (taste test: `goods`, then the
      JR-style `transit` mag-stripe, then `live`) — carries over from `mementos-not-tickets.md`.
- [ ] Freeze the D9 **field `type` catalog** (the closed set: text/money/date/datetime/
      station/venue/url/enum) — the template validator (T) encodes it.

**S — spec (when promoted out of `archive/`)**
- [ ] `docs/spec/data-model.md` — memento-era schema + goose migration sketch (D1–D4, D7,
      D8): `journeys`, `mementos(+kind_data)`, `translations`, `memento_photos`; PostGIS
      geometry columns; unique/upsert targets refreshed off `mementos(journey_id,
      source_ref)`.
- [ ] `docs/spec/api-contract.md` — public read (D5, D6) incl. flat `/api/v1/mementos`; admin
      surface behind Access-JWT (archive D4); chi subrouter map.
- [ ] `docs/spec/memento-templates.md` — promote the D9 registry: template file format,
      frozen field-`type` catalog, the `Validate(template, kind_data)` contract, and how the
      admin form + stub + i18n derive from one declaration. (Design draft:
      [`memento-templates.md`](memento-templates.md).)
- [ ] Refresh the importer spec — field-scoped upsert **× i18n provenance** (D3), route
      union/compose (D2), Dawarich track+visits ingest and visit-snap (place-as-derived-visit).
      (OCR/vision pre-fill deferred — §1 note.)

**T — TDD (first failing tests, memento order)**
- [ ] **Template registry + `kind_data` validation (D9).** `internal/domain`: load a template,
      validate a `kind_data` blob against it — required-missing, unknown-field, per-`type`
      mismatch, `anchor`↔geom consistency. Table-driven with `transit.yaml` + `live.yaml`
      fixtures. **This is the first pure-core code** (correctness-critical, zero I/O). Pure.
- [ ] WKB↔GeoJSON round-trip + 4dp coordinate rounding (orb). Pure.
- [ ] Douglas–Peucker simplify + gap-split → `MultiLineString` (archive B2). Pure.
- [ ] EXIF extract (lat/lng/time) from a sample preview JPEG. Pure.
- [ ] Timezone resolution (tzf) incl. a cross-tz trip fixture (archive A4). Pure.
- [ ] Route composition — authored legs ∪ track, timestamp-interleaved (D2). Pure.
- [ ] Three-class **× language** no-clobber upsert (D3): re-import + re-translate never
      overwrites an authored essay or a hand-corrected locale.
- [ ] Zero-diff idempotency (archive C2): second import → 0 repo writes, 0 object puts.

## Sources

Library survey (2026): pgx/sqlc consensus (brandur.org/sqlc, reintech, jetbrains go blog);
geometry go-geos/go-geom/orb (pkg.go.dev twpayne, github paulmach/orb); routers
(jetbrains go ecosystem 2025/2026); migrations goose/migrate/atlas (atlasgo.io);
image imaging/bimg/govips (github disintegration, kovidgoyal); S3 aws-sdk-go-v2/minio-go
(developers.cloudflare.com/r2, github minio/minio-go); gpx tkrajina/gpxgo; exif
dsoprea/go-exif; anthropic-sdk-go structured outputs (github anthropics, platform.claude.com).
