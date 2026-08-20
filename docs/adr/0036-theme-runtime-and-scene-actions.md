---
id: "0036"
title: "Reader Runtime and Scene Actions"
status: "accepted"
date: "2026-08-20"
related:
  - "0003"
  - "0031"
  - "0034"
---

# ADR 0036: Reader Runtime and Scene Actions

## Context

Felicia will have several reader themes that tell different stories with the
same journey data. The first new direction is a tabletop diorama: a curated
character follows the real route from the GPX, arrives at authored stops,
reveals a memento or artifact, and eventually returns to the journey's actual
destination or home when the source route supports that interpretation.

The themes should share the meaning of a journey without sharing one visual
language. A theme may use Three.js, MapLibre, SVG, or another renderer, but a
renderer choice must not become the data model or the event choreography.

## Decision

Use flat, actual package boundaries and keep Tabi nested under the reader's
incubating theme area until it is promoted:

```text
packages/
  felicia-model/           # reader data and public contracts
  felicia-runtime/         # renderer-neutral scene, event, action, timeline contracts
  felicia-components/      # reusable reader component contracts and implementations
  felicia-renderers/       # renderer seams; Three.js is a future implementation
  felicia-reader/
    src/theme-ui/          # named reader designs and styles
      _incubating/tabi/    # Tabi contracts and implementation before registry admission
```

The runtime package models:

- a `JourneyScene` with the true route origin, destination, route, and stops;
- semantic actions such as `travel`, `arrive`, `reveal`, `inspect`, `pickup`,
  `carry`, `place`, `celebrate`, and `return-home`;
- authored journey events that can be mapped to different visual actions;
- a timeline phase that makes automatic travel and manual inspection explicit.

The runtime does not model a generic game engine. It does not add dice,
scoring, combat, a speculative ECS, or a global event bus. Event meaning is
shared; the action mapping, assets, camera, layout, and choreography remain
owned by each theme.

Tabi remains incubating and unregistered until its full design rework is
approved. Its skeleton may define a manifest, model vocabulary, and action
vocabulary, but it must not expose the current renderer as an approved preview.

The package names describe their actual responsibility; none uses a generic
`shared` or `theme` suffix. Split further only when an independent consumer,
release cadence, or dependency-isolation requirement is real and documented.

## Consequences

- New themes have a known place for scene models, semantic actions, reusable
  component contracts, renderer adapters, and composite roots.
- Tests can verify route/event semantics without requiring a browser renderer.
- A theme can present the same `arrive` or `inspect` event as animation, a
  camera move, a card transition, or a static state.
- The first implementation still has to choose a concrete protagonist,
  palette, stop-authoring format, and renderer composition for Tabi.
- The runtime package gains a small amount of vocabulary before it has a second
  production theme; the contracts must therefore stay narrow and be revised
  when the next theme exercises them.

## Rejected alternatives

- **Put the entire Tabi implementation in the registry immediately:** this
  would make an unreviewed visual experiment part of the public theme contract.
- **Keep runtime, components, and renderer concerns inside the reader package:**
  this would make intended reuse boundaries cosmetic and force unrelated
  consumers through the reader facade.
- **Let each theme invent its own event names:** this would make cross-theme
  journey behavior impossible to test and compare.
- **Build a generic game engine:** this would solve speculative future mechanics
  while obscuring Felicia's small, authored journey interaction model.
