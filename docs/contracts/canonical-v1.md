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
