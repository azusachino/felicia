---
id: "0034"
title: "Journey Packages Transport Stop Candidates"
status: "accepted"
date: "2026-08-16"
related:
  - "0010"
  - "0022"
  - "0023"
  - "0030"
  - "0033"
---

# ADR 0034: Journey Packages Transport Stop Candidates

## Context

The CLI and the admin GUI each held half of one workflow (issue #79):

```text
CLI  local files → mementos OK      stop candidates MISSING
GUI  stop candidates OK             local files MISSING
```

A trip that entered through the CLI arrived with journeys, mementos, and photos
but no stop candidates, so `GET /journeys/{id}/stop-candidates` — the only thing
the intake inbox lists — returned nothing and the journey page said "No stop
candidates yet." The surface built for naming, merging, and discarding stops
could not be used on the trip at all: in the real-trip walkthrough all 20 stops
were unnamed (`stop_label_missing` ×20) and naming them meant hand-editing
`stops.json`.

Two fixes were available.

1. **Package import writes stop candidates.** The producer plans once; the
   package carries the result; the importer persists it.
2. **Intake planning runs over local sources.** The GUI's `Plan intake` re-runs
   the planner over the author's GPX instead of over the journey's configured
   Dawarich/Immich sources.

Option 2 was rejected. `Plan intake` is a server-side operation and the author's
GPX is a file on the author's machine, so it would need a second upload path
before it could work at all; it would put a second copy of the planning
algorithm behind the API, where the CLI already owns planning; and re-deriving
stops discards the producer's curation — a stop the author already discarded in
the workspace would come back proposed, which is the opposite of what ADR-0030
promises about review being explicit.

## Decision

A journey package may carry an optional `stops.yaml` member holding the stops
its producer already derived. `runtime/importer` decodes it into
`domain.StopCandidate` values and `ApplyPackage` persists them through
`UpsertStopCandidate`, alongside the journey, memento, and photo writes it
already performs. A package that omits the member imports exactly as before.

One entry:

```yaml
- candidate_key: derived-route:cluster-001 # required, producer's stable key
  derivation_version: gpx-stops-v1 # required, algorithm that derived it
  label: "" # optional source label
  coord: [135.7681, 35.0116] # required
  arrive: 2026-04-01T09:00:00+09:00 # required, RFC3339
  depart: 2026-04-01T10:30:00+09:00 # required, RFC3339
  confidence: 0.5 # optional, 0..1
  evidence: # optional
    - kind: route
      source: { system: local-track, external_id: derived-route:cluster-001 }
      locator: derived-route:cluster-001
```

Three properties follow from what the member deliberately does **not** contain:

- **Identity is the producer's, not the transport's.** `candidate_key` and
  `derivation_version` are both required, and together with the journey they are
  the candidate's identity — the same key both providers enforce as
  `UNIQUE (journey_id, derivation_version, candidate_key)`. A package may not
  omit the derivation version, because a candidate whose derivation is unknown
  cannot be matched against a later planning run or explained to a reviewer.
- **Review state is never transported.** There is no `state`, `merged_into`, or
  `authored_fields` field. An import seeds the inbox; the author reviews in the
  GUI. This is the ADR-0022/ADR-0033 authored-field rule applied to candidates,
  and it is enforced where those are enforced — in the providers' upsert, which
  refreshes source-owned columns and leaves state, the merge target, and an
  authored label alone. A re-import therefore cannot resurrect a discarded stop,
  undo a merge, or rename a stop the author named. A producer that has already
  discarded a stop simply omits it from `stops.yaml`.
- **Provenance is generated, not declared.** Every imported candidate gets one
  `Provenance` entry naming the package (`package:<package_id>`, external ID
  `candidate_key`) with the stop's own arrival as `observed_at` — the same
  convention the planner uses for a derived visit. When a stop carries no
  `evidence` of its own, one `visit` evidence ref pointing at
  `stops.yaml#<candidate_key>` is recorded instead of an invented upstream
  observation. ADR-0010 requires attributable derived data, and blank provenance
  on derived stops has already been a defect in this repository once.

No schema migration is involved: `tb_stop_candidates` already exists with the
same shape in `migrations/` and `providers/sqlite/schema.sql`, and the
cross-provider assertions live in `providers/contract`.

## Consequences

- A CLI-imported trip lands in the intake inbox and can be named, merged, and
  discarded in the GUI without touching `stops.json` by hand.
- Planning stays owned by the CLI. The package is still a transport envelope
  (ADR-0023), not a second implementation of the planner.
- Package producers must emit `stops.yaml` to benefit. Until the local-workflow
  packager does, its packages import as they do today — the member is optional
  and the importer's behaviour on packages that lack it is unchanged.
- Evidence is a whole-snapshot replacement per candidate, as it already was for
  planning. A package import therefore replaces the evidence of a candidate that
  shares its identity, which is correct for a refreshed derivation but means a
  package should carry the evidence it has.

## Rejected alternatives

- **Re-deriving stops from the package's `route.gpx` at import time.** Works
  without a producer change, but discards the producer's curation, duplicates
  the planner inside the importer, and ties import results to whichever planner
  config the importer happens to use.
- **Deriving candidates from the imported mementos.** Only stops that produced a
  memento would appear, and a memento is not a stop.
- **Transporting review state in the package.** It would let a file claim an
  author decision, which is exactly what ADR-0030 keeps explicit.
