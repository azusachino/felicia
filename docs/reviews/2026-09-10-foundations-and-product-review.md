# Felicia: foundations and product review

Reviewed on 2026-09-10 JST against `344ae46` and the current working tree. Recommendation, not implementation approval.

## Decision

Felicia needs a focused foundational rework and a more coherent authoring-to-publication workflow. It does not currently justify starting from zero, changing the stack, replacing the journey/memento model, or rebuilding its four reader designs.

The product idea is sound, but several implementation paths contradict its strongest promises: preserve authorship, retain originals, and publish only a deliberate safe artifact. These are release-blocking concerns for a mature journal, even though its happy-path MVP is complete. More visual polish or automated imports should follow their repair.

Felicia differs materially from Iroha. External GPS and photos are inputs, but essays, curation, captions, chosen locations and original media are valuable author-owned state. They cannot be assumed reconstructible from providers or a sanitized public site. A destructive rebuild of derived data must never imply permission to discard them.

## Scope and evidence

Read the current project instructions, direction, selected user journey, Make workflows, admin screens, publication/compiler, package ingestion, and SQLite/PostgreSQL write paths. Two independent source reviews covered ingestion/persistence and publication/media. Existing graph checkpoints were stale and were not treated as completion evidence.

The checkout began on `main` with 68 pre-existing modified documentation files, including `AGENTS.md`. Those changes are preserved. This review adds one new document. No private journal, credentials, deployed instance, external providers, or sibling checkout was accessed. No production command, publication or database migration was executed.

One destructive-build defect was reproduced against temporary paths with external commands mocked. Other findings below are source-backed reachable paths, not claims of observed production incidents. The UI review is of implemented interaction/state handling, not a rendered visual or accessibility certification. Historic bugs already described as fixed in the README are not automatically counted as current defects.

## Product contract to retain

The useful user story is: return from a trip, collect evidence, select the places and memories worth keeping, write and arrange them, preview the exact public result, publish, and revisit or revise it later without losing prior work.

Keep these foundations:

- Journey → memento → essay/photos, with derived visits as useful location anchors.
- SQLite local authoring and a static public artifact as the primary supported workflow.
- Shared publication contracts for live/static readers, and the Atlas, Cabinet, Techo and Cartography compositions.
- Provider adapters, pure intake planning, stable candidate identities and explicit review decisions.
- Authored-field masks, manual-edit revisions, transactional package metadata, image sanitization and published-state filtering. These mechanisms need consistent enforcement, not replacement with a universal data platform.

Do not equate automation with auto-publication. Importing evidence can become automatic; deciding what expresses a memory and what is safe to share remains an author action.

## Findings that change the implementation plan

### F1 — P0: the Pages preview workflow can delete authoring originals

`scripts/felicia.py:18` uses `.felicia/media`; lines 29–38 recursively delete the selected media directory whenever `PAGES_DB` is unset, including an explicit `PAGES_MEDIA_ROOT`. The server defaults to the same media root (`apps/felicia-server/config/config.go:25`), as does CLI import (`apps/felicia-cli/cmd/felicia/main.go:246`). `make pages-preview` reaches this script at `Makefile:179`. The separate `make site-build` path does not execute this particular deletion.

Reproduction: imported the actual Python module, redirected all roots to a temporary directory, placed a sentinel original in the selected media root, cleared environment overrides, and mocked `run()` to stop at the first build command. The sentinel was deleted before that command. No real data was touched.

Replace the shared default with an explicitly disposable publication workspace. Reset only a directory whose ownership the build can prove; reject overlap with the authoring database, originals and authored workspaces. A failing preview build must leave all originals intact. This is the first repair, before another local Pages preview.

### F2 — P1: package member names become shared media identity

`apps/felicia-runtime/importer/package.go:335–341` carries a package photo path into `ObjectKey`; `apps/felicia-cli/cmd/felicia/main.go:284–297` writes that path beneath the shared media root with `os.WriteFile`. Two different packages containing `media/ticket.jpg` can therefore replace one another's bytes despite having different photo identities.

Public image sanitization does not prevent a collision in the private source store. Store immutable originals by verified content identity, with package-member-to-blob mappings. Verify two packages with the same member name and different bytes preserve both images and their references.

### F3 — P1: database import commits before its media is durable

CLI `main.go:280` applies the package transaction; lines 284–297 subsequently copy files. `apps/felicia-runtime/importer/package.go:69–77` confirms the SQL transaction completes inside that call. Disk failure or interruption can leave committed references to absent or incomplete originals.

Stage and validate immutable files before publishing database references. A failed transaction may leave recoverable unreferenced blobs; it must not leave committed references to missing bytes. Define retry behavior and conservative orphan cleanup, and inject failures at each boundary. A general distributed transaction system is unnecessary.

### F4 — P1: package re-import bypasses authorship protection

`apps/felicia-runtime/importer/package.go:44–46` promises authored fields are not supplied, but lines 105–107 invoke `ApplyManualMementoPatch` with package-authored fields and state, without an expected revision. Lines 111–114 upsert photos; PostgreSQL `queries/query.sql:201–208` replaces caption, sequence and association.

