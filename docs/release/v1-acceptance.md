---
title: "v1 Acceptance Scenarios and Privacy Invariants"
status: "active"
date: "2026-08-04"
---

# v1 acceptance scenarios and privacy invariants

This is the R0 deliverable [`docs/roadmap.md`](../roadmap.md) names — "write the v1
acceptance scenarios and privacy invariants" — consolidated into one document instead
of staying scattered across `direction.md`, the ADRs, and each milestone's own exit
check. It does not introduce new decisions; it restates what's already committed
elsewhere as one testable checklist, so a reviewer can verify v1 without re-deriving
the invariants from prose.

**Open item:** every scenario below is written against "the selected real journey" —
per M0 issue #7, no specific real trip has been designated as the v1 acceptance
fixture yet (only synthetic/example data exists: `examples/preview/local-journey`,
`tests/fixtures/local-journey-mixed-state`, and the E2E harnesses' seeded journeys).
Selecting one and inventorying its route/visits/mementos/essays/photos is separate,
pending work — this document is written so that step is a drop-in, not a rewrite.

## Acceptance scenarios

Each scenario restates a milestone's exit check from [`roadmap.md`](../roadmap.md) as
a concrete pass/fail check against the selected real journey, once chosen.

### Reader (R2)

1. A first-time visitor loads the public site with no prior state and can identify
   that a journey index exists, without already knowing the map UI (roadmap R2 exit
   check; tracked open by issue #37 until 2026-08-04, since closed against the
   current default reader's index rail).
2. From the index, the visitor opens the selected journey, sees its route on a dark
   MapLibre map with visit/memento markers, and the layout is usable on both desktop
   and mobile viewports.
3. Clicking a memento animates it open (per [ADR-0031](../adr/0031-frontend-style-map-first-and-shared-element-open.md),
   the committed shared-element morph once implemented) into its essay and photo
   gallery, and the visitor can return to the map/index afterward.
4. The reader is verified in Japanese, English, and Chinese system UI; authored
   content (titles, essays, captions) renders exactly as entered, with no automatic
   translation.
5. Keyboard navigation reaches every interactive element, and `prefers-reduced-motion`
   is honored for the open interaction.

### Authoring and publish (R3)

6. The selected journey can be created or repaired entirely through the admin UI —
   journey metadata, mementos (driven by the declarative template registry), and
   photo curation with captions/ordering.
7. A draft can be previewed as it will appear publicly before publishing.
8. Publishing is an explicit, separate action from saving a draft; unpublished
   content never appears through the public API or the compiled static artifact
   (enforced by the shared `publication` boundary and covered by the live/static
   parity check in `scripts/test_journey_workflow.py`).
9. Re-running an import after authoring never overwrites already-authored fields
   (field-scoped importer, [ADR-0022](../adr/0022-unified-intake-and-draft-pipeline.md)) —
   editing a field by hand and then re-importing the same source leaves the authored
   value intact.
10. A stale-write (editing the same memento from two contexts) surfaces a revision
    conflict instead of silently discarding one side's edit.

### Ingestion (R4)

11. Importing the selected journey's real GPX/Dawarich/Immich sources produces no
    duplicate mementos on a second run, and any incomplete/missing source data
    leaves an auditable result rather than a silent gap.

### Deployment (R5)

12. The compiled static artifact, deployed via the GitHub Pages workflow (or an
    equivalent supported host), serves the selected journey identically to what the
    live admin-mode API would serve for the same published state — verified by the
    live/static parity check, not by manual comparison.
13. The owner can publish a new memento to the selected journey and see it appear at
    the public URL without a developer's help.

## Privacy invariants

These are enforced properties, not aspirations — each one names where it's actually
checked in code/tests, per [ADR-0025](../adr/0025-static-and-self-hosted-modes.md) and
[ADR-0026](../adr/0026-local-first-media-and-blob-storage.md).

1. **Drafts and originals never leave the machine.** The compiled static artifact
   contains only `published` content, EXIF-stripped public media derivatives, and
   rounded geometry. The admin app itself is never part of a deployed artifact.
2. **No raw GPS in public responses.** Public projections (`publication/public.go`,
   the live `/api/v1` handlers) never expose unrounded coordinates or a private
   track's full-precision points — checked by the roadmap's stated verification gate
   ("no raw GPS in public responses") but **not yet backed by an automated geometry-
   rounding check in code** (confirmed gap — no `Round(` call exists in the Go
   codebase today; tracked by open issue #27, not yet closed).
3. **Public images are resized and EXIF-stripped.** Every media file that crosses the
   public media boundary is a processed derivative, never the original upload —
   **also not yet automated as a compile-time or test-time check** (same gap as #2,
   tracked by issue #27/#24).
4. **A journey without published mementos has no public projection at all**, on
   either the live API or the static compiler — enforced by the shared `publication`
   package and covered by `scripts/test_journey_workflow.py`'s unpublished-journey
   exclusion check.
5. **Re-import never overwrites authored fields.** The field-scoped importer upserts
   only source-derived fields; anything the author has edited by hand is preserved
   across re-import — enforced by `runtime/importer/` and its contract tests.
6. **No credentials inside felicia.** No provider or deployment path stores third-
   party credentials in the database or the compiled artifact; GitHub deployment (M3
   of epic ADMIN-02) reuses the operator's own `git` credentials rather than storing
   a token.

Invariants 2 and 3 above are stated in the roadmap's verification gates but are
**not yet enforced by an automated check** — this document surfaces that gap
explicitly (feeding open issues #24 and #27) rather than letting the informal
"we round geometry / strip EXIF" claim stand unverified.
