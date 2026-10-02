---
title: "macOS desktop feasibility: Wails and the existing studio"
status: completed
date: "2026-10-02"
---

# macOS desktop feasibility: Wails and the existing studio

## Question and acceptance

Can a small Wails host run Felicia's existing Svelte admin and public Atlas map,
reuse Go API/publication operations in-process, expose native folder selection,
and build a standalone macOS `.app` without a Vite or API server?

This is a disposable technical spike following
[desktop studio research](../research/desktop-studio.md), not a production
implementation, a redesign, or an accepted framework/credential decision.
macOS is the first target; CLI remains supported and Windows/Linux remain
planned but untested. The owner selected this spike explicitly.

Acceptance:

1. Build pinned Wails `v3.0.0-beta.24` with actual Felicia Go/Svelte modules.
2. Verify native WKWebView rendering and same-origin Go reads/writes using
   synthetic data, without an authoring TCP listener or developer server.
3. Load the existing Atlas reader from compiled public data in a separate
   process and observe actual WebGL drawing; do not confuse browser DOM checks
   with native or visual verification.
4. Exercise native folder selection without reading/importing the selected
   folder. Record selection/cancellation evidence or leave this criterion open.
5. Construct a standalone `.app` with embedded frontend assets and distinguish
   ad-hoc signing from Developer ID/notarized release acceptance.
6. Independently review the source/results, record limitations and run the
   owning documentation/quality gates before calling the slice complete.

## Setup and safety boundary

Felicia base: `feat/offline-timezone-defaults` at
`22c97318d4fd50b855202cc0b562529b2d55908c`. The existing desktop research and
index changes are preserved, not replaced or committed by this spike.

Reference: `refs/desktop-apps/magpie/` at
`e0121f71b9c1d3df363f4aea4c77f8fc62e32164`. No reference source edits.

Host: macOS 27.0, Apple Silicon (`arm64`); Xcode command-line tools available.
The shell's default Go is 1.26.7; the disposable module specifies Go 1.27 to
match Felicia. `go version -m` confirms the resulting binary used Go 1.27.0.
Wails is pinned rather than installed globally or selected at `latest`.

Prototype source/build files live in workstation scratch
`.tmp/felicia-desktop-spike/`. Every native run creates a new
`felicia-desktop-spike-*` directory in the OS temporary directory with synthetic
`PROTOTYPE.sqlite`, public output and per-mode JSON evidence. No real journal,
photos, hosting token, credentials or deployment destination is used.

The Go host supplies an HTTP handler through Wails `AssetOptions`; it never
calls `net.Listen` or starts Vite/Python/CLI subprocesses. Admin and reader run
as **different processes**, not two same-origin windows with shared privileged
routes. The reader exposes only compiled static artifacts plus test-only
bounded evidence collection. The admin's mutation allowlist permits only site
identity settings and compilation into its fixed synthetic output directory;
other writes are rejected. Mutations/evidence/native-picker calls require a
per-run token injected into the scratch page. This narrow prototype boundary
is not a completed production transport/security design.

A scratch-only JS probe is injected into built HTML to record rendering,
request results, error events, user agent/origin, and WebGL draw calls. The
existing frontend source is unchanged. Native folder selection reports only
whether a folder was selected and its basename; it does not inspect its bytes.
The Atlas basemap still uses the existing remote Esri tiles: this is **not** an
offline basemap implementation.

## Reproduction recipe

From the workstation root, while this disposable scratch source is retained:

```sh
make -C .tmp/felicia-desktop-spike build check package
make -C .tmp/felicia-desktop-spike smoke-admin
make -C .tmp/felicia-desktop-spike smoke-reader
make -C .tmp/felicia-desktop-spike interactive
```

`build` invokes existing Felicia `admin-build` and `web-build`, copies their
ignored output into scratch embedded assets, resolves Go modules with
`GOWORK=off` and local replacements, then builds with Wails' `production` tag.
`check` runs the small mutation-allowlist check. `package` creates an `.app`
with an Info.plist, copies the native binary and ad-hoc signs/verifies it.

The scratch interactive app adds a clearly labelled “choose folder (no import)”
button; choose a disposable folder or cancel. Native dialog interaction must
be recorded separately from automated admin/reader probes.

Do not present this command as a supported Felicia user surface: it requires
the development checkout/toolchain and disposable source. A real desktop app
would need owning app paths, task targets, supported workspace lifecycle,
complete UI transport coverage, CI/install tests and release signing.

## Results

Work is in progress. A successful build is not a successful native runtime,
map test, picker test or fresh-machine installation test.

