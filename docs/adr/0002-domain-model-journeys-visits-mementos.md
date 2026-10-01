---
id: "0002"
title: "Domain Model: Journeys, Visits, and Mementos"
status: "accepted"
date: "2026-10-01"
related:
  - "0001"
---

# ADR 0002: Domain Model: Journeys, Visits, and Mementos

## Context

Travel journals typically oscillate between chronological blog feeds (unstructured essays) and raw GPS track logs (unreadable coordinate streams). Neither models how memory actually works: places anchor events, but physical artifacts and authored stories give them meaning.

## Decision

**Model travel around three core entities: Journeys, Visits, and Mementos.**

1. **Journey (Spatial Index):**
   - Represents a discrete trip with date bounds, title, and an orange route line drawn on a world map.
   - The map serves as the primary visual index rather than a chronological feed.

2. **Visit (Derived Spatial Anchor):**
   - Following how Dawarich and Google Timeline model location (`points → tracks → visits @ places → trips`), a visit represents a dwell-time cluster along the GPS route.
   - Multiple memories can stack at a single visit anchor.

3. **Memento (Collectible Story Unit):**
   - Physical paper tickets are dying, so mementos are `kind`-tagged data records (`transit`, `goods`, `live`, `stamp`, `receipt`, `souvenir`).
   - Mementos render as collectible stubs along the journey route that animate open into an essay and photo gallery.

## Consequences

### Positive

- Bridges the gap between automated data capture (GPS tracks, photos) and human storytelling.
- Declarative kinds allow customized stub artwork and metadata schemas per memento kind without database schema migrations.
- Places are derived from spatial proximity, keeping raw GPS noise separate from memory anchors.

### Negative / Trade-offs

- Requires an intake review step to reconcile raw GPS stops into meaningful visit candidates.
