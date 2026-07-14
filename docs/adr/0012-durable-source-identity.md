# ADR 0012: Durable Source Identity for Idempotent Ingest

- **Status:** Accepted
- **Date:** 2026-07-13
- **Related:** ADR 0010, `felicia:write-side-stability:task-3`

## Decision

Persist external identity on synchronized mementos as the pair
`(source_system, source_external_id)`. The pair is nullable for manual
mementos, constrained to be all-or-nothing, and protected by a unique partial
index. The existing `source_ref` remains as a compatibility/display seam.

During migration, legacy refs are split at the first colon: the first segment
becomes `source_system` and the remaining namespaced value becomes
`source_external_id`. Ingest resolves an existing memento by this durable pair
before falling back to the incoming local UUID. Therefore a source can
regenerate local IDs without creating duplicate mementos.

Manual mementos may omit source identity. Source identity is source-owned and
is never part of the manual authored field mask.

## Consequences

- Re-import idempotency no longer depends on provider-generated Felicia UUIDs.
- A source identity must be globally unique across journeys for memento-level
  synchronization.
- Legacy adapter refs remain readable while adapters migrate to
  `domain.SourceIdentity`.
- Multi-source reconciliation and identity collision policy remain follow-up
  work; a collision is currently rejected by the database constraint.
