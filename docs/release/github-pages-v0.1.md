---
title: "GitHub Pages v0.1 Release"
status: "proposed"
date: "2026-07-17"
---

# GitHub Pages v0.1 Release

This is the smallest forkable release path for the current public design demo.
It publishes the checked-in fixture projection and is intentionally not the
SQLite compiler or the self-hosted authoring server.

## Local release check

```bash
make static-publish
make pages-workflow-validate
make fork-smoke
make pages-preview
```

The preview only serves the existing combined artifact from
`apps/web-public/dist`; it does not import or compile data. Build the SPA and
compile the static publication separately, then use `make pages-preview` to
inspect the result.

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
