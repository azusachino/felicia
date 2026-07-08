# Research — from Notion sandbox to own stack

> 2026-06-15. The [Notion prototype](notion-prototype.md) is back-of-house only; it leaves
> the **moat untested** (map · designed stub · animation). This note draws the bridge: how
> Notion content crosses to our stack, and *what the first real build should be* — the moat,
> not the CRUD. Research-stage sequencing, not a spec. Under
> `felicia:decision:notion-schema-sandbox`.

## The trap to avoid

The tempting first build is the part Notion already does well: trip/memento CRUD + an
authoring UI. **Don't.** Notion *is* that, for free, indefinitely. Rebuilding it first spends
months re-creating a back-office while the thing that makes felicia worth existing — the
map-indexed, stub-rendered, animated scrapbook — stays unproven.

> **Invert the build order: front-of-house first.** The first thing on our stack is the
> moat reading Notion as its data source. Authoring stays in Notion until the moat is real.

## Step 1 — the moat spike (the real first build)

The smallest thing that tests what Notion *can't*, now as a web MVP: Svelte + TypeScript UI,
Tailwind styling, and MapLibre GL for the map. This keeps the liuaaron-shaped front-of-house
from the [teardown](liuaaron-teardown.md):

- **One trip**, real content, pulled from Notion (export JSON or the API — see below).
- **Dark map + one orange route** (hand-drawn GeoJSON or one GPX is fine — the route source
  isn't the point yet).
- **One memento** rendered **template-first** as a designed stub (pick the `goods` kind — the
  fuwamiku — since it's the furthest from a "ticket" and the best taste test).
- **One open-animation** on click → essay + gallery.
If that one screen feels like the artifact, the moat is real and the model survives contact.
If it doesn't, we learned it for the price of one page, not a product.

## Step 2 — the data bridge (Notion → Memento-creation seam)

Notion is the *source*, our importer is the *sink* — and it's the **same Memento-creation
seam** as every other source ([mementos-not-tickets](mementos-not-tickets.md)). So a
`notion` source is just one more interface impl behind that seam:

```
Notion API ──▶ notionSource (impl) ──▶ Memento-creation seam ──▶ core ──▶ Postgres + R2
   (pages,        maps page→draft        image + structured        (same as OCR /
    relations,    + re-fetches files     draft + route             Wallet / goods-photo)
    file URLs)
```

- **Pull, don't export.** CSV/MD export loses relation IDs and expires file URLs; the API
  returns structured pages + stable enough file URLs to re-fetch. Re-fetch each file →
  resize + **EXIF-strip** → R2 (the privacy invariant, free at the boundary).
- **Field-scoped, re-runnable.** Same rule as the importer everywhere: a re-pull never
  overwrites fields authored on our side. Early on there *are* no authored-on-our-side
  fields (Notion holds them), so this is trivial — and it stays correct when authoring later
  moves onto our stack.
- **Notion → model mapping** is 1:1 (that's why we built the template to the ER): Trip→Trip,
  Memento→Memento (`kind`, drafts, essay = page body → `description`), Photos→MementoPhoto.

## Step 3 — graduate authoring (only if/when it earns it)

Authoring leaves Notion **only** when our stack can do something Notion can't for the
author: live stub preview, GPX-snap placement, the animation picker. Until then the split is
a feature, not debt — Notion is a perfectly good admin UI for an author-of-one.

## Why this order is lowest-regret

- Tests the **differentiator first**, on real content, for the price of one page.
- The `notion` source is **throwaway-friendly**: it's one impl behind the seam, so deleting
  it later (when authoring moves in-stack) costs nothing structural — the seam stays.
- Keeps us honest about the stack leaning (Svelte + TypeScript + Tailwind for the MVP UI;
  Go core + Postgres/PostGIS + R2 when persistence graduates) without committing to the
  CRUD/auth/multi-tenant surface that's still deferred in [`direction.md`](../direction.md).

## Open

- Spike data feed: one-shot **export JSON** (zero auth, faster to start) vs. **API pull**
  (the real bridge) — probably export for the very first screen, API once it sticks.
- Which `kind` gets the first designed stub template (leaning `goods` for the taste test).
- Route source for the spike — hand-drawn GeoJSON vs. a single real GPX.
