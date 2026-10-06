# Publish your own site

> For someone who wants to run **their own** travel journal on GitHub Pages.
> For contributor setup see [`setup.md`](setup.md); for the delivery status of
> this path see [`roadmap/user-journey.md`](roadmap/user-journey.md).

## What leaves your machine

Only the compiled `dist/` — published mementos, EXIF-stripped public image
derivatives, and rounded geometry. Your SQLite journal, drafts, and original
photos stay local and are gitignored
([ADR 0001](adr/0001-personal-now-product-ready.md),
[ADR 0005](adr/0005-sqlite-storage-and-content-addressed-media.md)).

There is **no hosted admin**. The authoring GUI runs from your own machine and
is never deployed; `make admin` binds the local stack to `0.0.0.0` for access
over Tailscale. GitHub only ever holds the static site. Because the admin API is
unauthenticated, keep the host firewall and Tailscale ACLs as the access boundary.

## Two routes

|                            | **A — CI build**                  | **B — Local authoring** (recommended) |
| -------------------------- | --------------------------------- | ------------------------------------- |
| Content lives in           | JSON files committed to the repo  | local SQLite, never committed         |
| Authoring interface        | a text editor                     | the admin GUI                         |
| Built by                   | GitHub Actions                    | your machine                          |
| Site title / design choice | not available (defaults to Atlas) | set in the GUI                        |
| Privacy                    | journal content is in git history | only `dist/` leaves the machine       |
| Needs a fork               | yes (for `.github/workflows/`)    | no — any checkout plus an empty repo  |

Route A is how this repository publishes the production journey catalog. Route B is the
intended path for a personal journal.

---

## Route B — local authoring

### The trip folder contract

Put each trip's source files together before opening admin:

```text
my-trip/
├── route.gpx
├── photos/
│   ├── IMG_2699.jpeg
│   └── IMG_2708.jpeg
└── photos.jsonl              # optional EXIF/GPS/title overrides
```

The scanner treats paths in `photos.jsonl` as relative to `photos/`. Use one
JSON object per line; `at` is the capture timestamp and `coord` is
`[longitude, latitude]`:

```jsonl
{"path":"IMG_2699.jpeg","at":"2026-08-02T13:09:35+09:00"}
{"path":"IMG_2708.jpeg","at":"2026-08-02T15:12:39+09:00"}
```

The GPX and original photos remain private source evidence. Scan and preview
derive editable draft records from them; they do not rewrite the files. The
single-step CLI command `felicia-cli journey ingest --dir <trip-folder>`
performs the scan, installs media, and stages draft records directly into the
workspace (see [Trip folder contract](contracts/trip-folder-contract.md)).
Alternatively, `make journey-local` performs the scan and writes an intermediate
editable workspace under `.felicia/workspaces/<slug>`:

```bash
make journey-local GPX=~/my-trip/route.gpx PHOTOS=~/my-trip/photos \
  SIDECAR=~/my-trip/photos.jsonl SLUG=my-trip TITLE="My trip"
```

See [ADR 0005](adr/0005-sqlite-storage-and-content-addressed-media.md) for the
dry-run, provenance, draft-preview, and named-design boundaries.

### 1. Get the code and a site repo

```bash
git clone https://github.com/azusachino/felicia.git ~/felicia
```

Then create an **empty public repository** on GitHub to host the site (for
example `my-travels`). It will contain nothing but the built site. A fork works
equally well; nothing in this route depends on the checkout's git remote.

> GitHub Pages on a **private** repository requires a paid plan. The site repo
> holds only what visitors already see, so public is normally the right choice.

Prerequisite: **mise**. Run `mise install` once; `make` runs repository tools through
`mise exec` without requiring shell activation.

### 2. Bring in a trip

The primary, streamlined way to ingest a trip into Felicia is the single-step CLI intake command:

```bash
./bin/felicia-cli journey ingest --dir ~/my-trip
```

(Build the binary first with `make cli-build` if not yet built.)

This command executes the entire intake pipeline in one step:

1. **Scans** `~/my-trip` for `route.gpx` (or `timeline.json`), photos (in `photos/` or top-level, up to 20 MiB each), and optional `photos.jsonl` sidecar overrides.
2. **Extracts metadata**: reads track coordinates and photo EXIF timestamps/locations, merging any sidecar overrides.
3. **Plans candidates**: clusters dwell times (20+ minutes within 250 m) into stop candidates and matches photos to stops.
4. **Installs media**: hashes photos with SHA-256 and installs unique files atomically into the workspace media store (`~/.felicia/media/`).
5. **Stages draft records**: creates or updates the journey, stop candidates, and photos in the local database (`~/.felicia/felicia.sqlite`), respecting existing human-authored edits (`authored_fields`).

You can customize identity and workspace parameters:

