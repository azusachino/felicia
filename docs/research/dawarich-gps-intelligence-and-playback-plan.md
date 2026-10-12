# Research & Architectural Plan: Dawarich GPS Intelligence & Cinematic Spatial Playback

> **Document Status**: Proposal & Implementation Blueprint  
> **Target System**: `felicia` (Spatial Journal Engine)  
> **Key References**: Dawarich (`Freika/dawarich` v2.x architecture), `hyoban.cc` cinematic spatial teardown, OSRM/Mapbox routing semantics.

---

## Executive Summary

The user identified two critical shortcomings in the current implementation:

1. **The current GPX dataset is not accurate enough / synthetic**: The published route (`publication/journeys/izu-trip-2026-08-01/route.gpx`) is a 99-point coarse polygon with average 4.1 km gaps (up to 14.88 km gap) and no timestamps. This causes the camera and route animation to cut across mountains and water in harsh jagged straight lines rather than adhering to roads and coastlines.
2. **The playback and map experience lacks the polish and sophistication seen in projects like Dawarich & Hyoban**: The map needs proper route segmentation, distance-paced progress, road network snapping, Chaikin path corner smoothing, camera framing along route bearings, and an integrated HUD.

This document analyzes the exact mechanisms used by Dawarich and Hyoban, diagnoses why the current Izu dataset and route engine feel broken, and presents a concrete, step-by-step engineering plan.

---

## Part 1: Deep Architectural Analysis — What Dawarich Got Right

Dawarich (`github.com/Freika/dawarich`, 10k+ stars) solves the exact same problem: transforming passive, noisy, raw location feeds (OwnTracks, GPX, Google Timeline, Dawarich mobile app) into cohesive, legible trip representations and interactive video/map animations.

Our reverse-engineering of Dawarich's codebase (`app/javascript/controllers/trip_maplibre_controller.js`, `maps_maplibre/utils/route_segmenter.js`, `video_studio/route_timeline.js`, `video_studio/path_smoothing.js`) reveals five foundational pillars:

### 1. Distance & Time Threshold Segment Splitting (`RouteSegmenter`)

- **The Problem**: If a traveler rides a train, takes a ferry, sleeps at a hotel for 10 hours, or drives across a tunnel, naive polyline engines connect point A to point B with a straight line that slices through bays and mountains.
- **Dawarich Solution**:

  ```javascript
  // Dawarich splits points when:
  // 1. Distance jump > threshold (e.g., flight, teleport, tunnel dropout)
  // 2. Time gap > timeThresholdMinutes (e.g., overnight stop, multi-day hiatus)
  if (distance > distanceThresholdKm || timeDiff > timeThresholdMinutes) {
    segments.push(currentSegment);
    currentSegment = [curr];
  }
  ```

- **Felicia Application**: In Felicia, journeys span multiple days and multiple legs (`leg_mode: car | ferry | train | plane | foot`). Rather than flattening an entire journey into one rigid 99-point `LineString`, routes must be modeled as discrete, semantic route segments per leg/day.

### 2. Distance-Paced Timeline Navigation (`route_timeline.js`)

- **The Problem**: In our previous implementation, we animated between stop coordinates using raw point-to-point interpolation or basic turf slicing. Because GPS points vary in spacing (some 100m apart, others 14 km apart), time-based progress caused the camera and vehicle dot to crawl through dense areas and whip violently past sparse segments.
- **Dawarich Solution**:
  - Dawarich builds a continuous distance cumulative array: `entries = [{ coord, dist, seg }]`.
  - Animation progresses at **constant geographic speed** (`sliceAtFraction(timeline, fraction)`), interpolating cleanly along distance (`lerpCoord(prev, next, t)`) regardless of point density.
  - Inter-segment jumps (ferries, flights, or teleports) cost **zero playback time**, preventing awkward empty-ocean crawls.

### 3. Bounded Chaikin Corner-Cutting for GPS Smoothing (`path_smoothing.js`)

- **The Problem**: Raw GPS coordinates create sharp, angular turns (elbows) that make camera bearing adjustments jerky and erratic.
- **Dawarich Solution**:
  - Implements bounded Chaikin corner-cutting with strict constraints:

    ```javascript
    const DEFAULT_ITERATIONS = 2;
    const DEFAULT_MAX_CORNER_M = 30; // Never cut corners by more than 30m
    const DEFAULT_MIN_EDGE_M = 12;   // Don't touch edges already sub-pixel
    ```

  - This turns sharp polyline corners into gentle arcs so camera heading changes smoothly without drifting off the underlying road.

