# Intake experiments

Each case corresponds to one user story from
[`docs/research/intake-stop-candidates.md`](../../docs/research/intake-stop-candidates.md).
The cases are deliberately small and deterministic. They are source fixtures,
not public journey content.

Current status: these cases describe the proposed raw-intake contract. The
offline `felicia-cli journey plan` path is executable; partial cases identify
deliberate capability gaps rather than missing command coverage.

## Cases

| Case                       | Class | Story                                  | Current expectation                                             |
| -------------------------- | ----- | -------------------------------------- | --------------------------------------------------------------- |
| `US-01-plan`               | happy | Plan without mutation                  | pass: raw GPX/local media plan and deterministic output         |
| `US-02-review-stops`       | happy | Review evidence-backed stops           | partial: persistence/review API exists; session fixture pending |
| `US-03-missing-metadata`   | evil  | Handle photos without metadata         | partial: JSONL sidecar works; EXIF/confidence classes pending   |
| `US-04-mementos-from-stop` | happy | Create multiple mementos from one stop | partial: mementos work, stop link does not                      |
| `US-05-agent-suggestions`  | evil  | Agent suggestions preserve authorship  | partial: patch boundary exists, suggestion store does not       |
| `US-06-safe-publish`       | evil  | Publish only reviewed story            | works for prepared packages; raw-intake path blocked            |

Happy cases prove that the intended workflow is useful. Evil cases prove that
bad or incomplete inputs fail safely and remain explainable. Both are required.

## Implementation gates

The user stories are acceptance tests, not the first implementation target. Do
not build story-specific CLI commands before these model gates are settled.

### Gate 1 — Canonical model

Ratify the boundaries and identities for:

```text
Route / Visit / StopCandidate / MementoCandidate / Memento / MediaAsset
```

Decide whether stop candidates are persisted draft records or workspace-only
records, how evidence is linked, and how a memento references a stop without
creating a public places table.

### Gate 2 — Source capability API

Split source capabilities so connected and offline inputs share the same core:

```text
RouteSource   -> Dawarich, local GPX
VisitSource   -> Dawarich, local visit derivation
PhotoSource   -> Immich, local photo directory
```

The core must not depend on Dawarich, Immich, HTTP, SQLite, or PostgreSQL.

### Gate 3 — Pure plan operation

Implement a deterministic, read-only planner that returns routes, visits, stop
candidates, memento candidates, evidence, warnings, and unresolved items.
It must not mutate authored records or publish anything.

### Gate 4 — Persistence and review

Persist source observations, candidate state, evidence links, and authoring
decisions through narrow ports. Re-import must preserve authored fields and
ignored or merged candidate decisions.

### Gate 5 — CLI/server projections

Expose the same runtime operations through `felicia-cli` and the localhost
admin API/UI. They must not implement two different planning or curation paths.

## Experiment order

1. Model review: resolve entities, identities, evidence links, and draft
   persistence policy.
2. Happy baseline: timestamped route, three photo classes, and one clear stop.
3. Evil metadata case: photos without EXIF, then the same case with a JSONL
   sidecar.
4. Evil track case: out-of-order timestamps, impossible-speed jumps, sparse
   points, and a second trip.
5. Evil scale case: high-point GPX with memory, runtime, and deterministic-output
   measurements.
6. Provider parity: equivalent Dawarich, Immich, GPX, and local-photo inputs.
7. Evil re-import: source changes after authoring title, essay, and photo order.
8. Evil publication: private originals, unattached media, and raw route data.

## Running the current package path

The existing CLI can be exercised with a prepared ZIP fixture:

```bash
make cli-build
python3 scripts/build_preview_package.py
bin/felicia-cli package validate .felicia/preview.zip
bin/felicia-cli import --db .felicia/experiments/US-06-safe-publish/felicia.sqlite \
  --media-root .felicia/experiments/US-06-safe-publish/media \
  --apply .felicia/preview.zip
bin/felicia-cli static compile \
  --db .felicia/experiments/US-06-safe-publish/felicia.sqlite \
  --media-root .felicia/experiments/US-06-safe-publish/media \
  --out .felicia/experiments/US-06-safe-publish/site
```

Generated databases, media, reports, and sites belong under `.felicia/` and
must not be committed. The small source inputs and expected reports in this
directory are committed so the experiments remain reproducible.

The current read-only command is:

```text
felicia-cli journey plan --gpx <file.gpx> --photos <directory> [--sidecar <photos.jsonl>] --format json
```

That command should consume these source cases without mutating the database.