```bash
./bin/felicia-cli journey ingest --dir ~/my-trip \
  --slug kyoto-2026 \
  --title "Kyoto 2026" \
  --place "Kyoto, Japan"
```

The command emits structured JSON to stdout upon completion:

```json
{
  "mode": "ingest",
  "journey_id": "0192634e-8f2c-7431-b842-749e7cf93d8b",
  "slug": "kyoto-2026",
  "candidates": 5,
  "mementos": 4,
  "photos": 12,
  "conflicts": []
}
```

See [Contract: Trip folder intake and agent workflows](contracts/trip-folder-contract.md) for full details on directory layout, sidecar fields, and agent automation patterns.

#### Lower-level alternative: `make journey-local`

For workflows that require manual inspection or editing of intermediate JSON plan files (`journey.json`, `stops.json`, `mementos.json`) before committing to the database, `make journey-local` remains available:

```bash
make journey-local GPX=~/trip/route.gpx PHOTOS=~/trip/photos SLUG=kyoto-2026 TITLE="Kyoto 2026"
```

That writes an editable workspace to `.felicia/workspaces/<slug>` (printed
by the command): `journey.json`, `stops.json`, `mementos.json`, plus the
planner's `plan.json`. Pointing a workspace that already holds a _different_
journey at a new trip is a loud error, never a silent overwrite. Edit the
titles and selections, then package and import it — reuse the workspace path
the command printed:

```bash
uv run python scripts/local_journey.py package --workspace .felicia/workspaces/kyoto-2026
./bin/felicia-cli import --db .felicia/felicia.sqlite --media-root .felicia/media \
  --apply .felicia/workspaces/kyoto-2026/journey.zip
```

Repeat with a different GPX/`SLUG` for a second trip — it imports alongside
the first rather than replacing it.

Two things decide whether this produces anything:

- **Stops need dwell.** A stop candidate is 20+ minutes within 250 m. A track
  that never stops — a train ride, a drive — yields no stops, and therefore no
  mementos. `preprocess` reports success either way, so check the candidate
  counts rather than trusting the exit code alone.
- **Photos without EXIF.** Photos without EXIF capture timestamps or GPS tags cannot
  be attached to the track automatically. Supply a `photos.jsonl` sidecar in the trip
  directory with explicit `at` timestamps and/or `coord` coordinates:

  ```jsonl
  {
    "path": "IMG_0001.jpg",
    "at": "2026-04-18T00:25:00Z",
    "title": "Morning stop"
  }
  ```

### 3. Author

```bash
make admin
```

