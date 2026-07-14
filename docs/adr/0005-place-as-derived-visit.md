# ADR 0005: Places as Derived Visits

- **Status:** Accepted
- **Date:** 2026-07-09
- **Decisions:** `felicia:decision:place-as-derived-visit`

## Context

Our frontends need to display and cluster mementos by "places" (e.g. Kyoto, Shinjuku Station, Hakone). Creating a static, user-edited `places` table introduces massive overhead: the user would have to manually create and manage places, resolve duplicate naming (e.g. "Tokyo" vs. "Tokyo Station"), and link mementos to them.

## Decision

We decided that a "place" is a **derived visit**, computed dynamically from tracking data rather than stored as a primary database table.

Implementation details:

1. **Consume Dawarich Semantic Data:** Dawarich already implements a robust pipeline (`points -> tracks -> visits @ places -> trips`). When importing data, _felicia_ consumes Dawarich's `visits` and `places` endpoints directly.
2. **GPX Fallback:** If a raw GPX track is uploaded, the importer runs a local spatial-temporal clustering algorithm (based on dwell time and coordinates) to synthesize a `Visit` object.
3. **Double-Gated Snapping:** Point-geometry mementos are snapped to their corresponding visit at import time:
   - **Temporal check:** If the memento's timestamp falls within the arrive/depart window of a visit, snap to it.
   - **Spatial fallback:** If no visit overlaps temporally, find the nearest visit within a $500\text{m}$ radius. If none is found, fallback to the raw EXIF coordinate as a standalone, transient visit.
4. **No static table:** The frontend receives an ordered array of places per journey computed dynamically at query time (`places[] = { key, label, coord, seq, memento_count }`).

## Consequences

- Eliminates the need for a manual `places` CRUD interface in the admin panel.
- Ensures 100% semantic alignment with tracking providers (Dawarich / Google Timeline export via Dawarich).
- Standardizes spatial queries: mementos snap to meaningful "visits" rather than arbitrary raw GPS points, preventing location drifting.
