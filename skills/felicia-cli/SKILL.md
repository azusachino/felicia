# Felicia CLI agent skill

Use this skill when an agent must turn a user's offline travel exports into a
reviewable Felicia journey and preview it locally.

Felicia is contract-first. The canonical semantic contract is
[`contracts/canonical/v1/schema.json`](../../contracts/canonical/v1/schema.json).
The editable workspace files are an adapter of that contract, not the database
model or the public API. Read the relevant contract before inventing fields.

## Safety and authority

The default workflow is offline and local. Do not upload GPX, photos, sidecars,
or package contents. Do not call Immich, Dawarich, an HTTP server, or a remote
agent tool unless the user explicitly provides that authority.

Never:

- publish, deploy, push, or send a message without explicit user approval;
- overwrite authored fields during re-import;
- mark a stop as kept or publish a memento merely because a model suggests it;
- copy raw originals, private media, or unsanitized route data into a public
  artifact;
- invent attraction names, dates, coordinates, vendors, prices, or evidence;
- treat a missing timestamp or coordinate as proof that an asset is unrelated.

When evidence is insufficient, preserve the candidate as unresolved and explain
what the user must decide.

## Current command boundary

Build the real CLI through the repository task runner:

```sh
make cli-build
```

The binary is `bin/felicia-cli`. The CLI currently operates locally and does
not contact the Felicia server. The server's admin API is a separate transport
used only when the user explicitly asks for server authoring.

Available CLI commands:

```text
felicia-cli journey plan
felicia-cli journey apply
felicia-cli journey review
felicia-cli package validate
felicia-cli import
felicia-cli static compile
```

Do not claim that commands such as `journey diff`, `publish`, agent suggestion
acceptance, or multi-journey package import exist until the CLI help/source and
contract are updated.

## Required input

Minimum planning input:

- a valid GPX file;
- a stable journey UUID;
- a photo directory is optional but required to match local photos;
- a JSONL sidecar is optional and should be used when photo metadata is missing.

The sidecar is one JSON object per line. Use stable local paths and only factual
metadata, for example:

```json
{ "path": "photos/IMG_0001.jpg", "at": "2026-04-01T09:10:00Z", "coord": [135.5016, 34.6687], "source_ref": "local:photo:IMG_0001" }
```

Do not put prose guesses in the sidecar. Keep the original files unchanged.

## Happy path: plan and inspect

1. Build the CLI.
2. Run a plan without mutation.
3. Save the plan as evidence.
4. Inspect stops, mementos, issues, and source fingerprints.

```sh
make cli-build
./bin/felicia-cli journey plan \
  --journey 00000000-0000-0000-0000-000000000001 \
  --gpx /path/to/route.gpx \
  --photos /path/to/photos \
  --sidecar /path/to/photos.jsonl \
  --format json > plan.json
```

The JSON plan is source-derived evidence. It is not an authoring decision.
Repeated planning with unchanged inputs should be deterministic. Record:

- command and input paths;
- plan schema/version;
- source fingerprint;
- route/visit/stop/memento counts;
- warnings and errors;
- unmatched media count.

For streaming inspection, use JSONL:

```sh
./bin/felicia-cli journey plan \
  --journey 00000000-0000-0000-0000-000000000001 \
  --gpx /path/to/route.gpx \
  --format jsonl > plan.jsonl
```

JSONL records are currently `{"type":"stop", ...}` followed by one summary
record. Treat the summary as authoritative for counts and the stop records as
review candidates.

## Authoring workspace

The UV orchestration creates an editable workspace from a plan:

```sh
uv run python scripts/local_journey.py preprocess \
  --journey 00000000-0000-0000-0000-000000000001 \
  --gpx /path/to/route.gpx \
  --photos /path/to/photos \
  --sidecar /path/to/photos.jsonl \
  --workspace .felicia/local-journey
```

The workspace contains:

- `plan.json`: immutable-ish source evidence; regenerate rather than hand-edit;
- `journey.json`: authored journey metadata;
- `stops.json`: explicit stop selection and labels;
- `mementos.json`: authored memento order, kind, content, and media selection;
- `route.gpx`: private source route;
- optional `workspace.json`: root index when one directory contains multiple
  journey workspaces.

