# Felicia Desktop Studio Redesign: Ordered Execution Plan

> This is the execution contract. Research and design rationale live in
> `docs/research/desktop-studio-redesign.md`; live task state and claims live in
> Asobi. Do not dispatch a later slice while an earlier acceptance gate is open.
> The owner-feedback revision below supersedes the earlier modal-creation direction.

## Goal and constraints

Deliver Felicia as a polished local desktop studio: import → author → preview →
export, while retaining CLI and local read-only serving. Develop on Wails v3;
defer public release until the workflow matures.

- macOS 26+ first (owner-selected support floor), with Windows/Linux planned.
  Compilation, linking and bundle metadata must agree on that floor. Reuse existing Svelte, core, runtime,
  providers and publication modules. Application modules must not import other
  composition modules.
- SQLite, drafts, originals, credentials and admin assets stay out of public
  exports. Re-import preserves authored fields.
- Browser composition tests use real desktop handler/SQLite/assets through an
  authenticated, test-build-only loopback transport. They do not prove WKWebView
  rendering or native dialogs.
- Native acceptance uses synthetic-only state and app-window-scoped capture.
  Public preview/export checks use privacy sentinels.
- Functional green is separate from design acceptance. Screenshots, composition
  tests and native capability probes do not establish good appearance.
- No private macOS API dependency. Content and accessibility must work when
  translucency is unsupported or reduced.
- One source writer per checkout. No commit/push without explicit owner approval.
  Fresh independent Herdr verification is required before tier-1/2 completion.

The approved visual direction is a compact aligned library shell, real Felicia
branding, language in actual settings, no fictional account/profile chrome,
system-aware neutral surfaces with restrained amber, and full-page creation/import
tasks. The focused authoring workspace should adapt to available width. See
[`docs/research/desktop-studio-redesign.md`](../research/desktop-studio-redesign.md).

## Authoring design discussion

The owner selected **write and curate** as the primary task when opening a journey,
and **native macOS studio** as the authoring interface's visual character. The
workspace should prioritize selecting a memento, writing its story and arranging
photos; the map supplies context. The public reader's map-led identity does not
require the authoring editor to put the map first. Use compact toolbars, neutral
surfaces, system typography and restrained color; keep expressive journal styling
in reader previews.

The owner also selected a persistent memento item rail beside one focused editor,
explicit Save/keyboard-save with visible dirty/saving/saved/error states and a
leave-with-unsaved-changes guard, and private preview of the current unsaved
story/photos without publication. The compiled published-site preview remains a
separately named action.

The owner approved a studio-only SvelteKit static SPA migration before further UI
implementation. Use its layouts, routing and navigation hooks while preserving
the Go backend and the separate Vite public reader. Bun remains the package
manager and the real runtime for compatible commands; Node LTS is an explicitly
permitted fallback on demonstrated runtime capability failures. Keep Node-style
source, Vitest and Playwright. The abandoned pnpm migration is not a delivery.