### 4. Day-Differentiated Chromatic Layering (`day_routes_layer.js`)

- **The Problem**: Rendering the entire multi-day trip as a single orange stroke leaves the user disoriented about what happened on Day 1 vs. Day 2.
- **Dawarich Solution**:
  - Predefined high-contrast, perceptual palette per day:
    `Day 1: #6366F1 (Indigo), Day 2: #F43F5E (Rose), Day 3: #10B981 (Emerald), etc.`
  - In replay mode, previous days stay subtly visible at lower opacity, while the active day or active leg glows intensely.

### 5. Camera Tracking & Dynamic Bearing Tilt

- **The Problem**: Static birds-eye view (`pitch: 0`) feels flat like a paper atlas. Flying directly to a pin with zero bearing creates disconnected jump cuts.
- **Dawarich & Hyoban Solution**:
  - Cinematic 3D camera: `pitch: 45°-55°`, `bearing: routeHeading ± 15°`.
  - The camera doesn't just center on the pin; it tracks slightly ahead of the traveling head (`along(line, dist + leadOffset)`), providing forward perspective.

---

## Part 2: Diagnosis of the Izu Dataset & Current Limitations

### 1. Root Cause of "Coarse / Unrealistic Route"

We examined `publication/journeys/izu-trip-2026-08-01/route.gpx`:

- **Total Points**: Only **99 coordinates** across a **401.8 km** round-trip!
- **Point Density**: Average spacing of **4.10 km per point**.
  - Largest single jump: **14.88 km** straight line!
  - 33 out of 98 edges exceed **5 km**.
- **Lack of Timestamps**: The `<trkpt>` tags contain only `lat` and `lon` with no `<time>` attributes.
- **Misalignment with Mementos**:
  - `Memento 0 (Kujuppama Beach)` is **1.55 km away** from the nearest GPX point.
  - The GPX cuts straight across the Izu Peninsula from Ito to Shimoda without hugging Route 135 or the coastal highway.
- **Visual Artifacts**:
  - In our MapLibre viewer, the route draws straight lines cutting across mountains, bays, and residential zones.
  - When the camera zooms into Stop 1 (`zoom: 13.5`), the route appears as an arbitrary geometric cord floating 1.5 km off the coast instead of arriving at the beach parking lot.

### 2. Limitations in Current Playback Architecture

1. **Map Lib Topography vs. Route Detail**: While we added Esri Topo and Dark Canvas basemaps, having an ultra-detailed topographic basemap with high-res road networks makes an inaccurate 99-point coarse polygon look *worse*, because the discrepancy between the road network and the straight chords is glaringly obvious.
2. **Camera Transitions**: Jumping between Day overview and Stop 1 currently causes instant camera pitch and bearing swings.
3. **UI Polish**:
   - The top toolbar has overlapping elements in smaller viewports.
   - The brand pill (`felicia`) in the top-left appears twice (once in `App.svelte` and once in `FloatingToolbar.svelte`).
   - The stop card needs clearer visual hierarchy and tighter typography.

---

## Part 3: The Solution Blueprint

### Milestone A: High-Fidelity Road Network GPX Generation & Enrichment

To replace the 99-point coarse mock with an authentic, high-density GPS track:

1. **Map-Matched Waypoint Reconstruction**:
   - Trace the true route for the Izu trip from Tokyo/Kanagawa down Route 135 to:
     - **Stop 1**: Kujuppama Beach (九十浜海水浴場, Shimoda) — `[138.9807, 34.6667]`
     - **Stop 2**: Omuroyama Lift (大室山登山リフト, Ito) — `[139.0975, 34.9071]`
     - **Stop 3**: Jogasaki Coast (城ヶ崎海岸, Ito) — `[139.1394, 34.8918]`
2. **Road-Snapping**:
   - Use high-density road routing (OSRM / open road geometries for Route 135, Tomei Expressway, and Odawara-Atsugi Road) to generate a realistic polyline with points spaced every 50–100 meters (approx. 2,500–4,000 points).
   - Add realistic timestamps corresponding to the journey dates (`2026-08-01` and `2026-08-02`), matching Dawarich's expected input structure.
