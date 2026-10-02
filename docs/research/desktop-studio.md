---
title: "Desktop studio: Magpie study and delivery options"
status: research
date: "2026-10-01"
---

# Desktop studio: Magpie study and delivery options

## Scope and recommendation

This is research, not an accepted architecture change or implementation plan.
The audience is **other people installing Felicia**. The requested experience
is install → import → author → preview → export or publish, without deploying
an admin server or learning the CLI. macOS comes first; Windows and Linux stay
in the plan, and the CLI remains a supported interface.

**Recommended direction:** a Go desktop application using Wails, reusing the
Svelte admin, runtime/intake services, SQLite provider, and publication compiler.
Ship a complete offline authoring/export experience before adding account-based
publishing. Treat Wails v3 as the leading spike candidate, not an approved release
dependency: Magpie demonstrates its use, but the pinned Wails documentation still
calls v3 beta and v2 the stable release [W1].

Do not fork Magpie into a travel app, rewrite Felicia in Rust, or merely put the
current development launcher inside a window. Magpie is most valuable as an
implementation reference for native integration, lifecycle and distribution.

## What exists in Felicia, and what does not

Reviewed Felicia revision: `22c97318d4fd50b855202cc0b562529b2d55908c`.

| Area | Reusable foundation | Desktop gap |
| --- | --- | --- |
| Application logic | `apps/felicia-runtime/`, core domain, provider interfaces | A desktop composition root and native-facing operations |
| Authoring | `apps/felicia-admin/`: Svelte UI, typed API client, journey/memento editing | Desktop onboarding, native files, durable background jobs, error recovery |
| Intake | GPX/photos and Google Timeline local sources; shared plan/apply service | GUI source selection and semantic-visit import; current local-workspace endpoint requires `route.gpx` and `photos/` |
| Local persistence | SQLite and filesystem media | Stable per-user workspace, backup/restore, migrations, simultaneous CLI/desktop access policy |
| Publication | Publication projections/compiler; build/preview actions | A packaged reader bundle, complete portable export, destination validation |
| Deployment | Static hosting guide and planned GUI GitHub deployment | Guided account/target setup and reliable remote confirmation |

Sources: [layout](../development/layout.md), [publishing guide](../publish.md),
[GUI epic](../roadmap/admin-gui-v2-epic.md),
[`localjourney.go`](../../apps/felicia-server/api/localjourney.go),
[`compiler.go`](../../apps/felicia-publication/compiler.go), and
[`api.ts`](../../apps/felicia-admin/src/api.ts).

Important distinctions:

- The new CLI Timeline import does **not** mean the admin imports Timeline today.
  Reuse the visit source; do not fabricate route geometry for semantic visits.
- `scripts/admin.py` is development orchestration, not a distributable desktop
  runtime. It starts the API and Vite and defaults to `0.0.0.0` for Tailscale.
- The API has no authentication. A packaged desktop must not inherit the
  development networking default.
- The current server opens a second listener for public preview. That preview
  must remain a read-only, unprivileged surface in a desktop design.
- The GUI epic explicitly forbids credential storage and plans existing-user
  Git credentials. A friendly OAuth-based publisher is a **new decision**, not
  an implementation detail already authorized by that epic.

## Magpie: study boundary

Workstation reference: `[ref.magpie]` in `platform.toml`, cloned read-only at
`refs/desktop-apps/magpie/` using `make refs NAMES=magpie`; `make index`
regenerates the catalog. The initial tool-fetched temporary clone was not the
correct persistent reference home and is not the basis for future checkout paths.

Upstream examined: `yetone/magpie`, revision
`0eb16d4228498c32dbe4040f419d145038ada46e` [M1]. Its `go.mod` pins Go
`1.26.3`, Wails `v3.0.0-beta.24`, and `modernc.org/sqlite v1.44.3` [M2].
Its README describes desktop, TUI and CLI interfaces; an OS webview; a menu-bar
panel plus normal window; atomic configuration writes; and a plain HTML UI
rather than a frontend framework [M1]. These are upstream claims and source
observations, **not measured Felicia package-size or performance results**.

Concrete implementation lessons from the examined revision:

