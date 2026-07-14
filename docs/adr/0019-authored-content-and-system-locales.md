# ADR 0019: Authored Content and System Locales

- **Status:** Accepted
- **Date:** 2026-07-14
- **Supersedes:** ADR 0011 and the translatable-field portion of ADR 0006

## Context

Felicia has a multilingual interface, but the journal is a personal record.
Automatically translating titles, essays, captions, ticket metadata, links, or
media descriptions changes the author's voice and creates a second write model
that must be reconciled with authored content.

## Decision

Remove translation persistence and translation API fields entirely. Journey,
memento, photo, and media content is stored and rendered exactly as authored.

System-owned interface strings use static, typed frontend locale catalogs for
Japanese, English, and Chinese. Locale selection affects navigation, controls,
state labels, and errors only; it never rewrites API content.

The API exposes stable error codes and canonical authored values. The frontend
chooses the system locale using an explicit user preference, then browser
language, then Japanese as the fallback.

## Consequences

- The write model has no translation sidecar or translation provenance.
- Locale coverage is testable as a catalog completeness problem.
- User-authored content has one source of truth and no language-axis merge rules.
- Deliberately authored alternate-language content, if ever needed, must be a
  new explicit content model decision rather than an inferred translation.
