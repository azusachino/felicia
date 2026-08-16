---
id: "0035"
title: "Package Import Reuses the Write Boundary and Carries Edge Geometry"
status: "accepted"
date: "2026-08-16"
related:
  - "0013"
  - "0022"
  - "0023"
  - "0033"
  - "0034"
---

# ADR 0035: Package Import Reuses the Write Boundary and Carries Edge Geometry

## Context

[ADR-0013](0013-write-boundary-validation.md) split validation into pure domain
rules and an HTTP boundary so that "importers can reuse the same checks before an
ingest patch". [ADR-0023](0023-portable-journey-package.md) went further and
promised that package import would "validate the ZIP before writing: … coordinate
ranges, timestamps, and template fields".

Neither happened (issue #77). `domain.ValidateForState`,
`ValidateOccurredTimezone`, and `ValidateMementoGeometry` each had exactly one
caller — the admin API. `normalizeMemento` in `runtime/importer` checked UUID and
RFC3339 parsing, a non-empty `kind`, and coordinate range, and never consulted the
kind-template registry at all. So a package could seed a memento the admin GUI
would refuse to save for the rest of its life: an unregistered or misspelled kind
imports cleanly and then answers 400 `kind_not_registered` on every save.

The sharpest form of it was a whole kind that could not exist:

```text
core/kinds/transit.yaml   anchor: edge   ->  >=2-point LineString (core/domain/validate.go)
mementos.yaml `geom`      [lon, lat]     ->  one point, always
```

No CLI-imported memento could legally be `transit`. The importer accepted it, the
compiler published it, and the GUI rejected every later save with
`anchor_mismatch`. Adding the validator calls alone would not have fixed that — it
would have converted a silent corruption into a loud rejection and left `transit`
impossible through the CLI. Two things were needed: the checks, and a format that
can express what the registry declares.

## Decision

**1. `DecodePackage` is the import path's write boundary.** For every memento it
runs the same registry lookup and the same three domain validators the admin API
runs before a save: the kind must be registered, `kind_data` must satisfy its
template for the memento's lifecycle state, `occurred_tz` must be a usable IANA
zone, and the geometry must match the kind's `anchor`. Failures name the kind and
the same machine codes the API returns (`kind_not_registered`, `unknown_field`,
`required_missing`, `invalid_timezone`, `anchor_mismatch`, …), so a package is
diagnosed in the vocabulary the GUI would have used.

The checks live in `DecodePackage`, not `ApplyPackage`. `ApplyPackage` persists
stop candidates in a loop that runs before the memento loop
([ADR-0034](0034-packages-transport-stop-candidates.md)) and runs in no
transaction (issue #76), so a memento rejected there would leave written stops
behind. Validating at decode makes a package all-or-nothing without a transaction:
a rejected package never reaches a store.

**2. A memento's `geom` expresses both anchors through one key.**

```yaml
- kind: goods # anchor: point
  geom: [135.7681, 35.0116] # a [longitude, latitude] pair
- kind: transit # anchor: edge
  geom: # two or more such pairs
    - [135.7681, 35.0116]
    - [135.5023, 34.7025]
```

The element type decides the shape (numbers = one point, sequences = a line); an
absent, null, or empty value means "no geometry yet". The format does not decide
which shape is _legal_ — `domain.ValidateMementoGeometry` checks it against the
kind's anchor, exactly as the API does, so a point on `transit` and a line on
`goods` are both rejected with `anchor_mismatch`. Every package written before
this change still decodes unchanged, because a two-number array still means a
point.

`schemas/local-authoring-v1.schema.json` mirrors this for the editable workspace
(`coordinate` or `coordinate_line` or null), and its memento `kind` enum is now
the whole registry, guarded by a drift test — a kind the registry declares but the
workspace forbids cannot be authored locally at all.

**3. An ingested `candidate` stays allowed to be incomplete.** A candidate is
"source-derived, awaiting authoring" (`docs/contracts/memento-lifecycle.md` §1),
which is the state the importer itself creates, so the import boundary holds it to
a draft's leniency: required template fields and geometry may be missing. Present
values are still type-checked, the field set is still closed, and the kind must
still be registered. `authored`, `published`, and `archived` records must be
complete, which is the ADR-0013 rule the API applies. Nothing in
`core/domain` changed; the leniency is one explicit mapping in the importer.

## Consequences

- A `transit` memento is possible through the CLI end to end: workspace → package
  → import → SQLite `LineString` → publish, and it then passes the admin API's own
  validators, so it can be authored in the GUI afterwards.
- Packages carrying registry-invalid `kind_data` now fail at
  `felicia-cli package validate` / `import` instead of corrupting the journal. The
  checked-in mixed-state fixture and the Pages preview examples were exactly such
  packages and were corrected; that they were wrong for months is the argument for
  the check.
- A published memento can no longer omit its geometry or timezone through the
  CLI, because the GUI cannot save one either.
- The importer now loads the embedded template registry (`core/kinds/*.yaml`, the
  same data `server/cmd/api` loads). The registry is parsed once and is read-only.
- Import failure is per-package, not per-memento: one bad memento rejects the
  whole package. That is the price of "persists nothing" while issue #76 leaves
  the apply untransacted, and it keeps re-import safe.

## Rejected alternatives

- **A separate `geom_line` key.** Statically typed and unambiguous, but it gives
  one domain field (`Memento.Geom`) two package keys, invents an invalid state
  (both set), and forces every producer and schema to track which key a kind's
  anchor implies — the same "the format and the registry disagree" class of bug
  this ADR closes.
- **A GeoJSON object (`{type, coordinates}`)** as the admin API's wire format
  uses. Self-describing, but it breaks every existing package and hand-written
  fixture, and it lets a package _declare_ a geometry type that contradicts the
  registry, which then has to be reconciled. The anchor is already the single
  declaration of shape.
- **Validating in `ApplyPackage`.** Closer to the write, but stop candidates are
  persisted before the memento loop and there is no transaction, so a rejection
  there leaves rows behind.
- **Making `transit` point-anchored, or relaxing the `>=2` rule.** That would
  discard the reason edges exist: a transit memento composes into the journey's
  display route (`docs/research/transit-tickets.md`). The format was wrong, not
  the registry.
- **Holding candidates to full completeness, as `ValidateForState` does for every
  non-draft state.** It would refuse intake itself — an ingested candidate is
  incomplete by definition — and either break the importer's own creation state or
  require changing a domain validator the admin API depends on.
