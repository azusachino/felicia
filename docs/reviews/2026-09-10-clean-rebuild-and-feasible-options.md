# Felicia from zero: mature product and feasible delivery options

Proposal, 2026-09-10 JST. Companion to [the source-backed foundations review](2026-09-10-foundations-and-product-review.md), reviewed against `344ae46`. This document designs an alternative; it does not claim implementation or migration has been performed.

## Recommendation

If starting from zero, build **a local travel-journal studio that produces a deliberate, versioned public site**. Its center is the author's collection and writing; the map and collectible mementos are the reader experience. Reliable preservation, effortless intake and understandable publication are product features, not backend chores.

For the existing Felicia, the most feasible choice is to replace the unsafe storage/import/publication boundaries inside the current repository, then reorganize the authoring experience around them. A fresh implementation is feasible, but it must earn its migration cost. Keep the original system available until a full journal has been imported, edited, published, withdrawn and restored successfully in the replacement.

The goal is a complete, mature personal product. Early milestones below are delivery order, not permission to ship a reduced substitute. This proposal assumes single-author local operation remains the intended product; collaboration and hosted multi-user operation would change the architecture and should be a separate product decision.

## What the finished product feels like

1. **Connect or bring your trip.** Choose a date interval and timezone; connect GPS/photo sources or select local files. See what was found, what is missing and what needs interpretation. A trip can start without GPS or without photos.
2. **Review one inbox.** Routes, suggested stops and photos arrive together. Accept, merge or dismiss suggestions. Repeated intake updates the evidence without moving things the author deliberately arranged.
3. **Compose memories.** Group photos, choose a memento kind, write an essay, adjust place/time and order. Save drafts without completing every publication field. Recover local work after navigation, a failed save or a conflict.
4. **Preview a publication.** Choose which journeys and memories are public, what location detail is acceptable, the site design and identity. Review changes since the previous build, including removals. Preview the exact build to be deployed.
5. **Publish and return later.** A release has a stable identity and a clear outcome. Late photos and corrections enter the same workflow. A failed build retains the prior site. Withdrawal removes content from the next deployed artifact; backups restore the private journal, not merely its public rendering.

Automation belongs in collection, deduplication, candidate generation, derivative creation and build validation. It must not silently decide what becomes public or overwrite an authored memory. Do not require daily records or impose Iroha-style completeness on a curated trip.

## Architecture if there were no existing code

Use one Go application with a CLI and a local authoring HTTP interface, one SQLite database, a private immutable media directory, and a static publication compiler. Use Svelte for the authoring app and reusable reader compositions. Internal packages separate responsibilities; separate services and one module per concept are unnecessary.

The normal authoring authority is **SQLite plus original blobs**, with recoverable browser drafts for uncommitted edits. Imported packages are interchange inputs; exported recovery archives are backups. Neither silently overrides newer GUI edits. The static site is a sanitized projection, never the backup or primary database.

| Boundary    | Owns                                                                            | Does not own                                |
| ----------- | ------------------------------------------------------------------------------- | ------------------------------------------- |
| Collection  | Source credentials, cursors, file receipts, retryable acquisition               | Authored essays or publication decisions    |
| Intake      | Normalized evidence, deduplication, candidate proposals and review decisions    | Direct replacement of curated memories      |
| Journal     | Journeys, visits/anchors, mementos, writing, ordering and revisions             | Provider-specific payload formats           |
| Media       | Immutable originals, verified digests, attachment metadata and derivatives      | Destructive resets of authoring storage     |
| Publication | Public projection, privacy rules, complete build manifests and release identity | Editing the private journal                 |
| Application | Commands, API, authoring screens and reader hosts                               | A second implementation of the domain rules |

Use the same application commands from the CLI and GUI. Keep one primary local database engine for a fresh product. PostgreSQL is not intrinsically needed to author locally and publish static files.

For existing Felicia this is no longer an open question: [ADR-0032](../adr/0032-sqlite-first-v1-postgres-follow-up.md) is accepted, SQLite is the only v1 persistence contract, and PostgreSQL/PostGIS is a frozen non-v1 snapshot deferred to v1.1/v1.2. No inventory is required to decide it. What remains is enforcement, tracked separately, and the reclassification that follows: a provider gap SQLite does not share is deferred-provider work, not a v1 parity blocker, so no phase below carries a dual-provider test burden.

Avoid introducing a workflow engine, event sourcing for every field, a universal plugin system, microservices, remote object storage or a native mobile application without a demonstrated need. Retain adapter seams for actual sources. The existing Go modules can remain during an in-place rebuild; module consolidation is not a prerequisite for correctness.

## Concrete storage model