- admin GUI — `http://localhost:5174/` (use the host's Tailscale IP or MagicDNS name remotely)
- site preview — `http://localhost:8081/` (same Tailscale access)

Confirm stop candidates in the intake inbox, write essays in the memento editor,
then advance each memento `draft → authored → published`. Imported mementos
arrive as `draft`. **Only `published` mementos reach the artifact**, and a
journey with no published mementos is not published at all — a compile that
reports `Journeys: 0` usually means nothing has been published yet.

On the **Site & Deploy** page set the site title and description, pick one of
the four designs, and set the default language, theme, and accent colour. A
deployed site presents exactly one design.

Defaults: journal `.felicia/felicia.sqlite`, media `.felicia/media`, compile
output `.felicia/site`. Override with `--db` / `DATABASE_PATH`, `--media-root`,
or the Site & Deploy output directory.

### 4. Build a deployable directory

The artifact is the SPA and the compiled JSON/media **in one directory**. Build
the SPA with the base path your site will be served under, then compile your
journal into the same directory:

```bash
BASE_PATH=/my-travels/ make site-build
```

The compiler only removes files its own previous manifest listed, so the
co-located SPA build is left untouched.

`site-build` reads three optional environment variables, matching the defaults
above: `DATABASE_PATH`, `MEDIA_ROOT`, and `SITE_DIST`. Keeping the journal
outside the checkout is a good idea — it survives re-cloning:

```bash
DATABASE_PATH=~/felicia-data/felicia.sqlite MEDIA_ROOT=~/felicia-data/media \
  BASE_PATH=/my-travels/ make site-build
```

Equivalent from the GUI: set the Site & Deploy output directory to
`apps/felicia-public-site/dist` and press **Build** — but the SPA must already have been
built with the correct `BASE_PATH`.

!!! note "The Pages build is the publication path"

    `scripts/felicia.py publish` compiles the production journey catalog through the
    same SQLite import and static compiler used by the local authoring workflow.

Check the result before deploying:

```bash
BASE_PATH=/my-travels/ make site-verify
```

It asserts the base path reached `index.html`, that journeys are present, and
that every referenced photo exists in the artifact.

### 5. Deploy

Keep a clone of the site repo as a deploy directory, sync the build into it, and
push:

```bash
git clone git@github.com:<you>/my-travels.git ~/my-travels-deploy
rsync -a --delete --exclude .git --exclude CNAME apps/felicia-public-site/dist/ ~/my-travels-ops/
touch ~/my-travels-ops/.nojekyll
cd ~/my-travels-deploy && git add -A && git commit -m "deploy: site" && git push
```

`.nojekyll` is required — without it GitHub runs the output through Jekyll,
which drops files whose names begin with an underscore.

### 6. Enable Pages

**Only after the first push** — the branch must exist before it appears in the
dropdown:

Settings → Pages → Build and deployment → Source → **Deploy from a branch** →
branch `main`, folder `/ (root)` → Save.

Leave **Custom domain** empty; the site is served at
`https://<you>.github.io/my-travels/`. The URL appears at the top of that page
once the first deployment finishes.

### 7. Update

For an already-imported journey, make changes in the local SQLite journal through
Admin, then repeat **step 4** to rebuild and step 5 to sync, commit, and push the
artifact. The Pages settings never need to change.

Do not re-import the original package to update an existing journey. A package
can seed a new memento in its declared lifecycle state (normally a draft); the
journal owns that state after creation. A package may seed a published memento,
but import does not build or deploy the site. Re-importing content with a
conflicting lifecycle state is rejected rather than silently publishing or
unpublishing it. The import is transactional, so a rejected re-import leaves the
journal unchanged. Use step 2 only to bring a trip into the journal for the first
time. See the [memento lifecycle contract](contracts/memento-lifecycle.md#3-transition-table).

Unpublishing works the same way: step a memento back to `authored`, rebuild, and
manifest reconciliation removes it from the artifact.

---

## Route A — publish the production catalog with GitHub Actions

1. **Fork** the repository.
2. Actions tab → enable workflows on the fork.
3. Settings → Pages → Source → **GitHub Actions**.
4. Actions → **Publish production catalog to GitHub Pages** → Run workflow. (Forking creates no
   push, so the first run must be manual.)
5. The workflow publishes every catalog entry under
   [`publication/journeys/`](https://github.com/azusachino/felicia/tree/main/publication/journeys).
   Each journey directory contains one `journey.json`, selected stops, authored
   mementos, a rounded public route, and public image derivatives.
   Add future journeys by adding the same file set and one entry to
   `publication/journeys/catalog.json`; the publisher discovers them without
   source-code edits.
6. Verify locally with `make pages-preview` (`http://localhost:8082`). The
   preview builds into a disposable workspace of its own
   (`.felicia/pages-preview.sqlite` and `.felicia/pages-preview-media/`), which
   it empties on every run. It never reads or resets your authoring journal or
   original media. Pointing `PAGES_DB` or `PAGES_MEDIA_ROOT` elsewhere uses that
   location as given; the preview only empties a directory it can prove it owns.
7. Push to `main`; the workflow rebuilds and deploys.

The base path is derived from the repository name, so nothing needs editing for
a fork. The workflow needs no secrets, database, or credentials.

Site identity (title, design, accent) comes from the database and is authored in
the GUI; the Pages build uses the Atlas default when no site settings are stored.

---

## Base path reference

`BASE_PATH` must match the URL the site is served under; a mismatch produces a
page that loads with every asset and `.json` request returning 404.

| Site repository   | URL                                   | `BASE_PATH`    |
| ----------------- | ------------------------------------- | -------------- |
| `my-travels`      | `https://<you>.github.io/my-travels/` | `/my-travels/` |
| `<you>.github.io` | `https://<you>.github.io/`            | `/`            |
| custom domain     | `https://your.domain/`                | `/`            |

## Troubleshooting

| Symptom                                                 | Cause                                                                                                |
| ------------------------------------------------------- | ---------------------------------------------------------------------------------------------------- |
| Page loads blank, assets 404                            | `BASE_PATH` does not match the deployed URL — rebuild step 4                                         |
| Settings → Pages has no Source selector                 | private repository on a free plan, or you are not a repository admin                                 |
| The target branch is missing from the list              | it does not exist yet — push the artifact first, then set the source                                 |
| Some files are missing from the live site               | `.nojekyll` was not deployed                                                                         |
| The site keeps redirecting elsewhere                    | check that `CNAME` and the Pages Custom domain setting match your intended domain                             |
| A journey is absent from the site                       | it has no `published` mementos                                                                       |
| `site-build`: `resolve site settings: entity not found` | the journal is empty — author or import something first, or `DATABASE_PATH` points at the wrong file |

## Not available yet

Deploying from the GUI with one action, and the deployed-URL confirmation that
goes with it, are planned in
[FELICIA-ADMIN-02 M3](roadmap/admin-gui-v2-epic.md). Until then step 4 is a
manual `git push`.
