# felicia — Direction

> **Stage: implementation** (updated 2026-07-18; the research trail continues).
> Still deliberately unhurried (~6-month horizon). This doc is the light
> north star — _what felicia is_ and _which way we're pointed_ — not a spec.
> The decisions now live in `adr/`; delivery status lives in
> [`roadmap.md`](roadmap.md) and the target end-to-end journey in
> [`roadmap/user-journey.md`](roadmap/user-journey.md). Detailed early design
> thinking is parked in `archive/`; exploration lives in `research/`.

## The idea (one paragraph)

A map-based travel journal, centered on an interactive living journey playback canvas (modeled on [hyoban.cc](https://hyoban.cc/trip/2026-national-day-road-trip/), pivoting from the initial `liuaaron.com` ticket-stub baseline via [ADR 0008](adr/0008-pivot-to-hyoban-spatial-journal.md)). Each **journey** traces a multimodal route line on an edge-to-edge dark map; along it sit **mementos** — narrative essays, place chips, weather context, and tactile Live Photo Polaroid cards. An automated cinematic playback state machine guides viewers step-by-step through days, route legs, and dwell stops. The map is the index; the mementos are the stories.

> **Living playback and authentic moments over simulated ticket stubs.** Physical paper tickets are dying, and simulated perforation/barcode components introduced high layout complexity. Following ADR 0008, Felicia treats memory anchors as **Moments**: expressive typography, responsive Live Photos, and spatial journey playback. Full reasoning: [`docs/research/hyoban-trip-playback-teardown.md`](research/hyoban-trip-playback-teardown.md) and [ADR 0008](adr/0008-pivot-to-hyoban-spatial-journal.md).

## Direction: personal now, product-ready

Decided 2026-06-12 (ADR `felicia:decision:personal-now-product-ready`). The lowest-regret
path:

- **Build the personal artifact first** — a beautiful journal for one author (you). This
  is the liuaaron model and it's the right first goal.
- **Keep the seams swappable** so a future product pivot is _additive, not a rewrite_.
  "Product-ready" means **clean seams, not built features** — we do not build product
  surface now.
- The reasoning (and the competitive reality — Polarsteps already owns this space) is in
  [`research/product-vs-personal.md`](research/product-vs-personal.md).

### What shipped first: the web moat MVP

Exploring the SaaS angle produced a useful simplification, and the **web moat spike**
became the first thing built: Svelte + TypeScript + Tailwind, with MapLibre GL for the
dark route map. That spike has since grown into the public reader (four switchable
designs over one presentation-agnostic contract) backed by the working Go pipeline —
importer, intake planner, and the published-only static compiler. Original sketch:
[`research/notion-to-stack.md`](research/notion-to-stack.md).

### What "clean seams" means (cheap hedges, do these)

1. **Sources behind interfaces** — route source and photo source are interfaces; the pure
   core imports no source-specific types. Swapping a GPS/photo source never reaches the core.
2. **Trigger-agnostic importer** — the import _logic_ is a plain callable, not glued to CLI
   arg-parsing, so a future background sync or HTTP handler reuses it.
3. **A single journal/account root** — everything hangs off one root entity even though
   there's exactly one. Multi-tenant later = "add a second root," not "reshape every table."
4. **Privacy invariant stays load-bearing** — raw GPS lives only in the DB; public images
   are resized and EXIF-stripped. (A legal requirement the day it's ever a product.)

### Explicitly deferred (decided non-goals, not oversights)

Companion mobile app · OAuth photo providers (Google/Apple Photos) · background sync ·
`owner_id` columns / real multi-tenancy · app-level user auth · billing · GDPR tooling.

## The shape we keep believing in

These are _leanings_, not locks — they survive into the spec stage unless research unseats
them. Detail in [`archive/design.md`](archive/design.md).

- **Engine:** a pure importer core (join photos + GPS on timestamp, cluster waypoints,
  simplify route, OCR ticket metadata) behind I/O seams. This is the durable heart and the
  natural TDD target whichever direction we go.
- **Model:** Journey → Memento → {essay, extra photos, open-animation}; the memento is the
  click target. `kind` (goods · live · transit · stamp · receipt · souvenir · …) selects the
  stub form. (Was "Ticket"; generalized 2026-06-14 — see mementos-not-tickets.)
- **Stack leaning:** MVP UI in Svelte + TypeScript + Tailwind, MapLibre GL for the map, and Go
  (CLI importer + API), Postgres + PostGIS, S3-compatible storage when the data layer
  graduates past the spike.
- **Language leaning:** i18n is part of the product shape from the MVP: Japanese, English,
  and Chinese at minimum, with Japanese as the primary/default near-term language.

## Open research questions

- **The GPS track.** For a passive product it's the hard part; the web MVP side-steps it
  with a per-trip **GPX/GeoJSON file**. Open: is manual route input good enough, or is
  "no track" common enough to need the photo-trail fallback? (see saas-dataflow,
  product-vs-personal)
- **Stub rendering** — _resolved_ to **template-first** (rendered from data; photographed
  stub is the bonus), per mementos-not-tickets. Still open: _which_ `kind` forms ship first
  and how much design each earns.
- **Ticket-open animation** — _resolved_ to **shared-element morph** (2026-08-04), per
  [ADR 0003](adr/0003-single-flagship-reader-atlas.md), alongside
  the map-first/fixed-entry layout decision. Still open: the frontend style spec
  (per-kind stub anatomy, motion timing) and building a reader that matches it.
- **I18n shape** — how much of the first demo/spec is translated vs. just architected for
  Japanese/English/Chinese; Japanese should be the primary review path.

## How we move

Research → spec → TDD → build. We are in **implementation**: the backend pipeline
(field-scoped importer, intake planner, SQLite/PostgreSQL providers, public + admin
APIs, published-only static compiler) is built and tested, the public reader runs on
the real contract, and the admin authoring GUI is being assembled. Delivery status
lives in [`roadmap.md`](roadmap.md) and the per-stage journey status in
[`roadmap/user-journey.md`](roadmap/user-journey.md); the research trail in
`research/` continues alongside.

The earlier research tactic — prototyping the data model in a Notion sandbox before
committing to a stack — served its purpose and is recorded in
[`research/notion-prototype.md`](research/notion-prototype.md) (ADR
`felicia:decision:notion-schema-sandbox`).
