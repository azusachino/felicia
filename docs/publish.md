# Publish your own site

> For someone who wants to run **their own** travel journal on GitHub Pages.
> For contributor setup see [`setup.md`](setup.md); for the delivery status of
> this path see [`roadmap/user-journey.md`](roadmap/user-journey.md).

## What leaves your machine

Only the compiled `dist/` — published mementos, EXIF-stripped public image
derivatives, and rounded geometry. Your SQLite journal, drafts, and original
photos stay local and are gitignored
([ADR-0025](adr/0025-static-and-self-hosted-modes.md),
[ADR-0026](adr/0026-local-first-media-and-blob-storage.md)).

There is **no hosted admin**. The authoring GUI runs on `127.0.0.1` on your own
machine and is never deployed; GitHub only ever holds the static site.

## Two routes

|                            | **A — CI build**                        | **B — Local authoring** (recommended) |
| -------------------------- | --------------------------------------- | ------------------------------------- |
| Content lives in           | JSON files committed to the repo        | local SQLite, never committed         |
| Authoring interface        | a text editor                           | the admin GUI                         |
| Built by                   | GitHub Actions                          | your machine                          |
| Site title / design choice | not available (defaults to cartography) | set in the GUI                        |
| Privacy                    | journal content is in git history       | only `dist/` leaves the machine       |
| Needs a fork               | yes (for `.github/workflows/`)          | no — any checkout plus an empty repo  |

Route A is how the project's own demo site is published. Route B is the
intended path for a personal journal.

---

## Route B — local authoring

### 1. Get the code and a site repo

```bash
git clone https://github.com/azusachino/felicia.git ~/felicia
```

Then create an **empty public repository** on GitHub to host the site (for
example `my-travels`). It will contain nothing but the built site. A fork works
equally well; nothing in this route depends on the checkout's git remote.

> GitHub Pages on a **private** repository requires a paid plan. The site repo
> holds only what visitors already see, so public is normally the right choice.

Prerequisite: **nix** with flakes enabled. Everything else comes from the flake
(`make` enters it automatically).

### 2. Bring in a trip

The admin GUI has no journey-creation or file-upload control yet (its Import
buttons talk to Dawarich and Immich, not to your disk), so a trip enters through
the CLI. This step also creates the journey:

```bash
make journey-local GPX=~/trip/route.gpx PHOTOS=~/trip/photos
```

Each trip gets its own identity by default: the journey id and slug are
derived from the GPX track's own bytes, so re-running the same trip lands on
the same journey (idempotent — no duplicate) while a different trip always
gets a distinct id, slug, and workspace directory. Name the trip yourself with
`SLUG=` / `TITLE=`, or pin the id/journal/workspace explicitly with `JOURNEY=`
/ `JOURNAL=` / `WORKSPACE=`:

```bash
make journey-local GPX=~/trip/route.gpx PHOTOS=~/trip/photos SLUG=kyoto-2026 TITLE="Kyoto 2026"
```

That writes an editable workspace to `.felicia/local-journey/<slug>` (printed
by the command): `journey.json`, `stops.json`, `mementos.json`, plus the
planner's `plan.json`. Pointing a workspace that already holds a _different_
journey at a new trip is a loud error, never a silent overwrite. Edit the
titles and selections, then package and import it — reuse the workspace path
the command printed:

```bash
uv run python scripts/local_journey.py package --workspace .felicia/local-journey/kyoto-2026
./bin/felicia-cli import --db .felicia/local.sqlite --media-root .felicia/media \
  --apply .felicia/local-journey/kyoto-2026/journey.zip
```

Repeat with a different GPX/`SLUG` for a second trip — it imports alongside
the first rather than replacing it.

Two things decide whether this produces anything:

- **Stops need dwell.** A stop candidate is 20+ minutes within 250 m. A track
  that never stops — a train ride, a drive — yields no stops, and therefore no
  mementos. `preprocess` reports success either way, so check the counts in
  `stops.json` rather than trusting the exit code.
