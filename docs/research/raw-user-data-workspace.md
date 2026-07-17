---
title: "Raw user-data workspace"
status: "proposed"
date: "2026-07-17"
---

# Raw user-data workspace

This is the boundary between a user's original materials and Felicia's reviewed
public content.

## Layout

```text
.felicia/
  inbox/              # raw imports; never published automatically
  felicia.sqlite      # local canonical database
  import-reports/     # machine-readable dry-run and conflict reports

content/
  journeys/
    <slug>/
      journey.yaml
      route.gpx
      mementos.yaml
      media/           # selected, intentionally publishable assets

site/                 # generated static reader; disposable output
```

The `.felicia` directory belongs to one local installation and is always
gitignored. A user may keep `content/` in a private or public Git repository,
but raw photos, exports, and unreviewed package contents stay in `.felicia/inbox`
unless the author explicitly promotes them.

## Intake examples

- Google Timeline or another location export is copied into `.felicia/inbox`.
- A GPX file is copied into `.felicia/inbox` or included in a ZIP package.
- Local photos are copied into `.felicia/inbox` and referenced by checksum.
- An agent can assemble or inspect a package without touching SQLite.
- `felicia import --dry-run` writes only an import report.

## Incremental behavior

The package ID identifies one source envelope. Each journey, memento, route, and
media entry also has a stable source identity. Re-importing the same package is
idempotent. A later package adds or updates source-owned observations; it does not
renumber existing authored mementos or overwrite authored prose and curation.

Public ordering is deterministic:

- journeys: `date_start`, `slug`, stable ID;
- mementos: explicit `seq`, `occurred_at`, stable ID;
- photos: explicit `seq`, stable ID;
- GPX points: source track order, with timestamps validated when present.