These are logical responsibilities, not a requirement to introduce every table at once or mirror the old schema mechanically.

| Record                 | Identity and invariants                                                                                                        |
| ---------------------- | ------------------------------------------------------------------------------------------------------------------------------ |
| Journal                | Stable journal ID, site identity, timezone defaults and settings revision                                                      |
| Journey                | Stable ID, title/date interval, revision; imports cannot replace authored values                                               |
| Source instance        | Provider/account or local collection identity; separate from an individual import                                              |
| Import run and receipt | Source instance, input digest, acquisition interval, outcome and diagnostics; retry is distinguishable from new evidence       |
| Observation            | Stable source-scoped identity, normalized value and provenance; newer interpretation does not acquire authorship               |
| Stop proposal/review   | Stable derivation identity, evidence, decision and revision; re-planning retains accepted/ignored choices                      |
| Memento                | Journey owner, kind, content, anchor, lifecycle and revision; meaningful incomplete drafts are allowed                         |
| Authored overrides     | Explicit overridden fields, including intentional empty values; source refresh never clears them                               |
| Blob                   | Verified content digest, size/type and immutable storage key; package filenames are metadata ([ADR-0026](../adr/0026-local-first-media-and-blob-storage.md) already specifies this) |
| Attachment             | Memento owner, blob reference, caption, order and revision; curation is independent of byte identity                           |
| Build/release          | Input snapshot identity, privacy policy version, artifact digest, outcome and manifest; deployment acknowledgement is separate |

### Delta against the current schema

The table above reads as a fresh design, which understates how much of it exists. Measured against `apps/felicia-providers/sqlite/schema.sql`, nine of the eleven records are already present and the work is mostly at column level:

| Record                 | Today                                                         | Delta                                                                                           |
| ---------------------- | ------------------------------------------------------------- | ----------------------------------------------------------------------------------------------- |
| Journal                | `tb_journals`, `tb_site_settings`                             | no settings revision                                                                            |
| Journey                | `tb_journeys` with `authored_fields`                          | **no `revision` column** — this is the ingest race itself                                       |
| Source instance        | a `source_system` string only                                 | new record, or an explicit decision that the string suffices                                     |
| Import run and receipt | `tb_import_runs` (ADR-0014)                                   | add input digest and acquisition interval; retry is not currently distinguishable                |
| Observation            | `tb_source_observations` (ADR-0010)                           | identity is **run-scoped** — `UNIQUE (run_id, source_system, source_external_id)` — not source-scoped |
| Stop proposal/review   | `tb_stop_candidates`, `tb_stop_candidate_evidence`             | has `revision` and `derivation_version`; PostgreSQL lacks the shared transaction                 |
| Memento                | `tb_mementos`                                                 | already revision-aware                                                                          |
| Authored overrides     | `authored_fields` masks (ADR-0033)                            | enforcement, not schema                                                                          |
| Blob                   | `content_hash` on `tb_memento_photos`                         | **new record**; the hash is recorded but `object_key` is the package path                        |
| Attachment             | `tb_memento_photos`                                           | no `revision`, so curation has the same race as the journey                                      |
| Build/release          | —                                                             | **new record**                                                                                   |

So the storage delta is two new records, roughly four columns, three `revision` columns, and one uniqueness change. `revision` exists today on exactly `tb_mementos` and `tb_stop_candidates`.

Two consequences follow. The blob record is not a decision to be made — [ADR-0026](../adr/0026-local-first-media-and-blob-storage.md) is accepted and already specifies `originals/<sha256>/<filename>` with the content hash as the stable contract, so this is a defect against an existing contract rather than new design, and phase 1 inherits a specified target. And build/release identity is the one genuinely new decision in this document, so it is the one that warrants its own ADR.

Keep core ownership fields relational. Kind-specific details may remain validated JSON with versioned schemas; the renderer never gets to invent storage fields. Validate date/time semantics explicitly, preserve source timestamps and timezone uncertainty, and allow manual location correction without changing the original evidence.

Enforce foreign keys, source-scoped uniqueness and revision predicates in the database. A read/merge/full-row-write sequence is not a sufficient authorship guard. Candidate plus evidence changes share a transaction. A photo's sequence/caption changes belong to a revision-aware curation command, not a generic import upsert.

Use a small append-only command/import audit for diagnosis and recovery where valuable; do not require rebuilding all current state from an event log. Version database migrations and recovery formats independently from the public JSON contract.

## Three workflows that must be correct by construction

### Acquisition and import

Acquire into a private staging area, validate the complete input, hash the original bytes, install immutable blobs, then transact references and normalized evidence. A failed transaction can leave unreferenced immutable files for conservative cleanup; it must not leave published references to missing bytes. Retry by receipt/source identity.

