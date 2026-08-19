# Local authoring schema v1 review

Review date: 2026-07-18

This report closes the schema-v1 task sequence with the executable intake matrix,
the two-journey static preview, and provider checks. The raw intake report is
generated at `.felicia/experiments/intake/report.json` by:

```sh
make cli-build
uv run python scripts/run_intake_experiments.py \
  --out .felicia/experiments/intake/report.json
BASE_PATH=/ mise exec -- uv run python scripts/felicia.py publish
```

## Story results

| Story                    | Result  | Evidence and remaining boundary                                                                                                                                       |
| ------------------------ | ------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| US-01 plan               | Pass    | Real CLI plan is deterministic across repeated runs; six route points, two derived stops, and three media inputs.                                                     |
| US-02 review stops       | Partial | Stop candidate output is executable; the matrix does not yet drive an authored review action. The SQLite HTTP workflow covers the separate review/write API path.     |
| US-03 missing metadata   | Partial | JSONL sidecar promotes one asset; EXIF extraction and confidence classes remain incomplete.                                                                           |
| US-04 mementos from stop | Partial | Planner emits at most one generic candidate per matched stop. The local authoring file can contain multiple authored mementos, but the planner does not propose them. |
| US-05 agent suggestions  | Not run | No suggestion schema or persistence contract exists; the safe behavior is still non-mutating absence.                                                                 |
| US-06 safe publish       | Pass    | Both prepared packages pass CLI validation; raw GPX is not copied into the public artifact.                                                                           |
| Evil: malformed GPX      | Pass    | Invalid latitude is rejected.                                                                                                                                         |
| Evil: 20,000-point GPX   | Pass    | Current XML materialization completed in about 292 ms with about 15.6 MiB child RSS in this run; limits and streaming remain future hardening.                        |

## End-to-end evidence

- Both preview workspaces validate against the v1 JSON Schema.
- Two independent packages import successfully: two journeys, eight mementos,
  and eight media files compile into the static preview.
- `make test-workflow` passes against SQLite.
- `FELICIA_TEST_DATABASE_DSN=... make test-postgres` passes against the live
  Podman PostgreSQL/PostGIS service.
- The PostgreSQL admin API starts and direct journey/memento writes succeed.
- The original PostgreSQL HTTP workflow exposed two isolation/parity defects:
  fixed IDs and ports allowed stale state, and the PostgreSQL no-clobber upsert
  path also preserved draft values during a manual publish. The harness now
  creates per-run IDs, probes `/readyz` on a reserved port, and can create,
  migrate, and drop a unique PostgreSQL database. Manual authoring uses a
  separate PostgreSQL upsert path. The isolated SQLite and PostgreSQL workflows
  now pass.

## Schema-v1 decision

The contract is usable for the local-first workflow, with explicit boundaries:

- `journey.json`, `stops.json`, `mementos.json`, and optional `plan.json` are
  executable-validated documents.
- Authored memento fields use an explicit `authored_fields` mask and survive
  SQLite/PostgreSQL no-clobber imports.
- Public packaging accepts only public local JPEG/PNG/WebP images. Broader
  intake media kinds remain descriptive until a `memento_media` model exists.
- Authored translations are intentionally absent; the canonical model has no
  translation sidecar.

## Remaining gaps before calling v1 product-ready

1. Implement agent suggestion storage and an explicit review transition.
2. Add the future media attachment model for video, audio, documents, and
   trusted embeds.

Conclusion: schema v1 is suitable for the current offline authoring and static
preview experiment, but the two gaps above must remain visible in the next
implementation stage.