Authoring rules:

- select a stop only when its evidence supports a meaningful stay or place;
- rename labels only when the user or reliable source evidence supports it;
- one stop may contain multiple mementos;
- add a memento manually when the attraction is not inferable from GPS;
- preserve `candidate_key` joins;
- keep memento IDs stable when reordering; update `seq` only;
- leave uncertain fields empty or in a review note rather than hallucinating.

Validate before packaging:

```sh
uv run python scripts/validate_local_authoring.py .felicia/local-journey
```

## Package and preview

Package one reviewed journey:

```sh
uv run python scripts/local_journey.py package \
  --workspace .felicia/local-journey
```

Validate the actual transport archive:

```sh
./bin/felicia-cli package validate .felicia/local-journey/journey.zip
```

The current public package accepts only public local JPEG, PNG, and WebP image
attachments. Video, audio, documents, links, embeds, private media, external
URLs, and unsupported files must remain outside publication and should produce
an explicit error or unresolved report.

Run the complete local preview:

```sh
uv run python scripts/local_journey.py preview \
  --workspace .felicia/local-journey
```

This imports into workspace-local SQLite and compiles a static site. It does not
contact a server or publish anywhere. Inspect `site/` and verify that:

- only published mementos appear;
- route geometry is present;
- media paths resolve;
- private/unselected material is absent;
- repeated runs do not duplicate records.

The repository demo builds multiple journey packages with:

```sh
BASE_PATH=/ nix develop --command uv run python scripts/build_preview_package.py
```

## Applying a plan and reviewing stops

The CLI can apply a plan to SQLite:

```sh
./bin/felicia-cli journey apply \
  --db .felicia/felicia.sqlite plan.json
```

Review a persisted stop explicitly:

```sh
./bin/felicia-cli journey review \
  --db .felicia/felicia.sqlite \
  --candidate <candidate-uuid> \
  --state kept \
  --label "Dotonbori" \
  --expected-revision 1
```

Valid states are `proposed`, `kept`, `ignored`, and `merged`. A stale expected
revision must be treated as a conflict; reload before retrying.

## Server/API boundary

When the user explicitly requests server authoring, use the documented admin
routes in `apps/felicia-server/api/server.go`. Current important routes include:

- `GET /api/admin/journeys` and `POST /api/admin/journeys`;
- `GET /api/admin/journeys/{id}/mementos`;
- `GET/POST /api/admin/mementos/{id}` and `/api/admin/mementos`;
- `GET /api/admin/journeys/{id}/stop-candidates`;
- `POST /api/admin/stop-candidates/{id}/review`;
- `POST /api/admin/photos`;
- `GET /api/v1/journeys.json` and journey memento projections.

The admin API is authoring state; the `/api/v1` API is public state. Do not
expose admin endpoints through a public static deployment. Respect revision
conflicts and preserve the same canonical ownership rules as the CLI.

## Failure and evil paths

Stop and report clearly for:

- malformed XML, invalid latitude/longitude, missing track points, or huge GPX;
- photos without timestamps or coordinates;
- sidecar paths outside the photo directory;
- duplicate IDs, missing stop keys, invalid kinds, or invalid timestamps;
- private/unsupported media in a public package;
- path traversal, checksum mismatch, or duplicate package members;
- stale memento/stop revisions;
- a plan that has warnings but no trustworthy stops.

Do not “repair” an input silently. Keep the source and emit a deterministic
diagnostic that lets the user correct or explicitly accept the result.

## Agent handoff report

Every agent run should finish with:

1. inputs and command lines;
2. files created or edited;
3. selected/ignored stop decisions and their evidence;
4. mementos created, changed, or left unresolved;
5. media included and excluded with reasons;
6. validation/package/preview results;
7. remaining user decisions;
8. whether any publish or server mutation was performed.

If a contract field or command is missing, stop and identify the contract gap.
Do not invent a new shape inside a user workspace as a substitute.
