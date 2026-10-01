---
id: "0011"
title: "System-Only Internationalization"
status: "superseded"
date: "2026-07-11"
decisions:
  - "felicia:decision:system-only-i18n"
related: []
supersedes:
  - "ADR 0004"
---

# ADR 0011: System-Only Internationalization

## Context

Felicia needs a multilingual interface, but the travel journal itself is authored
content. Automatically translating or duplicating a user's titles, places,
essays, captions, and memento metadata changes the author's voice and creates
multiple competing versions of the same memory.

The earlier Japanese-first decision allowed translation sidecars for some
structural fields. That is too broad for the current product direction: the
system language and the user's content are separate concerns.

## Decision

Internationalization applies only to system-owned interface text:

1. Support Japanese (`ja`), English (`en`), and Chinese (`zh`) for navigation,
   controls, labels, counts, errors, help text, and accessibility text.
2. Keep user-authored journey and memento content exactly as entered. This
   includes titles, places, vendors, essays, photo captions, ticket text, and
   kind-specific metadata.
3. Do not automatically translate, normalize, or overwrite authored content
   when the system language changes.
4. The API may expose the authored content's language metadata, but clients must
   render the content as authored rather than manufacture a translated variant.
5. Existing translation infrastructure is reserved for system-owned vocabulary
   and future explicitly authored translations; it is not a requirement for
   every user-content field.

## Consequences

- The UI can switch languages without changing the journal's voice.
- Fixtures and seeds need one authored content representation, reducing
  duplication and drift.
- Clients need a system UI dictionary with a fallback order, independent of
  journey and memento data.
- Deliberately authored translations may be added later as content, but they
  are never inferred merely from the selected UI language.
