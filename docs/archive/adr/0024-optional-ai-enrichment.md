---
id: "0024"
title: "Optional OCR and AI Enrichment"
status: "accepted"
date: "2026-07-16"
decisions: []
related: []
supersedes: []
---

# ADR 0024: Optional OCR and AI Enrichment

## Context

Ticket and receipt images can contain stations, operators, fares, dates, and vendor names. OCR or a vision model could reduce authoring work, but it introduces provider cost, secrets, privacy
concerns, and uncertain output.

Felicia must remain useful without an AI subscription, network access, or configured provider.

## Decision

OCR and AI are optional enrichment providers. They produce suggestions for a draft; they are not a required import dependency or a publication gate.

```text
media/candidate → enrichment request → suggestions + confidence + provenance
→ author confirms/edits → authored field patch
```

Rules:

1. Manual entry and publishing work with no enrichment provider.
2. Providers may be local OCR, a user-supplied hosted API key, or a future Felicia service. Core depends on an enrichment interface, not a vendor SDK.
3. Suggestions are separate from authored values and include provider, model/version when available, request time, confidence, and source asset hash.
4. Suggestions never overwrite authored fields or publish automatically.
5. Secrets, full provider requests, and sensitive image payloads never enter normal logs or public API responses.
6. The author can accept, edit, reject, or ignore each suggestion.
7. The first useful capability is structured extraction for `transit` and `receipt`; free-form essay generation is deferred.

## Consequences

- Core import and template generation remain subscription-free.
- AI can be added without changing the memento model or ZIP format.
- The admin UI needs suggestion review and provenance display.
- Provider cost and privacy remain explicit author choices.

## Deferred

- Default hosted provider.
- Local OCR engine and packaging.
- Essay generation, translation, and autonomous publishing.
