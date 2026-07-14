# Research — arranging the set of mementos

> 2026-07-02. The MapLibre decision demo (`web/src/App.svelte`) proved the
> map→stub→essay→gallery loop, but a review surfaced a missing decision: **how the _set_ of
> mementos along a journey is arranged**. Today they live in two disconnected places with no
> ordering model. This note names the gap, lays out the options, and leaves the choice as an
> open fork (taste — author's call). Research-stage — a direction sketch, not a component spec.
> Sits beside [`ux-restyle.md`](ux-restyle.md) (the visual critique),
> [`liuaaron-teardown.md`](liuaaron-teardown.md) (the muse), and
> [`mementos-not-tickets.md`](mementos-not-tickets.md) (the model).

## The gap

Read against the demo as it stands, mementos appear in two places that never agree on order:

1. **Map markers** — placed by `coords` (`syncMarkers`, ~L219). Spatial, but unordered.
2. **A flat quick-list** — `kind` + `title` buttons at the bottom of the artifact panel
   (~L398–405), in raw array order.

Neither expresses the core promise from `AGENTS.md` — _mementos **sit along** the route; the
map is the index, the mementos are the stories_. There is no chronology, no along-route
sequence, no sense of a **collection** you assembled. `ux-restyle.md` flags the adjacent
symptoms (dead entry state, flat markers, "no visible index") but stops at the _layout_ fork;
it never decides how the memento **set** is composed. That composition is this note.

## Why it matters

The arrangement _is_ the index. It sets three things at once:

- **the spine** — what orders the story (date? route position? kind?),
- **the entry** — what a first-time reader sees before clicking any marker,
- **the identity** — diary (timeline) vs. map-as-index (route order) vs. collection (wall).

## The options

### A. Chronological timeline (diary spine)

Panel lists mementos top-to-bottom by date; selecting one flies the map. Date is the spine.

- **For:** closest to liuaaron and to how a trip is _remembered_; back-filled goods slot in by
  their authored date; obvious, low-build.
- **Against:** a pure rail day can have several mementos at near-identical times; date ordering
  under-uses the map.

### B. Along-route order (map-as-index spine)

Mementos numbered by position along the amber route line, not the calendar.

- **For:** most directly honors "the map is the index"; reads as a walkthrough of the line.
- **Against:** back-fillable goods and non-contiguous days fight a strict spatial order; needs
  a projection of each memento onto the route (nearest-point along the LineString).

### C. Collectible wall / grid (object spine)

Mementos as a grid of designed stubs — the _collection_ you assembled; map becomes a secondary
view/tab.

- **For:** leans hardest into the stub-as-object identity (`stub-templates`,
  `memento-not-ticket`); great for a "look what I gathered" share.
- **Against:** demotes the map, which is the whole hero; weakest at "map is the index."

## Decided (2026-07-02) — A, liuaaron-aligned

Author confirmed a continued preference for the **liuaaron.com** UX. That settles it:
**A (chronological timeline) as the primary index**, in a **left index rail → map → paper
detail** layout (liuaaron's proven shape), with an **understated open** (a clean detail panel,
not a tear/flip/scrollytelling morph). Along-route sequence numbers on markers are an optional
secondary read (borrow B's numbering, not B's ordering). C (the collectible wall) is a possible
_second view_ later — a "collection" tab — never the primary index. This also resolves the
`ux-restyle.md` layout fork toward _index rail + detail_ over _map-first_ and _immersive
scrollytelling_. ADR: `felicia:decision:memento-arrangement`.

## Reference to steal from

**AdventureLog** (see [`references`] / MEMORY) is a near stack-twin (MapLibre + PostGIS +
Svelte MapLibre) whose entry-list + map pairing is worth a real teardown the way
`liuaaron-teardown.md` did the muse — specifically how it lists entries beside the map and how
selection drives the camera. **CSS scroll-snap** is the cheap, high-polish path for whichever
list/gallery wins (timeline scroll or collectible wall).

## Open forks — author's call

1. **Primary arrangement:** A / B / C (lean: A + B's numbering).
2. **Second view:** ship a collectible-wall tab at all, or map+timeline only?
3. **Marker-list binding:** hover-sync both directions, or click-only (as today)?

## Next step

When (1) settles, fold it into the candidate `felicia:decision:frontend-style` ADR named in
`ux-restyle.md` (layout + open-anim + **arrangement**), then promote a frontend style spec
before refactoring `web/`. No app code yet — this stays research.
