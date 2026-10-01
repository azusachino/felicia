---
id: "0003"
title: "Single Flagship Reader: Atlas"
status: "accepted"
date: "2026-10-01"
related:
  - "0001"
  - "0002"
---

# ADR 0003: Single Flagship Reader: Atlas

## Context

Felicia previously maintained four distinct reader themes (Atlas, Cabinet, Techo, Cartography) and an on-screen switcher with hash-based route overrides.

Audits revealed that maintaining four independent art directions created massive complexity: over 33 interface defects (contrast, hit areas, keyboard navigation, reduced motion) had to be solved repeatedly across four themes. Following Iroha's 2026-09-28 reset to "Grapher only", Felicia examined its reader maintenance burden.

## Decision

**Consolidate the reader into a single flagship implementation: Atlas.**

1. **Atlas as the Sole Reader:**
   - Atlas (`journey rail → world route map → collectible stubs → memento story dialog`) directly embodies the core product vision.
   - Secondary themes (Cabinet, Techo, Cartography, Tabi) are retired from active maintenance.
2. **Eliminate Theme Registry & Switcher:**
   - Delete dynamic theme registries and switcher navigation.
   - Export `<Reader />` directly from `@felicia/reader`.
3. **Hash Deep-Linking:**
   - Use URL hash anchors (`/#journey-<id>`, `/#memento-<id>`) for shareable deep links compatible with static hosting.

## Consequences

### Positive

- Subtracted over 4,200 lines of high-churn Svelte and CSS.
- Drastically reduced reader bundle size and build times.
- Focuses 100% of frontend polish, accessibility, and mobile responsiveness on one excellent experience.

### Negative / Trade-offs

- Alternative browsing modes (such as Cabinet's flat greatest-hits carousel or Cartography's full-map index) are retired from the public site.