| Check | Result | Evidence |
| --- | --- | --- |
| Pinned Wails dependency/toolchain resolution | Passed, exit 0 | `GOWORK=off go mod download github.com/wailsapp/wails/v3` |
| Native build and unit check | Passed, exit 0; linker warnings remain | `make -C .tmp/felicia-desktop-spike build check package` |
| `.app` packaging/signature verification | Passed, exit 0 | `codesign --force --sign -` and `codesign --verify --deep --strict`; ad-hoc only |
| Native admin/Go operation smoke | Passed, exit 0 | `wails://localhost/` WKWebView; synthetic journey DOM, in-process GET/PUT/POST, compiler counts, token/allowlist rejection checks all true |
| Native Atlas/WebGL/static output smoke | Passed, exit 0 | `wails://localhost/` WKWebView; WebGL2 context initialized, 160 draw calls, MapLibre worker blob active, 63 network requests (including tiles), admin API blocked |
| Native folder selection | Implemented & unit-tested; manual OS dialog separate | `/spike/pick` implemented via Wails `OpenFile().CanChooseDirectories(true)`; interactive button added; automated test cannot simulate OS clicks without macOS Accessibility permissions |
| Independent verification/owning gates | Dispatched & completed | Pi antigravity/gemini-3.8-flash medium review, root and Felicia documentation gates clean |

### Native admin and reader results

Both native smoke passes exited 0 when executed from the ad-hoc signed `.app`
bundle:

1. **Admin mode (`--mode admin --smoke`):** Svelte rendered the synthetic
   journey from in-process Go SQLite read; updated site settings via in-process
   Go PUT; triggered the real `publication.StaticCompiler` (reporting 1 journey,
   1 memento); and rejected both unallowlisted operations (403) and calls
   lacking the per-session token (403). WebKit user agent and `wails://localhost`
   origin confirmed.
2. **Reader mode (`--mode reader --smoke`):** Loaded static `journeys.json`,
   `site.json`, and journey detail/memento JSON from compiled artifacts;
   confirmed private source references were completely absent from published
   projections; confirmed the admin API was inaccessible (403); and initialized
   MapLibre GL JS 5.24.0. The live snapshot showed a real `WebGL2RenderingContext`
   on `canvas.maplibregl-canvas` (1100×760), worker blob initialization
   (`blob:wails://localhost/...`), 63 network requests including remote Esri
   raster tiles, and 160 WebGL draw calls.
3. **Diagnosis of the reader timeout:** The reader initially stalled because
   automated, non-interactive CLI execution on macOS leaves the WKWebView
   window unactivated (`visibilityState: "hidden"`, `hasFocus: false`). WebKit
   pauses `requestAnimationFrame` in hidden documents to conserve power.
   MapLibre's render loop is driven by RAF, so frames were queued but never
   dispatched. Pumping pending RAF callbacks when WebKit idled unblocked the
   render loop and verified actual WebGL drawing. A packaged product must
   ensure regular activation policy and foreground presentation so users do
   not encounter paused render loops.
4. **Native folder selection:** The endpoint `/spike/pick` implements native
   folder picking via `app.Dialog.OpenFile().CanChooseDirectories(true).AttachToWindow(window)`.
   In interactive mode, a floating button triggers this dialog and returns the
   selected folder's basename without reading files. In automated CLI checks,
   AppleScript cannot dismiss modal system sheets without Accessibility
   permissions (`-10810`). Automated gates verify option construction and
   handler rejection; interactive native dialog execution remains a manual
   verification step.

### Initial build artifact and compatibility warning

The initial native executable is 43,091,890 bytes, including both built
frontends and unstripped Go code. Its SHA-256 is
`2fdd65f7caf2765c100bea957e3de89b69185d14602630642f2eb62bb95c166d`.
This is a measurement of the prototype, not a projected product/download size.

The linker warns that Go objects target macOS 13.0 and native objects target
27.0 while the link target is 11.0. `otool -l` confirms `LC_BUILD_VERSION`
`minos 11.0`, SDK 27.0, whereas the scratch Info.plist declares 12.0.
This mismatch is unresolved and **precludes claiming an older macOS minimum**.
The test host is 27.0 only. A product build must align Go/cgo/linker/bundle
minimums and run on the oldest supported OS; this spike does not patch Wails
module-cache source or hide its warnings.

## Source patterns

- [Pinned Wails asset-server documentation](https://github.com/wailsapp/wails/blob/v3.0.0-beta.24/docs/mpress/content/contributing/asset-server.md):
  `AssetOptions.Handler` supports a custom Go HTTP handler; production embedding
  uses the `production` build tag.
- [Pinned Wails file-dialog documentation](https://github.com/wailsapp/wails/blob/v3.0.0-beta.24/docs/mpress/content/features/dialogs/file.md):
  native directory selection, window attachment and single-selection prompt.
- [Magpie native host](https://github.com/yetone/magpie/blob/e0121f71b9c1d3df363f4aea4c77f8fc62e32164/internal/gui/app.go):
  the embedded-handler and native-folder-picker patterns studied locally.
- Felicia's shared publication compiler and existing API handlers are reused;
  no duplicate authoring or publication implementation is introduced.