- **Local photos have no timestamps.** EXIF is only read through Immich; a photo
  folder is timestamp-less to felicia and cannot be attached to the track. Supply
  a JSONL sidecar and pass it as `SIDECAR=`:

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

- admin GUI — `http://127.0.0.1:5174/`
- site preview — `http://127.0.0.1:8081/`

Confirm stop candidates in the intake inbox, write essays in the memento editor,
then advance each memento `draft → authored → published`. Imported mementos
arrive as `draft`. **Only `published` mementos reach the artifact**, and a
journey with no published mementos is not published at all — a compile that
reports `Journeys: 0` usually means nothing has been published yet.

On the **Site & Deploy** page set the site title and description, pick one of
the four designs, and set the default language, theme, and accent colour. A
deployed site presents exactly one design.

Defaults: journal `.felicia/local.sqlite`, media `.felicia/media`, compile
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
DATABASE_PATH=~/felicia-data/local.sqlite MEDIA_ROOT=~/felicia-data/media \
  BASE_PATH=/my-travels/ make site-build
```

Equivalent from the GUI: set the Site & Deploy output directory to
`apps/felicia-public-site/dist` and press **Build** — but the SPA must already have been
built with the correct `BASE_PATH`.

!!! warning "`make static-build` is not this"

    `make static-build` / `make static-publish` compile the **fixture demo**
    (`scripts/data.json`), not your journal. They exist as a design-demo helper.

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
rsync -a --delete --exclude .git apps/felicia-public-site/dist/ ~/my-travels-ops/
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

Repeat steps 2 to 4, then re-run the `rsync` + commit + push from step 5. The
Pages settings never need to change.

Unpublishing works the same way: step a memento back to `authored`, rebuild, and
manifest reconciliation removes it from the artifact.

---

## Route A — CI build

1. **Fork** the repository.
2. Actions tab → enable workflows on the fork.
3. Settings → Pages → Source → **GitHub Actions**.
4. Actions → **GitHub Pages design demo** → Run workflow. (Forking creates no
   push, so the first run must be manual.)
5. Replace the content in [`examples/preview/local-journey/`](https://github.com/azusachino/felicia/tree/main/examples/preview/local-journey):
   `workspace.json` lists the journey directories; each holds `journey.json`
   (requires `id` and `journal_id`), `stops.json`, `mementos.json`, and
   optionally `route.gpx`. Photos are committed to the repo and referenced by
   `media[].path`, resolved relative to the journey directory and then to the
   repository root.
6. Verify locally with `make pages-preview` (`http://localhost:8082`).
7. Push to `main`; the workflow rebuilds and deploys.

The base path is derived from the repository name, so nothing needs editing for
a fork. The workflow needs no secrets, database, or credentials.

Site identity (title, design, accent) comes from the database and is authored in
the GUI, so a route A site uses the defaults.

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
| Page loads blank, assets 404                            | `BASE_PATH` does not match the deployed URL — rebuild step 3                                         |
| Settings → Pages has no Source selector                 | private repository on a free plan, or you are not a repository admin                                 |
| The target branch is missing from the list              | it does not exist yet — push the artifact first, then set the source                                 |
| Some files are missing from the live site               | `.nojekyll` was not deployed                                                                         |
| The site keeps redirecting elsewhere                    | a stray `CNAME` file, or a Custom domain value that was saved by mistake                             |
| A journey is absent from the site                       | it has no `published` mementos                                                                       |
| `site-build`: `resolve site settings: entity not found` | the journal is empty — author or import something first, or `DATABASE_PATH` points at the wrong file |

## Not available yet

Deploying from the GUI with one action, and the deployed-URL confirmation that
goes with it, are planned in
[FELICIA-ADMIN-02 M3](roadmap/admin-gui-v2-epic.md). Until then step 4 is a
manual `git push`.
