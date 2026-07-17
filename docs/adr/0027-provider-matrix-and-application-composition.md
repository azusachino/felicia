---
id: "0027"
title: "Provider Matrix and Application Composition"
status: "accepted"
date: "2026-07-17"
decisions:
  - "Keep SQLite as the default local database and support PostgreSQL as the server database provider."
  - "Treat Valkey as optional cache/infrastructure, never as canonical storage or a correctness dependency."
  - "Compose selected providers at executable boundaries rather than from the domain or transport layer."
related:
  - "0016"
  - "0017"
  - "0018"
  - "0025"
  - "0026"
supersedes: []
---

# ADR 0027: Provider Matrix and Application Composition

## Context

Felicia must serve both a low-friction local installation and a more capable
self-hosted server. SQLite is appropriate for a single author and static builds;
PostgreSQL is appropriate when a deployment wants stronger operational tooling and
the existing spatial database path. Valkey can improve server caching and future
job/session workflows, but Felicia must remain correct when it is unavailable.

The repository now has five backend Go modules—core, runtime, providers,
publication, and server—and both SQLite and PostgreSQL providers. The static compiler currently
selects PostgreSQL directly, which prevents it from being a true static mode.

## Decision

The architecture is layered as follows:

```text
core       domain entities, validation, and provider ports
runtime    import, authoring, review, media, and publication use cases
providers  SQLite, PostgreSQL, local blob, S3, Valkey, and source adapters
composition selected provider wiring and application construction
delivery   CLI, HTTP server, and static publisher
```

The canonical provider matrix is:

| Capability      | Default/local    | Self-hosted option                | Required in static mode   |
| --------------- | ---------------- | --------------------------------- | ------------------------- |
| Database        | SQLite           | PostgreSQL                        | SQLite or published input |
| Media blobs     | Local filesystem | S3-compatible or local filesystem | Public derivatives        |
| Cache           | None             | Valkey                            | None                      |
| Public delivery | Static files     | HTTP server or static export      | Static files              |

Composition happens in the CLI, server, or publisher entrypoint. The domain and
runtime packages receive narrow interfaces and do not select concrete drivers.

Valkey is an optional cache. Cache misses, cache outages, and cache eviction must
not alter authored data, publication correctness, import idempotency, or public
read semantics.

The delivery targets are separated from the module responsibilities:

```text
felicia serve  → HTTP admin/API and optional public reader
felicia build  → static HTML/JSON/media projection
felicia import → source/package intake and review report
felicia export → portable source and authored-data backup
```

The existing `server` module may continue to host HTTP transport during the
transition, but static build logic must move behind provider-neutral ports and must
not instantiate the PostgreSQL provider itself.

## Consequences

- SQLite remains the fastest path for contributors and personal users.
- PostgreSQL remains a supported server deployment, not a second domain model.
- Valkey can be added without making the local or static experience heavier.
- Provider contract tests must run against both database implementations.
- Executable wiring becomes the explicit place to choose the deployment profile.
- A future CLI or desktop wrapper can reuse runtime logic without importing HTTP
  transport packages.

## Deferred

- A separate `common` Go module; shared code must first prove it belongs in core,
  runtime, or a concrete provider package.
- Background job infrastructure beyond the optional cache boundary.
- Multi-tenant database/provider selection.