3. **Multi-Segment Structure**:
   - Segment 1 (Day 1): Tokyo → Odawara → Atami → Ito → Shimoda (Kujuppama Beach).
   - Segment 2 (Day 2): Shimoda → Kawazu → Omuroyama → Jogasaki Coast → Tokyo.

### Milestone B: Dawarich-Inspired Progressive Route Engine

1. **Distance Timeline Engine (`RouteTimeline`)**:
   - Build a TypeScript `RouteTimeline` class inside `@felicia/reader` mirroring Dawarich's `video_studio/route_timeline.js`.
   - Compute cumulative distances using Haversine calculation.
   - Implement `sliceAtFraction(timeline, fraction)` and `sliceWindow(timeline, fromDist, toDist)` for smooth animated route reveal.
2. **Bounded Chaikin Polyline Smoothing**:
   - Implement `smoothSegmentCoords(coords, { iterations: 2, maxCornerM: 30 })` to ensure camera heading calculations never suffer from sharp 90-degree discontinuities.
3. **Day-Colored Routes & Multi-Modal Styling**:
   - Differentiate segments by day color or transport mode (solid orange for driving, dashed for ferry/walking).
   - Draw an ambient, translucent "full planned route" layer beneath the progressive "traveled route" glowing line.

### Milestone C: Cinematic Camera Choreography & Bearing Following

1. **Dynamic Camera Framing**:
   - On `intro`: Smooth fly-out to journey bounding box (`pitch: 25`, `bearing: 0`).
   - On `day`: Smooth fly-in to the day's spatial cluster (`pitch: 35`, `bearing: 15`).
   - On `stop`: Dynamic ease-in to the stop location with `pitch: 45°` and bearing calculated from the incoming direction of travel (`bearing = getHeading(prevCoord, stopCoord)`).
2. **Smooth Traversal during Playback**:
   - When playing through a segment, update the traveling head marker smoothly, with the camera gently panning along the trajectory without sudden jerky jumps.

### Milestone D: Reader UI Polish & Clean HUD Hierarchy

1. **Deduplicate Brand Pills**:
   - Remove redundant floating badge in `App.svelte` so the reader's `FloatingToolbar` is the single source of truth for branding, journey selection, and surface switching.
2. **Enhanced Stop Card**:
   - Tighter padding, crisp Japanese typography (`Zen Old Mincho` / clean sans-serif), responsive live photo / video tag, and clear step indicators.
3. **Robust Fallbacks & Zero Chinese Text**:
   - Ensure all UI strings and dataset fallbacks prioritize `lang` (Japanese/English) over Chinese.

---

## Part 4: Implementation Work Breakdown

| Step | Scope | Description | Verification Gate |
|---|---|---|---|
| **1** | Route Data | Generate a realistic, high-fidelity road-matched GPX for `izu-trip-2026-08-01` with ~2,500 points snapped to Route 135 and local access roads. | Run `uv run python scripts/felicia.py publish` and inspect route coordinates density. |
| **2** | Route Engine | Implement `RouteTimeline` (cumulative distance, fractional slicing) and Chaikin smoothing in `@felicia/reader`. | Unit test in `packages/felicia-reader/src/playback/timeline.test.ts` (100% passing). |
| **3** | Map Layers | Integrate multi-segment and progressive distance drawing into `AtlasMap.svelte`. | Verify animated route draws smoothly along coastal highways without straight-line mountain cuts. |
| **4** | Camera & Nav | Implement heading-aware camera transitions with pitch/bearing easing. | Visual verification in `agent-browser` with screenshots. |
| **5** | HUD & Branding | Remove duplicate brand pill in `App.svelte`, refine FloatingToolbar layout, and verify clean typography across cards. | Visual verification across mobile and desktop viewport sizes. |
| **6** | Gates & Validation | Run full `make check`, ensure all 74+ Go tests and Vitest tests pass cleanly. | `make check` exit 0. |

---

## Approval Request

Please review this research and architectural plan. Once approved, implementation will proceed systematically starting with Milestone A (high-fidelity road-matched GPX generation) followed by Milestone B (Dawarich-style distance timeline engine).
