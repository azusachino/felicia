# Local journey workflow

This is the smallest end-to-end authoring simulation for a local-first journey.
It intentionally leaves the human or an agent in charge of the semantic choices.

## One-command workflow

For the full human-in-the-loop flow, use one command:

```sh
uv run python scripts/local_journey.py run \
  --journey 0190cbde-f300-7000-8000-111111111111 \
  --gpx path/to/route.gpx \
  --photos path/to/photos \
  --sidecar path/to/photos.jsonl \
  --workspace .felicia/local-journey
```

This command preprocesses the sources, asks the user to confirm/rename stops,
allows manual stops and mementos, lets existing mementos be edited, and then
packages, validates, imports, and compiles the local preview. No intermediate
command is required.

## Individual stages

The stages remain available when an agent or a scripted experiment needs to
inspect or edit JSON between steps.

```sh
uv run python scripts/local_journey.py preprocess \
  --journey 0190cbde-f300-7000-8000-111111111111 \
  --gpx path/to/route.gpx \
  --photos path/to/photos \
  --sidecar path/to/photos.jsonl \
  --workspace .felicia/local-journey
```

The command runs the real `felicia-cli journey plan` and writes:

- `plan.json`: immutable-ish source-derived output and diagnostics;
- `journey.json`: journey metadata to complete;
- `stops.json`: the stop decisions to curate (`selected` and `label`);
- `mementos.json`: editable memento drafts seeded from matched media.

The JSON files are deliberately easy to edit. A user can remove a noisy stop,
rename “Osaka cluster” to “Dotonbori”, or add a memento for an attraction that
was not inferable from GPS. A local agent can make the same edits while keeping
the source plan unchanged.

## 2. Package and preview

```sh
uv run python scripts/local_journey.py package --workspace .felicia/local-journey
uv run python scripts/local_journey.py preview --workspace .felicia/local-journey
```

`package` includes only selected stops and their mementos, copies referenced
media into the package, and emits the existing version-1 portable package. The
`preview` command validates that package, imports it into a workspace-local
SQLite database, and runs the existing static compiler to:

```text
.felicia/local-journey/site/
```

The command is safe to repeat after editing the JSON, although the workspace
SQLite database and generated site are local build artifacts.

## Current boundary

The package importer carries essay/vendor/price and an explicit `authored_fields`
mask into the existing no-clobber write path. Translations are intentionally not
part of this v1 workflow: authored content has no translation sidecar and is
rendered exactly as entered. Media remains the separate image-support contract.
