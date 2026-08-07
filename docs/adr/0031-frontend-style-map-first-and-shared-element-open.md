---
id: "0031"
title: "Frontend Style: Map-First Layout and Shared-Element Open"
status: "accepted"
date: "2026-08-04"
decisions:
  - "felicia:decision:frontend-style"
related: ["0002"]
supersedes: []
---

# ADR 0031: Frontend Style — Map-First Layout and Shared-Element Open

## Context

[`docs/research/ux-restyle.md`](../research/ux-restyle.md) (2026-07-02) diagnosed the
early reader prototype — a dead entry state (the index was unreachable on first load),
no signature open moment (mementos didn't behave like physical objects being opened),
scaffolding leaking into the public reader, and a clashing visual identity (a warm JR
ticket next to generic Tailwind kind badges).

The note committed several directions as **high confidence** — warm paper as the
material system for every `kind`, killing the authoring scaffolding out of the reader,
fixing the entry so the index is always reachable, and ragged-right essay typography —
but left two **genuine taste decisions** explicitly open, asked of the author on
2026-07-02 with no answer until now (M0 issue #16, "Freeze memento visual and motion
direction," tracked this gap):

1. **Layout / navigation model** — index rail + detail vs. map-first fixed entry vs.
   immersive scrollytelling.
2. **Signature open interaction** — shared-element morph vs. tear/unfold vs. flip.

In the time since, `apps/web-public/src/{v1,v2,v3,v4}` grew four parallel reader
designs (`v1` "Map", `v2` "Collection", `v3` "Journal", `v4` "Atlas") as prior
exploration, but none of them is a direct, committed answer to either fork — they
predate this decision, not implement it.

## Decision

Settled 2026-08-04, author's call:

1. **Layout / navigation model: map-first, fixed entry.** A full-bleed map stays the
   hero. The reader lands on a journey-overview card (title, dates, memento count)
   plus a viewport-anchored index toggle — not a persistent side rail. This is the
   smallest structural change from the diagnosed prototype and keeps the map as the
   index, matching [ADR-0002](0002-mementos-not-tickets.md)'s "the map is the index"
   framing.
2. **Signature open interaction: shared-element morph.** The map stub grows into the
   full stub in the detail view — one continuous object, not a separate open/close
   panel. This is the option ux-restyle.md judged best reinforces map-as-index.

The other "high confidence" items from ux-restyle.md are frozen alongside these two
forks, as the same issue (#16) scoped them together:

3. **Warm paper is the system.** Every `kind` renders as a designed paper object in
   one warm accent scale (amber → orange) — ticket (admission stock), transit (the
   existing きっぷ magnetic ticket, kept as-is), stamp/goshuin (ink-on-washi), goods
   (hang-tag/kraft label), receipt (thermal-roll paper). No Tailwind-badge kind
   coloring.
4. **Entry is never dead.** The index (journey overview + memento quick-list) is
   reachable from first load, via a viewport-anchored affordance rather than one
   anchored to a panel that can slide away.
5. **Essay typography.** Ragged-right (no `justify`), a wider reading measure, and
   raised leading (~1.7).

## Consequences

- This freezes the fork so implementation doesn't reopen it (the acceptance bar
  issue #16 set) — `docs/research/ux-restyle.md`'s "Open forks" section and
  `docs/direction.md`'s open-question line are updated to point here instead of
  leaving the question open.
- **None of the four existing prototype designs is a clean match for "map-first,
  fixed entry."** `v1` is an always-visible left index rail (closer to the rejected
  "index rail + detail" option); `v3` is a two-page book-spread; `v4` is scroll-driven
  ("immersive scrollytelling," the other rejected option). Building the reader that
  actually matches this decision — full-bleed map, a journey-overview card, a
  viewport-anchored toggle instead of a persistent rail, and the shared-element open —
  is real follow-up implementation work, out of scope for this ADR. Whether that means
  adapting the closest existing prototype or building fresh is a call for whoever picks
  up that implementation issue.
- Per-kind stub material design (tokens, exact CSS) still needs a follow-up frontend
  style spec (ux-restyle.md's own next step) before implementation — this ADR freezes
  the _direction_, not a component spec.
