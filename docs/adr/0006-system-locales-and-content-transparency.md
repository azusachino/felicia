---
id: "0006"
title: "System Locales and Content Transparency"
status: "accepted"
date: "2026-10-01"
related:
  - "0001"
  - "0003"
---

# ADR 0006: System Locales and Content Transparency

## Context

Felicia's visual style and cultural framing originate in Japanese travel literature and stationery (*techo*, *memento stubs*, train journeys). However, readers may use English or Chinese browsers, while authored essays may be written in any language.

An earlier draft considered automated machine translation for authored stories, which created complex sidecars and produced awkward, lossy translations of personal memories.

## Decision

**Support Japanese, English, and Chinese for system chrome, while rendering authored content exactly as written.**

1. **System Locale Catalogs:**
   - Chrome elements (buttons, navigation landmarks, dates, kind labels, accessible descriptions) are translated via typed static catalogs for Japanese (`ja`, default), English (`en`), and Simplified Chinese (`zh`).
   - Browser preferences (`navigator.language`) set the default locale, persisted in `localStorage`.
2. **Transparent Authored Content:**
   - Essays, captions, notes, and titles carry no translation sidecars and are never run through automatic machine translation.
   - The journal presents the author's exact words in whatever language they were authored.

## Consequences

### Positive

- System interfaces are accessible and intuitive for multilingual readers.
- Eliminates brittle translation synchronization, AI translation dependencies, and corrupted voice.

### Negative / Trade-offs

- A monolingual visitor must read authored stories in the original language chosen by the author.
