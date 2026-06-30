# Research — Japanese-first i18n

> 2026-07-01. Decision: felicia supports at least Japanese, English, and Chinese, with
> Japanese as the primary/default language for the near-term MVP while the author is in
> Japan. Outcome ADR: `felicia:decision:jp-first-i18n`.

## Decision

Build the MVP with i18n as a first-class product constraint:

- **Japanese (`ja`)** — primary/default review and authoring language.
- **English (`en`)** — supported for international reading and development sanity checks.
- **Chinese (`zh`)** — supported for the author's bilingual context.

## Why it matters

felicia is centered on travel memory, place names, transit labels, stamps, tickets, and essays.
For Japan trips, Japanese is not a decorative translation layer; it affects station names,
stub templates, route labels, and the feeling of the artifact.

## Consequences

- UI copy should not be hard-coded into components once the MVP spec starts.
- Domain data should distinguish stable identifiers from localized display labels.
- Station/catalog fixtures should preserve Japanese labels alongside romanized names.
- Essay content can remain authored text; the app should not auto-translate personal writing
  unless a later decision explicitly adds that.

## Open

- Whether the current decision demo should immediately expose a language switcher, or whether
  the first pass only records the architecture and uses Japanese-first copy.
- Locale codes and fallback rules: likely `ja`, `en`, `zh-Hans` / `zh-Hant`, but this should be
  confirmed before implementation.
