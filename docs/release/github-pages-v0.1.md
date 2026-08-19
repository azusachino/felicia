---
title: "GitHub Pages v0.1 Release"
status: "active"
date: "2026-07-18"
---

# Production GitHub Pages publication

This is the repository's first production journey publication. The Pages workflow
runs the **real publication pipeline** against the production journey catalog:
`publication/journeys` → journey packages → SQLite import →
`felicia-cli static compile` → merged with the built SPA and deployed.

The catalog is data-driven. Add a new directory under `publication/journeys/`
with the same `journey.json`, `stops.json`, `mementos.json`, `route.gpx`, and
public `media/` set, then add its path and identity to
`publication/journeys/catalog.json`. The publisher discovers every manifest
entry; adding a journey does not require a Go, TypeScript, or workflow change.

## Deployment topology (why the repo stays clean)

The workflow uses GitHub's artifact-based Pages deployment
(`upload-pages-artifact` → `deploy-pages`): the site is built on an ephemeral
CI runner and handed to the Pages hosting environment directly. **No build
output is ever committed** — there is no `gh-pages` branch, no generated
files in git history, and `dist/`/`bin/`/`.felicia/` are gitignored locally.

The deployed site publishes only the sanitized journeys in the production catalog. Original photos,
the source GPX, drafts, and the local SQLite journal remain on the author's
machine; the repository carries only curated text, a rounded route, and public
image derivatives.
The compiled artifact contains published-only content and is reconciled
against `api/v1/manifest.json` on every compile, so unpublished or deleted
content never lingers in a reused output directory.

## Local release check

```bash
make web-install
BASE_PATH=/felicia/ mise exec -- uv run python scripts/felicia.py publish
BASE_PATH=/felicia/ make site-verify
make pages-workflow-validate
make fork-smoke
make pages-preview
```

The preview is the one-click local path: it builds packages from the production
catalog, builds the SPA, compiles the SQLite publication, and serves the combined
artifact from `apps/felicia-public-site/dist`. The underlying CLI commands
remain independently usable for automation.

For a project-site path, use the repository name as the base path:

```bash
BASE_PATH=/felicia/ mise exec -- uv run python scripts/felicia.py publish
```

## Fork workflow

1. Fork the repository.
2. Enable GitHub Pages with **GitHub Actions** as the source.
3. Push to the fork's `main` branch or manually dispatch **Publish production catalog
   to GitHub Pages**.
4. The workflow derives the project-site path from
   `github.event.repository.name`; no owner or repository name is hardcoded.
5. Review the deployment URL and test `/`, `#cabinet`, `#techo`, and `#atlas`.

The workflow requires no database, Valkey, S3/R2 credentials, API server, or
private secrets. It only publishes files produced by the repository build.

## Public artifact contract

The reader loads:

```text
/api/v1/journeys.json
/api/v1/journeys/<journey-uuid>.json
/api/v1/journeys/<journey-uuid>/mementos.json
```

Media is sanitized again by the static compiler before it reaches the artifact;
the checked-in source images are already public derivatives.

## v0.1 boundary

Included:

- public SPA with four switchable designs;
- static JSON projections;
- the curated journeys and their published mementos;
- project-site base-path support;
- local CLI/Compose preview;
- fork-safe GitHub Actions build and deployment.

The private admin server and connected-source ingestion remain separate local
authoring concerns.