An old authored package can overwrite a newer essay or photo arrangement. The ordinary source-import path and intentional archive restoration need different contracts. Default re-import should preserve later authored state; conflicting authored package values should produce a reviewable conflict. Explicit restore/replace should be a separate operation with a preview and backup. Test import → edit essay/caption/order → re-import, including lifecycle state changes.

### F5 — P1: journey ingest protection is a read/merge/write race

SQLite `store.go:398–409` and PostgreSQL `repository.go:266–277` read a journey, merge fields in memory, then call the ordinary full-row upsert. SQLite `store.go:430` replaces the authored values and mask. There is no revision predicate or row lock in these ingest methods; route sync can call them outside package transactions.

Interleaving: ingest reads → author changes title and marks it authored → ingest writes the stale copy while updating the route. The title and protection mask can be lost. Make field ownership checks and updates atomic against current database state, or use revisions/locking with a bounded retry. Test that exact interleaving on each supported provider. Single-user does not mean single-request.

### F6 — P1: PostgreSQL stop refresh can return failure after partial mutation

`apps/felicia-providers/postgres/stop_candidate.go:85–113` updates the candidate, deletes evidence, then validates/inserts replacement evidence without an internal transaction. `apps/felicia-runtime/intake/service.go:82–89` calls candidate upserts directly. SQLite's equivalent is transactional (`sqlite/stop_candidate.go:98–105,140–154`).

Validate the complete replacement first and transact candidate plus evidence together, including when called inside an existing transaction. Use one conformance case that injects invalid evidence and insert failure; both providers must retain the complete prior state. This is a current behavioral mismatch; historic table-name drift alone is not evidence of current schema drift.

### F7 — P1: journey index coordinates bypass the privacy boundary

`apps/felicia-publication/projection.go:62–68` emits raw point coordinates or the first line coordinate into `representative_dots`. The shared journey index projection is used by the compiler and live API. Detail geometry rounding does not cover this path.

Route all public coordinates through the same precision policy and test high-precision inputs in both index and detail JSON. Audit coordinate-bearing `kind_data` as a follow-up; this review does not claim it is a demonstrated second leak. Rounding itself is a precision policy, not a substitute for author-controlled omission of a sensitive place.

### F8 — P1: publication lacks a complete-artifact commit boundary

`apps/felicia-publication/fileio.go:85–110` atomically replaces individual files. That does not make a complete build atomic. A later compiler failure leaves mixed output. Server `api/server.go:1271–1282` only finalizes after compilation succeeds.

At `fileio.go:157–170`, an unreadable or malformed previous manifest becomes an empty inventory. Finalization then skips stale-file removal and replaces the manifest, forgetting older files. An unpublished image can remain directly reachable even if it disappears from the index. The malformed-manifest test currently accepts this behavior.

Build a complete artifact in a new owned directory from a consistent input snapshot, verify it, then switch the served release. Serialize builds targeting the same publication and keep the prior valid release on failure. Treat malformed ownership metadata as an error for reused output, not successful cleanup. Test unpublication, corrupt manifests, concurrent builds and failure midway through media processing. Preserve the public SPA alongside its matching data release.

### F9 — P1/P2: “pending build” ignores content changes and confuses built with deployed

`apps/felicia-server/api/buildstatus.go:17–23,59–89` compares only published memento ID membership. Editing an already-published essay, caption, journey title or site settings can leave zero pending items. No artifact also returns zero. `JourneyDetail.svelte:216–224` converts status-read errors into an empty pending set.

Track the revision or content digest of the complete publication input against the last successful local build. Distinguish “never built,” “changes since build,” “build failed/unknown,” and “built.” Track a deployed release separately only when a deployment workflow can provide evidence; a local output directory is not proof of what a remote site serves. Verify an essay-only edit marks the site dirty and a status error does not look clean.

### F10 — P2: conflict recovery discards the draft the author needs to reapply

`apps/felicia-admin/src/views/MementoEditor.svelte:246–247` handles “Reload and reapply” by calling `loadAll()`. Lines 175–179 hydrate the form from the server, replacing its local contents. The label at line 396 suggests a recovery action, but the unsaved draft is not retained or merged.

Preserve the attempted draft on conflict and show what changed before retrying. Add explicit unsaved-navigation handling or local draft recovery for essays and captions. This is a source-level interaction finding; no browser reproduction was performed. Test two editor sessions and intentional navigation during a save.

## Data-model recommendation

Retain the current domain tables. A memento with kind-specific data is a sensible model; no finding requires an all-purpose fact/event schema or a rewrite of every repository interface.

Strengthen the boundaries that the model currently expresses only partially:

1. Immutable original blobs, independent of filenames and package membership.
2. Source observations/import membership distinct from authored overlays and explicit restore commands.
3. Revision-aware aggregate writes for journeys and curated photos as well as mementos.
4. Candidate/evidence changes committed together, with provider conformance tests.
5. A publication input revision/digest and immutable build identity, separate from publication eligibility and deployment acknowledgement.

