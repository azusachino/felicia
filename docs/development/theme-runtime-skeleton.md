# Theme runtime skeleton

This document is the working implementation specification for [ADR-0036](../adr/0036-theme-runtime-and-scene-actions.md).
The ADR is the decision record; this page describes the first file boundary and
the order in which it should become real behavior.

The first concrete fixture is the [Tabi Izu storyboard](tabi-izu-first-storyboard.md).

## Objective

Give multiple Felicia themes one renderer-neutral vocabulary for journey scenes,
stops, events, and semantic actions while leaving visual interpretation to each
theme. The first consumer is the incubating Tabi tabletop diorama.

The first slice is deliberately a skeleton. It proves names, ownership, and
dependency direction. It does not register Tabi, add a Three.js dependency, or
replace the existing map-based themes.

## Package boundary

All of this work starts inside `packages/felicia-shared`. Hosts continue to
load the existing named registry. The shared package owns public reader
contracts and theme composition; hosts own bootstrapping, transport, and
authoring concerns.

Split packages only after at least one of these is demonstrated:

- a consumer needs the runtime without the reader components;
- a renderer needs a dependency set that the other themes should not install;
- the runtime and themes need independent release or compatibility policy.

## Skeleton layout

```text
packages/felicia-shared/src/theme-ui/
  runtime/
    actions.ts       # semantic action vocabulary
    events.ts        # journey event vocabulary
    scene.ts         # route/stops/scene model
    timeline.ts      # phase and playback state
    theme.ts         # theme capability metadata
  components/
    contracts.ts     # reusable board, stop, artifact, and archive inputs
  renderers/
    renderer.ts      # renderer-neutral mount/render/action seam
  _incubating/tabi/
    contract.ts      # Tabi manifest and theme-owned vocabulary
    composite.ts     # Tabi composition input, still unregistered
```

The files are contracts first. A concrete Svelte component or renderer is
added only when a visual slice needs it.

## Domain conventions

### Route truth

`JourneyScene.route[0]` and `JourneyScene.route.at(-1)` are the source route
endpoints. A theme may label the first endpoint `home` only when the journey
data and authored copy support that reading. It must not invent a geographic
return leg. A return-home phase means returning to the source endpoint or
authored home state, not silently modifying the GPX route.

### Stops and mementos

Stops identify authored visits. Mementos remain the existing data model and are
attached to a stop by `visitId`. A theme may turn a memento into an artifact,
memory card, ticket, or another visual object, but it does not change the
public memento contract.

### Events and actions

Events express what happened in the journey narrative. Semantic actions express
what a theme can animate or present. For example:

```text
arrive(stop-2) -> character: arrive + camera: focus-stop
reveal(memento-7) -> artifact: reveal + panel: open
inspect(memento-7) -> character: idle + memory: inspect
return-home -> character: travel + board: restore-home
```

The mapping belongs to the theme. The shared layer supplies the stable terms,
not a universal animation clip list.

## Implementation tracks

1. **Design:** approve the visual direction, protagonist sheet, palette,
   camera framing, stop-event grammar, and the first real GPX storyboard.
2. **Detail:** turn the approved direction into theme-owned models, action
   mappings, authored stop metadata, and reusable component props.
3. **Implementation:** add one thin vertical slice — load a real journey,
   travel to one stop, reveal one memento, inspect it, and return to the
   destination/home state.
4. **Composite:** compose the board, route, character, stop markers, artifact
   reveal, memory panel, and home archive without changing the runtime meaning.

Every track is verified independently. A renderer demo is not evidence that
route truth or event semantics are correct.

## Verification

The skeleton is complete when:

- `make web-check` passes;
- the shared package type-checks the new contracts;
- Tabi remains absent from `theme-ui/registry.ts`;
- no new renderer dependency is required;
- the existing `cartography`, `cabinet`, `techo`, and `atlas` themes remain
  unchanged.

Later vertical slices must add pure contract tests for route endpoints,
memento-to-stop association, event ordering, and the semantic-action mapping
before browser animation tests.

## Open design questions

- What is the authored representation for a stop's event sequence?
- Which protagonist model and palette survive the first design review?
- Does the first renderer use Three.js directly or a smaller scene abstraction?
- Which parts of the home archive are reusable across a second theme?
