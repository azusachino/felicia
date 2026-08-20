---
id: "0038"
title: "Named Frontend Package Boundaries"
status: "accepted"
date: "2026-08-20"
related:
  - "0034"
  - "0036"
---

# ADR 0038: Named Frontend Package Boundaries

## Context

`@felicia/shared` became a catch-all for model data, reader compositions,
reusable components, renderer contracts, and theme experiments. The name did
not tell a contributor what the package owned, and `theme-ui/` became a second
namespace hiding several intended reuse boundaries.

## Decision

Use flat packages named for their actual responsibility:

```text
packages/
  felicia-model/       @felicia/model
  felicia-runtime/     @felicia/runtime
  felicia-components/  @felicia/components
  felicia-renderers/   @felicia/renderers
  felicia-reader/      @felicia/reader
```

`felicia-reader` is the host-facing facade. It owns the named design registry,
existing reader compositions, localization, and styles. The other packages
are reusable boundaries and must not import reader hosts.

Tabi is an incubating theme, so it remains under:

```text
packages/felicia-reader/src/theme-ui/_incubating/tabi/
```

It becomes a package only when it is promoted to a supported reader design.
Infrastructure packages do not use `theme` in their names, and no new generic
`shared` package is introduced.

## Consequences

- Package names communicate ownership without requiring a `theme-ui` tour.
- The reader facade can compose the runtime, components, and renderers without
  making those packages depend on the facade.
- Existing `apps/felicia-runtime` remains the Go application module; the
  frontend package is distinguished by its `packages/` location and
  `@felicia/runtime` scope.
- Promoting Tabi later has a clear move from incubating reader code to its own
  package, without pretending that an unfinished theme is stable infrastructure.
