---
id: "0004"
title: "Japanese-First i18n"
status: "accepted"
date: "2026-07-01"
decisions:
  - "felicia:decision:jp-first-i18n"
related: []
supersedes: []
---

# ADR 0004: Japanese-First i18n

## Context

Travel logs in Japan are heavily tied to local terms (e.g. Japanese station names, ticket operators, stamps, and location tags). A plain English-only codebase would fail to capture the authentic feeling of the physical stubs. Furthermore, the author is bilingual/trilingual, requiring Japanese, English, and Chinese support from day one.

## Decision

We decided to treat **Japanese (`ja`) as the authored canonical language** for the application database, with English (`en`) and Chinese (`zh`) stored in a translation sidecar table.

Implementation details:

1. **Inline Japanese:** All core text fields on primary tables (e.g., `mementos.title`, `journeys.title`, `memento_photos.caption`) store Japanese directly. This ensures that the default Japanese render requires zero table joins.
2. **No Translation for User-Authored Content:** User-authored essays (`mementos.essay`) and photo captions are **not** translated; they remain solely in the primary Japanese language. i18n only applies to structural fields (e.g., transit station names, line names, and operators).
3. **Translation Sidecar:** A dedicated `translations` table stores non-primary translations (`en`, `zh`) keyed by `(owner_type, owner_id, lang, field)`.
4. **No-Clobber Provenance:** Each translation row carries a `provenance` field (`machine` or `authored`).
   - When importing or auto-translating, the system only overwrites translations marked `machine`.
   - Once a human hand-corrects a translation, the system updates its provenance to `authored` and never clobbers it on subsequent data syncs.
5. **Fallback Rules:** If a requested translation is missing, the client or API falls back to the inline Japanese value.

## Consequences

- High performance for the primary `ja` localization.
- Re-translation runs are completely safe and do not overwrite manual proofreading work.
- Slight write amplification on imports as translation rows are seeded.
