---
id: "0023"
title: "Portable Journey Package for Import and Agents"
status: "accepted"
date: "2026-07-16"
decisions: []
related: []
supersedes: []
---

# ADR 0023: Portable Journey Package for Import and Agents

## Context

The author may record a journey with several applications and later bring the material into Felicia. Connected APIs are not always available, export formats change, and an agent should be able to
prepare or inspect an import without using a proprietary UI.

## Decision

Felicia accepts a versioned ZIP as the portable journey interchange format. It is a transport envelope, not a database backup and not an executable plugin.

The v1 shape is:

```text
journey.zip
├── manifest.yaml                 # required: schema, identity, checksums
├── journey.yaml                  # optional journey metadata
├── route.gpx                     # optional route fallback
├── timeline.json                 # optional timestamped locations/events
├── visits.json                   # optional normalized visit hints
├── mementos.yaml                 # optional template-shaped candidates
├── notes/                        # optional Markdown notes
└── media/                        # optional photos and future assets
```

`manifest.yaml` contains `schema_version`, stable `package_id`, creation/source metadata, file paths, media kinds, byte sizes, SHA-256 checksums, record IDs, and optional timezone/coordinate precision
metadata.

Package import behavior:

1. Validate the ZIP before writing: schema, paths, checksums, file sizes, supported media kinds, coordinate ranges, timestamps, and template fields.
2. Reject path traversal, symlinks, executable content, duplicate record IDs, and checksum mismatches.
3. Produce a stable dry-run report of created, matched, changed, orphaned, and unresolved records.
4. Apply only after explicit confirmation, as a draft import run.
5. Derive source identities from `package_id` and record IDs. Re-importing the same package is idempotent; media deduplication additionally uses content hashes.
6. Preserve original package files privately when configured, but only derivatives with stripped metadata enter the public media path.

The agent-facing contract is command/API based:

```text
felicia package validate journey.zip
felicia import journey.zip --dry-run
felicia import journey.zip --apply
felicia journey diff <journey-id>
felicia publish <journey-id>
```

An agent may generate a package, validate it, and propose an import report. It may not publish or overwrite authored content without an explicit authoring operation.

## Consequences

- Google Timeline, GPX tools, photo managers, and agents target one stable adapter.
- The package is inspectable with ordinary archive and YAML/JSON tools.
- Schema evolution is explicit; unsupported versions fail before mutation.
- The importer distinguishes package identity, record identity, and media identity.
- Unsupported source metadata remains in `notes/` or is reported unresolved.

## Rejected alternatives

- **Database dumps:** couple interchange to storage and migrations.
- **Executable ZIP plugins:** create an unnecessary code-execution boundary.
- **UI-only import:** prevents offline preparation, dry runs, and agent workflows.
