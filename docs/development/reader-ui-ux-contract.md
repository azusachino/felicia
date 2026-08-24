# Reader UI and UX contract

This is the shared contract for Felicia's four public reader designs: Atlas,
Cabinet, Techo, and Cartography. It keeps the reader coherent while allowing
each design to tell the same journey in its own visual language.

## Design read

Reading this as a redesign of a personal, map-first travel journal for one
reader, with a tactile editorial atlas language, leaning toward midnight blue,
apricot route light, mint signals, warm paper, and restrained cinematic motion.

The working dials are:

- Design variance: 7. The four designs should feel authored, not cloned, but
  share enough structure that the reader never loses orientation.
- Motion intensity: 6. Movement explains route, selection, and opening a
  memento. It never runs for decoration alone.
- Visual density: 4. The map and the memento carry the page; controls stay
  quiet and sparse.

## What every design must preserve

- The map is the index. A reader can see the active journey and reach the
  journey/memento index from the first screen.
- A memento is a physical paper object. Kind is expressed through material,
  silhouette, and typography rather than a colored badge.
- Opening a memento is one continuous action. The selected object grows into
  its detail surface where the layout allows it; a panel reveal is the fallback
  on narrow screens.
- Hash navigation remains stable: `/`, `#cabinet`, `#techo`, and
  `#cartography` remain shareable entry points.
- Authored copy, locale behavior, and the `{ journey, visit, memento }` reader
  contract do not change as part of visual work.

## Shared interaction grammar

Every design exposes the same semantic controls, even when their placement and
visual treatment differ:

| Intent                                 | Required behavior                                                                              |
| -------------------------------------- | ---------------------------------------------------------------------------------------------- |
| Choose a journey                       | Updates the active route and keeps the selected journey visible.                               |
| Inspect a memento                      | Opens its story and gallery, updates map/index selection, and exposes a labelled close action. |
| Change language                        | Preserves the current design and selected content.                                             |
| Change light/dark mode                 | Updates the shell and map together, with readable controls in both modes.                      |
| Change design                          | Uses the existing hash links and exposes the current design to assistive technology.           |
| Recover from loading/error/empty state | Uses a shaped status surface and an actionable retry or explanation.                           |

Keyboard focus is visible, click targets are at least 44px on touch surfaces,
and every map marker has a real button name. `Escape` closes an open detail
surface where the design owns the document focus.

## Visual system

The shell uses one midnight-to-mist neutral family and two signal colors. Dark
mode is a blue-green midnight rather than pure black; light mode is a cool mist
derived from the same hue. Apricot marks selected states and the Felicia route;
mint carries the quieter route network and map signals. Warm paper remains a
physical material for mementos, and red is reserved for actual error or
destructive state.

The public shell may use liquid-glass surfaces for navigation, indexes, and
detail panels: translucent layered fills, a hairline highlight, soft elevation,
and backdrop blur where supported. Glass is an interface layer, not a material
replacement for the paper memento system. The public mark is a native SVG
waypoint: a paper note crossing a two-tone route on a midnight field.

The shape scale is intentionally narrow:

- 2px for map markers and paper edges;
- 8px for controls and quiet content surfaces;
- 16px only for the main detail surface when elevation communicates hierarchy;
- full rounding only for circular markers and icon controls.

Typography uses the existing project fonts and adds hierarchy through measure,
weight, tracking, and ragged-right reading text. Do not add a new font or UI
library for this pass.

## Motion contract

Motion is stateful and interruptible. The reader uses the existing Svelte
transitions and CSS rather than introducing a second animation runtime.

- **Text reveal:** a short stagger for first-entry headings and empty states.
- **Panel reveal:** translate, opacity, and a small blur for detail surfaces.
- **Card resize:** ease the index/detail state rather than snapping widths.
- **Shared-element open:** preserve the current marker/stub ghost behavior where
  it exists; do not fake a morph with unrelated simultaneous fades.
- **Route emphasis:** use D3 geometry, interpolation, and easing for route
  progress or active-segment emphasis. Do not animate layout properties when a
  transform or opacity is sufficient.

All motion has a `prefers-reduced-motion: reduce` path that removes travel and
stagger while retaining the state change. Map camera movement must use zero
duration for reduced motion.

Three.js is not a shared reader dependency. The ThreeUI reference informs
scene composition, control density, and visual depth; a real Three.js renderer
belongs to the incubating Tabi scene described by ADR-0036 and should be added
only when that theme has an approved scene contract.

## Theme roles

- **Atlas:** default map-and-story reader. It owns the clearest route,
  journey-overview, and memento-open path.
- **Cabinet:** collection browse. It emphasizes discovery across journeys while
  retaining the same detail and control semantics.
- **Techo:** notebook interpretation. It may be warmer and more editorial, but
  its map remains an actionable index rather than a decorative spread.
- **Cartography:** full map atlas. It owns the richest per-kind paper materials
  and route geography, not a separate interaction model.

## Verification bar

Before a reader slice is complete:

1. `make web-check` passes from a clean dependency install.
2. Each hash loads with the production catalog and no console errors.
3. A keyboard-only pass can switch design, language, mode, journey, and
   memento, then close detail.
4. Narrow viewport behavior keeps the map/index affordance reachable and does
   not trap detail content off-screen.
5. Reduced-motion mode keeps all content and state changes usable without
   animation.
