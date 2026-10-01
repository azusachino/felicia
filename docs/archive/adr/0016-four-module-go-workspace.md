---
id: "0016"
title: "Go Workspace Module Boundaries"
status: "accepted"
date: "2026-07-14"
decisions: []
related: []
supersedes: []
---

# ADR 0016: Four-Module Go Workspace Boundaries

## Context

Felicia currently combines HTTP transport, application workflows, domain
contracts, and infrastructure adapters in one Go module. The API server has
become difficult to review, while SQLite-first local operation and optional
PostgreSQL support require an explicit seam between application behavior and
storage/provider implementations.

## Decision

Felicia will use a committed Go workspace containing backend modules:

- `core`: domain entities, validation, errors, and provider contracts.
- `runtime`: application use cases and workflows; it depends only on `core`.
- `providers`: SQLite, PostgreSQL, external-source, and object-storage adapters.
- `publication`: public read projections and static compiler contracts.
- `server`: HTTP transport, operational concerns, and composition wiring.

The dependency direction is `core <- runtime`, `core <- providers`,
`core <- publication`, and `core + runtime + providers + publication <- server`.
Runtime code must not import a concrete provider. Cross-module packages must not
be placed under another module's `internal/` directory.

## Consequences

The boundaries make provider substitution and API review explicit, and each
module can have focused tests. The repository incurs additional `go.mod`,
workspace, and CI maintenance. The existing root module remains during the
incremental migration and is removed only after all packages have moved and
the workspace checks replace its checks.
