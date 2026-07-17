---
title: "GitHub Pages Design Demo"
status: "exploratory"
date: "2026-07-17"
---

# GitHub Pages Design Demo

This is an implementation experiment for Felicia's first portable reader. It
tries the existing public designs against one generated static data projection:

- `#collection` — collection/index-oriented presentation;
- `#techo` — memento/story-oriented presentation;
- `#atlas` — map/index-oriented presentation.

The experiment does not select a design or settle the application architecture.
The three surfaces are alternative readers of the same output and remain
available for product review.

## Reproduction

The original fixture-only reproduction is retired. From the repository root,
prepare a package and compile it with the real CLI:

```bash
felicia-cli package validate .felicia/inbox/journey.zip
felicia-cli import --db .felicia/felicia.sqlite --media-root .felicia/media --apply .felicia/inbox/journey.zip
felicia-cli static compile --db .felicia/felicia.sqlite --media-root .felicia/media --out apps/web-public/dist
```

`static-publish` only arranges and verifies `apps/web-public/dist`; it does not
commit or push. A fork's normal Git review workflow remains the publication
approval boundary.

For a local browser check using a compiled SQLite publication:

```bash
make pages-preview
# open http://localhost:8082
make pages-down
```

The preview uses Python's standard `http.server` in a disposable Compose
profile. It mounts `apps/web-public/dist` read-only and has no API, database,
cache, or cloud dependency. Caddy remains the server for the self-hosted/shared
runtime, where reverse-proxy behavior also needs to be tested.

The retired fixture command previously performed two operations:

1. `scripts/build_static_demo.py` converted the checked-in fixture at
   `scripts/data.json` into a static read projection under
   `apps/web-public/public/api/v1`.
2. Vite builds the public SPA with the project-site base path. The Pages workflow
   uses the repository name as that path automatically.

The generated API is ignored by Git and is intentionally treated as build
output. The source fixture remains the inspectable input for this experiment;
it is not evidence that JSON is Felicia's canonical data model.

The public read contract is extensionful in both publication modes:

```text
/api/v1/journeys.json
/api/v1/journeys/<id>.json
/api/v1/journeys/<id>/mementos.json
```

GitHub Pages reads these files directly. The Go server exposes equivalent
`.json` aliases alongside its existing extensionless routes, so a frontend can
switch between static and server publication without changing its read paths.

## Evidence

Run on 2026-07-17 in the local development environment:

| Check                                   | Result                                                  |
| --------------------------------------- | ------------------------------------------------------- |
| retired fixture build                   | passed; Vite transformed 139 modules                    |
| retired fixture project path build      | passed; project base path embedded                      |
| `make web-check`                        | passed; 0 Svelte diagnostics, ESLint and Prettier clean |
| `uv run --group dev ruff check scripts` | passed                                                  |
| retired fixture artifact verifier       | passed                                                  |
| journeys in static index                | 9                                                       |
| journey detail files                    | 9                                                       |
| demo media copied to `dist/`            | 3 files                                                 |
| generated artifact size                 | 4.2 MB locally                                          |

The output contains one index file, one detail file per journey, one memento
file per journey, the SPA bundle, and the three fixture images. The frontend
resolves API and media URLs through `import.meta.env.BASE_URL` and uses `.json`
files when no `VITE_API_BASE` is configured. With `VITE_API_BASE`, it keeps
extensionless server API paths. The same bundle can therefore be served at `/`
locally or at `/<repository>/` as a GitHub Pages project site.

## Decision evidence

Current result: **retain as a candidate v1 publication path; do not accept yet**.

The experiment supports these narrow claims:

- the current public designs can be built into one static artifact;
- a project-site base path can be carried through API and media URLs;
- static and server publication can share the same `.json` read paths;
- the public reader can consume a generated projection without an API server;
- local fixture media can be included in the artifact for a design review.

It does not yet support these claims:

- SQLite-to-static compilation works through `felicia-cli`; the local preview
  now consumes the same compiled artifact;
- GPX import and route normalization are wired into publication;
- a local filesystem media root is safely transformed into public derivatives;
- GitHub Actions can deploy this repository; the workflow is present but has
  not run on the protected branch;
- external map styles and tile providers are suitable for production traffic;
- the best reader design is known.

Therefore the provisional decision is to keep the Pages build as an isolated
publication adapter and use it for design feedback. The next discriminating
experiment is to replace the fixture reader with a SQLite reader while keeping
the generated public contract unchanged. If that fails, revise the adapter or
the contract before changing the domain model.

## Workflow boundary

`.github/workflows/pages.yml` builds on pushes to `main` and on manual dispatch,
then uploads and deploys the static `apps/web-public/dist` artifact. The local
`pages-preview` Compose profile provides the same artifact check without a
remote deployment. The workflow requires
no database, Valkey, object-storage credentials, or running API. Self-hosted
server mode remains a separate runtime path; `deploy/Caddyfile` and Compose are
not dependencies of this Pages experiment.

## Open gaps before calling this v1

- run the workflow after merge and record the actual Pages URL;
- inspect all three designs in a browser at the project-site base path;
- decide whether route geometry should be GeoJSON, a Felicia-specific shape, or
  both in the public projection;
- compile from SQLite and preserve source GPX provenance;
- define media size limits, derivative generation, cache headers, and a safe
  policy for originals versus public assets;
- measure a 100-journey fixture and lazy-load journey details;
- add accessibility, broken-media, and offline/rebuild checks.
