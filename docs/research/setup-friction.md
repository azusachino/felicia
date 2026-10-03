# Setup friction assessment

Source inventory on 2026-10-03 at `feat/desktop-studio-revision`, commit
`7123d5f4377d6f0b36a3f25045bebaf870609bf9`. The unfinished shadcn/full-page UI
work is still uncommitted; this assessment does not treat it as delivered.
This inventory predates implementation; its Node/pnpm baseline below was a
proposal, not the selected final direction.

## Subsequent owner decision

The owner rejected pnpm and approved a Bun workspace/package manager, actual
Bun execution where compatible, and Node LTS fallback for demonstrated capability
failures. Keep portable Node-style source and Vitest/Playwright. The owner also
approved SvelteKit for the studio only, as a static client-rendered app with the
Go backend unchanged. Persistent authoring layouts and framework navigation
guards justify that choice; the public reader remains Svelte + Vite.
Implementation is in progress and requires browser/full-gate independent
verification. Native design acceptance and shared authoring API parity are not
implied by these choices.

## Summary

There are six identifiable areas worth simplifying, not a measured percentage of
technical debt. Changing Bun to Node/pnpm addresses only part of the problem.
Most avoidable maintenance is in custom behavior and duplicated composition.

| Area | Current evidence | Recommended direction |
| --- | --- | --- |
| Handwritten frontend routing | The former route union, regex parser and shell rendering switch were replaced in `7123d5f`. | Keep the maintained Svelte router and extend its route table. This item is already fixed. |
| Frontend toolchain and test environment | `mise.toml` pins Bun but not Node. Root scripts are Bun-specific; eight test files import `bun:test`. `apps/felicia-public-site/src/api/source.test.ts` patches `import.meta.env` for its non-Vite test environment. | Prefer a pinned supported Node LTS, pnpm workspace and Vitest for Vite-aware tests, retaining Playwright. This is a migration proposal, not evidence that Bun is broken. |
| Separate studio control/style implementations | `JourneyList`, `JourneyDetail`, `MementoEditor` and `SiteDeploy` each contain their own control styles; the existing editor/detail/output styles include independent hardcoded palettes. | Finish the approved shadcn-svelte primitives and one shared theme. A package-manager change cannot fix visual inconsistency. |
| Duplicated authoring HTTP composition | `apps/felicia-desktop/handler.go` and `apps/felicia-server/api/server.go` separately route and implement overlapping journey, memento and site operations. The server has photo/intake routes absent from desktop. | Consider one shared Go authoring HTTP adapter composed by both hosts, retaining host-specific security, native pickers and assets. Existing instructions prohibit desktop importing the server composition module; a shared boundary requires an explicit design decision. |
| Hand-maintained orchestration and command drift | There are 22 Python scripts, including substantive tests/import tooling, not 22 disposable wrappers. Make delegates through mise/uv/Python and Go/Bun. `make tidy` invokes root `go mod tidy` despite no root Go module. `db-up` still describes Postgres even though Compose has no Postgres service. | Remove misleading/unused paths first. Keep useful process lifecycle and native build logic; simplify frontend orchestration with package scripts and workspace filters. Do not translate all Python scripts to JavaScript merely for uniformity. |
| Native framework and compatibility cost | Desktop pins Wails `v3.0.0-beta.24`; build scripts manage platform flags, embedded assets, packaging and signatures. | Treat the beta framework as an explicit risk decision. Keep it if Go reuse/native direction justify it; evaluate a stable alternative only against concrete missing capabilities. Node/pnpm does not remove native integration work. |

## Proposed frontend baseline

```text
Pinned supported Node LTS
pnpm workspace
Svelte 5 + Vite
svelte-spa-router
shadcn-svelte / Bits UI + shared Felicia tokens
Vitest for unit/component tests
Playwright for browser integration
```

Node is the runtime; pnpm manages packages/workspaces; Vite builds/dev-serves;
Vitest owns the Vite-aware test environment. They solve different problems.
Replacing only `bun install` with `pnpm install` would leave Bun-dependent tests,
mock serving and command wrappers behind. `scripts/browser_preview_mock.ts` uses
`Bun.serve` and `Bun.env`; that path also needs a deliberate replacement or retained
Bun dependency. Update lockfiles, CI, container builds, make/scripts and local
cache tool keys together; do not leave two package managers as competing defaults.

SvelteKit is an option if file-based routing and its application conventions become
valuable. Its static adapter can build a client-side app; it does not require SSR
for this use case. However, migrating the embedded desktop host is a larger change
than using the current Vite app with a standard router. No concrete need for that
migration was established here. React/Electron or a monorepo task framework would
also be larger decisions, not prerequisites for fixing the identified problems.

## Priorities and limits

1. Finish the already-approved common studio UI and full-page creation slice.
2. If the owner selects it, migrate the frontend toolchain in one bounded slice,
   preserving workspace dependencies, unit coverage and both browser workflows.
3. Make the shared authoring API boundary an explicit acceptance/design task;
   routing-library adoption alone does not resolve the desktop endpoint gaps.
4. Fix stale commands and docs as focused changes. In particular,
   `docs/development/layout.md` still describes retired frontend packages and
   cites ADR-0034 rather than ADR 0007's intended consolidation. The layout guide
   remains the canonical current-tree document; its stale claims need correction.
5. Do not flatten seven Go modules solely because there are seven. Their ownership
   boundaries are intentional in ADR 0007; no external-consumer or build-time
   evidence justifies a consolidation in this inventory.

The read-only backend/native inventory was independently performed by fresh
Herdr peer `felicia-friction-scan` (`wV:p4T`), Pi
`openai-codex/gpt-6-luna`, medium. It reviewed the same owning revision and made
no source changes, builds, native launches or remote writes. This is a bounded
source/docs assessment, not a full architecture audit, migration benchmark,
security review or native acceptance.

## Sources

Repository sources are authoritative for current configuration:

- [`package.json`](../../package.json), [`mise.toml`](../../mise.toml),
  [`Makefile`](../../Makefile), [`CI`](../../.github/workflows/ci.yml) and
  [`Compose`](../../ops/compose.yaml).
- [`admin`](../../apps/felicia-admin/package.json),
  [`public site`](../../apps/felicia-public-site/package.json),
  [`public source tests`](../../apps/felicia-public-site/src/api/source.test.ts) and
  [`preview mock`](../../scripts/browser_preview_mock.ts).
- [`desktop handler`](../../apps/felicia-desktop/handler.go),
  [`server API`](../../apps/felicia-server/api/server.go),
  [`desktop dependencies`](../../apps/felicia-desktop/go.mod),
  [`Go workspace`](../../go.work) and
  [ADR 0007](../adr/0007-application-and-package-layout.md).

Primary framework/tool references:

- [Vite getting started](https://vite.dev/guide/): supports multiple package
  managers; documents supported Node versions. This does not establish Bun failure.
- [pnpm workspaces](https://pnpm.io/workspaces): workspace membership and local
  package resolution, not a replacement for Go/native orchestration.
- [Vitest](https://vitest.dev/guide/): Vite-native test configuration and transforms.
- [SvelteKit static adapter](https://svelte.dev/docs/kit/adapter-static): static
  deployment and SPA fallback are supported alternatives to server rendering.
- [svelte-spa-router](https://github.com/ItalyPaleAle/svelte-spa-router): current
  Svelte 5/hash-routing foundation.
- [Bun Node compatibility](https://bun.sh/docs/runtime/nodejs-compat): compatibility
  claims are API-specific; no blanket incompatibility claim is made here.
