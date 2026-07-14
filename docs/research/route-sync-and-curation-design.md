# Research — Route Sync and Curation Design

This document records the design and workflow decisions for syncing location tracks from Dawarich, importing photo assets from Immich, and constructing mementos through the Admin UI.

Status: **Research Stage / Confirmed Design**  
Decisions: `felicia:decision:route-sync-curation`

---

## 1. Route Ingestion & Simplification

Dawarich passively logs high-density GPS track points (typically every 100 meters). Importing these coordinates directly creates visual clutter on the map and hurts performance.

### The RDP Simplification Pipeline

1. **Fetch:** The importer fetches raw GPX track points from Dawarich for the target journey date range.
2. **Simplify:** Before storing in the `journeys.gps_route` column (PostGIS `MultiLineString`), the track is simplified using the **Ramer-Douglas-Peucker (RDP)** algorithm.
3. **Threshold:** The simplification uses a configurable epsilon (e.g., `0.0001` degrees, ~10 meters) to discard redundant straight-line points and jitter, preserving only major route changes.

---

## 2. User-Input Routes (Flights & Lost GPS)

Passive GPS logs cannot capture high-altitude flights (due to airplane mode or signal loss). The system supports manual route additions.

### Ingesting Geodesic Arcs

1. **The Form:** In the Admin UI, the user clicks **[Add Leg]** and inputs origin and destination details (e.g., `HND` to `KIX`, or clicks directly on the map).
2. **Processing:** The Go server resolves the coordinates and generates a Great-Circle/Geodesic path (`LineString`) connecting the two coordinates.
3. **Append:** The generated path is appended directly to the journey's `gps_route` table column.

---

## 3. Manual Drag-to-Snap Curation UX

There is no automated LLM pre-fill of metadata. Curation is fully driven by the user through a tactile drag-and-drop workspace.

### Step-by-Step Workflow

1. **Initialize Journey:** The user inputs the title, slug, and date range. Felicia fetches the simplified Dawarich route and Immich photos.
2. **Split Workspace:**
   - **Left Pane (Map):** Shows the dark map with the orange route line.
   - **Right Pane (Tray):** Shows a grid of all synced photos from Immich for that trip date range.
3. **Drag and Placement:**
   - The user drags a photo from the tray.
   - The locator dot snaps magnetically along the orange route track on the map.
   - Dropping the photo opens the manual form drawer.
4. **Form Curation:**
   - The user selects the memento template kind (`transit`, `receipt`, `goods`, `stamp`, `live`).
   - The user inputs details (vendor, price, essay) manually.
   - On save, the memento is published.
5. **Adjusting Snaps:**
   - If the photo's EXIF timestamp is incorrect or missing, the user can manually drag the published memento dot to position it at the correct place (e.g., a specific train station) along the route.