A first migration slice reuses the existing screens, retains hash deep links,
and checks embedded assets, malformed URL recovery, history, locale-preserved
edits and the unsaved-change leave guard. SvelteKit hash mode disables SSR and
prerendering itself; do not also set those page options. References:
[SvelteKit SPA](https://svelte.dev/docs/kit/single-page-apps),
[router configuration](https://svelte.dev/docs/kit/configuration#router), and
[navigation guards](https://svelte.dev/docs/kit/$app-navigation#beforeNavigate).

Remaining S3 workspace details are still being discussed. The owner approved
continuing the framework and revised S1 slices; those choices are not a completed
workspace design or native acceptance. S2 remains gated on revised S1 acceptance.

## Owner-feedback revision — 2026-10-03

Acceptance issue: [#158](https://github.com/azusachino/felicia/issues/158).

The owner rejected the current studio's design consistency and scan workflow after
inspecting the interface and supplying a scan-dialog screenshot. Browser-functional
passes do not accept that design. The owner explicitly selected:

- Full-page New journey and Import tasks, not modal task forms.
- shadcn-svelte as the shared control foundation, with one Felicia theme across
  library, journey/editor and Site & Deploy. Component adoption alone is not visual
  acceptance; persistent inline status/errors must use one consistent pattern.
- A separate synthetic demo workspace, never automatic seeding of the real journal.
- macOS 26+ support, with one verified deployment target rather than suppressed
  linker warnings or an unsupported older-version claim.

The static Local workspace label/green dot must be removed: it is not a real health
or workspace-selection state. The existing Refresh action only reloads journey
summaries. Remove it from the primary toolbar unless a genuine user need remains;
any retained action must name its scope and refresh all state it claims to update.

### Ordered revision slices

1. **Platform contract:** set compilation/link/bundle minimum to macOS 26.0 across
   canonical native build and test targets, preserving other-platform behavior.
   Gate: a real packaged build has no deployment-target mismatch warnings; inspect
   Mach-O build-version metadata and plist; retain signature checks. An SDK version
   of 27 does not by itself require a deployment minimum of 27.
2. **Shared studio foundation:** add only needed shadcn-svelte primitives and
   shared tokens; apply a single toolbar, panel, field, button and inline feedback
   language to library, journey/editor and output pages. Remove misleading static
   status. Gate: all routes/states visibly share the foundation in ja/en/zh at the
   owning sizes/zoom, while existing authoring/publish behavior remains intact.
   This styling work does not claim S3's adaptive authoring-pane behavior is done.
3. **Full-page creation:** a dedicated navigable route with Back/Cancel, required
   title/place/dates, editable derived slug under optional details, visible errors
   and input preservation. Gate: no task dialog; create/reload persists; cancel has
   no database effect; collisions/date errors and keyboard flow still pass.
4. **Isolated demo:** explicitly open a synthetic sample workspace with journey,
   visits, mementos, essay and safe photos that exercise populated library, editor
   and published preview. Gate: author DB/media/output/credentials are never read,
   seeded or overwritten; reopening the demo is deterministic; real data is
   unchanged; the UI clearly identifies demo mode and how to return to real work.
5. **S2 source workflow:** replace the legacy scan popup with a full-page
   Sources → Review → Apply task. Explain Timeline visits versus GPX geometry,
   optional photos, detected dates/counts/warnings and the no-mutation-before-Apply
   boundary. Trip-folder discovery is an advanced shortcut, not required setup.
   Gate: native selection/cancel, review/input identity, re-import protection and
   malformed/empty/changed-input recovery as specified by S2 below. Current desktop
   Apply rescans inputs and the fingerprint covers only one source (or a constant
   fallback); S2 must bind the entire reviewed input set, not merely restyle it.

Slices 1–4 are the revised S1 acceptance scope. S2 source implementation remains
blocked until that revised S1 scope is accepted; retiring the scan modal in S1
must not masquerade as completed import. S3/S4 retain their deeper workflow gates.
Use grouped `make local-check` while iterating; invoke affected browser specs for
behavior and one final uncached owning gate per delivery boundary, not full gates
for every line. The owner subsequently approved local checkpoint commits and
immediate implementation. Pushes, releases and remote setting changes remain
unapproved.

Primary component references: [Vite integration](https://www.shadcn-svelte.com/docs/installation/vite),
[Button](https://www.shadcn-svelte.com/docs/components/button), and
[Popover](https://www.shadcn-svelte.com/docs/components/popover). Preserve existing
Svelte 5/Vite/Tailwind 4 tooling and inspect generated source; do not let a CLI
initialization overwrite existing global styling or add unused components.

### Routing foundation — interim checkpoint

The owner selected a maintained router instead of extending the handwritten
route union/parser. At checkpoint `7123d5f`, the admin used pinned
`svelte-spa-router` 5.1.1 for Svelte 5. This interim implementation is superseded
by the static SvelteKit migration below. The interim router provided a hash-based
path-to-component table, parameter decoding, history and
programmatic navigation. `src/routes.ts` owns the table; `src/router.ts` only
constructs URL strings. The small `RouteView` adapter forwards library parameters
to existing page props and live locale settings, remounting on identifier changes
but preserving drafts on language changes. Adding a page requires a table entry,
not a parser branch or shell rendering branch. Unknown URLs retain the library
fallback; malformed encoded identifiers redirect to it through library conditions.

References: [getting started](https://svelte-spa-router.italypaleale.me/docs/getting-started/)
and [navigation](https://svelte-spa-router.italypaleale.me/docs/navigation/).
Browser regression coverage includes deep links, trailing slashes, encoded IDs,
query strings, fallback, back/forward, identifier changes and live locale updates.
The desktop memento/locale test supplies only an empty per-memento photo list:
that endpoint is absent in the existing desktop handler. This is routing evidence,
not proof of desktop authoring API parity; the web-host authoring closed loop uses
its real photo endpoint. Track that desktop gap when authoring work is dispatched.

Fresh independent Herdr peer `felicia-router-review` (`wV:p4S`), Pi
`openai-codex/gpt-6-luna` medium, verified the staged router checkpoint on
`feat/desktop-studio-revision` at parent `1d08e3ee0fd835aaac3693eb08abaec25e9dac00`
(source/index tree `5292c9e33eed0d697e6317cf841f47a51b80f37b`, before this evidence
paragraph). It found no material routing issues. The exact staged source export,
without the unfinished shadcn/task-page work, passed locked installation,
`make admin-check` (101 unit tests), `make e2e` (30/30 Chromium/WebKit),
`make test-admin-e2e` (14/14 real web-host closed loop) and `make docs-build`.
Snapshot-only `make check` stopped at four existing Markdown exemptions because
its absent Git root changed path matching; the peer then ran canonical
`make check` in the owning checkout, exit 0. No rules or unrelated docs were
weakened. Browser evidence is not native visual acceptance. Local checkpoint
commits are approved; no push or release is approved.

### Static studio and portable tooling checkpoint

The migration replaces the interim route table with SvelteKit filesystem pages,
static adapter output at `dist/index.html`, hash navigation and framework guards.
`vite.config.ts` owns the configuration; the unused `svelte.config.js` is removed.
The shared shell/context preserves locale changes without discarding an editor.
Only malformed URL normalization is custom. Go still embeds static assets; the
public reader remains Vite and no JavaScript server is deployed.

Bun owns frozen workspace installation and actual compatible execution. TypeScript
source uses portable `node:*` APIs, Vitest and Playwright; Node LTS is a capability
fallback, not another package manager. The runtime assertion checks the actual
engine. Project-local mise selectors and both runtime identities enter local cache
keys; full gates never consume the iteration cache.

Fresh read-only Herdr peer `felicia-kit-review` (`wV:p40`), Claude Sonnet 5.5,
launched at medium effort, independently verified the working diff at
`7123d5f4377d6f0b36a3f25045bebaf870609bf9`. After review fixes, `make validate`
passed (74 Python and 120 Vitest tests), uncached desktop tests passed, browser
composition passed 32/32 Chromium/WebKit, and real web authoring passed 14/14.
The peer inspected real held-response and stored-value assertions, not only
successful test output. Save/keyboard-save submit the form and dirty photo rows,
retain later edits, and guard internal leave and browser reload. Photo persistence
is verified against the real web API; desktop photo-list parity remains open.

The new keyboard test exposed a desktop adapter that decoded essays but omitted
them from manual-patch fields. Local checkpoint
`f3958781438d8919da6bdce7e2a1edf2d94f9b5f` repairs that omission and tests
create/edit/clear on SQLite. The endpoint is full-payload: an omitted essay clears
its stored value, as required by blank-form serialization. This is not a general
API-parity fix. The peer was closed after its final report; raw logs are disposable
workstation scratch under `.tmp/felicia-desktop-feedback/kit-review/`.

Fresh checkpoint verifier `felicia-kit-checkpoint` (`wV:p54`), Claude Sonnet 5.5
at launcher-selected medium effort, verified staged tree
`42ad1a82a95b7302b7d64cc0c20df1d539072a2d` at parent `f3958781`. All 503 staged
blobs matched the clean export; unfinished controls/task pages, their dependencies
and creation/import translations were excluded. Frozen install, `make validate`
(74 Python and 120 Vitest tests), `make e2e` (32/32), `make test-admin-e2e` (14/14)
and `make docs-build` exited 0. Matching Bun/Node runtime probes passed; the
wrong-runtime control failed. The lead also ran canonical `make check`, exit 0.

The peer's findings were stale Make help labels and generated index churn. The
lead corrected help to name Bun/Vite or Bun/SvelteKit and excluded both generated
desktop indexes, preserving their worktree outputs; the frozen lock remained
byte-identical after gates. Composition gates rebuild embedded assets before use.
Native Wails confirmation/unload/keyboard behavior and visual acceptance are still
unverified. Item rail, private unsaved preview, full-page tasks, demo and S2 import
identity are not delivered by this slice.

### Shared foundation — first increment in progress

At checkpoint `f04eec52ce4827e550985278dff88f101ea19ef2`, the lead started the
shared-control slice: shadcn Button now serves library toolbar actions; Tailwind
semantic colors map to the existing system-aware studio tokens. The fictional
Local workspace indicator and ambiguous Refresh action are removed. Unused date
and animation dependencies were dropped; native date inputs remain the baseline.

Self-verification: `make admin-check` passed with 0 diagnostics and 104 tests;
targeted compact-shell/reduced-motion browser checks passed 8/8 Chromium/WebKit
at 1100×760, 900×600 and 720×600, including absence of fake status/Refresh and
existing creation/focus behavior. The browser creation-sheet image was inspected;
it still shows the legacy sheet, not the approved future full-page task. This
increment is uncommitted and has no fresh independent acceptance yet. Remaining:
shared editor/output controls, panels and inline feedback, locale/dark/zoom
coverage, then full-page creation and isolated demo. S1 is not complete.

Owner feedback extended this slice to locally persisted System / Light / Dark
appearance (default System), independently of published-site settings. Settings
now composes shadcn Popover primitives; ModeWatcher owns the document dark class.
Site output/identity/build panels use studio tokens and shared action Buttons;
the bounded output-path grid wraps long paths. The scan form's Browse action is
adjacent to, not nested inside, the input label. No generic Settings wrapper or
second studio palette was introduced.

Self checks passed: `make admin-check` (104 tests, 0 diagnostics) and targeted
`make e2e ARGS='--grep "settings|appearance|site folder|compact shell|motion" --workers=2'`
(20/20 Chromium/WebKit). These cover ja/zh preferences, OS-theme changes, saved
appearance after reload, unchanged published settings, Escape/focus restoration,
reduced motion, and Site action bounds at 1100/720px. Synthetic dark Site and
Settings captures were inspected. Browser testing caught and repaired an
undefined bound Button ref (Site render failure), missing popover dialog role,
and WebKit close-focus restoration. Picker execution/native acceptance and the
remaining journey/editor palettes and controls are not certified by this check.
Changes remain uncommitted and independently unaccepted; task 12 stays in progress.

### Icon-first action increment — independently verified, uncommitted

Owner preference: familiar Browse, Settings, Back and photo reorder controls use
icons with localized accessible names and hover/keyboard-focus hints. New, Scan,
Create, Save, Build and Import use icons with compact text; keep explicit text for
ambiguous lifecycle/save-and-back actions and destructive confirmations. New
icons share the Lucide set (`@lucide/svelte`), current color and decorative SVG
semantics. Shared Buttons retain native links for Back and existing handlers,
pending/disabled states and save protection.

The live library, Site, journey navigation/build and memento/photo action changes
were independently accepted by fresh task-created Herdr peer
`felicia-icons-verify` (`wV:p57`), Claude Sonnet 5.5, launcher medium, against
`feat/desktop-studio-revision` at `f04eec52ce4827e550985278dff88f101ea19ef2` plus
the working diff. Final gates: `make admin-check` exit 0 (104 tests, no
diagnostics), `make e2e ARGS='--workers=2'` exit 0 (46/46 Chromium/WebKit), and
`make test-admin-e2e` exit 0 (14/14 real authoring checks, including persisted
caption/reorder and compiled essay/site identity). Back link hover/keyboard hints,
activation, modal cancellation/focus restoration, locales, theme and zoom checks
pass. Pending labels were also reviewed in source; not every pending action was
held in a browser. Synthetic Site captures were inspected.

Review exposed a real native HTML dialog top-layer bug: the tooltip Portal mounted
before its trigger ref existed and fell back to the body. Mounting only while the
hint is open resolves the nearest open dialog at mount; the unchanged regression
then passed in both browsers. Settings also unmounts closed hints to avoid stale
hint nodes during keyboard navigation. Evidence:
`.tmp/felicia-desktop-feedback/icons-review/accepted-report.md`, with distinct
final `admin-check3.log`, `e2e3.log`, `admin-e2e3.log`; the prior failed `e2e2.log`
was 42/46. `cn`/Lucide usage follows the
[official manual setup](https://www.shadcn-svelte.com/docs/installation/manual).

This accepts the icon increment only, not the complete shared-foundation task.
Remaining independent journey/editor palettes, full-page tasks, synthetic demo,
picker API parity and native Wails acceptance remain open. Unfinished task views
and generated asset changes stay preserved and uncommitted; no push/deployment.

### PR publication scope

The owner approved publishing the verified checkpoints together from
`feat/desktop-studio-revision`: local gate/cache repair and macOS 26 build floor,
static SvelteKit/Bun tooling, desktop essay persistence, and the live
appearance/shared-control/icon increment. This PR does not complete revised S1
or close [#158](https://github.com/azusachino/felicia/issues/158).

The owner corrected the initial publication scope after
[PR #159](https://github.com/azusachino/felicia/pull/159): full-page creation is
part of this PR, not an excluded prototype. The PR is draft while the dedicated
`/journey/new` route is implemented and verified. Include `NewJourney.svelte`
and the Input/Label controls it uses. Library New opens that route instead of a
creation dialog. Required acceptance: direct entry and initial title focus,
Back/Cancel with no accidental write, guarded unsaved/pending navigation,
required title/place/dates, visible date-range validation, editable derived
slug, optional metadata, retained inputs on collision and successful retry,
real persisted creation/reload, one write while pending, locale and bounded
small/zoomed layouts. Browser checks do not establish native Wails acceptance.

Exclude unwired `ImportJourney.svelte`, unused Textarea/Badge scaffolds,
import-page-only translations, and generated desktop asset index churn. Keep
those worktree artifacts intact; legacy scan/apply is not the reviewed S2
source contract. The candidate includes the live journey Back translation. Verify an
isolated export of the exact index with frozen Bun install, uncached project and
browser gates, then a fresh independent pass before publication. Previous
working-diff reviews remain historical evidence, not acceptance of a changed
candidate. No release, deployment, native-window launch or repository-setting
change is authorized by opening this PR.

### Platform revision checkpoint

At `main` / `9c38309074645a425294fbdef9bc19c2c76d925a` plus the working diff,
`Info.plist` now selects macOS 26.0. The shared Go task runner derives desktop-only
compile/link/deployment flags from that metadata, including canonical production
and test-only builds/tests. Other modules, platforms and non-Darwin cross-targets
retain their environment. This corrects mixed 11/13/27 deployment targets without
suppressing warnings or making SDK 27 the support minimum.

Fresh independent Herdr peer `felicia-macos26-review` (`wV:p4R`), Pi
`openai-codex/gpt-6-luna` medium (lead-confirmed startup/footer), found no platform
slice issues. It ran `make local-check ARGS='--groups go scripts'` (Go format/vet/
lint/tests and 73 Python tests), `make desktop-package`, and `make e2e` (24/24
Chromium/WebKit), all exit 0. Production and headless E2E Mach-O metadata report
`minos 26.0`, `sdk 27.0`; the packaged plist is 26.0; signature verification passed.
Package/E2E logs contain no deployment mismatch warnings. No native window was
launched: runtime/visual acceptance on macOS 26 remains a separate gate.

The shared component foundation, full-page task navigation and isolated sample
are still unimplemented. Builds updated the already-dirty generated reader index;
it was left intact, not silently reset. This verified platform/tooling checkpoint
was committed as `1d08e3e` on `feat/desktop-studio-revision` after owner approval.

## S0 — integrated browser harness and native baseline

Owned paths: desktop test-only transport, admin Playwright config/fixtures/specs,
Make/CI and evidence docs.

1. Pin Playwright in the admin workspace and provide canonical `make e2e` and
   install/build targets. Launch the exact production-composition test binary
   against isolated synthetic SQLite/media/output state; use a random auth token,
   literal loopback binding, readiness deadline and exact-PID teardown.
2. Compile the transport only under an explicit e2e build tag. Normal builds
   expose no test listener or bypass. Exercise the real handler and assets.
3. Capture browser screenshots and accessible snapshots; retain strict console,
   network, persistence, isolation and cleanup checks. Add a macOS CI job, but
   distinguish structural CI review from an actual remote run.
4. Capture and inspect the normal Wails app's initial empty-library window with
   synthetic paths. This is baseline evidence only.

Acceptance: normal build cannot enable test transport; auth/origin guards work;
all browser engines pass; no test child/listener remains; app-only native baseline
is captured and inspected. This does not test native pickers or the redesigned UI.

## S1 — library shell and creation

Owned paths: shared admin shell/styles, `JourneyList`, locale catalogs/specs,
desktop window chrome adapter and creation handler tests.

1. Replace repeated brand/page chrome with one compact, aligned library toolbar.
   Keep actions reachable at short heights and align title/action geometry.
2. Use the real Felicia mark and coherent accessible icons. Remove fake YP/login
   identity. Put language selection in a compact, anchored Settings popover. On macOS
   allow native traffic-light clearance; on other OSes preserve native window chrome.
3. Use quiet system-aware neutral surfaces and restrained amber. Native material
   is platform-specific and optional; maintain opaque/readable fallback behavior.
   Do not treat a backdrop flag or CSS blur as proof of Liquid Glass.
4. Keep New journey creation in its own full-page task with Back/Cancel,
   route/focus behavior, scrollable body and stable action footer. Respect reduced
   motion for any transitions. Require title, place
   and dates, derive an editable slug, disclose optional metadata, surface slug
   collisions and server failures without discarding input. Source import is a
   separate Sources → Review → Apply task owned by S2, not another creation mode.
5. Keep app-level window intent independent of Wails OS settings. A small
   platform adapter maps that intent to macOS options; other platforms retain
   Wails native defaults unless and until their own treatment is implemented.
6. Assert screenshots/composition at 1100×760, 900×600, 1440×900 and responsive
   720×600; test ja/en/zh, long titles and 200% zoom. Inspect actual native chrome
   and material independently.

Acceptance: the revised owner-feedback slices 1–4 above are covered; create
persists and navigates; cancel adds no row; collisions preserve existing records;
invalid dates are rejected; actions and focus remain reachable;
all geometry and locale assertions pass. Native window must render and be visually
inspected before S1 is DONE.

## S2 — source selection and review

Owned paths: admin import components/API contracts, desktop picker/local import,
shared intake use cases only where needed, provider/handler/spec tests.

1. Implement import as a task separate from New journey creation, with explicit
   Sources → Review → Apply stages. Offer explicit Timeline file, optional GPX file
   and photo-folder selection; trip-folder discovery remains an advanced shortcut,
   not compulsory onboarding.
2. Explain semantic visits separately from track geometry. Show source dates and
   timezone, counts, duplicates, unsupported records and warnings before apply.
3. Bind Apply to the reviewed plan/input identity. Do not silently rescan changed
   inputs or overwrite authored fields.
4. Make browser-host behavior explicit. Component test doubles are not native
   dialog verification. Test native picker selection and cancellation separately.

Acceptance: Timeline-only and GPX/photos fixtures; malformed/empty input; native
pick/cancel; no mutation before Apply; duplicate/re-import and authored-field
protection; failure recovery preserves user state. Semantic visits do not invent
route geometry.

## S3 — focused authoring workspace

Owned paths: `JourneyDetail`, `MementoEditor`, shared layout/components, locale and
specs.

1. Use a journey item rail, active map/essay/gallery workspace and contextual
   inspector. Collapse global navigation first; move the inspector to a sheet
   when panes stop fitting.
2. Provide visible restore controls for collapsed panes and a focus mode that
   retains navigation. Do not permanently squeeze every editor into a dashboard.
3. Preserve existing memento vocabulary/lifecycle, dirty/saved/error/retry states,
   ordered media and keyboard operations. Do not change reader animation or
   navigation contracts.

Acceptance: author/save/reload via real desktop handler; failed saves retain
input; pane restoration and focus order; ja/en/zh and resizing; existing web
closed-loop workflow passes.

## S4 — public-output review and export

Owned paths: `SiteDeploy`/output UI, desktop publication/destination adapter and
publication tests. Preserve local read-only server and CLI contracts.

1. Label the real compiled-reader preview as published-only, distinct from private
   authoring. Build status and failures come from real artifacts, not placeholders.
2. Confirm export destination and public subset. Reveal/open only after success;
   reject unsafe destinations and preserve prior successful output after errors.
3. Keep private backup distinct. Account-based publishing/auth changes require a
   separate decision and are out of scope.

Acceptance: root/subpath preview; safe derivatives and artifact manifest; scans
exclude private sentinels/paths/credentials/SQLite/admin assets; failed build/export
has retry and prior output remains usable.

## S5 — native acceptance and delivery

1. Inspect actual Wails window with synthetic content in loading/library/sheet/
   import, editor and compiled-reader states. Check resize, scroll, focus, pick/cancel
   and map rendering.
2. Review each target OS's native chrome. Verify macOS material only where
   supported and readable reduced-transparency/motion, dark/light and macOS 26
   minimum-version fallbacks. Do not require private APIs. OS/CPU support remains a packaging
   contract decision.
3. Run owning gates, integrated suites and documentation builds. Obtain a fresh
   Herdr Luna medium review. Persist criterion-by-criterion evidence in Felicia
   docs and (after authorized delivery) the owning PR.
4. Obtain explicit commit/push approval. Direction approval is not delivery approval.

Acceptance: `make check`, `make validate`, integrated desktop and existing web
suites, `make docs-build`, packaging/signature checks, independent review and
native evidence. Missing native capture/automation is blocked/unverifiable, not
passed.

## Execution checkpoint

### S0 complete

Fresh Herdr peer `desktop-s0-review` (`w1:p56`), Pi
`openai-codex/gpt-6-luna` medium, verified the S0 working scope at
`15693e1418ac42a915b1005241e64505ae418c2c`. Normal-build isolation,
authenticated real-composition transport, pinned targets, persisted evidence and
observed child teardown met criteria. CI was reviewed structurally only. After
fixing the actual journey build-status route (shared artifact/content comparison,
not a zero-value stub), integrated browser tests passed 6/6; targeted Go tests,
lint, `make validate`, docs checks and docs build passed. The peer inspected the
1099×791 app-only empty-library baseline at
[`native-library-before.png`](../experiments/desktop-studio/native-library-before.png).
This completed S0, not S5 native workflow acceptance.

### S1 merged implementation — acceptance remains blocked

PR [#157](https://github.com/azusachino/felicia/pull/157) merged into `main` at
`9c38309074645a425294fbdef9bc19c2c76d925a`. The merged tracked tree matches the
PR head `8d512df6b467fccaeb6e7cc749524e2f0f7ffcde`; merge is not native acceptance.
The owner requested agent-only follow-up. S1 remains open and S2 is not dispatched.

Fresh task-created Herdr peer `felicia-s1-main-verify` (`wV:p4F`), Pi
`openai-codex/gpt-6-luna` medium (startup argv and live footer confirmed by the
lead), revalidated the merged revision. Initial follow-up results:

- `make e2e` passed 24/24 Chromium/WebKit tests after `make e2e-install` installed
  the missing project-pinned browsers. Compact toolbar and creation/persistence,
  cancel, collision/retry, focus, motion, locale and size checks passed on browser
  composition. Platform intent separation was verified from source. Native chrome,
  material/fallback appearance and visual acceptance remain unverifiable.
- `make validate` exited 2 at the legacy-root assertion: this checkout still has
  `apps/felicia-web/node_modules`. The leftover directory was not moved or deleted;
  the assertion was not weakened. Earlier gate passes below are historical.
- `make test-admin-e2e` exited 2 twice because `/readyz` returned HTTP 503,
  `database is not ready`. `apps/felicia-server/api/server.go` returns that response
  when `ListJourneys` fails; the underlying repository error is not identified by
  this evidence. The web-host closed loop is not currently verified.
- `make fmt-docs-check docs-build`, `make desktop-package` and
  `codesign --verify --deep --strict bin/FeliciaStudio.app` exited 0. No native
  window or capture was attempted. No matching test-server processes or
  Felicia-related TCP listeners remained in the peer's post-run check.
- E2E asset generation changed the tracked reader `index.html` asset hashes. The
  lead reverted only that generated diff to the reviewed content; no implementation
  changes were made. Raw run logs are disposable workstation scratch under
  `.tmp/felicia-s1-main/review/`; this checkpoint preserves the result.

#### Agent-only gate remediation

The lead preserved four retired directories (`apps/felicia-web` and
`packages/felicia-{components,renderers,runtime}`) intact under workstation
`.tmp/felicia-s1-main/legacy-output/`. They contained only ignored dependency
symlinks, not regular files; no layout assertion was changed.

The readiness probe identified a stale September `bin/felicia-cli`: it seeded a
SQLite schema without the journey `revision` column required by the current
server. Rebuilding the CLI made the same synthetic readiness probe pass.
`ensure_cli()` now always invokes the cached `make cli-build`; the existing-binary
regression failed before the fix and passed afterward. This shared fix covers all
local-workflow callers, not just the E2E seeder.

The next web-host run exposed an optimistic photo-reorder test race: reload
aborted sequence writes before persistence. The trace recorded aborted photo
POSTs. The spec now awaits both successful sequence responses before reload;
reload and published gallery-order assertions remain unchanged. The lead reran
`make test-admin-e2e` (14/14) and `make validate` (57 Python and 118 admin tests),
both exit 0. Fresh independent Herdr peer `felicia-s1-gate-review` (`wV:p4H`),
Pi `openai-codex/gpt-6-luna` medium (lead-confirmed startup/footer), reviewed this
five-path working diff at `main` / `9c38309074645a425294fbdef9bc19c2c76d925a`.
It reproduced `make validate` (57 Python and 118 admin tests), `make e2e` (24/24),
`make test-admin-e2e` twice (14/14 each), and `make fmt-docs-check docs-build`, all
exit 0. CLI freshness, both successful sequence writes, unchanged reload/public
order assertions, reversible dependency cleanup and fixture teardown met criteria;
no findings. Native acceptance remains unverified.

Remaining gates: obtain owner-device
native inspection of the library, Settings popover and New journey sheet. Do not
mark S1 DONE or advance S2 on browser results alone. No commit or push is
authorized by this follow-up.

#### Historical pre-delivery checkpoint

Working branch: `feat/desktop-studio-scaffold` at
`15693e1418ac42a915b1005241e64505ae418c2c`; lead-owned changes are uncommitted.
Owner-approved interaction refinement: Settings is a compact anchored popover,
not a sparse modal or toast; New journey remains a focused sheet with restrained
motion only when reduced motion is not requested. The legacy Scan trip folder
branch still shares that dialog until S2 implements a distinct Sources → Review →
Apply Import task. Do not count the S2 flow as implemented.

Fresh independent Herdr reviewer `felicia-s1-verify` (Pi,
`zai-coding-cn/glm-5.3-flash`, low effort) reviewed this working tree at the branch
and commit above. Criteria 1–4 passed on browser-composition evidence; no blocking
findings. The reviewer noted that pointer-outside dismissal does not restore focus;
Escape does. Native visual acceptance remains unverified and S1 is not accepted.

Current gates passed: `make admin-check` (118 Bun tests), `make e2e` (24 Chromium/
WebKit tests), `make validate`, `make fmt-docs-check docs-build`, and
`make test-admin-e2e` (14 web-host workflow tests). The collision case proves
original records and form input are preserved and retry creates a separate
journey. These checks and browser screenshots do not establish native Wails
rendering.

The window chrome has a platform seam: common startup calls
`applyPlatformWindowChrome`; the macOS implementation requests HiddenInset and
Liquid Glass, while non-macOS builds retain Wails native defaults. A macOS
production window with synthetic paths was captured blank except traffic lights;
the evidence is
[`native-library-s1-blocked.png`](../experiments/desktop-studio/native-library-s1-blocked.png).
A separately captured default-titlebar variant is also blank, so HiddenInset alone
is not the cause. Launching the same build from a `.app` bundle did not fix it.
A Normal-backdrop differential was built, but not captured. The owner reports
screen capture was granted during prior Animeko debugging. In this run, the
agent-launched process reports `CGPreflightScreenCaptureAccess() == false`, and
app-window `screencapture -l` fails. A synthetic AppKit control window explicitly
set to read-only sharing reported Core Graphics sharing state 1, yet the same
`/usr/sbin/screencapture -l` path also failed on it. This establishes that the
agent-launched capture path fails even for a capture-eligible control window; it
does not establish that the owner's system permission is absent, nor whether the
Felicia WKWebView rendered. The prior Felicia window's sharing state 0 therefore
cannot explain the capture failure by itself. Wails source review identified
native view layout/compositing and custom-scheme startup as hypotheses only; app
rendering remains unverified. After the latest interaction changes, the synthetic
app was rebuilt by `make desktop-package`; `codesign --verify --deep --strict`
passed. The fresh window (PID 47295, window 438, 1099×759) is onscreen with
sharing state 0. `CGPreflightScreenCaptureAccess()` from the Herdr/Ghostty shell
reported false, and app-only `/usr/sbin/screencapture -x -l 438` failed with
`could not create image from window`. On owner request, `CGRequestScreenCaptureAccess()`
was called from the same Herdr/Ghostty shell and returned `false`. The owner then
approved Ghostty in System Settings, but a fresh preflight still returned false
and a post-approval app-window capture again failed with `could not create image
from window`. The owner then clarified this Herdr session is accessed through
SSH and Felicia runs on the remote Mac, not on the owner device running Ghostty.
The local Ghostty grant therefore does not authorize capture on the remote host.
No remote setting change or reset occurred. The remote requester still reports
false and its capture fails; this does not establish remote permission state or
whether Felicia rendered. Native appearance remains unverified. S1 remains open;
S2 is not dispatched.

## Sample and theme polish — working slice

On `feat/desktop-sample-polish`, the uncommitted studio adds explicit **Open
sample trip** actions in Library and Settings. `--sample` selects a fresh
temporary database, generated media and compiled output before resolving an
author workspace. Native pickers and imports are unavailable in this sample.
The launcher switches the current window, not a second process. **Return to my
workspace** restores the original handler and cleans up the sample's database,
media and preview server; reopening creates a fresh sample. A standalone
`--sample` diagnostic launch has no original workspace and closes on exit.

The first implementation spawned a second process for isolation. The owner
rejected that interaction. The replacement keeps separate immutable workspace
handlers, waits for in-flight authoring requests before switching, and preserves
the original workspace. The frontend leaves through dirty/pending navigation
guards before switching the backend, so cancellation cannot leave an editor
pointing at a different repository.

Blank journey creation and folder import are distinct actions. The sample does
not expose unsupported folder scanning. Creation and Scan offer validation
feedback instead of silently disabling actions for empty fields. Synthetic
browser checks cover Library → New journey → validation → create → reload;
these checks do not establish native Wails acceptance.

Journey dates initialize to the actual local calendar day, not empty values that
WebKit may render as apparent defaults. Untouched defaults do not make a draft
dirty. The creation regression includes submitting without editing either date
and a local-day/UTC boundary. Pending creation blocks anchor clicks before
WebKit queues hash navigation, while route guards continue to protect history
and programmatic navigation. Sample exit labels are simply **Close** or
**Return**, with the banner explaining the workspace boundary.

Creation, journey, and editor return controls are outlined shared `Button`
components with an arrow and visible Return label, not bare ghost arrows.
The editor datetime uses a reusable Bits UI `DateField` with library-managed
segments, fixed year-month-day / 24-hour / seconds presentation, and localized
accessible segment labels. Formatting is the control's responsibility; users
are not shown a format instruction. Storage remains RFC3339, with the existing
UTC edit convention and separate timezone metadata; unchanged instants retain
subsecond precision. New Journey reuses that field in date-only mode with
`yyyy-MM-DD` presentation, required local-day defaults, and the existing range,
collision, dirty, and pending protections.

The owner exposed a missing desktop API seam: source-action URLs fell through
the generic journey-UUID route and returned 405 or misleading UUID errors. The
sample now composes isolated synthetic route/visit/media sources into the same
importer and intake services used by the server. Sync actually updates the
private route; visit/tray previews are read-only; intake persists stable stop
candidates without creating public mementos. Candidate listing reads that
store rather than returning an empty stub. Route snapping uses the repository's
composed route and validates coordinates. No sample source reads credentials,
contacts external providers, or opens the original workspace.

External providers in normal desktop workspaces and native candidate
promotion/review remain deferred S2 capabilities. The action-matrix audit also
found no desktop photo-upload endpoint: Add photos and its file chooser are now
hidden in the native editor with a localized capability explanation. Existing
photo curation still works; the web host keeps its actual upload workflow.
This is capability gating, not native upload implementation or S2 parity.
A fresh read-only peer `felicia-safe-launch-verify` (`wV:p5W`, actual
`gpt-6-luna`, medium) verified the final candidate: validation (105 admin tests,
zero Svelte diagnostics), 140/140 Chromium/WebKit, 14/14 web authoring, docs and
native packaging/strict signature all passed. Its source walkthrough found no new
defect and confirmed sample isolation before real-workspace resolution. Six
locale/runtime cases verify the unavailable chooser and persistent existing-photo
captions. Only generated admin `index.html` drifted during packaging; authored
source and the retained reader index stayed stable. This clears the source-only
safe-launch gate; native Wails appearance, Dock/controls and S1/S2 owner acceptance
remain open. Source control foundations, Settings/left-form polish and isolated
demo slices are reconciled as verified; native acceptance is not batch-closed.

A later all-widget sweep found two native dialogs omitted from the original AST
control inventory: the scan surface and compiled preview. That means the earlier
safe-launch review did not establish full library-first compliance. Both are now
Bits UI Dialog compositions (Title/Description, portal, overlay and focus scope),
and the AST guard rejects native `dialog` outside shared primitives as well.
Scan preserves first-field focus, Escape/close focus return and the import-pending
close guard; pending inputs and Scan are disabled rather than allowing a second
request to disturb that guard. Preview retains native titlebar inset and drafts.
The migration exposed that iframe Escape does not cross origin boundaries into
Bits UI: Chromium/WebKit reproductions failed before an explicit reader Escape
message bridge. The receiver requires the current iframe window and checked
loopback reader origin; wrong-window/wrong-origin messages are rejected, and the
message carries no data or authoring command. The reader respects already-handled
Escape events. Four browser regressions now pass, including focus return and pending guards.
The full suite then found the shared IconButton tooltip still targeted native
dialogs; its portal now targets the closest native or library dialog, matching
Select. A fresh read-only `felicia-dialog-final-verify` peer (`wV:p5Y`, actual
`gpt-6-luna`, medium observed in its UI footer) independently passed validation
(105 admin tests, zero Svelte diagnostics; 17 public tests), 144/144 desktop
Chromium/WebKit, 14/14 web authoring, docs and native package/strict signature.
Earlier runner stalls did not reproduce in this pass; no cause is asserted and no
check was weakened. All authored source stayed frozen. Only expected generated
admin index bytes changed; direct comparisons confirm both embedded indexes match
the current admin/reader builds, and generated outputs were retained.

This supersedes the prior safe-launch review and clears the current source-only
launch gate for the already-approved isolated synthetic `--sample` invocation.
The canonical Asobi graph is temporarily unreachable (DNS/request failure), so
live task/claim reconciliation remains blocked; no fallback graph or networking
change was made. Verified results are promoted here while that update waits.
Native Wails acceptance remains owner-only, with no real-workspace launch,
permissions, real photo/folder selection, commits or pushes authorized.

After the fresh pass, the lead launched the signed bundle with `--sample` only
(no path overrides): native PID **56042** was observed running the packaged
`FeliciaStudio.app/Contents/MacOS/felicia-desktop --sample`. This is startup/process
evidence, not native UI acceptance. No pre-existing packaged native process was
replaced. The sample window is retained for owner checks; native controls, fonts,
Dock, dragging, authoring and preview appearance still require that feedback.
Live-record reconciliation remains blocked by Asobi DNS, independently of the
source/test pass.

### Owner feedback: standalone Close must keep the studio open

The owner clicked Close in the reviewed native sample and the application quit.
The old standalone composition explicitly called `app.Quit()` because it had no
original workspace. The owner chose returning to an **empty isolated studio**,
not opening a real journal or merely renaming the action Quit. The `--sample`
startup now creates a separate empty temporary baseline before any default
workspace resolution, then opens its disposable sample through WorkspaceRouter.
Close restores the empty handler, closes/deletes sample resources and reloads the
library in the same window; reopening creates fresh sample data. The baseline
survives until application exit, when its own resources are cleaned up.

Isolation is independent of the visible Sample state: both temporary workspaces
reject author-path pickers/imports/output-root changes. The empty studio explains
that the journal is not connected and hides Scan; no native file dialog is opened.
Normal studio-to-sample Return preserves its original semantics. Go tests cover
empty return, fresh reopen, cleanup and forbidden baseline endpoints; six browser
locale/engine cases cover continued live transport and empty UI after Close,
with cancelled dirty navigation preserving the English sample draft. These are
source/headless evidence. Fresh read-only `felicia-close-verify` (`wV:p61`,
actual `gpt-6-luna`, medium observed in its UI footer) independently passed
10 focused Chromium/WebKit cases, standalone/workspace Go tests, admin checks
(105 tests, zero Svelte diagnostics) and current signature verification. Frozen
source, both binaries and full-checkpoint log SHA-256s matched before/after;
the passing 150-case full suite, 14-case web suite, validation, docs and package
results were inspected and reused rather than needlessly rerun. No findings
warranted another full suite. This owner-approved verification-efficiency choice
keeps the final quality bar and assertions unchanged. The initial native UI remains
the seeded sample; only Close shows the empty baseline, in the same process.
The signed, reviewed isolated sample was relaunched and the owner confirmed that
Close now leaves the application open in the empty temporary studio. This accepts
the bounded native Close fix, not the rest of S1/S2. The owner asked about the
“your journal is not connected” notice; it explains the isolated launch, rather
than a connection failure. The clearer proposed wording (“Temporary workspace —
changes are cleared when you quit”) has not yet been applied to the catalogs.
Asobi DNS continues to block live-record reconciliation. Unsupported actions are not
shown as enabled controls; the UI explains the boundary. These changes and
focused browser checks do not establish complete desktop/native UX parity.

The owner made library-first controls a hard rule after native selects/color
controls remained undersized and inconsistent despite earlier component work.
`AGENTS.md` records the rule: reuse shared UI components and Bits UI primitives;
any native/custom fallback needs a demonstrated failure and recorded evidence.
The independent control inventory found remaining view-level native inputs,
selects, textareas, and buttons, including Site, Settings, Editor, connectors,
and the scan dialog. The subsequent source migration replaces those widgets with shared Input,
Textarea and Button components and a reusable Bits UI Select. Native elements
inside shared primitives and ordinary form/label semantics remain intentional;
there is no demonstrated-failure fallback for a view-level widget. More options
uses Bits UI Collapsible, and photo upload has a shared visible button and hidden
shared file input. A Svelte-AST unit gate rejects raw view-level input, textarea,
select, button, details and summary elements. The overall UX is not yet accepted.

The native preview embeds the existing Go-served, last-built reader. It does not
start a JavaScript server, open an external browser or save unsaved input.
Returning from preview keeps the authoring page mounted. On macOS, the preview
starts below the shared native title-bar height; DOM top-layer dialogs cannot
cover native traffic lights safely. The sample-return control wraps within its
sidebar card instead of relying on a fixed single-line label width. Published appearance
starts from the built site's settings, independently of studio appearance. The
reader exposes labelled Sun/Moon controls for a local viewing override; these
controls do not save or republish site settings. The desktop journey-detail
shortcut uses this same in-app last-built preview, not the web-only external
browser link. Library header actions and the preview header button have a
36-pixel minimum height.

Both the packaged `.icns` metadata and Wails' runtime `Icon` option use the
existing default Felicia artwork. The runtime option is needed for executable
launches that do not obtain their Dock icon from bundle metadata.

The owner confirmed dragging and normal-looking fonts in one isolated native
sample. The CoreText warning was not reproduced; its cause is not established.
The owner then rejected the color treatment and requested further UI polish.
Do not treat the drag/font check as visual acceptance or revised S1 completion.

Rendered inspection found hard-coded warm authoring styles mixing with the
neutral studio palette, including unreadable dark-mode labels and badges.
Journey and Editor now reuse the shared semantic surfaces/status colors rather
than maintaining their own palette. Save is a sticky, visible action with a
dirty-state cue; preview has an explicit return action. Reader accent text uses
a separate readable ink role rather than borrowing the selected-fill color,
and its index collapses earlier to avoid covering content in compact previews.
The public paper-memento material remains intentional.

Targeted browser checks and rendered contrast measurements cover this slice;
final owning gates, fresh independent verification and renewed native visual
acceptance are still required. No commit, publication or release is approved.

### Bounded verification: authoring a newly created journey

The owner found that creation led to an empty detail page with no metadata edit
or first-memento action. Journey detail now exposes **Edit journey** and **Add
memento**. Editing reuses the full-page shared journey form and carries the
existing id, journal, source reference and revision; changing the title does not
silently rename its slug. The desktop handler retains the stored GPS route and
authored-field mask. A stale save is rejected without discarding input.

Add memento opens a full-page shared Input/Label/Button form with a Bits UI Select
for the registered kind. It creates a stable-id private draft, then opens the
existing editor for essay and other authoring. Failed creation retains input and
reuses the same id on retry. Return is guarded navigation; the journey footer's
Cancel is an explicit discard. Default shared inputs, buttons, date fields and
the new Select use physical 40-pixel sizing rather than short rem-based heights.
The existing SQLite create contract omits the existing-row revision constraint;
metadata updates continue to carry `expected_revision`.

At `feat/desktop-sample-polish` / `848aca80cb54e86ba38ceb871358806fd8d714db` plus
this uncommitted slice, fresh read-only Herdr peer `felicia-authoring-verify`
(`wV:p5Q`, Sonnet 5.5, startup `--model sonnet --effort medium`) independently ran
`e2e-desktop/authoring-entry.spec.ts` (6/6 Chromium/WebKit) and `make admin-check`
(104/104 unit tests, zero Svelte errors/warnings, formatting clean), all exit 0.
The runtime footer confirmed the model, not the effort setting. Browser evidence
covers create/edit/save identity, stale-writer rejection and cancellation,
first-memento essay save/reload, keyboard Select, input/button/Select sizing,
dirty Return, held-POST navigation and a 503 retry producing exactly one draft.
The initial review was corrected to cite the desktop handler, not the web server,
and to claim only new controls, not every control in changed views.

The new-memento next-sequence value remains a load-time snapshot; concurrent
adds are not verified. GPS/source retention, date-range/collision behavior in
metadata editing and date-field physical height have source evidence or existing
creation coverage, not new dedicated edit-browser acceptance. The all-control
migration remains open, and this bounded verification does not complete native
acceptance, S1/S2 or the overall UX. The retained native process has not been
replaced; no commit or push is approved.

### All-control migration gate checkpoint

The source migration standardizes form controls at physical 40-pixel sizing,
keeps entered text in its original case, and supplies foreground/surface colors
in shared primitives instead of inheriting muted label colors. Select exposes
its selected value as an accessible description and uses a dialog-aware portal.
Single-choice design controls report `aria-pressed`. WebKit skipped implicit
button tab stops; explicit default `tabindex=0` in shared Button restores the
verified Settings-to-Return keyboard path without changing OS preferences.

On the uncommitted polish candidate, lead gates passed: all 132 real-composition
Chromium/WebKit cases (including a 12-case en/ja/zh, light/dark and 1100/720 Site
control/save matrix), 105 admin unit tests with zero Svelte errors/warnings,
`make validate`, `make docs-build`, `make desktop-package` and strict bundle
signature verification. The unchanged web-host authoring loop passed 14/14,
including photo upload/curation, candidate promotion, publication and saved site
identity in real compiled output. The initial full-browser run failed on stale
Scan/Create assertions, inherited input contrast and WebKit keyboard focus;
those were addressed without lowering contrast thresholds or deleting checks.
The first web run overlapped packaging and lost transient saved feedback;
its serialized rerun passed. Do not run dev-server browser verification while
packaging regenerates the same SvelteKit outputs.

Fresh expanded-candidate independent verification and owner-native acceptance
remain required. No native process was replaced and no commit/push is approved.

### Review findings: fonts and bounded JSON

Fresh Luna-medium source review found that the embedded reader still fetched its
existing typefaces from Google Fonts. The owner chose bundling those same fonts,
not replacing the paper-memento typography with system fonts. The public reader
now links a base-aware local stylesheet. Its five families, styles, weights and
425 Unicode-subset font-face rules are retained, with 396 original WOFF2 assets
(approximately 8.7 MB), content-hash provenance and all five SIL OFL licenses in
`apps/felicia-public-site/public/fonts/`. No runtime font download or package
dependency is added. Existing map tiles remain an online ArcGIS dependency;
self-hosted typography is not a claim that the complete map is network-free.
Browser verification blocks external traffic and explicitly loads the original
Latin and Japanese faces while published stories remain readable.

Journey/memento upserts and photo metadata now decode one JSON object through an
8 MiB limit, including unknown-length request bodies; oversized JSON returns
413 before repository access and trailing objects return 400. This bounds the
local availability issue without changing binary-upload handling or persistence.
Fresh Luna-medium peer `felicia-final-verify` (`wV:p5R`, actual
`openai-codex/gpt-6-luna`, medium) independently rechecked validation, 134/134
Chromium/WebKit, 14/14 web authoring, docs and desktop Go gates, all exit 0. Its
source review verified local font coverage/licenses/hashes, unchanged reader
style declarations and bounded JSON; subpath behavior has structural evidence,
not a separate subpath-font browser run. It found missing embedded-reader WOFF2
MIME mapping. The handler now returns `font/woff2`, with a direct reader-mode
response test independently rerun in the desktop Go gate (exit 0). Native
WKWebView rendering remains unverified. Generated indexes are retained; expected
SvelteKit regeneration changes their hashes rather than authored-source behavior.
The approved isolated sample launch follows final safe-affordance verification.

### PR 160 owner feedback: dirty-page Return

The owner reported that Return from Add memento appeared frozen. The source
used synchronous browser `confirm()` for dirty navigation; the pinned Wails
macOS delegate has no JavaScript-confirm handler. A browser repro with
`window.confirm` returning false reproduced the blocked Return. This is not
proof of the full native failure's origin.

Internal dirty navigation now uses a shared Bits UI AlertDialog with localized
Keep editing / Discard changes actions. Journey creation/editing, Add memento,
the memento editor and Site share the guard. Pending writes and unload
cancellation remain protected; browser unload retains its browser-owned prompt.
Workspace switches await the actual discard decision before touching the
backend. Focused tests caught an additional SvelteKit boundary: a cancelled
`goto()` can resolve before that decision. The continuation now handles that
case, including standalone Close from a dirty editor.

Lead verification of this uncommitted slice:

- `make admin-check`: 110 tests pass, zero Svelte diagnostics, lint/format pass.
- Headless desktop rebuild plus 38 affected Chromium/WebKit cases pass:
  Return, draft retention/discard, pending creation, keyboard saves, reload
  cancellation, routing and same-window workspace/standalone Close behavior.
- `make test-admin-e2e`: 14 real web authoring cases pass.
- Source tests cover resolved cancellation, refusing discard, a write becoming
  pending while confirming, and pending/unload refusal. The Svelte AST control
  guard now also rejects view-level native `alert()`, `confirm()` and `prompt()`.

Evidence: workstation scratch `.tmp/felicia-desktop-polish/return-*.log`.
Fresh independent verification and actual native retesting remain open. The
native application bundle was not replaced or relaunched. Font packaging,
native image upload and sample output-location remediation are separate open
PR 160 feedback items; their four browser regression cases were not included
in this Return-only gate. No native/S1/S2 acceptance or issue is closed here.

### PR 160 owner feedback: font dependencies and photo creation

The owner rejected tracked font binaries. Reader typography now comes from five
pinned `@fontsource` 5.3.0 dependencies (Inter, Outfit, Share Tech Mono, Spectral,
Zen Old Mincho). The same 17 family/weight/style combinations and the packages'
Unicode subsets remain locally served, bundled by Vite and embedded in desktop
assets. Public/desktop copies of font binaries and the hand-maintained face/hash
manifests are removed from the working tree; licenses and provenance remain.
The owner-approved npm mirror was scoped temporarily to these packages. The
existing dependency versions were preserved and an offline frozen install passed.
Earlier branch commits still contain font binaries: no history rewrite or
force-push has been performed.

The old desktop upload notice was a capability gate for a missing backend, not a
macOS permission problem. Ordinary desktop image creation now uses the same
runtime photo service as web multipart uploads. JPEG, PNG and WebP are accepted;
HEIC/HEIF requires conversion. The service checks actual image bytes, a 20 MiB
limit, dimensions up to 50,000 per side and 50 million pixels. Generated keys
contain the content digest and a new photo UUID, never client filenames or source
paths. Existing photo identities remain unchanged. Sequence allocation is
serialized within each service instance, not across processes.

Private filesystem reads, writes and deletion use `os.OpenRoot`; writes use
owner-only temporary objects and atomic rename. Rejected metadata writes remove
only the new upload's original after a fresh lookup confirms that its identity is
absent. An uncertain committed write retains its original; an unavailable lookup
also retains bytes rather than risk deleting committed data. Such an uncertain
failure can leave a private orphan and must not be automatically retried.

Sample and empty temporary workspaces expose Add sample photo, generating PNG
bytes internally. They never expose a file input or invoke a native file picker;
ordinary file uploads are forbidden at the backend. Pending upload controls and
navigation remain guarded. Temporary output is automatically managed: the folder
picker is hidden, and backend browsing/output mutations return 403 instead of a
misleading 404 or touching author roots.

Focused lead checks cover format/size/dimension rejection, symlink confinement,
compensation and uncertain outcomes, photo sequencing, generated-only isolation,
three-locale generated-photo curation and ordinary desktop upload/curation in
Chromium and WebKit. These are source/headless checks, not Wails file-picker
acceptance. The final lead checkpoint passed `make validate` (including 111 admin
and 17 public-reader tests), focused runtime/provider/server race tests, desktop
Go tests, Markdown checks and the documentation build. Web authoring passed all
14 cases in a sequential run. Desktop coverage comprises 158 distinct cases:
120 passed before the harness's 120-second deadline interrupted the full run;
a bounded WebKit remainder passed 39 cases with one overlap. No test failure was
reported in that interrupted desktop run. This is not a single completed
`make e2e` receipt. A first concurrent web attempt lost a caption draft; its cause
is unproven. The sequential rerun passed without source or assertion changes.
Build-producing gates should not share the checkout concurrently. After the
review correction below, a serialized final `make e2e` completed successfully:
**158/158, exit 0** (`rework-desktop-complete.log` and `.exit`). That replaces the
interrupted-run receipt as the final desktop gate, without diagnosing the earlier
web failure or weakening assertions. Final `make validate`, Markdown checks and
docs build also passed (`rework-postreview-validate.log`).

Evidence is in workstation scratch `.tmp/felicia-desktop-polish/`:
`rework-validate-final.log`, `photo-go-final.log`, `rework-desktop-final.log`,
`rework-desktop-remainder.log`, `rework-web-sequential.log` and
`rework-docs-final.log`. Fresh read-only pi-subagents review completed on the
owner-selected `zai-coding-cn/glm-5.3-flash` at observed medium effort, run
`d22c2b70-05f8-4fbd-a722-2b03763c2ad6`. It met all four source criteria with no
proven blocking findings, inspected exact source/test copies and matched their
SHA-256s to frozen working state
`f4c7a8e89f3fa6adee8c64f125f8efb50b57ef0a8cc1fe08b34d02cdf9a3dbc6`, and reused
owner-permitted matching gate evidence. Report: workstation scratch
`rework-independent-review.md`. This is source verification, not native acceptance.

A review coverage note exposed a real provider gap: SQLite `GetPhoto` returned
raw `sql.ErrNoRows`, so compensation's domain-not-found check could not remove a
rejected upload's new original. A regression first failed on that exact error;
the provider now maps only missing rows to `domain.ErrNotFound`. Focused
SQLite/runtime/provider/server race and desktop tests pass, with the actual
`-race` command retained in `photo-mapping-green.log`. A final fresh, narrowly
scoped pi-subagents GLM Flash/medium review met the mapping, regression and safe
compensation criteria with no findings (`rework-mapping-review-result.md`). It
inspected the actual provider/test/service files and reused the focused gate;
the initial broader review alone does not verify the changed provider.

Canonical Asobi connectivity recovered on the verified remote graph. The owning
S1 task's stale branch, commit and next action were reconciled while retaining
`BLOCKED_ON owner-native-acceptance`. Tracker-only duplicate implementation todos
were superseded, not counted as native completion. The native application bundle
has not been replaced or relaunched. No broader native/S1/S2 acceptance, merge,
release or deployment is claimed.

## Evidence and closeout

For each slice, record branch/full commit and working-diff ownership, exact commands
and results, fixture/evidence paths, independent review, and blockers in Asobi.
Promote durable findings here and to the owning PR only after delivery approval.
Do not weaken assertions, skip tests or silently rebaseline to force green. Initial
S0 evidence demonstrates the previous defects; it is not design approval. The
initial Create-button probe omitted required date state and did not prove a button
bug.
