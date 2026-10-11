---
id: "0008"
title: "Pivot to Hyoban Spatial Journal and Cinematic Playback"
status: "accepted"
date: "2026-10-11"
related:
  - "0001"
  - "0002"
  - "0003"
---

# ADR 0008: Pivot to Hyoban Spatial Journal and Cinematic Playback

## Context

Felicia was originally modeled on [liuaaron.com](https://liuaaron.com/) ("Aaron's Waypoints"), interpreting travel memories through simulated physical collectible ticket stubs (admission passes, receipts, transit tickets with barcodes and perforations) displayed in a split-screen catalogue.

In practice, physical paper tickets are dying, and simulating artificial ticket textures in CSS introduced high layout debt without matching real modern travel records. Furthermore, static pin inspection fails to capture the living, temporal narrative of a journey.

An in-depth architectural teardown of [hyoban.cc](https://hyoban.cc/trip/2026-national-day-road-trip/) (`docs/research/hyoban-trip-playback-teardown.md`) demonstrated a superior paradigm:

1. An immersive, edge-to-edge spatial map canvas with floating frosted-glass HUD controls (`backdrop-blur`, `color-mix`).
2. An automated, cinematic **Journey Playback Engine** (`intro` → `day` banner → `stop` flythrough → `summary` scorecard) with content-aware adaptive dwell pacing.
3. Multimodal transport leg visualization (`car`, `ferry`, `train`, `plane`) with solid and dashed line rendering.
4. Native Apple Live Photo integration (`.webp` + `.mp4` video companions) with silent hover playback.
5. Dual-surface navigation via a seamless animated pill switch between the spatial map canvas and a linear chronological journal stream.

Because Felicia's reader design is not yet settled, pivoting directly to this model aligns with the core vision of *"The map is the index; the mementos are the stories"* while delivering a dramatically more engaging and authentic travel experience.

## Decision

**Adopt `hyoban.cc` as Felicia's visual and architectural North Star, replacing the `liuaaron.com` ticket-stub simulation model.**

1. **Retire Simulated Ticket Stubs:**
   - Deprecate simulated ticket components (`TicketStub.svelte`, mock barcodes, perforation borders).
   - Standardize story atoms on **Moments**: clean typography, place chips, weather badges, and Polaroid-style photo cards with natural micro-rotations (`-6deg`, `0deg`, `+6deg`).

2. **Full-Bleed Viewport & Floating Glassmorphism HUD:**
   - Make the full-bleed MapLibre canvas (`h-dvh w-full overflow-hidden`) the default viewport.
   - Replace the heavy split-pane sidebar with floating, frosted-glass controls (Top toolbar with trip picker and animated mode switch; bottom-left playback controller; bottom-right detail drawer).

3. **Journey Playback Engine:**
   - Implement the chronological state machine: `intro` (tilted photo fan & stats) → `day` (weather badge & route milestones) → `stop` (camera flythrough, live photo loop, essay) → `summary` (journey metrics).
   - Use dynamic content-aware stop pacing (`Math.max(5000, photos*2000, text*120)`).

4. **Dual Surface Information Architecture:**
   - Provide two complementary views under one unified reader:
     - **Map Canvas (`/` or `/map`)**: Spatial playback and interactive route inspection.
     - **Journal Stream (`/journal`)**: Vertical chronological reading feed of moments grouped by date.

5. **Data Model & Ingestion Enrichment:**
   - Extend route leg definitions with `leg_mode` (`car`, `ferry`, `train`, `plane`, `foot`).
   - Pair Apple Live Photos during ingestion, storing `.webp` images alongside compressed `.mp4` video clips in content-addressed derivative storage.

## Consequences

### Positive

- Replaces artificial paper ticket simulation with authentic, photo-forward travel storytelling.
- Drastically simplifies reader component architecture and eliminates complex perforated ticket CSS hacks.
- Elevates Felicia from a static pin-viewer into a cinematic, documentary-style travel playback experience.
- Naturally supports modern media formats, including Apple Live Photos.

### Negative / Trade-offs

- The original `liuaaron`-style collectible ticket stub art direction is retired.
- Ingestion and derivative pipelines must now handle video transcoding (`.mp4` clips for Live Photos).
