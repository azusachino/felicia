# Research — Notion as the schema sandbox

> 2026-06-14. A research-stage tactic: prototype the **data model + authoring flow** in a
> Notion template *before* a spec freeze, then implement on our own stack. De-risks the open
> question — *does the `Memento` model hold one shape across kinds?* — where it's cheap to
> get wrong. Sits beside [`mementos-not-tickets.md`](mementos-not-tickets.md) and
> [`saas-dataflow.md`](saas-dataflow.md). ADR: `felicia:decision:notion-schema-sandbox`.

## What this validates — and what it can't

Notion is **back-of-house**: schema, authoring, real content. The **front-of-house** — the
map, the designed stub render, the open-animation — is the differentiating, hard part, and
Notion proves nothing about it. That split is the whole point: validate the cheap-to-get-
wrong layer cheaply; keep the moat for the real build.

| Validates (do in Notion) | Can't touch (needs the real stack) |
| --- | --- |
| Schema — does `Memento` + `kind` hold across ticket/goods/receipt/stamp? Relations *are* the FKs | The **map** — no geo; route / GPX / click-stub-on-map is the index metaphor |
| Authoring flow — the "E" half of A+E; essay + gallery editing | Designed **stub render** — template-first typeset stubs |
| Real content now — accumulate mementos that become migration seed data | The open-**animation** (tear / flip / morph) |

## Extraction path (bank this now)

Treat Notion as a **throwaway sandbox with a known exit**, never a creeping source of truth:

- Export = Markdown + CSV; **relations export as page-name strings**; **file URLs expire**.
- The real migration uses the **Notion API** (structured JSON: relation IDs, file URLs to
  re-fetch) → our importer's Memento-creation seam. Same seam as every other source.

## The template (1:1 with the saas-dataflow ER)

Three related databases. If it feels right here, the schema is probably right.

**Trips**

| Property | Notion type | Notes |
| --- | --- | --- |
| Title | title | trip name |
| slug | text | url slug |
| date-start / date-end | date | |
| summary | text | |
| cover | files | |
| gpx | files | **out-of-band** — Notion has no geo |
| map-link | url | placeholder for the route |

**Mementos**

| Property | Notion type | Notes |
| --- | --- | --- |
| Title | title | the memento |
| Trip | relation → Trips | the FK |
| kind | select | `ticket · goods · receipt · souvenir · stamp` |
| vendor | text | draft (OCR / Wallet / email / vision-LLM) |
| price | number | draft |
| occurred_at | date | draft — **back-fillable** |
| place | text | **no real Point** — the map layer Notion can't hold |
| stub_image | files | photographed stub *or* (later) generated card |
| animation | select | open style |
| seq | number | order along the route |
| *essay* | **page body** | where Notion shines — the authored piece |

**Photos** *(mirrors `MEMENTO_PHOTO`)*

| Property | Notion type | Notes |
| --- | --- | --- |
| Title | title | short label |
| Memento | relation → Mementos | the FK |
| image | files | |
| caption | text | |
| seq | number | |

> Alternative: drop photos straight into the memento **page body** instead of a Photos DB —
> faster to author, looser relationally. Worth feeling out; the trade is fidelity vs.
> ergonomics.

The two deliberate divergences from the ER — `place` as text (not a geo `Point`) and
`gpx`/`map-link` as out-of-band fields — *are* the map layer. Their awkwardness in Notion is
the signal that the map is the real-stack's job, not a gap in the model.

## Bootstrapping

CSV import files live in [`notion-prototype/`](notion-prototype/): import **trips.csv**
first, then **mementos.csv**, then **photos.csv**. Notion imports relation columns as text —
after import, change `Trip` / `Memento` columns to **relation** type; they match by title.
Seed rows include the fuwamiku-at-a-live example so the `goods` kind is exercised from day
one.

## Open

- Does `kind` survive contact with real entries, or do goods/stamps want their own fields?
- Photos: related DB vs. page-body gallery — which authoring feel wins?
- How much OCR/vision prefill is missed by hand-entry here (i.e. what the real importer adds)?
