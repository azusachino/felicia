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

## Evidence and closeout

For each slice, record branch/full commit and working-diff ownership, exact commands
and results, fixture/evidence paths, independent review, and blockers in Asobi.
Promote durable findings here and to the owning PR only after delivery approval.
Do not weaken assertions, skip tests or silently rebaseline to force green. Initial
S0 evidence demonstrates the previous defects; it is not design approval. The
initial Create-button probe omitted required date state and did not prove a button
bug.
