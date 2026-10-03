# Felicia Desktop Studio Redesign: Ordered Execution Plan

> This is the execution contract. Research and design rationale live in
> `docs/research/desktop-studio-redesign.md`; live task state and claims live in
> Asobi. Do not dispatch a later slice while an earlier acceptance gate is open.

## Goal and constraints

Deliver Felicia as a polished local desktop studio: import → author → preview →
export, while retaining CLI and local read-only serving. Develop on Wails v3;
defer public release until the workflow matures.

- macOS first, with Windows/Linux planned. Reuse existing Svelte, core, runtime,
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
system-aware neutral surfaces with restrained amber, and an accessible creation
sheet. The focused authoring workspace should adapt to available width. See
[`docs/research/desktop-studio-redesign.md`](../research/desktop-studio-redesign.md).

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
4. Keep New journey creation in its own accessible sheet with focus return, Escape
   and cancel behavior, scrollable body and stable action footer. Use restrained
   open/close motion only when reduced motion is not requested. Require title, place
   and dates, derive an editable slug, disclose optional metadata, surface slug
   collisions and server failures without discarding input. Source import is a
   separate Sources → Review → Apply task owned by S2, not another creation mode.
5. Keep app-level window intent independent of Wails OS settings. A small
   platform adapter maps that intent to macOS options; other platforms retain
   Wails native defaults unless and until their own treatment is implemented.
6. Assert screenshots/composition at 1100×760, 900×600, 1440×900 and responsive
   720×600; test ja/en/zh, long titles and 200% zoom. Inspect actual native chrome
   and material independently.

Acceptance: create persists and navigates; cancel adds no row; collisions preserve
existing records; invalid dates are rejected; actions and focus remain reachable;
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
   supported and readable reduced-transparency/motion, dark/light and older-OS
   fallbacks. Do not require private APIs. OS/CPU support remains a packaging
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

### S1 in progress — native appearance blocked

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
