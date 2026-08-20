---
id: "0037"
title: "Private Local Development Layout"
status: "accepted"
date: "2026-08-20"
related:
  - "0025"
  - "0035"
---

# ADR 0037: Private Local Development Layout

## Context

Felicia's local authoring and preview data is private and already belongs under
the gitignored `.felicia/` root. The old defaults repeated that fact in child
names such as `.felicia/local-journey` and `.felicia/local.sqlite`, which made
the active layout harder to read and encouraged commands to grow one-off local
prefixes.

## Decision

Use `.felicia/` as the only local/private namespace and give its children
semantic names:

```text
.felicia/
  felicia.sqlite       # default local authoring database
  workspaces/<slug>/   # editable journey workspaces
  media/               # local media store
  site/                # compiled local preview
```

Generated journey slugs use `journey-<digest>` when the author does not supply
one. The CLI may still be named `local_journey` because it describes the
workflow, and API routes or test fixtures may retain historical names when
changing them would alter a public or test contract. Active defaults and new
documentation must not introduce `.felicia/local-*` paths.

Existing private workspaces are migrated by moving them under
`.felicia/workspaces/`; data is not deleted or silently copied.

## Consequences

- A contributor can identify the private root once and then read child names as
  roles rather than storage qualifiers.
- `make dev`, `make admin`, and `make journey-local` share predictable defaults.
- Historical ADRs and feature-specific test fixtures retain their original
  names as provenance.
- A user who already has data under the old paths needs one explicit move or an
  explicit `DATABASE_PATH`/`--workspace` override during migration.