| Magpie source | Observed pattern | Felicia lesson |
| --- | --- | --- |
| [`internal/gui/app.go`](https://github.com/yetone/magpie/blob/0eb16d4228498c32dbe4040f419d145038ada46e/internal/gui/app.go) | Supplies an HTTP handler through Wails `AssetOptions`; single-instance dispatch, platform activation policy, saved window dimensions/zoom, shutdown hook | Existing Go handlers may be reusable without starting the development API server; define workspace and long-job lifecycle explicitly |
| [`internal/gui/api.go`](https://github.com/yetone/magpie/blob/0eb16d4228498c32dbe4040f419d145038ada46e/internal/gui/api.go) | Embeds UI assets; one handler serves static files and JSON operations; boot preferences precede first paint | Keep frontend transport adaptable; load theme/language before rendering; avoid rewriting Svelte just to match upstream |
| [`internal/gui/web.go`](https://github.com/yetone/magpie/blob/0eb16d4228498c32dbe4040f419d145038ada46e/internal/gui/web.go) | Separate browser host implementation disables native operations; per-run key/cookie guard protects network-served settings | Distinguish embedded UI from a network server; browser fallback must not silently gain native powers |
| [Release workflow](https://github.com/yetone/magpie/blob/0eb16d4228498c32dbe4040f419d145038ada46e/.github/workflows/release.yml) | Dispatches build/sign/notarize/publish to `yetone/magpie-releases`; signing secrets are stated to live there | A tagged build is not the complete release pipeline; this repository alone does not verify downstream signing or notarization |

The embedded-handler pattern is especially relevant: it may preserve Felicia's
current Svelte `fetch` contracts while avoiding wholesale generated-binding
conversion. Confirm webview origin, asset URLs and privilege isolation in a
spike; a source pattern is not proof that every current endpoint is safe to expose.

### Shared logic, persistence, and release lessons

The source investigation also examined the entrypoint, backend, directory
resolver, backup, updater and build/test definitions:

- **Desktop and CLI share Go code, not subprocess command parsing.**
  [`main.go`](https://github.com/yetone/magpie/blob/0eb16d4228498c32dbe4040f419d145038ada46e/main.go)
  initializes common hooks before interface selection;
  [`gui_on.go`](https://github.com/yetone/magpie/blob/0eb16d4228498c32dbe4040f419d145038ada46e/gui_on.go)
  and [`gui_off.go`](https://github.com/yetone/magpie/blob/0eb16d4228498c32dbe4040f419d145038ada46e/gui_off.go)
  select desktop versus `nogui` builds. The
  [GUI backend](https://github.com/yetone/magpie/blob/0eb16d4228498c32dbe4040f419d145038ada46e/internal/gui/backend.go)
  calls shared gateway/catalog packages and can observe another running gateway.
  Felicia should share operations likewise, but does not need a gateway or TUI.
- **Separate installed state, cache and portable state.**
  [`appdir.go`](https://github.com/yetone/magpie/blob/0eb16d4228498c32dbe4040f419d145038ada46e/internal/appdir/appdir.go)
  centralizes XDG-aware config/cache paths and detects adjacent `data/` or
  `.portable`. On macOS portable data stays beside, not inside, the signed
  `.app`; directory resolution remains stable during executable replacement.
  Portable mode does not make keychains and OS integration portable. Adopt the
  distinction, not necessarily Magpie's exact default paths.
- **Backup is a separate private operation.**
  [`backup.go`](https://github.com/yetone/magpie/blob/0eb16d4228498c32dbe4040f419d145038ada46e/internal/backup/backup.go)
  implements AES-256-GCM with random salt/nonce and PBKDF2-SHA256, validates its
  envelope and can omit credentials. Its secret filtering is heuristic; restore
  reports partial results rather than providing a proven global transaction.
  [GUI backup handling](https://github.com/yetone/magpie/blob/0eb16d4228498c32dbe4040f419d145038ada46e/internal/gui/backup.go)
  uses atomic output writes and mode `0600`. Neither this nor a live SQLite
  file copy proves a consistent Felicia backup or equivalent Windows ACLs.
- **Native lifecycle includes platform-specific failures.** The app source
  delays Windows window display until webview navigation completes, restores
  size/zoom, and routes subsequent Windows/Linux launches using config-directory
  identity. macOS uses a different activation/link path. Felicia needs tests
  for second-launch imports and close-versus-quit behavior, not just a window
  that opens once.
- **An updater is substantial, security-sensitive product code.**
  [`update.go`](https://github.com/yetone/magpie/blob/0eb16d4228498c32dbe4040f419d145038ada46e/internal/update/update.go)
  stages a macOS ZIP, checks its code signature and signing team, renames the
  old bundle aside, attempts rollback, and handles relaunch/admin installation.
  Its comments explicitly distinguish Windows/Linux/terminal updates without
  that macOS signature check. The examined team comparison does not explicitly
  reject empty identifiers; this investigation did not audit the complete
  download/replacement chain. Hashes alone are not a signed trust policy.
- **Build coverage is not install/runtime coverage.** Magpie's
  [`Makefile`](https://github.com/yetone/magpie/blob/0eb16d4228498c32dbe4040f419d145038ada46e/Makefile)
  uses a macOS 12.0 minimum, native Linux GTK3/WebKitGTK 4.1 builds, and separate
  cgo-free terminal builds. Its
  [test workflow](https://github.com/yetone/magpie/blob/0eb16d4228498c32dbe4040f419d145038ada46e/.github/workflows/test.yml)
  runs tests on macOS/Linux but only compiles tests on Windows, citing HOME and
  Unix permission assumptions. This is a known coverage gap, not a pattern to
  reproduce. Upstream Linux build choices also differ from the pinned Wails
  default; record Felicia's own exact tags/runtime support.

The registered shelf synced a newer revision,
`e0121f71b9c1d3df363f4aea4c77f8fc62e32164`. A byte comparison of the 17
source/build/documentation files cited in this Magpie section found only
`internal/gui/app.go` and `internal/appdir/appdir.go` changed. Their reviewed
[diff](https://github.com/yetone/magpie/compare/0eb16d4228498c32dbe4040f419d145038ada46e...e0121f71b9c1d3df363f4aea4c77f8fc62e32164)
adds an explicit portable Windows WebView2 profile path, `data/webview2`, while
preserving the installed default. This is another useful lesson: browser
cookies/cache/storage must follow workspace portability policy too; moving only
SQLite and media is insufficient. The earlier pinned citations remain valid
for the original source pass; the entire newer repository was not re-audited.

Not covered by this source pass: frontend state/rendering implementation,
visual/accessibility quality, complete secret-store internals, all updater
paths, downstream release execution, or a full authorization/security audit.
Follow-up exploration should target those areas rather than claiming the
whole application has been learned or verified.

Magpie is MIT licensed [M3]. Borrowed code requires preserving its copyright
and permission notice; icons, bundled assets and dependencies still need their
own license review. Prefer studying patterns over copying large subsystems.

This investigation is source-based. Magpie has not been built, launched, visually
reviewed, or security-audited here. Felicia has not been prototyped in a native
webview. Native appearance, accessibility, map rendering and package footprint
therefore remain empirical spike questions.

## Desktop framework comparison

| Option | Fit for Felicia | Costs / uncertainty | Position |
| --- | --- | --- | --- |
| Wails v3 | Go services in-process; existing Svelte assets; native OS webview; Magpie reference | Beta dependency, platform webview differences, native packaging gates | Leading spike candidate |
| Wails v2 | Same language/UI fit; upstream stable release [W1] | Different API; assess required native features before selecting | Stability fallback, not automatic rejection |
| Tauri v2 | Can keep the web UI and run the Go backend as a sidecar [T1] | Adds Rust/toolchain, IPC, process lifecycle and per-target sidecar packaging | Reconsider if Wails fails the native spike |
| Electron | Existing web UI; bundled Chromium with main/renderer/preload separation [E1] | Go sidecar plus Node/Electron lifecycle; larger runtime surface | Fallback if consistent Chromium behavior outweighs Go-first simplicity |
| Native SwiftUI | Strong macOS-specific integration | UI rewrite and a separate Windows/Linux strategy | Poor fit for this cross-platform Go/Svelte product |

Wails is not a visual style system. Keeping Svelte is compatible with Wails [W1];
replacing it with Magpie's plain HTML would discard working UI without removing
the real authoring complexity. Avoid unmeasured size/speed comparisons.

Wails v3's pinned FAQ documents Windows WebView2, macOS Intel/Apple Silicon,
and a Linux GTK4/WebKitGTK 6.0 default with a legacy GTK3 build for older
supported distributions [W1]. Consequently, “Linux support” needs an explicit
supported-distro/runtime matrix, not just a cross-compiled Go binary.

## Recommended architecture to test

```text
Desktop (Go/Wails + packaged Svelte)     CLI (Go)
                 \                       /
             runtime/application operations
               /           |           \
          intake        authoring     publication
             \              |              /
                  core + provider ports
                    SQLite / local media

Publication artifact → export OR a target-specific publisher
Public reader → static host, with no authoring bridge
```

This is a proposed boundary, not a request to create a generic framework.

1. Add a desktop composition root; do not import CLI command implementation
   from the GUI. Keep native window/filesystem integration outside core/domain.
2. Reuse application operations. Extract only HTTP-owned orchestration that the
   desktop actually needs; do not move every server handler preemptively.
3. Test two transports in the spike: a narrow generated Wails service bridge
   [W2], and embedded existing HTTP handlers through the webview asset handler.
   Prefer reuse without a publicly reachable listener. Confirm the chosen
   mechanism against the pinned framework implementation.
4. If loopback HTTP is unavoidable, bind only `127.0.0.1`, select an ephemeral
   port, authenticate requests with a per-run secret, validate Origin/Host,
   and prevent remote content from reaching privileged operations. Loopback
   alone is not an authorization mechanism.
5. Preview the compiled reader separately from the privileged authoring UI.
   External links go to the system browser; third-party sites cannot load in
   the bridge-enabled window. Apply a restrictive CSP and sanitize authored
   rich content.
6. Bundle built admin/reader assets at release time. End users should need no
   Go, Bun, Node, Vite, Python, or source checkout to import, edit and export.
   Audit media processing and map/network dependencies for hidden tool needs.
7. CLI and desktop use the same workspace/schema and publication contract.
   Define a workspace lock or supported concurrency policy, migration ownership,
   cancellation boundaries and transaction behavior before sharing writes.

## Product workflow: what makes this a studio

A desktop wrapper alone will not meet the goal. The first workflow should be:

1. **Create/open a journal.** Show where private data lives; offer a default
   application-data location and an explicit portable workspace option.
2. **Choose sources.** Native picker or drag/drop for GPX, Timeline, photos and
   journey packages. Show supported variants, timezone interpretation,
   duplicates and unsupported records before committing changes.
3. **Review an import plan.** Display route tracks and semantic visits distinctly,
   proposed stops/mementos, media counts and warnings. Preserve the existing
   plan/apply and authored-field protections. Avoid a raw JSON-only experience.
4. **Author.** A journey library and map/timeline workspace, focused editor,
   visible save/error status, understandable published/draft state. No first-run
   demand to arrange files into `route.gpx` plus `photos/` manually.
5. **Review public output.** A published-only preview with the real design,
   rounded geometry and safe derivatives. Clearly distinguish private originals
   from public photos; display total artifact size and target-limit warnings.
6. **Export or publish.** Export folder/ZIP always works without an account.
   Publishing chooses a target and presents the destination and privacy summary
   before network writes. Preserve a previous successful artifact on failure.
7. **Confirm and maintain.** Reveal the exported folder or open the live URL;
   distinguish uploaded from verified live. Explain how unpublishing propagates
   and how backups differ from public exports.

Use Magpie's compact hierarchy, platform-aware interactions and clear operation
states as inspiration. Do not copy its menu-bar-first information architecture:
editing maps, routes, prose and media needs a normal resizable window. A tray
agent, startup service, always-on sync and TUI are not first-release requirements.

## Static export and hosting

The artifact boundary is the product's portability boundary. The export must
contain the built reader plus published JSON and safe media, work under `/` and
a configured project subpath, and contain no SQLite, drafts, original EXIF-rich
photos, local source paths, tokens or admin assets. Validate a fresh output
folder; never recursively upload a user-selected arbitrary workspace.

| Destination | Practical path | Constraints / user burden | Suggested sequence |
| --- | --- | --- | --- |
| Any static host | Complete folder/ZIP + deploy instructions | Manual upload; no provider credentials inside Felicia | First release |
| GitHub Pages | Generated branch with `.nojekyll`, or an artifact-only Actions workflow | Account/repository/source setup, base path, permissions; site ≤1 GB [G1–G2] | First direct-publishing target |
| Cloudflare Pages | Direct Upload folder/ZIP in dashboard, or Wrangler [C1] | Account/project setup; Wrangler adds npm; Direct Upload cannot switch that project to Git integration | Guided export first; native publisher only after API spike |
| Cloudflare Workers Static Assets | Native manifest/upload-session/bucket upload/version deployment [C3] | Different product/API, not a Pages adapter; requires account/token setup | Later alternative if chosen deliberately |
| Netlify | Native API supports digest/missing-file upload or whole-site ZIP [N1] | Public integrations must use OAuth2; docs require client key/secret, so desktop auth needs separate design | Later, after auth feasibility review |
| S3-compatible/CDN, rsync | Upload exported files with user's existing tools | Credentials, endpoint, invalidation and deletion policy vary | Export compatibility, not first-release adapters |

Cloudflare Pages' documented Free limit is 20,000 files and 25 MiB per asset;
paid projects can raise file count to 100,000 with configuration [C2]. These
limits and GitHub's 1 GB site limit matter for photo journals. Preflight counts
and sizes against the chosen target; do not silently omit oversized media.

### GitHub: two intentionally different experiences

**Existing contract:** use installed Git and its credential helper, push only
compiled output to a dedicated target branch, then confirm the public build id.
This aligns with the GUI epic, but it is not frictionless for a new nontechnical
user. Never force-push or overwrite an existing user's branch as a default;
work in an isolated deployment checkout and require explicit target setup.

**Proposed end-user experience:** browser authorization, repository/Pages setup,
then publish from the application. GitHub's device flow needs no client secret,
requires enabling the flow in the app registration, and specifies polling
interval/expiry/error handling [G3]. Evaluate a repository-scoped GitHub App
versus an OAuth App and disclose actual permissions before choosing. Do not
assume adding PKCE removes the client-secret requirement from GitHub's documented
OAuth web flow; the same source still lists that secret as required.

Authentication does not upload a site by itself. Spike repository creation or
selection, Pages source configuration, output commit/upload, deployment triggering
and public manifest verification. Do not assume the Pages REST deployment API
is an arbitrary local-folder uploader.

A publisher should accept only an immutable validated artifact plus a target;
use build id/hash, retries/backoff and explicit job states. Keep credentials in
the OS credential store, outside workspace/config/export/logs; provide disconnect
and revocation guidance. This requires replacing the epic's credential rule via
an explicit decision. No token service or hosted Felicia control plane is
assumed by this research.

## Distribution and data safety

| Platform | Release work to plan | Acceptance evidence |
| --- | --- | --- |
| macOS first | Decide minimum OS; Apple Silicon/Intel or universal build; signed `.app`/DMG; Developer ID, hardened runtime, notarization and stapling [A1, W3] | Fresh-machine install/launch without quarantine bypass; dialogs, map, import/export; upgrade preserves workspace |
| Windows next | x64/ARM64 scope; NSIS packaging; WebView2 availability/bootstrap; signing and SmartScreen handling [W1, W4] | Fresh supported Windows machine, Unicode/long paths, locked files, safe upgrades |
| Linux next | Supported distros; GTK/WebKit dependencies; selected package formats [W1, W5] | Native runner per supported baseline, display-server/file-dialog coverage, credential-store fallback policy |

Avoid relying on private macOS APIs for essential features. Wails documents them
as opt-in [W3]; native polish must not make portability or distribution depend
on undocumented behavior. An App Store release is a separate sandbox/entitlement
project, not implied by shipping a notarized download.

Start with signed manual updates/check-for-update links rather than writing a
self-updater. Before automatic updates: verify artifact authenticity, protect
against rollback, retain recovery paths, test app/workspace migrations and
interrupted installation. Back up SQLite and media consistently, excluding
credentials by default. Public export is **not** a private journal backup.

Treat imported paths/ZIPs, malformed JSON/GPX, huge images and rich text as
untrusted. Validate symlinks and resolved destination boundaries, cap extraction
and processing, avoid shell interpolation, redact secrets and coordinates from
routine diagnostics. Native filesystem access expands risk compared with a
browser upload form; preserve a narrow operation interface.

## Proposed slices and decision gates

These are research recommendations; no dates or implementation estimates are
committed.

| Slice | Deliverable | Must prove before continuing |
| --- | --- | --- |
| 0: native feasibility | Disposable Wails v3/v2 comparison with actual Svelte admin/map and Go operation | Map/webview behavior, bridge isolation, packaging, native files, no end-user dev tools; select framework/version |
| 1: macOS offline studio | Install/open workspace → source plan → author → complete export | GPX + photos and Timeline visit paths; duplicate/re-import protections; private-data exclusion; folder/ZIP portability; signed fresh-machine install |
| 2: publication | One chosen GitHub flow + guided Cloudflare export | Explicit credential decision; target confirmation, subpath/custom-domain behavior, retry/failure/revocation, live build-id confirmation |
| 3: supported platforms | Windows and defined Linux builds, same CLI and workspace contracts | Native CI/install tests and migration parity, documented dependencies |
| 4: convenience | Additional native host adapters, optional auto-update | Demonstrated demand, auth design, update/rollback safety |

Testing should combine shared Go operation/contract tests, CLI/desktop parity,
existing Playwright admin tests, and native-webview smoke tests. Chromium-only
Playwright passing does not prove WKWebView/WebKitGTK behavior. Add privacy
fixtures that search exports for source coordinates, EXIF, drafts, tokens and
local paths, plus root/subpath static-host tests and failed-import/build/publish
recovery. No real provider deployment or native prototype was performed here.

### Decisions needed before implementation

1. Wails v3 beta versus v2 after the native feasibility slice.
2. Default workspace location, portable-workspace support, backup and concurrent
   CLI/desktop policy.
3. Minimum macOS/CPU scope and release signing ownership.
4. Git-credential-assisted versus app-managed GitHub authorization; corresponding
   amendment to the GUI epic/ADR and permission model.
5. Cloudflare Pages versus Workers for an eventual native publisher; whether
   guided dashboard upload is sufficient initially.
6. Concrete Windows/Linux support baselines and how many native release runners
   the project can maintain.

## Research verification record

A fresh Pi peer, `antigravity/gemini-3.8-flash` with medium thinking, reviewed
the working research and reference catalog independently from the workstation
root. It confirmed model access, all five acceptance criteria, source revision
reconciliation, relevant primary-source claims, and the existing Felicia
credential/privacy boundaries; no blocking defects were found.

Reviewed bases: workstation `main` at
`5e84e70d6594570a9ac0d16ea8aad9e4920d9df6` and Felicia
`feat/offline-timezone-defaults` at
`22c97318d4fd50b855202cc0b562529b2d55908c`, including this new research page,
its index link, and the root catalog working diff.

Commands passed (exit 0):

- Independent reviewer: root `make check`, `make build-docs`;
  `make -C vendor/felicia fmt-docs-check`, `make -C vendor/felicia docs-build`.
- Lead: root `make check`, `make build-docs`;
  `make -C vendor/felicia check`, `make -C vendor/felicia docs-build`.

These establish structural/documentation conformance and a source walkthrough,
not native behavior or deployment success. The reviewer suggested linking this
research back from the GUI epic when an implementation spike is accepted; that
follow-up is deferred rather than changing the active epic's contract now. No
catalog gist change is required: it already follows the workstation convention.

## Primary sources

Framework and service versions below are pinned where possible. Hosting docs
are live documentation and limits must be rechecked when implementing.

- [M1]: [Magpie README at examined revision](https://github.com/yetone/magpie/blob/0eb16d4228498c32dbe4040f419d145038ada46e/README.md).
- [M2]: [Magpie go.mod](https://github.com/yetone/magpie/blob/0eb16d4228498c32dbe4040f419d145038ada46e/go.mod).
- [M3]: [Magpie MIT license](https://github.com/yetone/magpie/blob/0eb16d4228498c32dbe4040f419d145038ada46e/LICENSE).
- [W1]: [Wails v3 beta.24 FAQ](https://github.com/wailsapp/wails/blob/v3.0.0-beta.24/docs/mpress/content/faq.md).
- [W2]: [Wails service bindings](https://github.com/wailsapp/wails/blob/v3.0.0-beta.24/docs/mpress/content/features/bindings/services.md).
- [W3]: [Wails macOS packaging](https://github.com/wailsapp/wails/blob/v3.0.0-beta.24/docs/mpress/content/guides/build/macos.md).
- [W4]: [Wails Windows packaging](https://github.com/wailsapp/wails/blob/v3.0.0-beta.24/docs/mpress/content/guides/build/windows.md).
- [W5]: [Wails Linux packaging](https://github.com/wailsapp/wails/blob/v3.0.0-beta.24/docs/mpress/content/guides/build/linux.md).
- [T1]: [Tauri v2: embedding external binaries](https://v2.tauri.app/develop/sidecar/).
- [E1]: [Electron process model](https://www.electronjs.org/docs/latest/tutorial/process-model).
- [A1]: [Apple Developer ID distribution](https://developer.apple.com/developer-id/).
- [G1]: [GitHub Pages limits](https://docs.github.com/en/pages/getting-started-with-github-pages/github-pages-limits).
- [G2]: [GitHub Pages publishing source](https://docs.github.com/en/pages/getting-started-with-github-pages/configuring-a-publishing-source-for-your-github-pages-site).
- [G3]: [GitHub OAuth authorization and device flow](https://docs.github.com/en/apps/oauth-apps/building-oauth-apps/authorizing-oauth-apps).
- [C1]: [Cloudflare Pages Direct Upload](https://developers.cloudflare.com/pages/get-started/direct-upload/).
- [C2]: [Cloudflare Pages limits](https://developers.cloudflare.com/pages/platform/limits/).
- [C3]: [Cloudflare Workers Static Assets Direct Upload](https://developers.cloudflare.com/workers/static-assets/direct-upload/).
- [N1]: [Netlify API authentication and deployments](https://docs.netlify.com/api-and-cli-guides/api-guides/get-started-with-api/).

[M1]: https://github.com/yetone/magpie/blob/0eb16d4228498c32dbe4040f419d145038ada46e/README.md
[M2]: https://github.com/yetone/magpie/blob/0eb16d4228498c32dbe4040f419d145038ada46e/go.mod
[M3]: https://github.com/yetone/magpie/blob/0eb16d4228498c32dbe4040f419d145038ada46e/LICENSE
[W1]: https://github.com/wailsapp/wails/blob/v3.0.0-beta.24/docs/mpress/content/faq.md
[W2]: https://github.com/wailsapp/wails/blob/v3.0.0-beta.24/docs/mpress/content/features/bindings/services.md
[W3]: https://github.com/wailsapp/wails/blob/v3.0.0-beta.24/docs/mpress/content/guides/build/macos.md
[W4]: https://github.com/wailsapp/wails/blob/v3.0.0-beta.24/docs/mpress/content/guides/build/windows.md
[W5]: https://github.com/wailsapp/wails/blob/v3.0.0-beta.24/docs/mpress/content/guides/build/linux.md
[T1]: https://v2.tauri.app/develop/sidecar/
[E1]: https://www.electronjs.org/docs/latest/tutorial/process-model
[A1]: https://developer.apple.com/developer-id/
[G1]: https://docs.github.com/en/pages/getting-started-with-github-pages/github-pages-limits
[G2]: https://docs.github.com/en/pages/getting-started-with-github-pages/configuring-a-publishing-source-for-your-github-pages-site
[G3]: https://docs.github.com/en/apps/oauth-apps/building-oauth-apps/authorizing-oauth-apps
[C1]: https://developers.cloudflare.com/pages/get-started/direct-upload/
[C2]: https://developers.cloudflare.com/pages/platform/limits/
[C3]: https://developers.cloudflare.com/workers/static-assets/direct-upload/
[N1]: https://docs.netlify.com/api-and-cli-guides/api-guides/get-started-with-api/