Choose one authoring authority for the normal workflow: the local database plus retained originals and author drafts. JSON workspaces/packages are import/export interchange or explicitly selected alternative authoring workflows; they must not silently compete with later GUI edits. A public artifact is deliberately lossy and cannot serve as the recovery archive.

Keep SQLite as the primary release path. Do not expand PostgreSQL-specific features while correctness parity is incomplete. If PostgreSQL remains supported, every relevant invariant must pass on both; otherwise explicitly narrow its support status before release. Do not remove it merely to reduce the review's task list.

## Workflow and UX recommendation

The four reader designs are already a product asset. The larger UX return is in the authoring loop:

- One trip inbox combines route/photo intake outcomes and candidate review. Show missing evidence and partial import failures without inventing memories.
- Curate first, then write. Preserve selections and essays across refreshes, re-imports and conflicts. Allow incomplete drafts without forcing final metadata prematurely.
- Separate “include in next publication” from “build preview” and “deploy.” Show the exact changes included in a build, including edits and removals.
- Preview the exact artifact intended for deployment. Surface privacy exclusions, missing media and unresolved conflicts before build promotion.
- Make backup/export and restore part of the supported author workflow, including originals, essays, revisions and curation. Rehearse restoring into a fresh local installation.

The current admin includes internal milestone language and implementation instructions in user-facing copy (`SiteDeploy.svelte:190–199,219–220`). Replace these after the underlying workflow is truthful; better copy alone will not solve preservation or stale build state.

Do not copy Iroha's daily coverage model into Felicia. A journey is deliberately curated and may have sparse memories or no route. Automatic collection should reduce repeated effort, but daily rows and periodic imports are not Felicia's core success criterion.

## Concrete repair sequence

| Order | Work                                                                                    | Acceptance before proceeding                                                                                                              |
| ----- | --------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------- |
| 1     | Protect private roots and isolate disposable publication workspaces (F1)                | Temporary sentinel originals survive default/overridden preview paths and forced build failure.                                           |
| 2     | Inventory retained data and establish a recovery archive                                | Fresh-machine restore reproduces essays, captions, ordering, original bytes and identity links. No destructive repair before this gate.   |
| 3     | Make original-media identity and import publication safe (F2–F3)                        | Same filenames across two trips do not collide; interrupted/retried imports preserve references and bytes.                                |
| 4     | Separate re-import from restore; enforce atomic authorship and evidence updates (F4–F6) | Re-import after editing and concurrent edit/import preserve authorship; provider failure tests leave no partial state.                    |
| 5     | Close every public coordinate projection and stage whole artifacts (F7–F8)              | High-precision data passes the policy; failed builds retain the previous release; unpublication removes direct access in the new release. |
| 6     | Add accurate build identity/status and conflict draft recovery (F9–F10)                 | Essay-only edits mark changes; status failure is unknown; conflicts retain local work; built/deployed are distinct.                       |
| 7     | Connect intake, curation, preview and deployment around those contracts                 | Complete two-trip workflow with revisits, late photos, corrected route, withdrawal, failure/retry and restore; verify all four readers.   |

This is a finite repair program within the existing repo. Several findings can share a carefully scoped change, but media migration, authored-state migration and artifact promotion should remain separately reviewable. Estimate effort after the retained-data and current test baseline are known; this review does not invent a delivery date.

## Verification and limits

The temporary Python reproduction for F1 passed: the current code removed the sentinel before its first external build command. This validates the defect, not a fix. No product code was changed.

`make check` stopped in formatting: two pre-existing dirty documents (`docs/adr/0010-media-pipeline.md` and `docs/research/tabi-design-reset.md`) fail Prettier. They were not modified by this review. Frontend dependencies are missing; `scripts/format.py:62–65` reports this but returns success for the frontend formatting step, so a green aggregate formatting result would not certify those files. The new review document is checked separately with the repository Markdown formatting options. `make test` was attempted separately and stopped during CLI test setup because downloading `modernc.org/sqlite@v1.23.1` failed certificate validation. No Go test pass is claimed. No real provider/device trial, rendered UI walkthrough, database migration, restore rehearsal or deployment was run. The source findings must become focused regression cases in the repair work; existing unit test counts alone do not prove these boundary guarantees.

The recommendation is to repair foundations first, then complete the authoring UX using the existing product. A clean rebuild would still need all the same preservation and publication decisions, while adding migration risk for the most valuable data Felicia owns.

### Follow-up: local Markdown tooling

The PR also replaces the Markdown Prettier path with Nix-provided rumdl. `make fmt-docs` fixes Markdown and `make fmt-docs-check` checks it. Local `make check` includes formatting; CI runs `make check-ci` (vet, lint, tests and feature contracts), which does not require rumdl. No rumdl installation was added to mise or CI.

After fixing the remaining Markdown findings, rumdl passes all 128 files. Local `make check` now passes its formatting stage but is blocked in Go vet by the SQLite dependency download certificate error. The frontend formatter reports missing dependencies and skips its portion, as noted above. These are not full repository passes.
