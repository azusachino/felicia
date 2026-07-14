# ADR 0009: Route Ingestion, Simplification, and Manual Curation UX

- **Status:** Accepted
- **Date:** 2026-07-09
- **Decisions:** `felicia:decision:route-sync-curation-ux`

## Context

Passive GPS location tracking (Dawarich) produces highly detailed coordinate tracks (typically at ~100m intervals). Storing and rendering these raw coordinates directly on a web client results in visual clutter, slow rendering, and unnecessary database storage.

Additionally, transit types like long-distance flights or areas with signal loss create spatial tracking gaps. Lastly, automated metadata extraction via LLM vision models is out-of-scope for this phase of the personal-now product, meaning ingestion must transition to a manual, highly tactile curation workflow.

## Decision

We decided to resolve these constraints through three coordinated edge behaviors and an interactive Admin Workspace:

1. **RDP Track Simplification:**
   - When importing location coordinates from Dawarich, the pipeline runs a **Ramer-Douglas-Peucker (RDP)** simplification algorithm.
   - Redundant vertices are discarded using a default deviation epsilon of `0.0001` degrees (~10 meters), keeping only major route changes.
2. **Geodesic Transit Legs (User-Input Routes):**
   - Gaps (such as flights from Tokyo to Osaka) are resolved by allowing the user to manually add route legs in the Admin panel.
   - The backend dynamically generates a Great-Circle (geodesic) arc connecting origin and destination coordinates, appending the resulting `LineString` to the journey's `gps_route`.
3. **Split-Screen Ingestion Workspace:**
   - Curation is driven visually inside the Admin App using a split-pane layout: a Map workspace on the left and a synced photo tray on the right.
4. **Drag-to-Snap Placement:**
   - Grabbing a photo from the tray and dragging it over the map magnetically snaps the memento placeholder along the journey's track (using EXIF timestamps or proximity).
   - Dropping the photo opens a blank curation drawer to select the kind template and manually input details (vendor, price, essay).
   - If EXIF timestamps are absent, dragging and dropping the dot on the map route manually anchors its coordinate.

## Consequences

- **Client Performance:** MapLibre GL renders clean, lightweight paths instead of high-density, noisy tracks.
- **Transit Cleanliness:** Flight routes are visualized as clean, natural arcs instead of straight lines cutting through maps.
- **Tactile Scrapbooking:** Ingestion feels like pasting collectibles onto a real map rather than filling tabular databases, preserving a premium user experience without external LLM dependencies.
