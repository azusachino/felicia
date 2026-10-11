# hyoban.cc teardown — interactive travel playback and spatial journal

> Live-site and bundle analysis of [hyoban.cc/trip/2026-national-day-road-trip/](https://hyoban.cc/trip/2026-national-day-road-trip/) ("国庆东北自驾"),
> conducted 2026-10-11 via network/asset inspection, CSS/JS reverse-engineering, and DOM decompilation.
> Complements `liuaaron-teardown.md` and provides the blueprint for Felicia's cinematic playback evolution.

## Method

1. Headless asset inspection and HTTP request capture against `hyoban.cc/trip/2026-national-day-road-trip/`.
2. Extracted and decompiled the Astro client bundle (`CalendarMap.astro`, `CalendarMap.js`) and CSS stylesheets.
3. Extracted JSON data islands (`data-calendar-map-locations`, `data-calendar-map-trips`, `data-calendar-map-trip-data`).
4. Reconstructed the playback state machine, dynamic pacing algorithm, Element Capture API recording loop, and Web Audio mixing pipeline.

## Architectural Findings

### 1. Unified Spatial Canvas & Floating Glassmorphism HUD

The entire viewport is an edge-to-edge map canvas (`<main class="relative isolate h-dvh w-full overflow-hidden">`), over which floating frosted-glass controls are suspended using modern CSS:

```css
background: color-mix(in srgb, var(--color-bg) 82%, transparent);
border: 1px solid color-mix(in srgb, var(--color-border) 82%, transparent);
backdrop-filter: blur(1rem);
```

Key UI components:

- **Top Pill Toolbar (`calendar-map-toolbar`)**:
  - Author identity pill (`avatar` + handle).
  - Centered trip picker dropdown (`<select data-calendar-map-trip-select>`): switch between `全部地点` (Global Locations Atlas) and `旅程回放` (Curated Journeys).
  - Animated view mode switch (`calendar-mode-switch`): Smooth sliding pill indicator between **日历 (Calendar stream)** and **地图 (Map canvas)**.
- **Floating Playback Card (`calendar-map-trip`)**: Bottom-left card housing segmented progress indicators, stop information, controls, and speed multipliers.
- **Floating Details Drawer (`calendar-map-drawer`)**: Bottom-right slide-in panel showing deep stop narratives and mementos.

### 2. The Journey Playback Engine (`旅程回放`)

Rather than relying on static pins, the site runs an automated chronological playback state machine:

```text
[ Intro ] ──> [ Day 1 Banner ] ──> [ Stop 1 ] ──> [ Stop 2 ] ──> [ Day 2 Banner ] ──> ... ──> [ Summary ]
```

#### Step Structure

- **`intro`**:
  - Displays title, date range, total days, stop count.
  - Tilted Polaroid photo fan (3 cover images rotated at `-6deg`, `0deg`, `+6deg`).
  - Frames global bounding box of the entire journey.
- **`day`**:
  - Day transition banner: `第 {day} 天 · {date}`.
  - Weather widget: e.g. `阴 · 16–21°C` with Lucide weather icon.
  - Route sequence: e.g. `无为 → 南京 → 烟台港客运站`.
- **`stop`**:
  - Camera eases in (`easeTo`/`flyTo`) with smooth pitch and bearing to center on the stop.
  - Active leg draws forward up to the stop.
  - Displays stop essay/text, place chips, and photo cards.
  - Autoplays Live Photo video clips if available.
- **`summary`**:
  - Zooms out to the full journey bounding box.
  - Aggregated metrics scorecard: `[ 7 天 ]  [ 18 条记录 ]  [ 12 个地点 ]  [ 142 张照片 ]  [ 2,890 公里 ]  [ 36 小时车程 ]`.

#### Content-Aware Adaptive Dwell Algorithm

The playback pace adjusts dynamically according to textual and visual density:

```javascript
function getStepDuration(step) {
  if (step.type === 'intro') return 3000;
  if (step.type === 'day') return 2200;
  if (step.type === 'stop') {
    // 5s base, +2s per photo, +120ms per character of text
    return Math.max(5000, step.stop.images.length * 2000, step.stop.text.length * 120);
  }
}
```

### 3. Multimodal Transport & Route Intelligence

The route data contract differentiates transport modalities:

- **`legModes`**: Array assigning transport modes per leg: `car`, `ferry`, `train`, `plane`.
- **Route Styling**:
  - `car`: Solid colored line tracking roads.
  - `ferry`: Dashed nautical track (`line-dasharray: [2, 3]`).
  - `plane`: Curved great-circle arc.
- **Trailing Waypoints Animation**: Waypoint coordinates along each leg are progressively drawn as the camera travels.

### 4. Apple Live Photos (实况照片) Pipeline

Every photo object in the payload supports an optional live video companion:

```json
{
  "src": "...-960w.webp",
  "srcset": "...-480w.webp 480w, ...-960w.webp 960w, ... 1440w",
  "thumbnail": "...-480w.webp",
  "video": "...-live.mp4",
  "width": 1440,
  "height": 1080
}
```

- **Inline Hover/Playback**: Silent looping playback on stop dwell or hover.
- **Lightbox Integration**: Dedicated modal with a `播放实况` toggle button.

### 5. In-Browser Video Reel Recorder & Chapter Generation

The client script includes a browser-native video export tool:

1. **Element Capture API**: Uses `navigator.mediaDevices.getDisplayMedia` combined with `CropTarget.fromElement(container)` to crop recording directly to the presentation canvas.
2. **Aspect Ratios**: Supports `portrait` (9:16 for Reels/Shorts) and `landscape` (16:9 for Bilibili/YouTube).
3. **Web Audio BGM Mixing**: Decodes audio with `AudioContext`, pipes it through `createMediaStreamDestination()`, and combines video + audio into `MediaRecorder` at 12 Mbps with automatic fade-in and fade-out.
4. **Automatic Chapter Export**: Tracks step transitions to generate a timestamp file (`${title}-章节.txt`) ready for video platform descriptions.

## Implications for Felicia

1. **Evolution from Ticket Atlas to Cinematic Playback**:
   Felicia's original model (`liuaaron.com`) focused on static ticket stubs. Hyoban demonstrates how to evolve this into a dynamic, temporal narrative where journeys play like documentaries while preserving mementos as stories.
2. **Retain Felicia's Core Strengths**:
   - Felicia's Go backend, SQLite storage, authorship safety (`authored_fields`), and single-step CLI ingestion remain untouched.
   - The evolution takes place in the **data enrichment pipeline** (Live Photos, transport modes, weather) and the **public reader/desktop presentation layer**.
