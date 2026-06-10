# Plan — felicia

Methodology: **design → spec → TDD → implementation**, unhurried (~6-month horizon).
Build vertical slices smallest → biggest; the importer core is the TDD spine.

## Current phase

**Research / spec — complete.** Design and importer spec written; project initialized
(tooling + agent infra, no application code).

## Roadmap

1. **Design & spec** ✅ — `design.md`, `importer-spec.md`, workflow research, decisions (rosemary).
2. **Project init** ✅ — layout, mise/nix/Makefile, agent infra.
3. **Importer TDD core** — pure functions first (gpx parse + simplify, EXIF, clustering, OCR
   mapping, field-scoped no-clobber upsert on memrepo). Fixtures, no network. (importer-spec §7/§11)
4. **Sources** — Immich PhotoSource (recorded HTTP fixtures), Dawarich RouteSource, objectstore
   (image resize + EXIF strip + S3 upload), Postgres repository + migrations.
5. **End-to-end import** — `waypoints import`/`sync`/`export` wired; manual-file path first,
   then Immich, then Dawarich.
6. **API** — read endpoints for journeys/tickets; serve SPA data.
7. **Public SPA** — dark Mapbox map, orange routes, ticket markers, animated detail view.
8. **Admin SPA** — auth, essay editor, photo curation (drag-drop + Immich picker), animation picker.
9. **Deploy** — docker-compose on the Pi, Cloudflare Tunnel, R2 wiring.

## Open decisions

- Ticket-open animation style (flip / morph / tear) — prototype in step 7.
