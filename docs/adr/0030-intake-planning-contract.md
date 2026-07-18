---
id: "0030"
title: "Intake Planning Contract and Candidate Review Boundary"
status: "proposed"
date: "2026-07-18"
decisions:
  - "felicia:decision:intake-candidate-boundary"
  - "felicia:decision:intake-source-capabilities"
related:
  - "0022"
  - "0024"
---

# ADR 0030: Intake Planning Contract and Candidate Review Boundary

## Context

Felicia must normalize Dawarich/Immich records, local GPX/photos, and portable
packages into one authoring workflow. A route or visit is evidence, not an
authored story. The importer also needs durable review state so re-import does
not resurrect ignored candidates or overwrite authored fields.

## Decision

The intake model has four distinct layers:

```text
Route / Visit / MediaAsset
    -> StopCandidate
    -> MementoCandidate
    -> Memento
```

`StopCandidate` is private draft state. It is not a public places table. It
contains a stable candidate identity, evidence references, confidence, source
provenance, and review state (`proposed`, `kept`, `ignored`, or `merged`). A
memento candidate may refer to a stop through its stable `StopKey`; persistence
can later resolve that key to a local UUID.

Evidence references require a source identity, evidence kind, and source-local
locator. This makes candidate explanations and re-import reconciliation
observable rather than relying on matching coordinates alone.

Source capabilities are split into independent ports:

```text
RouteSource -> Dawarich, local GPX
VisitSource -> Dawarich, local visit derivation fallback
PhotoSource -> Immich, local photo directory
```

`TrackSource` remains as a compatibility composite while runtime callers
migrate to the narrower capabilities. When both connected and local sources
are present, the runtime must record the conflict and apply an explicit source
precedence policy; it must not silently merge contradictory semantics.

## Consequences

- The pure planner can operate with routes, optional visits, and optional media
  without knowing a provider or database.
- Candidate review requires a private persistence seam in addition to source
  observation history.
- Public publication remains memento- and privacy-based; candidate state and
  raw source evidence stay out of the public contract.
- Candidate identity and evidence schemas must be versioned before clustering
  or CLI commands are implemented.
- A future migration can replace the compatibility `TrackSource` without
  changing Dawarich's or Immich's normalized domain values.

## Pushback retained

- Derived stop identity can change when the clustering algorithm changes. The
  derivation version is therefore part of the candidate identity, and a plan
  must report when a new version invalidates prior matches.
- A photo-rich location is not automatically an attraction. Candidate ranking
  remains a review aid, never an authoring decision.
- A missing source capability is not always an import failure. The planner must
  distinguish unavailable optional evidence from invalid required input.
