# Research — the transit ticket creator (JR · Metro)

> 2026-06-19. felicia needs a **ticket creator**: a manual authoring path where the user enters
> a transit ticket — from→to, date, line, fare — and gets a memento back. MVP scope: **Japan
> Rail + Tokyo Metro**. This note records why transit is a _distinct shape_ (edge-anchored, not
> point-anchored) and how it folds into the existing model. Outcome ADR:
> `felicia:decision:transit-ticket-creator`. Research-stage — model vocabulary, not a spec.
> Sits beside [`mementos-not-tickets.md`](mementos-not-tickets.md),
> [`source-connectors.md`](source-connectors.md).

## Why a creator, not an importer

IC-card travel (Suica / Pasmo / ICOCA) and e-tickets leave **no scannable stub** — the
[memento generalization](mementos-not-tickets.md) already flagged that physical tickets are a
shrinking supply. Transit is the sharpest case: you took the train, but there's often nothing
to photograph or OCR. So transit is **authored by hand** — the **E** (essay/author) half of
the A+E model, the manual sibling of the [auto-ingest connectors](source-connectors.md). Both
land through the _same_ Memento-creation seam.

## The structural twist: transit is edge-anchored

Every memento so far is **point**-anchored — one `coords`. A transit ticket is **edge**-anchored:

```
point memento:   • (one place)            ticket | stamp | goods
transit memento: •──────▶• (from → to)    kind: transit
```

This isn't a cosmetic field difference — it changes what the memento _is_ on the map:

> **A transit leg renders as a segment of the journey route.** The amber line stops being only
> a passive Dawarich track — for a rail trip the legs the user authors **are** the route.

So the route line has two sources that compose: **authored transit legs** (precise, the rail
backbone) ∪ **passive GPS track** (the walking-around-a-city fill). For a Japan Golden Route
trip, the JR + Metro legs draw most of the map on their own.

## The model

A `transit` kind (or `ticket` with a `transit` sub-shape — decide at spec time) carries:

```
Transit memento
  operator   JR East | Tokyo Metro | …      (MVP: these two)
  line       Tōkaidō Shinkansen | Ginza Line | …   (optional)
  from       Station { name; coords }
  to         Station { name; coords }
  date       2026-05-11  (+ optional time)
  fare       ¥…
  + shared memento fields: serial, essay, photo gallery, open-animation
```

`from` / `to` resolve to coordinates via a **station catalog** (below). Everything else is the
ordinary memento — same stub render, same essay + gallery, same open-animation.

## Station resolution (the MVP's one real dependency)

To turn "Tokyo → Shin-Osaka" into two points + a leg, we need station→coords. MVP keeps it
small and offline:

- **Bundled station catalog** (JSON) of JR + Tokyo Metro stations on the trip's lines —
  `{ name_en, name_ja, operator, line, lat, lon }`. Sourced from an open dataset (ekidata /
  OpenStreetMap), trimmed to scope, committed as fixture data. No live geocoding service.
- **Autocomplete** in the creator form against that catalog; pick `from`, pick `to`.
- Out of scope for MVP: every private railway, exact track-following polylines (a straight or
  lightly-curved leg between station coords is enough), fare tables (fare is typed, not computed).

## The creator form (E path)

A small authoring surface (admin app; for the MVP a single form is fine):

```
operator ▾   line ▾(optional)
from [⌕ station autocomplete]  →  to [⌕ station autocomplete]
date [____]   fare [¥____]
→ creates a transit memento + a route leg on the map
```

Because it produces the same shape, a JR/Metro **ticket photo** pulled via Immich + vision-LLM
could later _pre-fill_ this exact form — the manual creator and the connector converge on one
transit memento. Manual first (no artifact to rely on); enrichment is the bonus, mirroring the
goods-photo logic in [`mementos-not-tickets.md`](mementos-not-tickets.md).

## MVP scope

- **In:** JR + Tokyo Metro; from→to + date + line + fare; bundled station catalog with
  autocomplete; leg rendered as a route segment; standard stub + essay + gallery.
- **Out:** other railways, computed fares, exact rail-geometry polylines, real-time data,
  reverse-geocoding services.

## Open

- `transit` as its own top-level `kind` vs. a sub-shape of `ticket` — leans top-level, since
  the edge-anchoring changes rendering and route assembly.
- Stub template for a transit ticket (the JR-style orange mag-stripe ticket is iconic — a
  strong taste test, like `goods` was for the [first stub](notion-to-stack.md)).
- Route composition order: how authored legs + GPS track merge into one ordered journey line
  (timestamp interleave?).
- Station catalog provenance + licensing (ekidata vs. OSM extract) and how it's refreshed.
