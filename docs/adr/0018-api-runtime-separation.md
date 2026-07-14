# ADR 0018: API Transport and Runtime Separation

- Status: Accepted
- Date: 2026-07-14

## Context

`internal/api/server.go` currently contains routing, request DTOs, validation
coordination, application orchestration, and provider-facing behavior. This
makes the write-side lifecycle difficult to exercise independently of HTTP.

## Decision

The `apiserver` module owns HTTP routes, transport DTOs, probes, metrics,
structured request logging, graceful shutdown, and dependency wiring. The
`runtime` module owns journey, memento, import, and publication use cases. The
runtime receives core contracts and never depends on HTTP or concrete storage.

## Consequences

HTTP tests can focus on status and representation while runtime tests exercise
workflows without a server. The initial extraction will require temporary
compatibility shims and careful contract tests; no API endpoint changes are
allowed unless separately specified.
