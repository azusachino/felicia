# ADR 0013: Lifecycle-Aware Write-Boundary Validation

* **Status:** Accepted
* **Date:** 2026-07-13
* **Related:** ADR 0010, `felicia:write-side-stability:task-4`

## Decision

Validation is split between pure domain rules and the HTTP boundary. The
domain validates template data, lifecycle state, IANA occurrence timezone, and
geometry independently from `kind_data`. Drafts may be incomplete while being
edited; authored and published mementos must satisfy required template fields,
anchor rules, valid coordinate ranges, and geometry shape.

The API parses input into typed values and rejects malformed geometry rather
than coercing it to a zero coordinate. The domain remains free of HTTP and
database concerns, so importers can reuse the same checks before an ingest
patch.

## Consequences

* Incomplete authoring work can be saved as a draft without weakening checks
  on complete records.
* Invalid coordinates and timezone identifiers fail before persistence.
* Rich media/embed policy and template version negotiation remain separate
  concerns until those payloads have a concrete write endpoint.
