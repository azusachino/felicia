# Research — mementos, not tickets

> 2026-06-14. The click-target on the map was modeled as a **ticket stub**. Two pressures
> break that framing: physical tickets are dying, and the thing that anchors a memory often
> *isn't* a ticket at all. This note records the generalization and the moat it sharpens.
> Outcome ADR: `felicia:decision:memento-not-ticket` (under `personal-now-product-ready`).
> Research-stage — model vocabulary, not a spec. Sits beside
> [`product-vs-personal.md`](product-vs-personal.md) and [`saas-dataflow.md`](saas-dataflow.md).

## Two pressures on "ticket"

1. **Physical tickets are dying.** Tap-to-pay, e-tickets in Apple Wallet, QR codes on a
   phone screen — the paper stub is a *shrinking supply* of raw material. A journal premised
   on "the stub you collected" runs out of stubs.
2. **The entrypoint isn't always a ticket.** At a live event the admission ticket is *one*
   anchor — but so is the **fuwamiku** you bought, the tour shirt, the acrylic stand, the
   *omiyage*. The thing that holds the memory is an **object**, and "ticket" was only ever
   one kind of it.

Neither pressure is a problem once you see what the reference already does: liuaaron's
tickets are **hand-built HTML/CSS rendered from data** (see
[`liuaaron-teardown.md`](liuaaron-teardown.md)) — he never scans paper. So *constructing* the
stub isn't a compromise. It's the design.

## The generalization

> **Ticket → Memento:** a *collectible that anchors a memory*, tagged with a `kind`. The
> `kind` selects the rendered stub **form**. Everything else is unchanged.

```
Journey → Memento → { essay, photo gallery, open-animation }
          kind: ticket | goods | receipt | souvenir | stamp | …
```

The click-target abstraction, the open-animation, and the **Memento-creation seam** all
survive intact — this is a rename + a widened source set, not a reshape. `Ticket` becomes
just `kind: ticket`.

## Stub rendering: template-first (resolves an open question)

`direction.md` had stub rendering open: *type-templates vs. photographed-stub fallback.*
The two pressures settle it.

- **Template-first.** A memento is a **designed form rendered from structured data**
  (vendor / type / price / when / where). The `kind` picks the template — a transit-pass
  form, a concert-ticket form, a goods tag, a receipt, a stamp.
- **Photographed real stub is the bonus**, not the baseline — the delightful case when you
  *did* keep the paper.

## Sources widen behind the one seam

The data behind a memento can arrive many ways; all of them end at the same place the
[`saas-dataflow.md`](saas-dataflow.md) seam already defines — *an image + a structured
draft*:

| Source | Yields |
| --- | --- |
| OCR of a photographed stub | type / vendor / price / datetime draft |
| Apple Wallet `.pkpass` / e-ticket | structured fields directly |
| Email confirmation | structured fields (parse) |
| **Goods photo + vision-LLM** | "what is this object, when/where on this trip?" draft |

## The quiet superpower: objects outlive the moment

A goods is **back-fillable**. You still have the fuwamiku on your shelf — photograph it
*months later* and back-place it on the trip where you got it. The moment's when/where is
*authored / reconstructed*, not *required-captured*.

This directly kills the "stub capture habit is mixed" problem (in-moment discipline is hard):
the object doesn't need it, because the object is still here.

## The moat this sharpens — vs. Dawarich + Immich (and Polarsteps)

Those tools are **exhaustive automatic logs**: every GPS point, every photo, chronological,
bound to the capture timestamp. That's their nature and their ceiling. felicia is the
opposite on four axes they *structurally* can't cross:

| | Dawarich + Immich (log) | felicia (scrapbook) |
| --- | --- | --- |
| **Index** | photo / GPS point — everything | **memento** — the few things you chose |
| **Voice** | assembles; no narration | **the essay** — your words behind the stub |
| **Render** | shows the photo as-is | **designed collectible** — typeset form + animation |
| **Time** | prisoner of the capture timestamp | **back-fillable** — author the when/where |

One line: **they're the source data; felicia is the designed, authored, object-first
artifact you make from it — plus the objects they never captured.**

This sharpens the open differentiation thesis in
[`product-vs-personal.md`](product-vs-personal.md): the niche is the **collector mindset**
(*eki* stamps, *goshuin* seal books, character goods, *omiyage*) — a taste-niche no log
occupies, and exactly the "out-execute on design taste / a niche" path already chosen.

## What stays open

- **The stub template library.** How many `kind` forms ship first, and how much design each
  earns (ties into the existing stub-rendering question — now scoped to *which forms*, not
  *whether to template*).
- **Goods-photo enrichment UX.** How much to trust the vision-LLM draft of an object vs.
  force review (mirrors the OCR-confidence question in `saas-dataflow.md`).
- **Back-placement.** When/where is authored — map-drag is the baseline; is a "pick the trip
  day" assist worth it?
