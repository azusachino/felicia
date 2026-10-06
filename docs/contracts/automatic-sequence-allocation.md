# Automatic sequence allocation

## Decision and delivery status

For the remaining concurrency criterion in [#161](https://github.com/azusachino/felicia/issues/161), the owner selected unique automatically assigned photo/memento positions across processes. Stable duplicate ties alone do not satisfy this criterion. The source and isolated headless criteria are locally implemented and independently verified on `feat/desktop-followup-auto-sequences`, based on `c2181ff`. The owner approved source-only commit, push and draft-PR publication; normal/native delivery remains open.

The contract concerns automatic creation, not a uniqueness constraint on all stored ordering values. Explicit authored sequence changes remain supported. Existing identities, source references, captions, original bytes and lifecycle/revision behavior must not change as a side effect of allocating a position. Existing tied rows are not renumbered.

## Acceptance

- Distinct photo uploads through independent services/connections to the same SQLite workspace each obtain a distinct automatic position, above the existing maximum for their memento.
- Automatic new-memento creation obtains distinct positions within its journey even when two forms were opened from the same list snapshot.
- Allocation and creation are one atomic persistence operation. A per-process/service mutex or a separated read-max/write is insufficient. Do not assume a deferred read transaction serializes concurrent writers.
- Explicit authored reordering and existing stored identities/values remain unchanged. No blanket `(parent_id, seq)` uniqueness constraint is implied.
- Failure leaves no partial row and retains the existing original-byte compensation boundary: never remove bytes whose committed state is uncertain.
- Tests use disposable synthetic state. A deterministic independent-writer reproduction must fail the uniqueness assertion before the fix and pass after it, without timing sleeps or a race-and-hope loop. Real process isolation must be verified before claiming cross-process acceptance; separate connections alone are narrower evidence.
- Run affected runtime/provider/handler tests with the race detector, the owning full gates and fresh independent verification. Record exactly which creation paths are covered; unrelated transit-leg and curation lost-update behavior is not implicitly fixed.

## Increment order

1. Reproduce independent-service photo upload allocation with two SQLite connections and a barrier at original storage. Preserve the failing receipt separately from green acceptance.
2. Implement the smallest atomic photo-creation seam and verify existing explicit photo updates and compensation behavior.
3. Reproduce the stale new-memento form case, then give automatic creation a server/provider allocation path distinguishable from explicit authored ordering. Preserve stable retry identity and optimistic edit revision checks.
4. Verify actual cross-process writes and owning gates; publish source/runtime evidence with the delivered PR.

## Automatic memento creation boundary

The new-memento form uses a distinct `POST /api/admin/mementos/create` operation, with its once-generated ID and authored values but no client-selected sequence. The existing `POST /api/admin/mementos` explicit upsert/edit contract remains unchanged. Both desktop and server transports support the create operation because the same admin bundle can target either host through `VITE_API_BASE`.

Creation and allocation must be a single SQLite write operation. The shared runtime derives the manual creation ownership mask from the supported creation values, including the assigned ordering field; the HTTP caller must not supply or clear that mask. Authored values and ordering must survive subsequent ingest patches. A retry with the same ID and matching creation values returns the stored row without reallocating or incrementing its revision. A different journey or incompatible values for an existing ID is a conflict, never permission to update that row. Intervening edits must not be overwritten by a create retry; a conflict is acceptable when the stored authored values no longer match. Existing editor revision checks remain on the edit path.

## Photo increment verification

The photo-only increment is locally verified on `feat/desktop-followup-auto-sequences`, based on `c2181ff`. The subsequent memento verification is recorded below. Source-only draft-PR delivery is approved; normal/native delivery remains open.

- The preserved independent-service reproduction failed because both writers allocated position 5 above the seeded position 4.
- Both the two-connection and real helper-process regressions now require persisted positions exactly `{5,6}`, original bytes and identities preserved, through the shared verifier in `apps/felicia-providers/sqlite/photo_sequence_test.go`.
- A subprocess harness failure was separately reproduced: an early successful exit consumed the exit result before its receipt, and cleanup then waited for the consumed result. One owner now drains stdout before `Wait` and publishes completion through a closed channel that cleanup can join repeatedly.
- The lead reran the combined regressions with `-race -count=10` (20 passes), affected runtime/provider/server/desktop packages with `-race`, and `make check` and `make validate`; all passed. Protected embedded admin assets remained unchanged.
- Fresh independent verification accepted the photo source and available logs after correcting an erroneous exact-position finding. Command exits and file hashes are lead receipts, not reviewer re-executions. The live record is `felicia:followup-161-auto-sequences` in Asobi.

Dedicated new provider regressions for duplicate identity/content-hash failures, explicit reorder/tied positions, and supplied caption/source values remain coverage gaps. The plain insert, unchanged schema and explicit update path were inspected; those gaps are not claimed as newly exercised behavior.

## Memento provider verification

The provider-only foundation is locally verified; the subsequent transport/UI verification is recorded below. Before implementation, two actual new-memento forms opened above a seeded position 4 successfully submitted distinct IDs and persisted positions `[4,5,5]` in both Chromium and WebKit. This was an allocation assertion failure, not a harness or HTTP failure.

`Repository.CreateMementoWithNextSequence` now performs atomic insertion/allocation and returns the persisted row. Provider regressions exercise actual helper processes, matching retries, incompatible or edited retries, preserved tied rows and explicit ordering, full representative authored metadata, draft-only creation, foreign-key errors and cancellation. The lead reran the four photo/memento regressions with `-race -count=10` (40 passes) and the full SQLite package with `-race`; both passed.

Fresh verification accepted the corrected provider foundation. Its initial review identified missing metadata and no-mutation assertions; the lead additionally caught an incorrect numeric SQLite constraint classification. The implementation now uses named `UNIQUE`/`PRIMARYKEY` constants and does not treat `NOTNULL` (1299) as an identity duplicate. The corrected review independently inspected source and available logs; exits and hashes remain lead/writer receipts, not reviewer re-executions.

## Memento integration verification

Desktop and server now expose the dedicated create endpoint, and the new form no longer reads the list maximum. Its caller-generated ID survives a failed response. Creation requests contain no sequence, edit revision or ownership mask; invalid fields, including explicit nulls, are rejected. The shared runtime assigns the canonical ownership keys `journey_id`, `kind`, `seq`, `occurred_at`, `occurred_tz`, `title`, `place` and `kind_data` on a cloned payload.

An integration defect was separately reproduced before acceptance: a missing caller mask produced an empty stored mask, and an ingest patch then replaced authored title, place, ordering, time and kind data. Real HTTP/SQLite tests on both hosts now prove those values and the server-derived mask survive ingest. Matching creation retries return the persisted ID, ordering and revision unchanged; incompatible or edited retries conflict rather than overwrite.

Final lead checks passed affected core/runtime/provider/desktop/server packages with `-race`, 40 repeated photo/memento provider cases, `make check`, `make validate`, 115 admin unit tests, and 14 Chromium/WebKit cases covering the existing authoring journey, two stale forms obtaining `{5,6}` above a preserved seed, and a real committed-but-lost response followed by an identical retry without a second row or revision increment. The existing new-form failure/retry test required only its exact interception path to migrate to `/mementos/create`; its pending-navigation, failure, retained-input, stable-ID and single-draft assertions stayed intact.

A freshly compiled admin bundle was embedded into the headless transport through a temporary Go 1.27 overlay. The 57 mappings target only virtual admin asset paths; the lead checked their backing-file hashes, the test-binary identity and unchanged physical assets during the final build/validation. The original owner index retained its known hash. This is current-source headless evidence, not normal/native embedded-asset delivery.

Fresh independent source/log review accepted the corrected integration with no remaining findings. It withdrew an incorrect attribution of the pre-existing owner asset diff and acknowledged the missed authorship defect. Gate exits and hashes remain lead/writer receipts, not reviewer-run commands. The live verification and delivery record is `felicia:followup-161-auto-sequences` in Asobi. The owner approved source-only commit, push and draft-PR publication, not normal/native delivery.

Scope remains automatic photo uploads and manual new-memento creation. This change does not alter explicit upserts or importer/transit-leg allocation and does not fix curation lost updates. Native interaction and issue closure remain outside acceptance.

## Persistence reference

[SQLite transaction semantics](https://www.sqlite.org/lang_transaction.html) allow
multiple readers but only one simultaneous writer, including separate processes.
A write statement starts a write transaction; a deferred transaction that reads
first may fail when upgraded after a concurrent write. The photo increment uses
an atomic insert/allocation operation rather than separated read-max/write.
[SQLite RETURNING](https://www.sqlite.org/lang_returning.html) can return the
assigned position from that operation. Target-driver and real-process tests are
required; these references alone do not establish acceptance.

Native windows, pickers, S1/S2 acceptance, release/deployment and issue closure are outside this contract.
