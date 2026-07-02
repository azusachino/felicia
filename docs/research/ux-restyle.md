# Research — UX & style rethink

> 2026-07-02. The React-TS MVP (`web/`) proved the map→stub→essay→gallery loop works, but a
> close read of the running prototype surfaced UX gaps and a split visual identity that a
> beautiful personal artifact can't ship with. This note is the critique + the direction we
> commit to on paper, before touching CSS. Research-stage — a design north star for the
> frontend, not a component spec. Sits beside [`liuaaron-teardown.md`](liuaaron-teardown.md)
> (the reference) and [`mementos-not-tickets.md`](mementos-not-tickets.md) (the model).

## The diagnosis (what the running MVP gets wrong)

Read against `web/src/App.tsx` + `index.css` as they stand:

### Structural / UX

1. **Dead entry state.** `isCollapsed` initializes `true`, so on load the side panel is
   translated fully off-screen — and its reopen button is anchored *to the panel*
   (`left: -50px`), so it rides off-screen too. The **welcome view and the memento
   quick-list are unreachable on first load**. The only way in is clicking a ~32px marker you
   have to already know is there. "The map is the index" ships with **no visible index**.
2. **No signature open moment.** `direction.md` names the click→*animates open* as the core
   promise (flip / tear / morph — open). Today it's a plain panel slide; the memento never
   behaves like a physical object being opened.
3. **Scaffolding leaks into the product.** A `TSX Prototype` badge in the header and an
   **"Add Ticket"** floating action button — the latter directly contradicts the decided
   *memento, not ticket* vocabulary (`felicia:decision:memento-not-ticket`).

### Visual identity

4. **Two clashing stub languages.** The warm JR/Metro ticket (amber thermal paper, guilloché
   grain, black mag-stripe) is genuinely good — but the generic `memento-stub` sits beside it
   with **stock-Tailwind kind badges** (`kind-ticket #6366f1` indigo, `kind-stamp #ec4899`
   pink, `kind-goods #eab308`, `kind-transit #06b6d4` cyan). A cool SaaS palette fighting a
   warm paper object; it reads as two apps.
5. **Flat markers.** Every kind renders the same 32×44 icon-badge; only the glyph changes.
   The "collectible stub" identity that makes the reference sing doesn't exist *on the map*.
6. **Justified essays.** `.essay-text { text-align: justify }` produces rivers and ragged
   inter-word spacing on a narrow measure — degrading the one thing the detail view exists
   for: reading.

The engineering is solid. The gaps are **object-ness** (stubs don't feel like things) and
**the way in** (no index, no opening).

## Direction — committed on paper

### Identity: warm paper is the system (high confidence)

Extend the JR ticket's material language to **every `kind`**, and delete the Tailwind badge
palette. On the dark map, each memento is a *designed paper object*:

- **ticket** — admission stock (the reference's Jeju/UNESCO look; perforated stub).
- **transit** — the existing きっぷ magnetic ticket (keep as-is; it's the exemplar).
- **stamp / goshuin** — ink-on-washi: vermilion seal, brushed strokes, paper tooth.
- **goods** — a **hang-tag / kraft label** (the fuwamiku's swing ticket), string hole + eyelet.
- **receipt** — thermal-roll paper, monospace, `react-barcode`-style bars (reference does this).

One warm accent scale (amber → orange, our existing `--accent-orange`) replaces indigo/pink/
cyan. Kind is read from **form and material**, not a colored pill.

### Kill the scaffolding (high confidence)

Remove the `TSX Prototype` badge and the `Add Ticket` FAB from the reading experience.
Authoring/creation is the **admin (E) surface** per `felicia:decision:authoring-publish-flow`
— it does not belong in the public journal chrome. The manual transit creator stays, but as
an authoring affordance, not a headline button on the reader.

### Fix the entry (high confidence)

The index can never be unreachable. Land with a **journey overview** visible (title, dates,
memento count, the quick-list) and a **persistent affordance** to reopen it — anchored to the
viewport, not to the panel that slides away.

### Essay typography (high confidence)

Drop `justify` → ragged-right. Widen the reading measure, raise leading (~1.7 is fine),
consider a serif for essay body to lean *travel book* over *dashboard* (ties to the layout
fork below).

## Open forks — your call

These are genuine taste decisions, left undecided (asked 2026-07-02, author away):

1. **Layout / navigation model.**
   - *Index rail + detail* — left rail lists journeys → mementos (photo-count badge, sort
     toggle), map center, paper detail panel. Proven (this is liuaaron); fixes the dead entry
     structurally.
   - *Map-first, fixed entry* — keep the single right glass panel over a full-bleed map, but
     land on a journey-overview card + a viewport-anchored index toggle. Smallest change;
     keeps the map the hero. **(current lean)**
   - *Immersive scrollytelling* — scroll drives the camera along the route; mementos surface
     as reached. Cinematic; biggest build.

2. **Signature open interaction.**
   - *Shared-element morph* — the map stub grows into the full stub in the detail view; one
     continuous object. Best reinforces map-as-index. **(current lean)**
   - *Tear / unfold* — perforated stub tears along its dashed line to reveal the essay.
     Tactile; thematically perfect for "warm paper is the system."
   - *Flip* — front (stub face) → back (essay + photos). Cheapest to do well.

## Implications & next step

- No app code yet — this stays research. When the two forks settle, promote a **frontend
  style spec** (design tokens, per-kind stub anatomy, motion spec) and *then* refactor `web/`.
- The refactor is mostly `index.css` (tokens + per-kind stub CSS) + `App.tsx` entry-state and
  scaffolding removal; the domain model and map wiring are sound.
- Candidate ADR once forks settle: `felicia:decision:frontend-style` — "warm paper as the
  system, scaffolding out of the reader, [layout] + [open-anim]."
