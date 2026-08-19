---
title: "GitHub Pages v0.1 Release"
status: "active"
date: "2026-07-18"
---

# GitHub Pages v0.1 Release

This is the smallest forkable release path for the public site. The Pages
workflow runs the **real publication pipeline** against the example workspace:
`examples/preview/local-journey` → journey packages → SQLite import →
`felicia-cli static compile` → merged with the built SPA and deployed. The
legacy fixture path survives only as the `make static-publish` demo helper.

## Deployment topology (why the repo stays clean)

The workflow uses GitHub's artifact-based Pages deployment
(`upload-pages-artifact` → `deploy-pages`): the site is built on an ephemeral
CI runner and handed to the Pages hosting environment directly. **No build
output is ever committed** — there is no `gh-pages` branch, no generated
files in git history, and `dist/`/`bin/`/`.felicia/` are gitignored locally.

The deployed demo publishes the checked-in **example data only**. Personal
journal content is not meant to live in this repository: originals stay on
the author's machine (local-first), and a personal site is a fork or private
repository that carries its own workspace data through the same workflow.
The compiled artifact contains published-only content and is reconciled
against `api/v1/manifest.json` on every compile, so unpublished or deleted
content never lingers in a reused output directory.

## Local release check

```bash
make static-publish
make pages-workflow-validate
make fork-smoke
make pages-preview
```

The preview is the one-click local path: it imports ZIPs found in
`.felicia/inbox`, builds the SPA, compiles the SQLite publication, and serves
the combined artifact from `apps/felicia-public-site/dist`. The underlying CLI commands
remain independently usable for automation.

For a project-site path, use the repository name as the base path:

```bash
BASE_PATH=/felicia/ make static-publish
```

## Fork workflow

1. Fork the repository.
2. Enable GitHub Pages with **GitHub Actions** as the source.
3. Push to the fork's `main` branch or manually dispatch **GitHub Pages design
   demo**.
4. The workflow derives the project-site path from
   `github.event.repository.name`; no owner or repository name is hardcoded.
5. Review the deployment URL and test `/`, `#collection`, `#techo`, and `#atlas`.

The workflow requires no database, Valkey, S3/R2 credentials, API server, or
private secrets. It only publishes files produced by the repository build.

## Public artifact contract

The reader loads:

```text
/api/v1/journeys.json
/api/v1/journeys/<journey-uuid>.json
/api/v1/journeys/<journey-uuid>/mementos.json
```

Media in this v0.1 fixture is copied into the artifact root. The local fixture
mapping is temporary; safe local-filesystem derivatives are a later task.

## v0.1 boundary

Included:

- public SPA with four switchable designs;
- static JSON projections;
- fixture media;
- project-site base-path support;
- local CLI/Compose preview;
- fork-safe GitHub Actions build and deployment.

Deferred:

- GitHub workflow input wiring for SQLite-backed publication;
- agent CLI import/diff over journey packages;
- GPX-to-canonical route compilation;
- local filesystem media derivatives and EXIF stripping;
- the private admin server and connected-source ingestion.
