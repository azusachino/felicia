---
title: "Epic: GUI site configuration and deployment (admin GUI v2)"
status: "in-progress"
date: "2026-07-19"
---

# Epic FELICIA-ADMIN-02 — GUI site configuration and deployment

**Epic key:** `FELICIA-ADMIN-02`
**Status:** M0–M2 landed; M3–M4 planned. Delivery status is tracked in
[`../roadmap.md`](../roadmap.md); this document records scope and
acceptance only.
**Goal:** the author configures resources, picks the public design, builds,
and deploys — entirely from the local admin GUI, with a clickable URL to
verify the result. The CLI remains available but is no longer required for
the routine authoring-to-deployment loop.

## The target experience (user-stated)

> Start the admin locally and configure everything offline. Compile is the
> build command. Only the static assets go to GitHub — the admin never
> lives on a server. After deploying, the GUI hands back a real link to
> verify.

The architecture already guarantees the privacy half of this (local-only
admin, published-only artifact, ADR-0025); this epic closes the remaining
GUI gaps: no compile/deploy action, no design/style selection, no
resource-upload path.

## Design constraints

- **Artifact stays the only thing that leaves the machine.** Every deploy
  path ships the compiled `dist/`; drafts, originals, SQLite, and the
  admin app itself never do.
- **`site.json` goes through the shared publication boundary.**
  Site-level settings are one more projection from `apps/felicia-publication/`, served
  identically by the live API and the static compiler (parity-tested, in
  the manifest).
- **No credentials inside felicia.** GitHub deployment reuses the
  author's existing git credentials by shelling out to `git`; deployment
  confirmation polls the public site URL for the new build stamp rather
  than calling authenticated GitHub APIs.
- **Browser sandbox honesty.** A web GUI cannot read local filesystem
  paths; files enter as uploaded bytes over localhost HTTP (multipart),
  with a paste-a-path fallback handled server-side.

## Milestones and child tasks

### M0 — Offline deployment to a local folder (landed)

| Task                             | Status | Outcome                                                                        |
| -------------------------------- | ------ | ------------------------------------------------------------------------------ |
| 02.0a Site page + compile action | done   | `#/site` → `POST /api/admin/compile`; `out_dir` defaults to `site.out_dir`.    |
| 02.0b Built-in preview server    | done   | `site.preview_port` (default 8081): compile output + the pre-built public SPA. |
| 02.0c Verification + docs        | done   | E2E gained a build→preview-JSON step; docs-sync per `AGENTS.md`.               |

### M1 — Authoring controls and polish (landed; user-tested gaps)

| Task                                   | Status | Outcome                                                                               |
| -------------------------------------- | ------ | ------------------------------------------------------------------------------------- |
| 02.1a Unpublish                        | done   | Bidirectional (published ↔ authored); rebuild removes it via manifest reconciliation. |
| 02.1b Delete a memento                 | done   | `DELETE /api/admin/mementos/{id}` + confirm; hard delete, cascades photos.            |
| 02.1c Discard a stop candidate         | done   | Maps to `ignored` state (avoids resurrection on next intake plan).                    |
| 02.1d Build shortcut on journey detail | done   | "Build & preview" shortcut reuses the compile endpoint.                               |
| 02.1e Topbar spacing                   | done   | Refresh separated from the profile icon.                                              |

### M2 — Site identity: design pick, style, `site.json` (landed)

| Task                                                 | Status | Outcome                                                                                                      |
| ---------------------------------------------------- | ------ | ------------------------------------------------------------------------------------------------------------ |
| 02.2a `site.json` contract                           | done   | Title/description/design/language/theme/accent via `GET/PUT /api/admin/site-settings` → `/api/v1/site.json`. |
| 02.2b Reader consumption — one design per deployment | done   | Deployed site boots into `site.json`'s design (default v1); switcher removed.                                |
| 02.2c Site page: design picker + site info + style   | done   | Four design cards + site info + style fields, saved through 02.2a.                                           |

Bounded to site-level _tokens_, not a theming engine (fonts, per-design
palettes, layout knobs stay out).

### M3 — GitHub Pages deployment, URL confirmation, takedown (planned)

- **02.3a Deploy target settings.** Git remote + branch (default
  `gh-pages`) and derived base path/URL; one-time GitHub Pages setup
  documented in the GUI.
- **02.3b Deploy action.** Deploy = compile with the target base path →
  commit onto the target branch (orphan; `main` keeps zero generated
  files) → `git push` with the author's own credentials.
- **02.3c Link confirmation.** Compile stamps a build id; the GUI polls
  `https://<owner>.github.io/<repo>/api/v1/manifest.json` until it
  matches, then shows the live URL.
- **02.3d Site takedown.** A guarded action deploys an empty artifact to
  the same target, plus guidance for disabling Pages on GitHub. The
  offline target's equivalent is already covered: rebuild after
  unpublishing.

Deferred within M3: GitHub Actions via a stored token, and arbitrary
rsync/scp targets (conflicts with ADR-0025's self-hosted deferral).

### M4 — Resource intake from the GUI (planned)

- **02.4a Photo upload.** Multi-file picker/drag-and-drop → multipart
  upload → server stores originals, hashes them, creates photo rows;
  EXIF stripping stays where it is today. Paste-a-path fallback ingests
  large local folders without browser upload.
- **02.4b GPX / route upload.** Local file picker for route files on the
  journey detail's import section, reusing the `sync-route` compilation
  path.
- **02.4c Journey package import.** Upload a package zip → server runs
  `package validate` → dry-run diff preview in the GUI → confirm to
  `import --apply`. Field-scoped importer guarantees hold unchanged.
- **02.4d Workspace settings panel.** Media root, upstream sources
  (Dawarich/Immich), and defaults surfaced in the GUI and persisted to
  `felicia.toml` (env overrides keep precedence).
- **02.4e AI memento artwork generator.** Optional local AI agent in the
  memento editor generates ticket artwork (kind/title/place) for a
  photo-less memento, attached after review (ADR-0024).

## Open decisions (to settle before their milestone starts)

1. M3: does the deploy button stop at push-and-confirm (current plan) or
   optionally trigger GitHub Actions with a stored token?
2. M4: upload-only vs upload + paste-a-path dual track (current plan says
   dual).
3. M2: the one-design rule ships as a runtime lock first; excluding
   unused designs (bundle slimming) is a possible later optimization, not
   v2 scope.
4. M1: whether a deleted memento leaves a tombstone blocking re-seeding,
   or deletion is plain and re-import may restore it (current plan:
   plain delete, explained in the confirm dialog).

## Explicitly out of scope

Theming engine, registry-driven dynamic form engine (still ADMIN-01's
deferral), media byte processing changes (EXIF pipeline stays put),
self-hosted always-on serving (ADR-0025), multi-user/auth, any credential
storage inside felicia.

## Estimates (working days, at the current orchestration cadence)

| Milestone | Estimate |
| --------- | -------- |
| M0        | landed   |
| M1        | 1–1.5    |
| M2        | 1.5–3    |
| M3        | 1.5–2    |
| M4        | 2–3.5    |
