# tabi — incubating, full rework required

Parked here on 2026-08-20. The first cartoon/game cat-trip attempt was built
reactively: emoji → hand-drawn SVG cat → CSS filter for "cartoon" → MapLibre
style-layer stripping → pixel-sprite rebuild → chrome redesign, each step
landed only after the previous one turned out wrong in review. The mechanics
(D3 timers/interpolation/easing, the MapLibre runtime-style technique,
`d3-geo` for real distance) remain useful, but the visual direction was never
decided before implementation.

This directory contains only the replacement contract/composite skeleton. The
rejected renderer is intentionally not part of the skeleton PR. Tabi is not in
`registry.ts`; do not expose an implementation as a preview or treat it as the
approved design. The design reset, shared runtime skeleton, and Izu storyboard
are the inputs for the next deliberate rework.
