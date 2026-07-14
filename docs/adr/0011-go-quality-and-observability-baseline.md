# ADR 0011: Go Quality and Observability Baseline

- **Status:** Accepted
- **Date:** 2026-07-13
- **Decisions:** `felicia:decision:go-quality-observability`

## Context

The canonical data layer will sit on write boundaries and source adapters,
where small inconsistencies can become duplicated records or lost authorship.
The project also runs as a long-lived local service, so failures in imports,
database writes, and upstream calls need useful operational signals without
logging secrets or high-cardinality payloads.

## Decision

All backend changes follow four quality gates:

1. **Go style:** `gofmt` and `goimports`, small interfaces at seams, wrapped
   errors with context, and the Uber Go Style Guide as the review baseline.
2. **Static analysis:** `golangci-lint` with the repository's versioned v2
   configuration. `make check` is the required pre-commit gate.
3. **Structured logging:** use `log/slog`; log operation, source system,
   observation kind, run ID, outcome, and duration where useful. Never log API
   keys, tokens, raw payloads, essays, or precise private coordinates.
4. **Metrics:** instrument source fetches, canonicalization, ingest patches,
   repository writes, validation failures, and cache behavior. Metric labels
   must be bounded (source system, operation, outcome, and coarse kind only);
   external IDs, UUIDs, URLs, and error strings are values, not labels.

Use OpenTelemetry-compatible metric names and APIs at the application seams;
keep the exporter and SDK wiring outside the domain package. OpenTelemetry
metrics and traces are the preferred community-standard interoperability path,
while application logs remain `slog` until the OTel logs signal is mature enough
to justify a migration.

## Consequences

- Every future connector and write operation has the same review and diagnostic
  expectations.
- Unit tests cover canonicalization and write semantics; feature tests exercise
  the seeded API contract through `uv run`.
- Metrics add a small amount of boundary code, but make failed or partial
  ingestion visible without exposing private travel data.
