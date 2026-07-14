# ADR 0010: Canonical Observations Between Sources and Writes

- **Status:** Accepted
- **Date:** 2026-07-13
- **Decisions:** `felicia:decision:canonical-data-layer`
- **Related:** `felicia:decision:go-quality-observability`

## Context

Felicia will accept data from more than Dawarich and Immich, and memento kinds
will continue to grow. Provider DTOs have different identities, timestamp
formats, location precision, and confidence semantics. At the same time, the
database must remain safe for re-import: an external observation must be
identifiable across runs without being allowed to overwrite authored fields.

A generic mapping or ETL language would move product-specific assembly into
configuration and make the system harder to reason about. The existing source
interfaces already provide a better seam, but their output needs an explicit
canonical envelope and provenance vocabulary.

## Decision

Introduce a small canonical observation layer between provider adapters and
the write side:

```text
provider DTO → source adapter → canonical observation → ingest patch → repository
```

The canonical layer owns these concepts:

| Concept            | Meaning                                                                                                 |
| ------------------ | ------------------------------------------------------------------------------------------------------- |
| `SourceIdentity`   | Stable `(system, external_id)` key for idempotent re-import                                             |
| `Observation`      | Envelope containing kind, source, observation time, confidence, and canonical payload                   |
| `Route`            | Time-bounded normalized track segment                                                                   |
| `Visit`            | Time-bounded normalized place candidate                                                                 |
| `MediaAsset`       | Normalized image, video, audio, document, link, or approved embed with optional coordinate and checksum |
| `MementoCandidate` | Normalized memento suggestion before authored fields and publication state are applied                  |
| `Provenance`       | Origin, observation time, and confidence metadata attached to canonical data                            |

Provider-specific DTOs, pagination, authentication, field names, and recovery
behavior remain inside the adapter packages. The canonical layer contains no
network or database code. External embeds carry a provider and approved URL;
raw arbitrary HTML is excluded.

Manual authoring is a first-class input path. A user can create a ticket-like
memento, choose a registered template, enter its fields, and attach media or
links without a source identity. Source identity is required only when an
external observation is being synchronized.

The lifecycle is:

```text
candidate → draft → authored → published → archived
```

Source adapters may create candidates. Only the authoring write path may move a
candidate into authored or published state; re-import may refresh source-owned
fields while preserving authored values. A manual ticket starts at draft and
does not need a source identity.

The write API exposes two distinct operations:

- `ManualMementoPatch` receives an explicit field mask derived by the authoring
  service and records those fields as authored.
- `IngestMementoPatch` receives a source-owned field mask and cannot add,
  remove, or replace authored ownership.

Neither operation accepts a caller-controlled `authored_fields` array. The
repository merges a patch with the current row before persisting it. Optimistic
concurrency and aggregate transactions are a follow-up task, so this first
seam is not yet the final concurrent-write protocol.

Essays remain Markdown in the first authoring slice. Rich block content is a
follow-up decision, with Portable Text and ProseMirror/Tiptap JSON investigated
as established structured-content options rather than storing arbitrary HTML.

Authorship is deliberately separate from provenance. Provenance answers
“where did this value come from?”; the write model answers “may this value be
changed?” through explicit ingest and authoring operations.

## Consequences

- A new source implements an adapter into canonical shapes rather than changing
  persistence or public projections.
- A new memento kind remains a template concern; it does not require a new
  source adapter or table.
- Source identity becomes the future idempotency key, replacing ambiguous
  nullable `source_ref` behavior.
- The abstraction is intentionally small. A third-source spike must validate
  it before adding generic orchestration or configurable mappings.
- Canonicalization and future ingest writes follow the Go quality and
  observability baseline in ADR 0011.
- Existing `SourceRef` fields remain a compatibility seam during migration;
  new write-side work should use `SourceIdentity` and `Provenance`.

## Rejected alternatives

- **Provider DTOs in the repository:** couples storage and domain behavior to
  external API versions.
- **Generic runtime mapping DSL:** encodes product assembly as configuration
  before there is evidence that the mappings genuinely rhyme.
- **One large universal event type:** loses compile-time meaning and encourages
  provider-specific payloads to leak into the core.
