# ADR 0015: Transactional Memento Aggregates with Revisions

* **Status:** Accepted
* **Date:** 2026-07-13
* **Related:** ADR 0010, `felicia:write-side-stability:task-6`

## Decision

Memento authoring uses a monotonic `revision` on the memento aggregate. An
authoring patch may provide `expected_revision`; PostgreSQL updates only when
that revision still matches and reports a write conflict otherwise. New rows
start at revision 1, and successful updates increment the revision.

The repository exposes a separate aggregate operation that opens one database
transaction, applies the memento patch, and writes child photos,
and commits only if every operation succeeds. A failed child write rolls back
the memento change as well.

## Consequences

* API clients receive HTTP 409 for stale authoring writes and must reload.
* Memento and media cannot be partially committed through the
  aggregate seam.
* Import and aggregate orchestration remain separate until broader workflow
  transaction boundaries are needed.
