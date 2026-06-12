# felicia — Direction

> **Stage: research.** Deliberately unhurried (~6-month horizon). This doc is the light
> north star — *what felicia is* and *which way we're pointed* — not a spec. Detailed
> design thinking is parked in `archive/` and drawn on when we choose to move
> to the spec stage. Exploration lives in `research/`.

## The idea (one paragraph)

A map-based travel journal, modeled on [liuaaron.com](https://liuaaron.com/) ("Aaron's
Waypoints"). Each **journey** is an orange route line on a dark world map; along it sit a
few **ticket stubs** — receipts, transit passes, admission tickets you actually collected.
The stub is the hero: click it and it animates open into a written **essay** and a photo
gallery. The map is the index; the tickets are the stories.

## Direction: personal now, product-ready

Decided 2026-06-12 (ADR `felicia:decision:personal-now-product-ready`). The lowest-regret
path:

- **Build the personal artifact first** — a beautiful journal for one author (you). This
  is the liuaaron model and it's the right first goal.
- **Keep the seams swappable** so a future product pivot is *additive, not a rewrite*.
  "Product-ready" means **clean seams, not built features** — we do not build product
  surface now.
- The reasoning (and the competitive reality — Polarsteps already owns this space) is in
  [`research/product-vs-personal.md`](research/product-vs-personal.md).

### Leading candidate for what ships first: the SaaS-manual model

Exploring the SaaS angle produced a surprise: a **manual web-upload** product is *simpler*
than the personal passive-ingest pipeline, not harder. User creates a trip, uploads a GPX,
uploads ticket images (OCR prefills, they edit), attaches photos. No Immich, no Dawarich,
no timestamp-join — **the user is the joiner**, and OCR is the only automation worth
keeping. The passive self-hosted pipeline becomes a *later power feature* behind the same
Ticket-creation seam. Full sketch: [`research/saas-dataflow.md`](research/saas-dataflow.md).
Leading candidate for the first thing built — still research, not locked.

### What "clean seams" means (cheap hedges, do these)

1. **Sources behind interfaces** — route source and photo source are interfaces; the pure
   core imports no source-specific types. Swapping a GPS/photo source never reaches the core.
2. **Trigger-agnostic importer** — the import *logic* is a plain callable, not glued to CLI
   arg-parsing, so a future background sync or HTTP handler reuses it.
3. **A single journal/account root** — everything hangs off one root entity even though
   there's exactly one. Multi-tenant later = "add a second root," not "reshape every table."
4. **Privacy invariant stays load-bearing** — raw GPS lives only in the DB; public images
   are resized and EXIF-stripped. (A legal requirement the day it's ever a product.)

### Explicitly deferred (decided non-goals, not oversights)

Companion mobile app · OAuth photo providers (Google/Apple Photos) · background sync ·
`owner_id` columns / real multi-tenancy · app-level user auth · billing · GDPR tooling.

## The shape we keep believing in

These are *leanings*, not locks — they survive into the spec stage unless research unseats
them. Detail in [`archive/design.md`](archive/design.md).

- **Engine:** a pure importer core (join photos + GPS on timestamp, cluster waypoints,
  simplify route, OCR ticket metadata) behind I/O seams. This is the durable heart and the
  natural TDD target whichever direction we go.
- **Model:** Journey → Ticket → {essay, extra photos, open-animation}; the ticket is the
  click target.
- **Stack leaning:** Go (CLI importer + API), Postgres + PostGIS, S3-compatible storage,
  Vite + Mapbox SPA — but the data *sources* are the open question, not the stack.

## Open research questions

- **The GPS track.** For a passive product it's the hard part; the SaaS-manual model
  side-steps it with a per-trip **GPX upload**. Open: is manual upload good enough, or is
  "no track" common enough to need the photo-trail fallback? (see saas-dataflow,
  product-vs-personal)
- **Stub rendering** — type-templates vs. photographed-stub fallback; how much is worth it.
- **Ticket-open animation** — flip vs. shared-element morph vs. tear/unfold; prototype later.

## How we move

Research → (when ready) spec → TDD → build. We are in **research**. No application code,
no spec freeze, no failing-test plan yet — those were archived as premature. When the open
questions above settle, we promote a spec out of `archive/` and tighten it.
