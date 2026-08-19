---
title: "Public Read Contract"
status: "exploratory"
date: "2026-07-17"
---

# Public Read Contract

This contract is the seam between Felicia's canonical storage and its public
readers. It is deliberately independent of whether the reader is served by a
Go API or GitHub Pages files.

## Paths

```text
GET /api/v1/journeys.json
GET /api/v1/journeys/<journey-uuid>.json
GET /api/v1/journeys/<journey-uuid>/mementos.json
```

The Go server currently retains extensionless aliases for compatibility. New
frontend code and static publication use the `.json` paths.

## Projection rules

- The index contains only journeys with at least one published memento.
- Journey detail contains stable UUID identity, dates, place metadata, optional
  route geometry, source reference, and authored-field markers.
- Memento detail contains stable identity, ordering, occurrence time/timezone,
  optional point geometry, authored text, kind data, source reference, and
  ordered photo metadata.
- Geometry uses GeoJSON `Point`, `LineString`, or `MultiLineString` coordinates
  in `[longitude, latitude]` order.
- Public projections never contain drafts, private originals, or raw package
  files.
- Photo `object_key` values are public derivative references, not local source
  paths.

## Current evidence

- `apps/felicia-public-site/src/api/source.ts` requests the three `.json` paths.
- `apps/felicia-server/api/server.go` exposes matching `.json` aliases.
- `scripts/verify_static_artifact.py` validates the index, every journey detail,
  every memento file, project-site base path, and published media.
- Go API tests and frontend tests pass for the current fixture/provider baseline.

## Still open

- Generate this exact contract from SQLite rather than hand-authored reader data.
- Compare a SQLite server response byte-for-byte by normalized JSON with its
  static projection.
- Define whether route timestamps/elevation need public extension fields.
- Add media derivative and privacy assertions to the verifier.

This is contract evidence, not an accepted canonical data-model ADR.
