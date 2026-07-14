# felicia — Direction

> **Stage: research.** Deliberately unhurried (~6-month horizon). This doc is the light
> north star — _what felicia is_ and _which way we're pointed_ — not a spec. Detailed
> design thinking is parked in `archive/` and drawn on when we choose to move
> to the spec stage. Exploration lives in `research/`.

## The idea (one paragraph)

A map-based travel journal, modeled on [liuaaron.com](https://liuaaron.com/) ("Aaron's
Waypoints"). Each **journey** is an orange route line on a dark world map; along it sit a
few **mementos** — the objects that anchor a memory: an admission ticket, but equally a
souvenir, a goods (the fuwamiku you bought), a receipt, a stamp. Each renders as a
collectible **stub**; click it and it animates open into a written **essay** and a photo
gallery. The map is the index; the mementos are the stories.

> **Memento, not ticket.** Physical tickets are dying and the anchor often isn't a ticket
> anyway, so the click-target generalizes: `Ticket → Memento` with a `kind`
> (ticket · goods · receipt · souvenir · stamp · …), and the stub is _rendered from data_,
> template-first, not scanned. Full reasoning:
> [`research/mementos-not-tickets.md`](research/mementos-not-tickets.md)
> (ADR `felicia:decision:memento-not-ticket`).

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

### Leading candidate for what ships first: the web moat MVP

Exploring the SaaS angle produced a useful simplification, but the lowest-regret MVP is now
a **web moat spike**: Svelte + TypeScript + Tailwind, with MapLibre GL for the dark route map.
The app reads one real trip, renders one designed memento stub, and opens it into an essay +
gallery. No passive Immich/Dawarich pipeline. The user is still the joiner: trip content can
come from the Notion sandbox, one GPX/GeoJSON route, and local/R2-backed images. Full sketch:
[`research/notion-to-stack.md`](research/notion-to-stack.md). Leading candidate for the first
thing built — still research, not locked.

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
  click target. `kind` (ticket · goods · receipt · souvenir · stamp · …) selects the stub
  form. (Was "Ticket"; generalized 2026-06-14 — see mementos-not-tickets.)
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
- **Ticket-open animation** — flip vs. shared-element morph vs. tear/unfold; prototype later.
- **I18n shape** — how much of the first demo/spec is translated vs. just architected for
  Japanese/English/Chinese; Japanese should be the primary review path.

## How we move

Research → (when ready) spec → TDD → build. We are in **research**. No application code,
no spec freeze, no failing-test plan yet — those were archived as premature. When the open
questions above settle, we promote a spec out of `archive/` and tighten it.

Current research tactic: **prototype the data model in a Notion template** (Trips / Mementos
/ Photos, 1:1 with the saas-dataflow ER) to settle "does the `Memento` model hold?" and start
real content now, then pull one trip into a Svelte/TypeScript/Tailwind web moat spike. The
map / stub-render / animation stay out of Notion and belong to the real MVP. See
[`research/notion-prototype.md`](research/notion-prototype.md) (ADR
`felicia:decision:notion-schema-sandbox`).
