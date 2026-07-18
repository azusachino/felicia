# Local authoring schema v1

The local workflow has four distinct JSON documents. `plan.json` is generated
evidence; the other three are editable authoring state.

| File            | Schema identifier                      | Ownership | Purpose                                          |
| --------------- | -------------------------------------- | --------- | ------------------------------------------------ |
| `plan.json`     | `felicia.intake.plan` + `version: "1"` | Felicia   | Reproducible source-derived plan and diagnostics |
| `journey.json`  | `felicia.local.journey.v1`             | Author    | Journey metadata and date range                  |
| `stops.json`    | `felicia.local.stops.v1`               | Author    | Keep/ignore decisions and stop labels            |
| `mementos.json` | `felicia.local.mementos.v1`            | Author    | Memento order, kind, content, and selected media |

The machine-readable definitions are in
[`schemas/local-authoring-v1.schema.json`](../../schemas/local-authoring-v1.schema.json).

## v1 rules

- The `schema` identifier is exact; a future incompatible shape gets a new
  identifier rather than silently changing v1.
- `plan.json` is regenerated from GPX/provider inputs and is never hand-edited.
- `candidate_key` is the stable join from a curated memento to a stop. A
  missing stop key is an authoring error, not an orphan to publish.
- `selected: false` excludes a stop and all mementos linked to it from the
  generated package. It does not delete source evidence.
- Memento IDs and `seq` are explicit. Reordering changes `seq`, not identity.
- `kind_data` is open for kind-specific fields; common authored fields remain
  top-level so importers and readers can handle them consistently.
- `media.path` is a local source reference in the workspace. The package
  builder resolves, hashes, and rewrites it to a safe package object key.
- Unknown top-level fields are invalid in v1. The schema must be versioned before
  adding a new field; reserved authored fields are already represented even when
  the current importer has not mapped every one yet.

## Deliberate boundaries and gaps

This task freezes file shape, not every downstream capability. The following
remain explicit next-task work:

- essay/vendor/price and the explicit `authored_fields` mask must be mapped
  through the importer without being dropped;
- translations are intentionally not part of v1: the canonical model has no
  translation sidecar, and authored content is rendered exactly as entered;
- `media` is still image-shaped in the current package/publication path;
- multiple journeys are currently multiple workspaces/packages, not one root
  workspace document;
- JSON Schema validation is defined here but executable validation belongs in
  the implementation task.
