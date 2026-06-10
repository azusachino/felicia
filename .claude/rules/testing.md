---
paths:
  - "**/*_test.go"
  - "internal/**/testdata/**"
---

# Testing conventions — felicia

- **Test-first** for the importer core. The pure packages (`domain`, `geo`, `exif`, `gpx`,
  `ocr` mapping) are exhaustively unit-tested with no network and no DB.
- Fixtures live in `internal/<pkg>/testdata/`: recorded Immich JSON, sample images (small,
  EXIF intact), a sample GPX, recorded vision-LLM responses, and golden YAML for `sync`.
- Use the in-memory `store/memrepo` for repository tests. The canonical invariant test:
  import → set an authored essay → re-import → essay survives, ingested fields refresh.
- Run with the race detector: `make test` (`go test -race -cover ./...`).
- Network/DB-touching tests are integration tests, build-tagged and excluded from the unit
  run; they target a throwaway PostGIS / a local MinIO, never live Immich/Dawarich.
