# Epic FELICIA-ADMIN-01 — Minimal authoring GUI (admin GUI MVP)

> Phase 2 of the selected end-to-end journey
> ([user-journey.md](user-journey.md)): upgrade stage 3 (authoring) from the
> file/CLI path to the smallest closed GUI loop in `apps/felicia-web`. Server
> API and data contract already exist; this epic is almost entirely frontend
> plus one small server endpoint.

## Design constraints (decided up front)

- **One editor, presentation-agnostic.** The editor edits the canonical
  contract (`kind` + `kind_data`, validated server-side against
  `apps/felicia-core/kinds/*.yaml`). The public designs v1–v4 are downstream projections;
  nothing in the editor may depend on any of them.
- **Kind source of truth is the registry.** The editor's kind list and field
  hints come from `GET /api/admin/templates` (the embedded `apps/felicia-core/kinds`
  registry). MVP hardcodes two forms (`transit`, `goods`) but their field
  sets must match the registry; the dynamic form engine stays deferred.
- **No new dependencies.** Svelte 5 + hash-based routing, same as
  `web-public`'s design switcher; the existing `src/api.ts` boundary grows,
  it is not replaced.
- **Importer discipline holds.** The GUI only writes through
  `POST /api/admin/mementos` (manual patch path) so authored-field tracking
  and revision checks stay server-owned.

## Tasks

All eight tasks landed (see Milestones below for the M1–M4 grouping); 01.7 is
marked separately as it landed early, in parallel with M1.

| Task                                                           | Status | Outcome                                                                                                                                    |
| -------------------------------------------------------------- | ------ | ------------------------------------------------------------------------------------------------------------------------------------------ |
| 01.1 Navigable journey shell                                   | done   | Navigating list → detail → back needs no reload; deep links work.                                                                          |
| 01.2 Import & preview triggers                                 | done   | A trigger's outcome (counts or error) is visible without opening dev tools; the no-clobber guarantee is asserted in the E2E pass.          |
| 01.3a Intake over HTTP (server prerequisite)                   | done   | Plan → promote over HTTP yields a draft memento whose geometry and time come from the candidate; promoting is idempotent-safe.             |
| 01.3b Intake inbox (GUI)                                       | done   | Reviewing a candidate updates the inbox and the memento list without reload.                                                               |
| 01.4 Memento editor (MVP forms)                                | done   | A promoted candidate can be authored (essay + kind_data) and published entirely in the GUI; validation issues render inline.               |
| 01.5 Revision concurrency                                      | done   | Two overlapping edits produce a visible conflict, never a silent overwrite.                                                                |
| 01.6 Publish-and-compile action                                | done   | After publishing in the GUI, the reported artifact matches `felicia-cli static compile` output for the same DB.                            |
| 01.7 Kind contract alignment (drift fix) — landed early, in M1 | done   | The drift test fails if `apps/felicia-core/kinds` and frontend kind lists disagree; a published `live` memento renders with a proper stub. |
| 01.8 Closed-loop verification                                  | done   | `make test-admin` is green in `make validate`; the E2E script passes locally against the disposable server.                                |

## Milestones

Milestone numbers here are **epic-local** (they are not the roadmap's
R-milestones — see the numbering note in [`../roadmap.md`](../roadmap.md)).

- **M1** = 01.1 + 01.2 (navigate + import/preview)
- **M2** = 01.3a + 01.3b (intake over HTTP, then the inbox)
- **M3** = 01.4 + 01.5 (editor + concurrency)
- **M4** = 01.6 + 01.8 (publish loop, verification) — 01.7 (drift fix) was
  parallel-safe and landed early with M1

Each milestone is one PR with docs-sync per `AGENTS.md`; GitHub issues are
the status ledger, this document records scope and acceptance only.

## Explicitly out of scope

Registry-driven dynamic form engine, media byte upload from the GUI, conflict
merge UI, auth/multi-user, translation sidecar, any change to the public
designs beyond the 01.7 kind alignment.
