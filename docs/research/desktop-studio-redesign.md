---
title: "Desktop studio redesign: reference study and proposed plan"
status: proposed
date: "2026-10-03"
---

# Desktop studio redesign

## Status and approval boundary

The owner approved this research direction and requested execution through a concrete
[execution plan](../roadmap/desktop-studio-redesign.md). Detailed tokens and layouts
still require verification; this research page itself contains no implementation. The request is to change the design and layout,
not merely prove that the existing web admin renders inside a desktop window.

Baseline: Felicia `feat/desktop-studio-scaffold` at
`15693e1418ac42a915b1005241e64505ae418c2c`, follow-up PR
[#157](https://github.com/azusachino/felicia/pull/157). PR
[#156](https://github.com/azusachino/felicia/pull/156) was merged as `6114c17`;
its checks do not cover the later fixes. An uncommitted debug-listener change in
`apps/felicia-desktop/main.go` is outside this documentation slice. Do not ship it
as an unauthenticated authoring server: its current address is not restricted to
loopback, and loopback alone is not authorization.

Workstation route: workstation-task, tier 1. Live coordination is
`felicia:desktop-redesign`; implementation/e2e work remains separately tracked as
`felicia:desktop-e2e`. Commit and push require explicit confirmation under
[project instructions](../../AGENTS.md). Direction approval has been received; the execution plan's slice gates now govern
implementation.

## Evidence and limits

The owner reported incorrect desktop layout and design. Earlier browser screenshots
were incorrectly described as "pixel-clean" and used as evidence of native usability.
They establish only what those browser captures displayed; they are not design
approval, native-webview verification, or a successful end-to-end workflow.

This study combines Magpie source, Felicia source/contracts, and first-party product
documentation. Bear's pane-layout illustration and Linear's display-options illustration
were visually inspected. The applications themselves were not launched or interactively
reviewed for this study. Adobe documentation fetches returned HTTP 403; Lightroom claims
below are search excerpts from first-party documentation, not a full-page walkthrough.
Apple HIG pages were JavaScript-rendered; their guidance is likewise supported by
first-party search excerpts, pending full-page confirmation before implementation.
No popularity ranking or comparative performance claim is implied by reference selection.

## Reference comparison

| Reference | Evidence | Borrow | Do not copy |
| --- | --- | --- | --- |
| Magpie | Read-only shelf at `e0121f71b9c1d3df363f4aea4c77f8fc62e32164`; source [M1–M4] | Quiet system-font controls, compact toolbar, separate scroll regions, short-window tests, platform-aware chrome | Menu-bar-first navigation, provider dashboard, tiny control dimensions without accessibility review |
| Bear | Pane modes and first-party illustration [B1–B2] | Library → item list → focused editor; collapse nonessential panes while keeping a way back | Its tagging model or forcing three panes at every window width |
| Craft | Styling documentation [C1] | Contextual inspector for appearance rather than putting every option above the editor | Its sharing/accounts model or ornamental document chrome for application controls |
| Linear | Display options and favorites [L1–L2]; illustration inspected | View-local search/filter/actions; secondary options in a labeled popover; predictable selected state | Task-management vocabulary, status taxonomy, or its entire command system |
| Lightroom | First-party local-access/export search excerpts [A1–A2] | Distinct source selection, review, editing and destination confirmation; local files need not imply cloud sync | Cloud-first onboarding or all photographic editing tools |
| Apple | HIG materials and sidebar guidance [H1–H2] | Navigation/control layer distinct from content; platform chrome and readable fallback materials | Assuming a Wails backdrop automatically makes Svelte components native Liquid Glass controls |

### Magpie source observations

Pinned source links are preferable to moving upstream screenshots:

- [M1] `app.css:2–102` defines semantic light/dark surfaces and system-font stacks;
  `:106–116` uses a 13px base and explicit focus outlines. Borrow the distinction
  between chrome and content, not every numeric value.
- [M1] `app.css:156–163` gives the toolbar a bounded height and macOS traffic-light
  inset that accounts for page zoom. `:257–267` contains progressively tighter header
  variants; `:291` gives the content view `min-height: 0` and its own vertical scrolling.
- [M2] `library.css:65` contains a save-status/action region; `:249–251` keeps empty
  states concise. These are source patterns, not measured usability results.
- [M3] editor-short-window, header-zoom-fit and header-centre tests encode prior
  layout regressions. They are valuable acceptance examples, but do not establish
  that Magpie has comprehensive browser/native e2e coverage.
- [M4] window size/zoom persistence and platform handling are desktop mechanics to
  evaluate. A tray panel is not a good primary home for Felicia's maps and essays.

References remain read-only at `refs/desktop-apps/magpie/`. No new reference clones
or borrowed assets are required; if code is later copied, review its MIT notice and
asset licenses separately.

## Current layout pressure points

| Root | Source | Proposed correction |
| --- | --- | --- |
| Repeated identity consumes working area | `App.svelte`: sidebar brand, workspace eyebrow, topbar Felicia label, hard-coded YP mark; `JourneyList.svelte`: page eyebrow/header | One compact workspace identity; remove fictitious profile chrome; move locale to real settings |
| Large inline setup displaces the library | `JourneyList.svelte:214–269`: seven-field creation or folder scan above the journey list | Separate New journey sheet and Import review flow; the library remains a browsing workspace |
| Desktop shell retains page-like spacing | `apps/felicia-admin/src/app.css:31–126` shell/content/header rules | Bound toolbar height, pane-based scrolling, shared edges, content-first density |
| Folder convention leaks into onboarding | Scan form expects a workspace path; desktop importer discovers root JSON/GPX/photos | Explicit Timeline, GPX and photo selection; keep trip-folder import as an advanced shortcut |
| Build/export context competes with library content | `JourneyList.svelte` build status/action region | Keep publication state in contextual toolbar/status and a dedicated output screen |

These are source-backed design pressures, not a confirmed explanation of every native
rendering defect. Before CSS changes, capture the actual native baseline using synthetic
data and inspect loading, list, form, editor and preview states.

The one-shot Create-button investigation was also inconclusive: creation dates initialize
as empty strings (`JourneyList.svelte:35–36`), and the button requires both dates
(`:231`). Date-widget appearance is not proof of populated Svelte state. Integrated tests
must fill labeled dates explicitly and assert the persisted journey; do not record a
separate disabled-button defect without reproducing it through valid user input.

## Recommended direction

**A content-first desktop studio: compact library shell, focused journey workspace,
and separate import/output tasks.** Keep Svelte and shared application services; do
not fork a second authoring implementation or create a generic desktop framework.

The public reader keeps its existing Atlas contract. The historical
[UX restyle](ux-restyle.md) governs the reader, not the authoring shell. The older
[reader/admin research](reader-admin-surfaces.md) contains superseded server-auth and
Postgres assumptions: use current [AGENTS.md](../../AGENTS.md),
[layout](../development/layout.md), and [publication](../publish.md) as authority.

### Alternatives considered

| Direction | Advantage | Why not recommended |
| --- | --- | --- |
| Restyle the existing long admin page | Smallest CSS diff | Does not address workflow separation or actions below the fold |
| Put every task into a permanent three-pane workspace | Fast expert switching | Overcrowds the default window and exposes irrelevant controls |
| Make the public map the entire authoring shell | Strong reader identity | Blurs private authoring/public output and makes prose/media tasks secondary |
| Library plus adaptive editor, task sheets for import/output | Clear hierarchy; familiar authoring pattern | Requires deliberate routing and capability handling, but reuses existing services |

### Shell and visual language

- Retain Felicia's amber accent and warm content identity. Use quiet neutral chrome,
  system sans-serif UI text, and restrained headings; paper/serif expression belongs
  primarily to authored content and public preview, not every setup form.
- Proposed starting dimensions: 48–56px toolbar, 180–200px expanded navigation,
  14px UI body, 20–24px page titles, 8/16/24px spacing rhythm. These are proposed
  tokens to validate in wireframe review, not copied measurements or binding thresholds.
- Primary navigation: Journeys and Site/output, with Settings below. Import is a
  library action, not another permanently empty top-level page. Export remains
  discoverable from a journey and from Site/output.
- Show local/private workspace identity and real save/job/error states. Remove
  decorative account chrome. Never represent unsupported actions with silent no-ops.
- Native picker actions appear only when the desktop capability exists; browser admin
  gets an explicit supported alternative. Share components and operations across hosts.
- Liquid Glass is optional chrome polish. Keep editor text/media on stable readable
  surfaces; respect reduced transparency/motion, contrast, and older OS fallback.
  Do not assume a backdrop option alone produces the intended appearance.

### Screen wireframes

These are layout proposals, not screenshots of implemented UI.

```text
Journey library · default 1100 × 760
┌─────────────────────────────────────────────────────────────┐
│ window chrome   Journeys          Search   Import   New      │
├─────────────┬───────────────────────────────────────────────┤
│ Felicia     │ All · Drafts · Published       Sort / View     │
│ Journeys    │                                               │
│ Site/output │ [cover] Kyoto       dates · draft · 8 mementos │
│             │ [cover] Izu         dates · published          │
│             │ [cover] ...                                   │
│             │                                               │
│ Settings    │ concise empty/error state when applicable     │
├─────────────┴───────────────────────────────────────────────┤
│ Private local workspace                 import/build status │
└─────────────────────────────────────────────────────────────┘
```

```text
Journey workspace
┌─────────────────────────────────────────────────────────────┐
│ Back · Kyoto         Saved     Review public output   Export │
├─────────────┬───────────────────────────┬───────────────────┤
│ Mementos    │ Active task: map / essay  │ Context inspector │
│ Visits      │ / gallery                 │ only when needed  │
│ Photos      │                           │                   │
│             │ selected content          │ metadata/visibility│
│ + Memento   │                           │                   │
├─────────────┴───────────────────────────┴───────────────────┤
│ Draft · private         changes / error / retry status       │
└─────────────────────────────────────────────────────────────┘
```

At narrower sizes, collapse global navigation first and present the inspector as a
sheet; do not squeeze map, prose, list and inspector simultaneously. Focus mode hides
secondary panes with a visible restore action, following Bear's pattern.

```text
New journey sheet                  Import task
┌───────────────────────────┐      ┌───────────────────────────┐
│ New journey           ×   │      │ Sources → Review → Apply  │
│ Title                     │      │ Timeline [Choose file]    │
│ Place                     │      │ GPX      [Choose file]    │
│ Start date · End date     │      │ Photos   [Choose folder]  │
│ More: slug/country/region │      │ warnings, counts, dates   │
│ inline validation/error   │      │ visits ≠ route geometry   │
├───────────────────────────┤      ├───────────────────────────┤
│ Cancel             Create │      │ Back          Apply import│
└───────────────────────────┘      └───────────────────────────┘
```

New-journey required fields remain title/place/dates; derive and validate the slug
without hiding collisions. Secondary metadata is disclosed on demand. Sheet content
scrolls independently while actions remain accessible. Cancel preserves the library
and does not create a database row.

Public-output review uses the real compiled reader, explicitly labeled published-only.
Export confirms destination and explains the private/public boundary. Keep private
backup separate. Account publishing and credential changes are not part of this redesign.

## Implementation sequence after direction approval

| Slice | Owned area | Acceptance before next slice |
| --- | --- | --- |
| 0: baseline and integrated harness | Desktop test entrypoint, admin Playwright config/specs, Makefile | Committed Bun/Playwright harness, synthetic isolated DB/media/output, explicit readiness and teardown; actual native baseline captured |
| 1: shell/library | `App.svelte`, shared styles, `JourneyList.svelte`, locales | Library content/actions fit default and short windows; create sheet persists a valid journey and reports errors |
| 2: source selection/review | Admin import components; desktop native adapter; shared intake contracts | Choose Timeline alone or GPX/photos, cancel safely, show review before apply, preserve authored fields on re-import |
| 3: focused authoring | Journey detail and memento editor | Adaptive panes; readable essay/gallery; keyboard save/retry and visible dirty/saved/error state |
| 4: public review/output | Site/output, publication UI and desktop host | Real published-only preview; export privacy checks; CLI/local read-only serving retained |
| 5: native polish and independent acceptance | Wails chrome, platform packaging, docs | Native visual walkthrough, fallback accessibility, fresh independent review and owning gates |

Do not introduce placeholder UI for missing backend capabilities. Source selection needs
a reviewed runtime/adapter contract; current folder discovery is not a complete multi-file
picker/import-review implementation. Display pending operations honestly.

## Verification contract

Expose a single documented `make e2e` entrypoint, using project-pinned Playwright with
Bun and committed specs/config. A desktop suite may be selected within that entrypoint;
reconcile with the workstation's existing `harus-workstation:playwright-verify:task-3`
before inventing a competing target. No `/tmp` scripts, `bunx` version drift, swallowed
click failures, or API-only probes count as UI e2e.

- Boot the real desktop composition against isolated synthetic state. A test transport
  must be explicitly enabled, loopback-only, authenticated, and unavailable in normal
  packaged operation. The current uncommitted debug listener does not meet this bar.
- WebKit and Chromium specs fill by accessible label, click actual actions, assert the
  response/navigation and persisted results, and record browser errors. Native-picker
  cancellation/selection needs native coverage; test doubles must be labeled as such.
- Exercise empty/populated library; create validation/success/failure; Timeline-only
  import; GPX/photos import; malformed input; cancellation; duplicate/re-import;
  authored-field preservation; edit/save/reopen; build failure/retry and export privacy.
- Geometry/visual cases: 1100×760 default, 900×600 compact, 1440×900 expanded; a browser
  reflow case at 720×600 and 200% zoom. Proposed native minimum must be reconciled with
  this matrix. No horizontal overflow, hidden primary actions, obscured focused controls,
  toolbar collisions or lost pane-restoration controls. Test Japanese/English/Chinese,
  long titles, light/dark, reduced motion and opaque fallback.
- Commit deliberate visual baselines with synthetic fixtures. Snapshot diffs require
  human inspection; never update them blindly to make a failing gate green. Passing
  snapshots do not establish that the design is good.
- Native macOS acceptance is separate: inspect real Wails window, resize/scroll, keyboard
  navigation, save/error state, pick/cancel, map rendering and published-reader preview.
  Use app-scoped capture or a documented native automation route; do not capture unrelated
  desktop content. Missing permission/tooling leaves this criterion unverifiable, not passed.
- Before a commit run `make check`; before PR run `make validate` and the integrated
  e2e gate. Run documentation gates for this plan. Tier-1 completion also requires a fresh
  independent peer review. No release or merge follows merely from passing local checks.

## Decisions requested from the owner

1. Approve the library → focused editor direction instead of restyling the long page.
2. Approve neutral compact chrome with amber/warm content, keeping glass to navigation.
3. Approve the slice order: baseline/e2e and library/create first, then import/editor/output.

Specific tokens and breakpoint behavior remain wireframe/prototype validation questions.
Native capture availability is a verification prerequisite, not a reason to ask the owner
to debug every layout failure.

## Independent plan review

A fresh Herdr Pi peer (`redesign-review`, local pane `w1:p55`) reviewed the
proposal and navigation link at Felicia HEAD
`15693e1418ac42a915b1005241e64505ae418c2c`, including the uncommitted documentation
scope. Runtime model access was confirmed as `openai-codex/gpt-6-luna`, medium
thinking, the owner-selected alternative. The existing dirty desktop `main.go`
was excluded and preserved.

All six documentation acceptance criteria were met: cited reference exploration
with explicit limits; layout alternatives and wireframes; reader/privacy/CLI/shared
Svelte boundaries; approval gate and staged plan; integrated Bun/Playwright and
separate native verification requirements; navigation and documentation gates.
No blocking findings were reported for the plan slice. This is not approval of
implemented UI, native rendering, or an e2e harness that does not yet exist.

Reviewer commands: `make -C vendor/felicia fmt-docs-check`,
`make -C vendor/felicia docs-build`, and `git -C vendor/felicia diff --check`,
all exit 0. Lead documentation gates also passed. The reviewer confirmed that
creation dates start empty and must be filled explicitly; the earlier disabled-button
probe did not establish a separate defect. Direction approval was subsequently received; implementation verification remains separate.

## Sources

- [M1] [Magpie shell stylesheet](https://github.com/yetone/magpie/blob/e0121f71b9c1d3df363f4aea4c77f8fc62e32164/internal/gui/assets/app.css).
- [M2] [Magpie library stylesheet](https://github.com/yetone/magpie/blob/e0121f71b9c1d3df363f4aea4c77f8fc62e32164/internal/gui/assets/library.css).
- [M3] [Short-window editor test](https://github.com/yetone/magpie/blob/e0121f71b9c1d3df363f4aea4c77f8fc62e32164/internal/gui/tests/editor-short-window.test.cjs), [header zoom fit](https://github.com/yetone/magpie/blob/e0121f71b9c1d3df363f4aea4c77f8fc62e32164/internal/gui/tests/header-zoom-fit.test.cjs).
- [M4] [Magpie native application](https://github.com/yetone/magpie/blob/e0121f71b9c1d3df363f4aea4c77f8fc62e32164/internal/gui/app.go).
- [B1] [Bear pane modes and illustration](https://bear.app/faq/hide-the-sidebar-and-note-list-on-mac-and-ipad/).
- [B2] [Bear sidebar](https://bear.app/faq/about-the-sidebar-in-bear/).
- [C1] [Craft document styling](https://support.craft.do/hc/en-us/articles/10293992410780-Document-Styling).
- [L1] [Linear display options and illustration](https://linear.app/docs/display-options).
- [L2] [Linear favorites](https://linear.app/docs/favorites).
- [A1] [Lightroom local photos](https://helpx.adobe.com/lightroom/desktop/add-import-and-capture-photos/access-photos.html) — search excerpt only; full-page fetch blocked.
- [A2] [Lightroom export](https://helpx.adobe.com/lightroom/desktop/save-share-and-export/save-share-photos.html) — search excerpt only; full-page fetch blocked.
- [H1] [Apple HIG materials](https://developer.apple.com/design/human-interface-guidelines/materials) — search excerpt only; full-page confirmation pending.
- [H2] [Apple HIG sidebars](https://developer.apple.com/design/human-interface-guidelines/sidebars) — full-page confirmation pending.
