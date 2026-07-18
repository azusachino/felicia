# Epic FELICIA-ADMIN-01 — Minimal authoring GUI (admin GUI MVP)

> Phase 2 of the selected end-to-end journey
> ([user-journey.md](user-journey.md)): upgrade stage 3 (authoring) from the
> file/CLI path to the smallest closed GUI loop in `apps/web-admin`. Server
> API and data contract already exist; this epic is almost entirely frontend
> plus one small server endpoint.

## Design constraints (decided up front)

- **One editor, presentation-agnostic.** The editor edits the canonical
  contract (`kind` + `kind_data`, validated server-side against
  `core/kinds/*.yaml`). The public designs v1–v4 are downstream projections;
  nothing in the editor may depend on any of them.
- **Kind source of truth is the registry.** The editor's kind list and field
  hints come from `GET /api/admin/templates` (the embedded `core/kinds`
  registry). MVP hardcodes two forms (`transit`, `goods`) but their field
  sets must match the registry; the dynamic form engine stays deferred.
- **No new dependencies.** Svelte 5 + hash-based routing, same as
  `web-public`'s design switcher; the existing `src/api.ts` boundary grows,
  it is not replaced.
- **Importer discipline holds.** The GUI only writes through
  `POST /api/admin/mementos` (manual patch path) so authored-field tracking
  and revision checks stay server-owned.

## Tasks

### ADMIN-01.1 — Navigable journey shell

Hash routes: `#/` journey list → `#/journey/{id}` detail (mementos ordered by
seq, state badges, stop-candidate count). Replaces the flat read-only
overview.

Acceptance: navigating list → detail → back needs no reload; deep links work.

### ADMIN-01.2 — Import & preview triggers

Buttons on journey detail: `sync-route` (writes the route) plus `visits` and
`tray` (read-only source previews — they persist nothing), each with
per-action pending/success/error status derived from the structured API
responses.

Acceptance: a trigger's outcome (counts or error) is visible without opening
dev tools. The no-clobber guarantee (re-import never overwrites authored
fields) lives on the memento upsert path, not these triggers, and is
asserted there in the E2E pass.

### ADMIN-01.3a — Intake over HTTP (server prerequisite)

Candidates are currently generated only by the CLI (`intake.Plan`/`Apply`),
and reviewing one never creates a memento. Two small endpoints close the
loop:

- `POST /api/admin/journeys/{id}/intake/plan` — run the intake planner over
  the journey's sources and persist the proposed candidates.
- `POST /api/admin/stop-candidates/{id}/promote` — mark the candidate `kept`
  and create a draft memento from it (kind from the request, geometry and
  occurred window carried over from the candidate, source ref preserved).

`ignored` / `merged` keep using the existing review endpoint. Candidate
states follow the existing domain vocabulary: `proposed` / `kept` /
`ignored` / `merged` — there is no "confirm".

Acceptance: plan → promote over HTTP yields a draft memento whose geometry
and time come from the candidate; promoting is idempotent-safe (a second
promote conflicts instead of duplicating).

### ADMIN-01.3b — Intake inbox (GUI)

List `GET /journeys/{id}/stop-candidates`; per proposed candidate: promote
(with kind picker), ignore, or merge. Promoted candidates leave the inbox
and appear in the journey's memento list.

Acceptance: reviewing a candidate updates the inbox and the memento list
without reload.

### ADMIN-01.4 — Memento editor (MVP forms)

Edit view per memento: common fields (title, place, occurred_at/tz, essay,
vendor, price, **location** — lat/lng inputs with the existing
`POST /journeys/{id}/snap` helper, since non-draft saves must pass the
kind's anchor geometry validation) + kind-scoped `kind_data` form for
`transit` and `goods`, aligned to the registry field specs; other kinds get
a read-only JSON view. Promoted mementos arrive with the candidate's
geometry already set. State transitions draft → authored → published as
explicit actions. Photo list with caption/seq editing via `POST /photos`
(public media boundary stays enforced by packaging/import — the GUI never
uploads bytes in this epic).

Acceptance: a promoted candidate can be authored (essay + kind_data) and
published entirely in the GUI; server-side validation issues (including
geometry) render inline.

### ADMIN-01.5 — Revision concurrency

Every save sends `expected_revision`; a stale write (server write-conflict)
shows a "reload and reapply" prompt. No merge UI.

Acceptance: two overlapping edits produce a visible conflict, never a silent
overwrite.

### ADMIN-01.6 — Publish-and-compile action

Small server addition: `POST /api/admin/compile {out_dir}` runs the shared
`publication.Compiler` against the live store (SQLite path) and returns the
`BuildReport`. The GUI's publish flow = mark memento(s) published → trigger
compile → show report (journeys/mementos/media counts).

Acceptance: after publishing in the GUI, the reported artifact matches
`felicia-cli static compile` output for the same DB (same shared compiler, so
this is a smoke assertion, not a new parity suite).

### ADMIN-01.7 — Kind contract alignment (drift fix)

Known drift: backend registry has `live` (no `souvenir`); the public SPA's
`MementoKind` union and v4 stub registry have `souvenir` (no `live`). Fix:
align frontend lists with `core/kinds` (add `live`, decide `souvenir` —
either add a registry yaml or drop it from the frontend). Frontend sites to
touch: `apps/web-public/src/data.ts` (union), `apps/web-public/src/v4/stubs.ts`
(+ `stubs.test.ts`, which currently asserts `live` is unknown), and
`apps/web-public/src/api/adapt.ts` (known-kind allowlist). Add a drift test
that compares the registry kinds against the frontend lists so the two can
never silently diverge again.

Acceptance: the drift test fails if `core/kinds` and frontend kind lists
disagree; a published `live` memento renders with a proper stub, not the
photo fallback.

### ADMIN-01.8 — Closed-loop verification

Extend the disposable-server harness with an admin-GUI E2E pass (browser
automation against `bun run dev` + the workflow server): import → review
candidate → author → publish → compile → assert the artifact contains the
authored essay. Component-level tests (`bun test`) cover the api client and
form validation mapping.

Acceptance: `make test-admin` (bun tests) is green in `make validate`; the
E2E script passes locally against the disposable server.

## Milestones

- **M1** = 01.1 + 01.2 (navigate + import/preview)
- **M2** = 01.3a + 01.3b (intake over HTTP, then the inbox)
- **M3** = 01.4 + 01.5 (editor + concurrency)
- **M4** = 01.6 + 01.7 + 01.8 (publish loop, drift fix, verification)

Each milestone is one PR with docs-sync per `AGENTS.md`; GitHub issues are
the status ledger, this document records scope and acceptance only.

## Explicitly out of scope

Registry-driven dynamic form engine, media byte upload from the GUI, conflict
merge UI, auth/multi-user, translation sidecar, any change to the public
designs beyond the 01.7 kind alignment.
