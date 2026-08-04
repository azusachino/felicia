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
admin, published-only artifact, ADR-0025); this epic closes the GUI gaps:
today the GUI exposes no compile or deploy action at all, no design/style
selection, and no resource-upload path.

## Design constraints

- **Artifact stays the only thing that leaves the machine.** Every deploy
  path ships the compiled `dist/` (published-only JSON + safe media + SPA);
  drafts, originals, SQLite, and the admin app itself never do.
- **`site.json` goes through the shared publication boundary.** Site-level
  settings become one more projection produced by `publication/` and served
  identically by the live API and the static compiler (parity-tested, in
  the manifest) — never invented per surface.
- **No credentials inside felicia.** GitHub deployment reuses the author's
  existing git credentials (SSH agent / credential helper) by shelling out
  to `git`; deployment confirmation polls the public site URL for the new
  build stamp instead of calling authenticated GitHub APIs.
- **Browser sandbox honesty.** A web GUI cannot read local filesystem
  paths; files enter the workspace as uploaded bytes over localhost HTTP
  (multipart), with a paste-a-path fallback handled server-side. No
  server-side filesystem-browser API.

## Milestones and child tasks

### M0 — Offline deployment to a local folder (landed)

The smallest end of the loop: build from the GUI and click a local link.

- **02.0a Site page + compile action.** New `#/site` route ("Site &
  Deploy" nav entry): a Build button calling the existing
  `POST /api/admin/compile`; `out_dir` defaults server-side to
  `site.out_dir` (default `.felicia/site`); the BuildReport
  (journeys/mementos/media/removed) renders in the page.
- **02.0b Built-in preview server.** The admin process serves the compiled
  site on a second local port (`site.preview_port`, default 8081): a union
  file server over the compile output (`api/v1/` + media) and a pre-built
  public SPA directory (`site.spa_dist`, default `apps/web-public/dist`).
  The Site page shows the preview URL as a clickable link and warns when
  the SPA build is missing.
- **02.0c Verification + docs.** The closed-loop E2E grows a Site-page
  step (build → report visible → preview port serves the compiled JSON);
  docs-sync per `AGENTS.md`.

Acceptance: with no CLI use, an author can press Build in the GUI and open
`http://localhost:<preview_port>/` to read the site exactly as it would
deploy. A journey published in the editor appears there after the next
build; unpublished content never does.

### M1 — Authoring controls and polish (landed; user-tested gaps)

Direct gaps found while operating the M0 build (2026-07-19 review):

- **02.1a Unpublish.** The editor's lifecycle actions become bidirectional:
  a published memento can step back to `authored` (the upsert already
  accepts any target state; only the GUI is one-way today). Rebuilding
  after an unpublish removes it from the artifact via the existing
  manifest reconciliation.
- **02.1b Delete a memento.** New `DELETE /api/admin/mementos/{id}` with a
  GUI confirm step. Hard delete (photos cascade); the importer's
  field-scoped upsert may re-seed source-derived mementos on a future
  import — the confirm copy says so.
- **02.1c Discard a stop candidate.** Candidate identity is source-stable,
  so a hard delete would resurrect on the next intake plan; "delete" in
  the inbox therefore maps to the existing `ignored` review state, and the
  UI presents it as discard with that explanation.
- **02.1d Build shortcut on journey detail.** A compact "Build & preview"
  action on the journey page (same compile endpoint), so publish → deploy
  happens without switching pages; the Site page stays the home of
  site-level settings.
- **02.1e Topbar spacing.** Separate the Refresh button from the profile
  icon (mis-click risk flagged in review).

### M2 — Site identity: design pick, style, `site.json` (landed)

- **02.2a `site.json` contract.** Journal-level site settings (site title
  and description, the single active design, default language, default
  theme, accent) stored in the DB, exposed at
  `GET/PUT /api/admin/site-settings`, and projected to
  `/api/v1/site.json` by the shared publication layer (live/static parity,
  manifest-tracked). Absent file = current demo behavior, so the example
  deployment is unaffected.
- **02.2b Reader consumption — one design per deployment.** A deployed
  site presents exactly one design (user decision, 2026-07-19): the reader
  boots into the design named by `site.json` (default v1), the switcher is
  not rendered, and the other designs are unreachable. First pass is a
  runtime lock (unused design code still ships in the bundle, invisible to
  visitors); excluding it from the build is a later optimization. Accent
  token wired for the designs that already use CSS variables (v1/v4
  first).
- **02.2c Site page: design picker + site info + style controls.** The
  four designs as selectable cards (v1 preselected), site name/description
  fields, and the style fields, saved through 02.2a — all above the
  existing Build section so the order reads configure → build → preview.

Deliberately bounded: this is site-level _tokens_, not a theming engine
(fonts, per-design palettes, and layout knobs stay out, like the dynamic
form engine in ADMIN-01).

### M3 — GitHub Pages deployment, URL confirmation, takedown (planned)

- **02.3a Deploy target settings.** Git remote + branch (default
  `gh-pages`) and derived base path / public URL, stored with the other
  settings; one-time GitHub-side setup (enable Pages, deploy-from-branch)
  documented in the GUI.
- **02.3b Deploy action.** Deploy = compile with the target base path →
  commit the artifact onto the target branch (orphan; `main` keeps zero
  generated files) → `git push` using the author's own credentials.
- **02.3c Link confirmation.** Compile stamps a build id into the
  manifest; after pushing, the GUI polls
  `https://<owner>.github.io/<repo>/api/v1/manifest.json` until the stamp
  matches, then shows the live URL. No tokens involved.
- **02.3d Site takedown.** A guarded action that deploys an empty artifact
  to the same target (the site goes blank through the same mechanism, no
  new concepts), plus guidance for fully disabling Pages on GitHub. The
  offline target needs no equivalent: rebuilding after unpublishing is
  already the takedown.

Deferred within M3 (revisit on demand): triggering GitHub Actions with a
user-supplied token, and arbitrary rsync/scp targets (conflicts with
ADR-0025's deferral of self-hosted serving).

### M4 — Resource intake from the GUI (planned)

Covers the "browse local files from the Import & preview section" gap.

- **02.4a Photo upload.** Multi-file picker + drag-and-drop in the editor's
  photo section → multipart upload → server stores originals in the media
  root, computes content hashes, creates photo rows. EXIF stripping stays
  where it is today (public-derivative time), so import-alignment metadata
  survives. A paste-a-path fallback lets the server ingest a large local
  folder without pushing bytes through the browser.
- **02.4b GPX / route upload.** The journey detail's import section gains a
  local file picker for route files, which land in the workspace and reuse
  the existing route compilation path (same code as `sync-route`).
- **02.4c Journey package import.** Upload a package zip → server runs
  `package validate` → dry-run diff preview in the GUI → confirm to
  `import --apply`. Field-scoped importer guarantees hold unchanged.
- **02.4d Workspace settings panel.** Media root, upstream sources
  (Dawarich/Immich), and defaults surfaced in the GUI and persisted to
  `felicia.toml` (env overrides keep precedence).
- **02.4e AI memento artwork generator.** Optional local AI agent connection
  in the memento editor: when a memento lacks physical photos, the author can
  trigger a local AI agent to generate ticket artwork based on kind/title/place
  and attach it after review (aligned with ADR-0024).

## Open decisions (to settle before their milestone starts)

1. M3: does the deploy button stop at push-and-confirm (current plan) or
   optionally trigger GitHub Actions with a stored token?
2. M4: upload-only vs upload + paste-a-path dual track (current plan says
   dual).
3. M2: the one-design rule ships as a runtime lock first; excluding unused
   designs from the artifact build (bundle slimming) is a possible later
   optimization, not v2 scope.
4. M1: whether a deleted memento leaves a tombstone that blocks re-seeding
   by a future import, or deletion is plain and re-import may restore
   source-derived rows (current plan: plain delete, explained in the
   confirm dialog).

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
