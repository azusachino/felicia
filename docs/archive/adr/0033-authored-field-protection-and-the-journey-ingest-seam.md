---
id: "0033"
title: "Authored-Field Protection and the Journey Ingest Seam"
status: "accepted"
date: "2026-08-16"
related:
  - "0017"
  - "0022"
  - "0025"
---

# ADR 0033: Authored-Field Protection and the Journey Ingest Seam

## Context

[ADR-0022](0022-unified-intake-and-draft-pipeline.md), AGENTS.md, and
`docs/roadmap/user-journey.md` all promise the same thing: the importer is
field-scoped and "never overwrites authored fields — re-import is always safe."
The code recorded the authored mask and then ignored it (issue #73).

Three separate holes produced that:

1. **Mementos.** The ingest patch path merged its whole field mask into the
   destination row without consulting `dst.AuthoredFields`
   (`mergeMemento` / `mergeMementoFields`). Every admin save claims
   `title, place, kind_data, occurred_at, geom, kind, seq, …` as authored, so a
   re-import silently reverted all of them.
2. **Journeys.** `UpsertJourney` is a single write path shared by the importer,
   the intake service, and the authoring API. SQLite's version assigned every
   column and `authored_fields = excluded.authored_fields`, so a package import
   both clobbered authored values and reset the mask — disabling the one guard
   that did work (the importer's `gps_route` skip).
3. **Provider divergence.** PostgreSQL guarded journey columns _inside_ the
   upsert with `authored_fields @> ARRAY['…']`. Because the guard reads the
   stored mask, it also blocked _authoring_ edits: a journey field could be
   claimed once and then never edited again. Meanwhile the same statement reset
   `authored_fields` from `EXCLUDED`, so the guard was easy to walk past. The two
   providers therefore disagreed about what `UpsertJourney` means.

A shared upsert cannot fix this on its own: it cannot tell an import from an
edit, and the two intents need opposite behavior.

## Decision

**Write intent is explicit in the port, and the authored-field decision is made
in Go, once, for both providers.**

- Journeys get the split port mementos already had:
  - `UpsertJourney` is the **authoring** write. It assigns every column and takes
    the caller's authored mask at face value. An author can always re-edit a
    field they already claimed.
  - `ApplyIngestJourneyPatch(ctx, *domain.IngestJourneyPatch)` is the **source**
    write. It loads the stored row, applies only the masked fields the row has
    not claimed as authored, and writes the stored mask back verbatim.
  - `ports.JourneySyncStore` — the seam the importer and the intake service hold
    — exposes only the ingest method. An import cannot reach the authoring write.
- `domain.IngestableFields(mask, authored)` and `domain.MergeIngestJourney` live
  in `core/domain`, so both providers share one implementation of the rule
  instead of two dialects of SQL. This follows the reasoning `runtime/intake`
  already recorded for derived date bounds.
- The per-column `authored_fields @> ARRAY[…]` guards were removed from
  PostgreSQL's `UpsertJourney`. They made invariant 3 unreachable and duplicated a
  decision that now lives in one place.
- The invariants are pinned in `providers/contract`, so SQLite and PostgreSQL are
  asserted against the same cases (AGENTS.md development-flow constraint 2).

The four invariants, stated so they can be tested:

1. An ingest write never modifies a field in the destination row's authored mask.
2. An ingest write never shrinks or resets the destination's authored mask.
3. An authoring write can write any field and claim it as authored.
4. Running the same import twice is a no-op on authored fields and does not error.

## Consequences

- Re-import is safe for both mementos and journeys, which is what the docs
  already claimed.
- The importer's package mask claims `gps_route` only when the package actually
  carries a route, so a route-less re-import cannot blank an imported track.
- The importer's `gps_route` early return in `SyncRoute` is now redundant with the
  seam. It is kept because it also avoids a pointless source fetch.
- `runtime/intake`'s `adopt` authored check is likewise redundant with the seam.
  It is kept because it decides whether a write is needed at all.
- Adding a write path for a new journey field means extending
  `domain.MergeIngestJourney`; a field missing from that switch is simply never
  seeded by an import, which fails visibly rather than silently clobbering.
- Not addressed here: the admin journey API still sends `authored_fields`
  wholesale, so a client that omits the array can clear the mask. That is an
  authoring-side concern (invariant 3 territory), not an ingest one.
