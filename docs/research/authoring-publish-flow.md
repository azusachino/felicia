# Research — the authoring → publish flow

> 2026-06-19. The product-side flow, sketched against the MVP: *import a track, curate the
> places that mattered, author each into a memento (ticket / goods / words), publish a subset
> to the public.* This note tightens that into the **E (authoring) half** of the A+E model and
> nails the **public/private boundary** — the piece not yet written down. Outcome ADR:
> `felicia:decision:authoring-publish-flow`. Research-stage — flow vocabulary, not a spec.
> Sits beside [`saas-dataflow.md`](saas-dataflow.md), [`source-connectors.md`](source-connectors.md),
> [`transit-tickets.md`](transit-tickets.md), [`mementos-not-tickets.md`](mementos-not-tickets.md).

## The flow

```
1. INGEST    GPX / Dawarich track  +  Immich photos
                track  → the amber route line
                photos ⨝ track on timestamp → CANDIDATE anchors (auto-proposed)

2. CURATE    keep the few anchors that matter, drop the noise
                + add what the track can't give:
                    a transit ticket (the creator)         — IS a route segment
                    a back-filled goods (object kept, shot months later)

3. AUTHOR    per memento → pick kind (selects the designed stub template)
                attach photos / goods, write the WORDS (the essay = your voice)

4. PUBLISH   journey → public, read-only SPA:
                dark map + amber route + designed stubs + essays + galleries
```

The MVP prototypes **step 4** — it's the public artifact, the output of the pipeline. The
auto-ingest connectors ([source-connectors](source-connectors.md), the **A** half) feed steps
1–2; the [transit ticket creator](transit-tickets.md) (the **E** half) feeds step 2.

## Three things the naive reading gets wrong

### 1. The user *curates*, doesn't *place*

"Select points on the GPX" sounds like dropping pins by hand. The leverage is the inverse: the
importer **proposes** candidate mementos from the photo×track timestamp join, and the user
mostly **confirms / rejects + writes**. Hand-placement is the fallback, not the main motion —
that is the entire reason for joining Immich + Dawarich ([saas-dataflow](saas-dataflow.md)). A
"snap to track" assist (place → snap coords to nearest track point / by timestamp) covers the
manual case.

### 2. GPX is the backbone, not a hard prerequisite

Two memento kinds need no track at all:

- **Transit tickets** *are* route segments (edge-anchored) — a pure rail trip can assemble its
  line from authored legs with zero GPX ([transit-tickets](transit-tickets.md)).
- **Back-fillable goods** — the object outlives the moment; its when/where is authored later
  ([mementos-not-tickets](mementos-not-tickets.md)).

So a journey must be able to exist **track-only**, **legs-only**, or a **mix**. "Import GPX
first" is the *common* path, not the *required* one.

### 3. "Open to public" is a boundary, not a switch

Publishing exposes a *bounded subset*, governed by the privacy invariant
(`.claude/rules/config.md`):

| Public — the artifact | Private — never served |
| --- | --- |
| essay, curated gallery, designed stub | photo originals |
| the *shape* of the route | raw GPS precision, EXIF, exact timestamps |

Granularity, three nested levels:

- **per-journey** — the publish unit.
- **per-memento** — a visibility toggle (some anchors stay private).
- **per-field** — public images resized + **EXIF-stripped**; raw track geometry stays in the
  DB; the public route is **simplified / snapped**, never the precise trace.

## Net

Read the model as **ingest → curate (confirm proposals) → author the words → publish a bounded
subset**, with GPX as the usual-but-optional backbone. It composes with what's already decided:
the field-scoped importer guarantees a re-import **never clobbers** the words authored in step 3
(design §5), so steps 1 and 3 stay independent and re-runnable.

## Open

- Curate UI: a review queue of proposed candidates (accept / merge / reject) vs. a map-first
  "pins to confirm" surface — which feels less like data entry?
- Publish unit: journey-at-once vs. memento-by-memento drip.
- Public route fidelity: how aggressively to simplify/snap so the line reads well *and* leaks
  no precise GPS (ties to the §2 privacy invariant).
- Where authoring lives during research: Notion ([notion-to-stack](notion-to-stack.md)) vs. an
  in-stack admin app — the flow above is agnostic, but the curate/snap assists are the trigger
  to graduate off Notion.
