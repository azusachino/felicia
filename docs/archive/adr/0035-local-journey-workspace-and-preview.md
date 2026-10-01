---
id: "0035"
title: "Local Journey Workspace and Authoring Preview"
status: "accepted"
date: "2026-08-19"
decisions:
  - "Use a folder of source files as the local journey intake contract."
  - "Scan and dry-run are read-only; import is an explicit, validated apply."
  - "Admin edits source-linked drafts and never mutates the original GPX or photo files."
  - "Draft preview and published preview are separate products."
  - "Atlas is the default published design; preview can switch among the shared named-design registry, and saving site settings selects the deployed design."
related:
  - "0022"
  - "0023"
  - "0025"
  - "0030"
  - "0034"
supersedes: []
---

# ADR 0035: Local Journey Workspace and Authoring Preview

## Context

The author has a GPX track and a small set of phone photos, but the current
admin GUI starts from an already-imported journey. A useful local-first flow
needs a stable place to put source files, a scan that explains what Felicia
found, and a preview that can be edited before anything is published.

The browser cannot safely infer a local folder from a path typed into the UI.
The local launcher therefore owns the workspace boundary; the future admin
scan action will call the same scanner and show its result before applying it.

## Decision

The author prepares one trip folder:

```text
izu-trip-2026-08-01/
├── route.gpx
├── photos/
│   ├── IMG_2699.jpeg
│   └── IMG_2708.jpeg
└── photos.jsonl              # optional metadata overrides
```

The folder may live anywhere outside the repository. When staged by Felicia,
it belongs under `.felicia/inbox/<slug>/`, which is gitignored. The source GPX
and original photos are private inputs and are never copied into the public
artifact.

`photos.jsonl` uses one JSON object per photo, with paths relative to the
`photos/` directory:

```jsonl
{"path":"IMG_2699.jpeg","at":"2026-08-02T13:09:35+09:00"}
{"path":"IMG_2708.jpeg","at":"2026-08-02T15:12:39+09:00","coord":[139.14,34.89]}
```

Supported metadata fields are `path`, `at` or `timestamp`, optional `coord`,
`title`, and `kind`. Explicit sidecar values override decoded EXIF values.
Absent metadata remains absent; the scanner must not invent coordinates or
titles. Original HEIC/Live Photo handling remains a separate media boundary;
the current public package accepts JPEG, PNG, and WebP derivatives.

The workflow has four distinct phases:

```text
source folder
  → scan / plan (read-only)
  → draft workspace and arrangement preview (editable, still private)
  → explicit validated import and publish/build
```

The scan produces an immutable-ish source plan plus editable derived files:

| File            | Meaning                                                         | Editable?                          |
| --------------- | --------------------------------------------------------------- | ---------------------------------- |
| `plan.json`     | route, matched assets, derived stops, diagnostics, provenance   | no                                 |
| `journey.json`  | journey identity/title/date fields                              | yes, with source identity retained |
| `stops.json`    | selected/ignored stops and labels                               | yes                                |
| `mementos.json` | draft kind, title, essay, geometry, order, and media references | yes                                |

Admin edits the derived draft records and records authored-field provenance.
Re-scanning the same source may refresh source-owned values, but it must not
erase authored fields or reorder authored photos silently.

The scan/dry-run response must show:

- source files and checksums;
- route date bounds and point count;
- each derived day/stop and confidence;
- each photo's timestamp/GPS provenance and selected match;
- unresolved, duplicate, unsupported, or unmatched inputs;
- the exact changes that an apply would make.

Scan and dry-run do not write the canonical database. Apply requires explicit
confirmation after package validation and is transactional through the
provider-neutral repository seam. The real-trip workspace still must not be
applied by the admin GUI until its scan/import surface exists.

Draft preview is not the public compiler. It may display draft mementos and
private local media to the author. The public compiler remains published-only
and strips private source data. The preview must expose the shared named design
registry — Atlas, Cartography, Cabinet, and Techo — as an instant choice. Atlas
is the fallback when a site has no saved settings. The selected design is
persisted to site settings only when the author saves the site configuration
and builds the publication.

Each story-bearing memento must list `essay` in `authored_fields`. This keeps
the importer from treating an authored story as source-owned data, while the
public compiler still emits it only after the memento reaches `published`.

## Consequences

- The author has one predictable folder contract for GPX, photos, and metadata.
- CLI and admin use the same scanner and produce the same dry-run plan.
- Original files remain the evidence layer; admin edits are reversible derived
  authoring state.
- A two-day trip with photos on only day two remains a two-day journey with two
  day-two mementos; the UI must not fabricate a day-one photo.
- Package safety and local-media correctness remain prerequisites for an
  apply-capable admin import screen.