Show an import result that distinguishes acquired, interpreted, applied, skipped and conflicted. Partial runs remain visible. Re-import can add source evidence and suggestions but cannot apply a manual author patch. An explicit restore/replace command is a separate, previewed operation with a recovery checkpoint.

Connected-source refresh can be scheduled while the local application is running, with persisted checkpoints and retries. “Automatic” must not imply background operation while the computer is asleep or the app is stopped. Start with bounded catch-up on opening a journey and optional periodic refresh; introduce a daemon only if unattended collection is an actual requirement.

### Authoring

Save through revision-checked commands that return the committed revision. Keep the submitted draft until the save outcome is known. On conflict, preserve both versions and show a comparison or field-level choice; a reload must not discard the only copy of the author's work. Browser draft recovery is local and must respect the private-data boundary.

Distinguish publication eligibility from editing state. Editing an eligible memory creates changes for a future build; it must not silently alter a remotely served static release. Keep manual photo order, captions, exclusions and anchor corrections independent of source refresh.

### Publication and recovery

Capture a consistent journal snapshot and its blob references. Produce all JSON, sanitized media and matching reader assets in a new owned build directory. Verify references, public-field allowlists, coordinate policy, draft exclusion, base paths and manifest hashes. Then promote the complete release; a failed build never changes the active one.

For local serving, switch a release pointer atomically under a single build/promotion lock. For static hosts, use their complete-artifact deployment capability and record the acknowledged release; do not claim universal atomic deployment on arbitrary file-copy hosting. Serialize promotion and do not let an older build supersede a newer selected release by finishing late.

Show “changes since build,” “built,” and “deployed” separately. Content edits, settings changes and removals affect the input digest. A network/status error means unknown, not clean. Unpublishing requires a new release and confirmed deployment; locally changing a flag does not retract previously deployed bytes.

A recovery archive includes a consistent database snapshot, referenced original blobs, configuration needed to interpret it, version metadata and checksums. Restore into an empty location and validate before switching. Keep credentials out of portable publication artifacts; document how an author reconnects sources after recovery.

## Complete feature port, not a minimal replacement

| Capability           | Rebuild requirement                                                                                                                                                       |
| -------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Journey collection   | Multiple trips, stable IDs, dates/timezones, local GPX/photos and supported Dawarich/Immich inputs                                                                        |
| Intake               | Routes, derived stops, evidence/confidence, review/merge/ignore, late arrivals and safe repeated import                                                                   |
| Writing and curation | Existing memento kinds and kind data, essays, attachments/captions/order, anchors, drafts, validation and conflict recovery                                               |
| Reader               | Private reader and public reader hosts; Atlas, Cabinet, Techo, Cartography; map routes/places, memento detail, galleries, deep links, supported system locales and themes |
| Site                 | Site identity/settings, inclusion decisions, privacy precision/exclusion, exact-artifact preview, static build and base-path deployment                                   |
| Interchange          | Existing workspace/package import with declared compatibility, export/recovery and identity mapping                                                                       |
| Operations           | CLI/GUI parity for supported workflows, actionable failures, retry/catch-up, backup/restore and failed-build recovery                                                     |
| Existing deployments | Inventory provider/database/API consumers; preserve or explicitly migrate them before retiring the old path                                                               |

The inventory must include real data distributions and unsupported media behavior. Preserve originals even when the current public renderer cannot display them; explain that boundary rather than silently dropping attachments. Current AI enrichment and remote object storage deferrals do not automatically become rebuild requirements.

## Feasible options for the existing repository

| Option                            | Concrete scope                                                                                                                             | Benefit                                                                     | Cost/risk                                                  | Assessment                                                              |
| --------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------ | --------------------------------------------------------------------------- | ---------------------------------------------------------- | ----------------------------------------------------------------------- |
| A. Repair existing boundaries     | Fix the ten review findings, add recovery/build identity, then connect the existing admin flow                                             | Fastest route to trustworthy daily use; preserves current ports and readers | Existing split workflows remain until explicitly unified   | Best when current authoring broadly fits the user                       |
| B. Replace the core in place      | Establish safe blob/import/publication commands; move existing callers onto them; redesign the authoring flow around one journal authority | Coherent product without rebuilding reader assets or every integration      | Requires careful data conversion and temporary adapters    | Recommended balance for a substantial rework                            |
| C. Fresh successor with full port | New local studio implementing the architecture above; import the old journal and port all agreed capabilities before cutover               | Cleanest implementation boundaries                                          | Largest duplicate effort, parity burden and migration risk | Feasible if concrete structural blockers make B harder than replacement |

