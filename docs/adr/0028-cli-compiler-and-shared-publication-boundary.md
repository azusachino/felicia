---
id: "0028"
title: "CLI Compiler and Shared Publication Boundary"
status: "proposed"
date: "2026-07-17"
decisions:
  - "Use the existing canonical domain model as the source of truth for both server and CLI workflows."
  - "Make package import, publication, and storage seams provider-neutral before adding server wiring."
  - "Introduce felicia-cli as a separate Go composition root in the existing go.work workspace."
  - "Move the reusable public projection/compiler boundary out of server."
related:
  - "0023"
  - "0025"
  - "0027"
supersedes: []
---

# ADR 0028: CLI Compiler and Shared Publication Boundary

## Context

The current repository already contains most of the canonical model, SQLite and
PostgreSQL providers, importer use cases, and a public JSON projection. The
composition is incomplete, however:

- `server/cmd/build` constructs PostgreSQL directly;
- the reusable projection package is now under `publication`;
- no `felicia-cli` composition root exists;
- the Python scripts build a checked-in design fixture, not a user package;
- the portable ZIP shape is specified by ADR 0023 but is not yet implemented.

The next milestone must prove the portable journey path with local inputs before
server ingestion or remote providers are wired into it.

## Proposed boundary

```text
core         domain entities, validation, templates, provider-neutral ports
runtime      import planning, no-clobber merge, publication use cases
publication  public JSON contract, visibility filtering, static compiler ports
providers    SQLite, PostgreSQL, local filesystem, and later S3-compatible implementations
cli               felicia-cli composition root
server    postponed server composition root using the same seams
```

`cli` may depend on `core`, `runtime`, `publication`, and selected providers.
Those packages must not depend on the CLI or HTTP server. The server can be
added later without changing package or publication contracts.

## Data layers

Felicia has three related but distinct representations:

1. **Canonical store model** — `Journal`, `Journey`, `Memento`, `MementoPhoto`,
   `TransitLeg`, and source observations in SQLite or PostgreSQL.
2. **Portable package model** — a versioned ZIP containing `manifest.yaml`,
   `journey.yaml`, optional `route.gpx`, `timeline.json`, `visits.json`,
   `mementos.yaml`, notes, and media. This is an import transport, not a database
   dump.
3. **Public publication model** — filtered `.json` projections and public media
   derivatives. It contains published content only and never exposes drafts,
   source payloads, private originals, or database files.

The CLI pipeline is therefore:

```text
ZIP/package files -> validate and plan -> canonical SQLite writes
  -> public projection query -> static JSON + media artifact
```

The first implementation may use a package directory fixture for fast tests, but
the accepted transport boundary remains the ZIP defined by ADR 0023.

## Abstract APIs

The exact Go names may evolve during TDD, but the seams must express these
responsibilities without importing a concrete database or HTTP package:

```go
type PackageReader interface {
    Read(ctx context.Context, source PackageSource) (*Package, error)
}

type PackageValidator interface {
    Validate(ctx context.Context, pkg *Package) (ValidationReport, error)
}

type ImportPlanner interface {
    Plan(ctx context.Context, pkg *Package, store ReadModel) (ImportPlan, error)
}

type ImportApplier interface {
    Apply(ctx context.Context, plan ImportPlan, store WriteModel) (ImportReport, error)
}

type PublicationCompiler interface {
    Compile(ctx context.Context, input PublicationInput, output ArtifactWriter) (BuildReport, error)
}
```

The package reader owns ZIP/path/checksum safety. Route and media adapters expose
normalized inputs to runtime. The importer owns no-clobber semantics. The
publication compiler owns visibility filtering, deterministic JSON paths, and
safe media copying. Composition roots choose SQLite versus PostgreSQL and local
filesystem versus S3-compatible blob storage.

## CLI surface

The initial real binary is `felicia-cli`:

```text
felicia-cli package validate journey.zip
felicia-cli import journey.zip --db felicia.sqlite --dry-run
felicia-cli import journey.zip --db felicia.sqlite --apply
felicia-cli static compile --db felicia.sqlite --media-root media --out dist
felicia-cli publish --db felicia.sqlite --media-root media --out dist
```

`publish` is explicit. `validate`, `import --dry-run`, and `static compile` must
be safe to repeat and must not contact a server. The CLI must support
machine-readable reports and must never log photo contents, package secrets, or
private source payloads.

## Consequences

- SQLite becomes the first complete local CLI path.
- PostgreSQL and the server remain supported by composing the same interfaces.
- The existing Python fixture scripts remain design-demo tooling only.
- The existing PostgreSQL compiler must be retired or rewritten behind the
  shared publication compiler before it is production code.
- GPX parsing, local media copying, checksum validation, and arbitrary `kind_data`
  preservation become testable without a running server.

## Open decisions

- Whether `publication` is a new Go module or a package first moved into
  `runtime`.
- Whether package YAML is decoded directly into import DTOs or first normalized
  into a versioned intermediate representation.
- Whether static compilation writes media directly to `dist/media/` or delegates
  to a blob-store adapter that exposes public derivatives.
