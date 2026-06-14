# Research — personal artifact vs. sellable product

> 2026-06-12. Triggered by a real question: *the CLI import feels unfriendly — what if I
> want to sell this?* This note records the analysis so the **why** behind the
> [direction](../direction.md) survives. Outcome ADR: `felicia:decision:personal-now-product-ready`.

## The complaint, diagnosed

`waypoints import --immich-album "Jeju"` feels bad as a *product* surface. But the CLI
isn't the real problem — it's **correct** for an author-of-one (technical, owns the data,
runs the Pi). What actually breaks under "sellable" is the **assumptions beneath** the
import, not the command:

| Personal (today) | Required for a product |
| --- | --- |
| Self-hosted Dawarich — *you* run a GPS logger | Customers don't run loggers — **the hard problem** |
| Self-hosted Immich on a Pi | OAuth to Google/Apple Photos, or upload |
| `--immich-album` manual trigger | Invisible background sync |
| CLI on your laptop | Web/mobile onboarding wizard |
| One tenant, your data | Multi-tenant, auth, billing, GDPR-for-location |

So "make it sellable" is not a UX tweak to the importer — it's a different product.

## The market reality

This product **already exists and nailed the consumer UX: Polarsteps.** Passive GPS via
*their own app*, automatic photo import, a beautiful map, monetized with printed photo
books. Also in the space: Journi, Wanderlog, plus Strava / Google Timeline adjacent.

To sell against that you must out-execute on *something*: design taste (the memento → essay
format is genuinely differentiated and not what Polarsteps does), or a niche.

> **Sharpened 2026-06-14** ([`mementos-not-tickets.md`](mementos-not-tickets.md)): the
> differentiation is *object-first authored scrapbook* vs. *log*. Polarsteps, Dawarich, and
> Immich are all exhaustive automatic logs (every photo/GPS point, capture-timestamp-bound).
> felicia is selective, narrated (the essay), designed-rendered, and **back-fillable** —
> indexed by the few **mementos** you chose, not by everything captured. The niche is the
> **collector mindset** (*eki* stamps, *goshuin*, character goods, *omiyage*) — a taste-niche
> no log occupies.

## The crux: the GPS track

The single hard, unique part is **the passive GPS track**. Normal people have no Dawarich.
A product's options all cost real work:

- **Companion app that logs in the background** — i.e. you are now building Polarsteps' app.
  This *is* the moat and *is* the pain.
- **Import from existing sources** — Google Maps Timeline export, Strava, Apple Significant
  Locations. Lower effort, worse coverage, clunky onboarding.

Photos are comparatively easy (OAuth). The track is the whole game.

## Why "personal now, product-ready" is lowest-regret

The architecture leaning already separates a **pure core** (join-on-timestamp, cluster,
route-simplify, OCR precedence) from **I/O seams** (route source, photo source as
interfaces). That core *is* the product engine in either future — only the **sources** and
the **trigger** change:

- Sources: swap Dawarich/Immich impls for companion-app / OAuth impls.
- Trigger: swap CLI for background sync / HTTP handler.
- Privacy invariant (GPS-in-DB-only, EXIF-stripped public images): nicety → legal head start.

So building the personal version *with clean seams* costs almost nothing extra and keeps
the product door open. Building product features now (app, multi-tenancy, billing) is a
much larger commitment with a strong incumbent — premature.

**Decision:** personal artifact first; pay only for swappable seams; defer all product
surface. See [`../direction.md`](../direction.md) for the concrete hedges and non-goals.

## If we ever do pivot — open questions to revisit

- Companion app vs. import-from-Timeline for the track (coverage vs. effort).
- Differentiation thesis vs. Polarsteps — sharpened to *object-first scrapbook vs. log*
  (see mementos-not-tickets); open: is the collector-niche framing enough to sell on?
- Location-data compliance (GDPR) as a first-class design constraint.