These options have different scopes, not different quality bars. A can still deliver a mature product. B is not a permanent dual-system design. C must not be called complete when only one reader or a clean demo dataset works.

The three-way framing overstates the choice, now that the storage delta above is measured. A and B share nearly all of their work: two new records, a few columns, and the enforcement to go with them. What actually separates them is whether the authoring flow is then redesigned around one journal authority, which is a decision that can be taken after the repairs rather than before them. So the real sequence is: do the boundary repairs, which nothing here disputes, and decide on the authoring redesign separately once daily use is trustworthy.

Before choosing C, compare one representative vertical slice: import two trips with colliding media names, author and revise memories, re-import, build, withdraw and restore. Identify which current abstractions prevent that slice from being implemented cleanly. The existing review finds broken boundaries, but does not establish that the current codebase is fundamentally incapable of enforcing them — and the delta above is evidence in the other direction, since nine of eleven records and both revision-aware write paths already exist.

## Recommended implementation sequence for option B

| Phase | Deliverable                                                                    | Exit evidence                                                                                               |
| ----- | ------------------------------------------------------------------------------ | ----------------------------------------------------------------------------------------------------------- |
| 0     | Repair the destructive preview path; capture current capability/data inventory | Preview cannot delete originals; retained inputs and author state are enumerated                            |
| 1     | Recovery archive and verified immutable blob mapping                           | Restore a representative multi-trip journal; colliding filenames retain correct bytes                       |
| 2     | Atomic source intake and revision-aware author commands                        | Late input, retry, concurrency and old-package tests preserve all authored values and review decisions      |
| 3     | Consistent snapshot compiler and complete release promotion                    | Failed/concurrent builds retain a coherent active release; withdrawn files are absent from the new artifact; high-precision input sits on the public grid in **index and detail** JSON alike |
| 4     | One inbox/editor/publication flow using those commands                         | No routine terminal step between available evidence and local public preview; conflicts retain drafts; an essay-only edit marks the site as changed and a status error reads as unknown |
| 5     | Verify reader behavior unchanged; port deployment and interchange              | All capability rows pass; old data and consumers have an explicit disposition                               |
| 6     | Rehearsed cutover and real authoring acceptance                                | Author uses a new trip plus a revision to an old trip, deploys, withdraws, and restores without lost work   |

Phase 5 says *verify* rather than *port* because option B does not rebuild the readers. Under option C that row becomes a port, and it is the largest single line item in the option.

Keep the source-backed review's specific regression cases as the acceptance ledger. Split phases into focused changes when implementation starts. Do not schedule a fixed completion date before measuring archive/restore and one full slice; those establish the real effort better than estimates from file count.

### These phases are mostly an existing backlog

The phases are an ordering and a set of gates, not a new inventory of work. Most of the work already exists in the issue ledger, and saying so keeps the plan from being read as a second, competing plan:

- phase 0 is the P0 preview defect, plus the inventory;
- phase 1's recovery archive is the standing backup/restore issue, and its blob mapping is the ADR-0026 violation;
- phase 2 and phase 3 correspond to the filed import, authorship and publication defects;
- phase 4's inbox and editor work is already filed across the intake and admin-GUI gaps;
- phase 6 is the standing end-to-end release rehearsal.

The plan's own contribution is the order, the gates, and the rule that no destructive repair happens before the archive exists.

## Migration and cutover

Inventory database state, originals, authored workspaces, decisions, site settings and external consumers before designing conversion. Stop writers for a consistent export. Preserve old IDs where possible; record explicit mappings otherwise. Verify media byte hashes and authored fields, not just row counts or whether the homepage renders.

Run the replacement against a restored copy, with no permanent dual writes. Compare private-state preservation and intended public output separately; privacy fixes may intentionally change public output. At cutover, take a final checkpoint, import/convert, validate and only then reopen authoring. Keep the previous application/data available for rollback. After new edits begin, rollback must carry those edits and blobs back or preserve them in a recoverable archive; restoring an old database alone is data loss.

The first two foundation repairs should not wait for a final decision about a successor. Preserve the current originals and archive the journal before experimentation.

## Decision to carry forward

Adopt option B if the intention is to spend meaningful effort improving Felicia itself. Keep option C as an implementation alternative with the same feature and recovery gates. Retain option A if the authoring workflow proves satisfactory after targeted fixes.

The product needs fewer contradictory write/build paths and clearer ownership of the author's work. Starting from zero is useful for defining that destination; it is not itself evidence that throwing away working readers, parsers and tests is the most feasible route there.

This is a design proposal based on the prior local source review, not a new competitor survey or a tested replacement. Existing review verification limits still apply.
