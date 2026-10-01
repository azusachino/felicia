---
id: "0039"
title: "Journey Authoring Writes and Mask Ownership"
status: "accepted"
date: "2026-09-11"
related:
  - "0022"
  - "0033"
---

# ADR 0039: Journey Authoring Writes and Mask Ownership

## Context

[ADR-0033](0033-authored-field-protection-and-the-journey-ingest-seam.md) split
journey writes into an authoring write and an ingest write, and closed the
ingest side. Its Consequences named what it left open:

> Not addressed here: the admin journey API still sends `authored_fields`
> wholesale, so a client that omits the array can clear the mask. That is an
> authoring-side concern (invariant 3 territory), not an ingest one.

That is issue #89. It is still open, and it is unreachable today only because
the admin GUI has no journey create or edit affordance at all (#82). Building
#82 first makes the defect live on the new feature's happy path, so the
authoring contract has to be settled before the form exists, not after.

Two things are wrong with `handleUpsertJourney`
(`apps/felicia-server/api/server.go:609`) rather than one:

1. **The mask is whatever the client says.** `if req.AuthoredFields == nil {
   req.AuthoredFields = []string{} }` (`:616`) turns an omitted field into an
   explicit empty mask, and that value goes into `UpsertJourney` — the
   authoring write, which per ADR-0033 takes the caller's mask at face value.
   `apps/felicia-admin/src/api.ts:105` declares `authored_fields?: string[]`
   optional, so a client that never sets it is type-correct. The next import
   then legally overwrites title, place and route, because the mask no longer
   claims them.
2. **An omitted route blanks the stored route outright.** `gpsRoute` stays nil
   when the request carries no route (`:631-642`), `UpsertJourney` assigns every
   column, and `marshalJSON(nil)` writes `gps_route = 'null'`. This needs no
   later import to lose data; the save itself does it.

The second fault is not recorded anywhere. Both have the same root: the
authoring endpoint treats a partial request as a complete description of the
row.

Mementos do not have either problem, and the difference is instructive.
`handleUpsertMemento` ignores the client's `authored_fields` entirely and passes
a **hardcoded field list** — the fields the authoring form actually writes — to
`ApplyManualPatch` (`:1180-1183`). It carries `ExpectedRevision`, so a
concurrent write fails with `ErrWriteConflict` and a 409 rather than clobbering.
It also treats an omitted state as "keep the current state", with a comment
saying why. Journeys got none of this.

## Decision

**The server decides a journey's authored mask from the fields it wrote. The
client never sends one.**

- `handleUpsertJourney` stops reading `req.AuthoredFields`. The field is removed
  from `upsertJourneyRequest` and from `apps/felicia-admin/src/api.ts`, so
  clearing the mask is not something a client can express, rather than something
  a client is trusted not to do.
- The authoring write claims exactly this list:

  ```text
  slug, title, place, country, region, date_start, date_end
  ```

- **`gps_route` is never authored.** It is absent from that list deliberately.
  Ingest owns the trace permanently, so Dawarich can keep refreshing it for the
  life of the journey. An author curates stops and mementos; the passive GPS
  track is not theirs to pin. This is what keeps the model free of un-authoring:
  every field an author can claim is a field they set deliberately and rarely
  want to un-set, so the mask only ever grows and no field needs a way out.
- Because the route is never authored, the authoring write must not write it
  either. A journey save preserves the stored `gps_route` instead of assigning
  the request's. An absent route means "unchanged", never "empty".
- `expected_revision` is added to the journey request, mirroring mementos: a
  stale write fails with `ErrWriteConflict` and the API answers 409. PR #120
  closed this race on the ingest side; this closes it on the authoring side.

ADR-0033's invariant 3 — "an authoring write can write any field and claim it as
authored" — is narrowed, and this ADR is the record of that. An authoring write
can still write and re-edit every field an author owns; it can no longer claim
`gps_route`, and it can no longer be told which fields to claim.

The invariants, stated so they can be tested:

1. A journey authoring write claims exactly the seven fields above, whatever the
   request contains.
2. A journey authoring write never modifies `gps_route`.
3. A journey authoring write with a stale `expected_revision` fails and changes
   nothing.
4. No sequence of authoring writes can shrink a journey's authored mask.

## Consequences

- #89 stops being reachable by construction rather than by validation. There is
  no request shape that clears the mask.
- #82 gets cheaper. A journey form PUTs the fields it edited and nothing else —
  it never tracks, echoes, or merges `authored_fields`, and it cannot blank a
  route by not knowing about one.
- An author cannot pin a hand-corrected GPS trace. If that becomes a real need,
  it arrives as a deliberate "adopt this route" action plus the only case of
  un-authoring in the model, and it supersedes this ADR rather than bending it.
- A new journey field an author should own means adding it to the list in one
  place. A field missing from the list is simply never claimed, so an import
  keeps refreshing it — visible, and recoverable, rather than silent data loss.
- `UpsertJourney`'s port signature is unchanged: it still takes a whole journey
  and a mask. What changes is that the mask is now assembled server-side, so the
  two providers need no new agreement (AGENTS.md development-flow constraint 2
  does not apply).
- Journeys and mementos now have the same authoring contract — derived mask,
  hardcoded field list, optimistic concurrency. The asymmetry that made this
  defect possible is gone.
- #84 (a journey can never be deleted) becomes more urgent, not less. Once #82
  ships, an author can create a journey by mistake for the first time, and
  nothing can remove it.
