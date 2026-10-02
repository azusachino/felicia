# Canonical contract v1

The canonical contract is Felicia's stable semantic model. It is independent of
SQLite, PostgreSQL, the local workspace, HTTP, and GitHub Pages. The machine-
readable record envelope is [`contracts/canonical/v1/schema.json`](../../contracts/canonical/v1/schema.json).

## Record ownership

- `journal` and `journey` are authored aggregate identity and metadata.
- `route` and `visit` are source-derived evidence. They may be absent or
  incomplete when a source cannot provide them.
- `stop_candidate` is private review state, never public place data.
- `memento` is the authored/public story object. Its lifecycle and revision are
  canonical; source evidence cannot silently overwrite authored fields.
- `media` is an attachment reference with source identity, visibility, and
  publication metadata. A media kind is not automatically renderable.
- `suggestion` is a non-mutating proposal. Acceptance is a separate authoring
  operation.

## Stable semantics

All coordinates are `[longitude, latitude]`. Timestamps are RFC 3339 strings and
must preserve the source timezone where it is meaningful; `occurred_tz` records
the display timezone separately from the instant. Source identity is the
idempotency key across re-imports. Local UUIDs identify Felicia records.

Memento writes use optimistic revisions. A stale write is a conflict, not an
implicit merge. Ingest and authoring writes use different ownership rules:
ingest may update source-owned fields, while authoring explicitly claims fields.

## Offline timezone defaults

Intake and package import resolve missing `occurred_tz` values offline from
`[longitude, latitude]`. Point memories use their point; transit lines use their
departure point. An explicit value, including `UTC`, wins over inference. An
admin edit that omits the zone retains the existing value. Re-import still
respects the journal's authored-field mask.

The intake journey fallback is the first usable coordinate in source order:
route, supplied visits, then media. Photos without GPS use that journey zone,
not the zone of whichever stop happens to match them. Coordinate-based local
calendar days also inform journey date bounds; conversion never changes an
instant or mutates source input. When no journey zone can be derived, UTC is
the display default and date bounds retain any meaningful source offset.

Packages may specify an IANA `manifest.yaml` `timezone` as the no-coordinate
fallback. Otherwise they derive it from the route, stops, then memento geometry,
finally UTC. Invalid supplied zones still fail validation. Coordinates continue
to be validated at write boundaries; `(0,0)` is treated as missing GPS for lookup.
No timezone field or migration is added to journey persistence.

The runtime pins [tzf v2.1.2](https://github.com/ringsaturn/tzf/tree/v2.1.2),
using one concurrent-safe `NewEmbeddedFinder` and embedded IANA `time/tzdata`.
This keeps lookup independent of network access and host zoneinfo installation.
The simplified boundary dataset has approximately 111 m uncertainty near borders;
these are editable defaults, not a claim of exact border precision. The full
finder's roughly 147 MiB retained heap is avoided for the local CLI/server flow.
Code is MIT-licensed; embedded boundary data comes from
[tzf-dist](https://github.com/ringsaturn/tzf-dist) and
[timezone-boundary-builder](https://github.com/evansiroky/timezone-boundary-builder)
under ODbL. Updates require an explicit dependency/data bump and regression run.

## Media capability boundary

Canonical media kinds are `image`, `video`, `audio`, `document`, `link`, and
`embed`. Each adapter declares what it can ingest, store, preview, and publish.
The presence of a canonical media record does not authorize publication. A
public projection must apply visibility, provider trust, derivative, and MIME
rules before emitting an attachment.

## Projection map

```text
canonical/v1
  ├── workspace/v1       editable files and review controls
  ├── apps/felicia-cli/v1             plan JSON/JSONL and command reports
  ├── admin-api/v1       write/read transport DTOs
  ├── storage             normalized relational persistence
  └── public-api/v1      published, redacted static/server projection
```

No projection may add provider-specific semantics to the canonical record or
publish private source evidence by default.
