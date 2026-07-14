# ADR 0020: Retire the Transitional Root Go Module

- **Status:** Accepted
- **Date:** 2026-07-14

## Context

The four-module workspace was introduced before all production packages were
relocated. The root module still owns the executable in `cmd/api`, HTTP
transport in `internal/api`, configuration, source connectors, the PostgreSQL
provider, and the static build command. `apps/apiserver` is currently only a
module placeholder, while SQLite and runtime code already live under `apps/`.

This split is misleading: package ownership suggests the new architecture, but
the root module remains the actual application boundary.

## Decision

Treat the root Go module as transitional and retire it after the following
relocations:

1. Move the API executable, HTTP transport, configuration, and publication
   projection into `apps/apiserver`.
2. Move PostgreSQL/sqlc code and external source adapters into
   `apps/providers` beside SQLite.
3. Move the build command into the owning application module and update
   container, workflow, and UV-script entrypoints.
4. Move the embedded kind registry into `apps/core` so no application package
   imports the root module.
5. Remove the root `go.mod`, root `embed.go`, and root entrypoint directories;
   keep migrations and shared scripts as repository-level non-Go assets.

The relocation must preserve the public HTTP contract, SQLite-first local
workflow, optional PostgreSQL parity, and existing integration-test gates.

## Consequences

- A temporary compatibility commit may leave root packages during migration,
  but new application code must not be added there.
- The final `go.work` will list only the owned `apps/*` modules.
- sqlc generation and provider tests will have one clear owner under
  `apps/providers`.
- Removing root imports will make module boundaries enforceable by the Go tool,
  rather than conventions alone.
