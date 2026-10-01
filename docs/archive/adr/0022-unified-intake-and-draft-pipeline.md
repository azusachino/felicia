---
id: "0022"
title: "Unified Intake and Draft Pipeline"
status: "accepted"
date: "2026-07-16"
decisions: []
related: []
supersedes: []
---

# ADR 0022: Unified Intake and Draft Pipeline

## Context

Felicia needs two practical ways to bring a journey into the product:

1. Pull route/visit data from connected services such as Dawarich and media from Immich.
2. Import a user-provided package containing route, timeline, photos, notes, and optional pre-authored memento data.

These are different transport mechanisms, not different domain models. Both must feed the same authoring, template, provenance, and publish rules.

## Decision

Felicia has one canonical journey/memento model and one write boundary. It accepts three input forms:

| Input form               | Examples                          | Role                                      |
| ------------------------ | --------------------------------- | ----------------------------------------- |
| Connected source adapter | Dawarich, Immich                  | Pull and normalize external observations  |
| Portable package adapter | Felicia ZIP, GPX + photos + notes | Normalize user-provided records and files |
| Direct authoring         | Admin form, manual memento        | Write intentional authored values         |

The first two are **intake modes**. Direct authoring is not a second data model; it is the explicit author operation applied to an imported draft or new journey.

Every intake mode follows the same pipeline:

```text
intake adapter → canonical observations → identity/deduplication
→ draft journey + candidate visits/mementos/media
→ author review and authored patches → preview → explicit publish
```

Rules:

1. An intake run always produces a draft or updates an existing draft. It never publishes directly.
2. Connected observations use `SourceIdentity{system, external_id}`. Package records use a stable package namespace and record ID, for example `felicia-package:<package_id>:<record_id>`.
3. Source-owned fields remain governed by the ingest field mask. When the author edits a field, the authored field mask wins and future intake runs preserve it.
4. Package values are import input until the author confirms them through the same authoring/publish boundary.
5. Missing, ambiguous, unmatched, and conflicting records remain visible as review items. The importer does not invent confident place or story data.
6. Media uses the common media pipeline. Content hashes and source identities make repeated imports idempotent.
7. Dawarich visits are consumed when available; GPX/package data uses the local visit derivation fallback from ADR 0005.

The minimum user workflow is:

```text
New journey → import package or connect sources → review draft → edit spots
→ choose memento templates → preview public view → publish
```

## Consequences

- Connected providers and packages share validation, provenance, deduplication, authoring, preview, and publish behavior.
- A user can begin with a ZIP and later connect Dawarich or Immich without changing the journey model.
- Import failures become recoverable draft problems rather than partial public writes.
- The admin UI needs an import review surface, not only CRUD forms.
- Import adapters remain separate from the presentation-agnostic public API.

## Open details to settle during M0/M1

- Which Google Timeline export formats receive first-class adapters.
- The exact package manifest and versioning rules in ADR 0023.
- How unmatched photos, visits, and route segments are grouped for review.
- Whether an author can merge two imported journeys or only append to one draft.
- The public privacy policy for route precision and imported originals.
