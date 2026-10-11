# Implementation Plan: Hyoban-Style Cinematic Journey Playback for Felicia

> Target: Evolve Felicia's public reader and desktop studio from static ticket stubs
> into an immersive, spatial, temporal journey playback engine modeled on `hyoban.cc`.
> Reference Teardown: `docs/research/hyoban-trip-playback-teardown.md`

## Architecture & Work Breakdown

```text
Phase 1: Ingestion & Model Enrichment (Go / SQLite / Contracts)
  ├── 1.1 Live Photo (.heic/.jpg + .mov) pairing & transcoding to WebP + MP4
  ├── 1.2 Multimodal transport leg classification (car, ferry, train, flight)
  └── 1.3 Historical weather snapshot enrichment during compilation

Phase 2: Fullscreen Reader & Glassmorphism HUD (packages/felicia-reader)
  ├── 2.1 Edge-to-edge MapLibre viewport with frosted-glass HUD (color-mix + blur)
  ├── 2.2 Top navigation toolbar: Trip picker + dual-mode switch (Map vs Stream)
  └── 2.3 Polaroid cover fan & responsive typography cards

Phase 3: Interactive Journey Playback Engine (Svelte 5 / MapLibre)
  ├── 3.1 State machine orchestrator: Intro -> Day -> Stop -> Summary
  ├── 3.2 Adaptive dwell timer: Math.max(5000, photos*2000, text*120)
  ├── 3.3 Camera flythrough with pitch, bearing, and route path animation
  └── 3.4 Floating bottom-left playback controller & segmented progress bar

Phase 4: Cinematic Stage Mode & Media Export (Element Capture / Web Audio)
  ├── 4.1 Fullscreen Stage mode (presenter view) with Live Photo auto-loop
  ├── 4.2 In-browser video recording via CropTarget + MediaRecorder (9:16 / 16:9)
  ├── 4.3 Web Audio BGM mixer with smooth fade-in/fade-out
  └── 4.4 Automated chapter timestamps generator (.txt export)
```

---

## Detailed Phases

### Phase 1: Ingestion & Model Enrichment

- **Component:** `apps/felicia-core`, `apps/felicia-runtime`, `apps/felicia-providers/sqlite`
- **Tasks:**
  1. **Live Photo Pairing:**
     - Update `felicia-cli journey ingest` to detect Apple Live Photo pairs (image + companion `.mov` with identical base name or matching asset identifier).
     - In `BlobStore` derivatives generation, emit scaled `.webp` alongside compressed `.mp4` video clips (H.264/AAC or VP9).
  2. **Transport Mode Annotation:**
     - Extend route segmentation to tag `legMode` (`car`, `ferry`, `train`, `plane`, `foot`).
     - Allow manual curation in Desktop Studio or infer from GPX speeds / candidate stops.
  3. **Weather Enrichment:**
     - During compilation, optionally fetch day-level historical weather snapshots (icon code + temperature range) via Open-Meteo archive API and embed into static payload.

### Phase 2: Fullscreen Reader & Glassmorphism HUD

- **Component:** `packages/felicia-reader`, `apps/felicia-public-site`
- **Tasks:**
  1. **Fullscreen Layout:**
     - Migrate `Atlas.svelte` into a full-bleed layout (`h-dvh w-full overflow-hidden`).
     - Remove split-screen ticket drawer as default view; replace with floating HUD.
  2. **Top Pill Toolbar:**
     - Build `TripPicker` dropdown and animated sliding-pill `ModeSwitch` (Stream vs Map).
  3. **Visual Polish:**
     - Implement the signature Polaroid 3-photo tilted fan (`-6deg`, `0deg`, `+6deg`).

### Phase 3: Interactive Journey Playback Engine

- **Component:** `packages/felicia-reader/src/playback/`
- **Tasks:**
  1. **State Machine (`createPlaybackMachine`):**
     - Manage steps: `intro`, `day` banners, `stop` dwells, `summary`.
     - Implement adaptive dwell algorithm: `Math.max(5000, photos*2000, text*120)`.
  2. **Camera & Route Orchestrator:**
     - Drive MapLibre `easeTo` and `fitBounds` with synchronized coordinate progression.
     - Render mode-aware route layers: solid line for `car`, dashed line (`line-dasharray`) for `ferry`/transit.
  3. **Playback Controls HUD:**
     - Bottom-left floating card with segmented tick progress bar, play/pause, prev/next, and speed toggles (1x, 1.5x, 2x).

### Phase 4: Stage Mode, Live Photos & Video Export

- **Component:** `packages/felicia-reader/src/presentation/`
- **Tasks:**
  1. **Live Photo Player:**
     - Seamless `<video>` companion overlay in Polaroid cards and Lightbox with `LIVE` indicator.
  2. **Video Export & Chapter Generator:**
     - Implement in-browser recording via `CropTarget` and `MediaRecorder` at 12 Mbps.
     - Web Audio API integration for BGM playback and stream mixing.
     - Emit `${title}-${ratio}.mp4` and `${title}-章节.txt`.

---

## Verification & Acceptance Criteria

1. **Ingestion Contract:** Ingesting a sample folder with Live Photos produces paired `.webp` and `.mp4` derivatives in `.felicia/derivatives`.
2. **Reader Visual Gate:** MapLibre renders full-screen with frosted glass controls on both desktop and mobile viewports (`@container`).
3. **Playback Functional Gate:** Pressing Play advances through Intro, Day banners, and Stops, smoothly animating the camera and route line.
4. **Live Photo Gate:** Live Photos silently play their motion clip on stop focus and in the lightbox.
5. **Video Export Gate:** Export produces a valid, playable video file with mixed audio and matching chapter timestamp text file.
