# AGENTS.md — felicia

Single source of truth for humans and agents working in this repo.

## Project Overview

**felicia** is a map-based travel journal (modeled on [liuaaron.com](https://liuaaron.com/),
"Aaron's Waypoints"). Each **journey** is drawn on a dark world map as an orange route line;
along it sit **mementos** — the objects that anchor a memory (an admission ticket, but equally
a souvenir, a goods, a receipt, a stamp), each rendered as a collectible **stub**. Clicking a
memento animates it open into an **essay** and a **photo gallery**. The map is the index; the
mementos are the stories. (`kind`-tagged; physical tickets are dying, so stubs are rendered
from data — see `docs/research/mementos-not-tickets.md`.)

North star: [`docs/direction.md`](docs/direction.md) (direction: _personal now,
product-ready_). Earlier design/spec drafts are parked in [`docs/archive/`](docs/archive/).
Status: **implementation stage** (research trail continues), unhurried (~6-month horizon).
Delivery status lives in [`docs/roadmap.md`](docs/roadmap.md); the selected end-to-end
journey and its per-stage status live in
[`docs/roadmap/user-journey.md`](docs/roadmap/user-journey.md).

## Tech Stack & Architecture

- **Backend:** Go 1.27 — API, runtime, provider, and core modules in one `go.work` workspace.
- **DB:** SQLite is the only v1 persistence contract ([ADR 0005](docs/adr/0005-sqlite-storage-and-content-addressed-media.md)).
  The PostgreSQL/PostGIS provider has been retired.
- **Media storage:** provider-neutral `BlobStore` with private, content-addressed local originals;
  publication emits resized, metadata-stripped derivatives ([ADR 0005](docs/adr/0005-sqlite-storage-and-content-addressed-media.md)).
- **Frontend:** SvelteKit static SPA for Admin authoring; Svelte + Vite for the Atlas public reader.
  Both share a Bun workspace. Go remains the backend; no JavaScript server is deployed.
- **Locales:** static system UI catalogs support Japanese, English, and Chinese. Authored content
  has no translation sidecar and is rendered exactly as entered.
- **Host:** self-hosted container deployment; Cloudflare Tunnel is an optional ingress.
- **Ingestion sources:** Immich photos and Dawarich routes/visits via API; local GPX/photo intake;
  and Google Timeline exports as named local visit input. The current Timeline adapter does not
  import activity geometry. Inputs create reviewable candidates; automatic vision-based metadata extraction
  is not implemented.

**Authoring model (A+E):** an auto-ingest pipeline seeds _ingested_ fields; the desktop studio / admin UI is
where you author _essays / photo curation / animation_. The importer is **field-scoped** and
**never overwrites authored fields** — re-import is always safe (see design §5).

### Agent Operations & Intake Boundary

- **The CLI is the only intake tool for agents:**
  - AI agents interact with Felicia intake strictly via `felicia-cli`. Never invoke or look for Python wrappers.
  - To ingest a trip into the user's unified workspace (`~/.felicia` or `$FELICIA_WORKSPACE`):

    ```sh
    felicia-cli journey ingest --dir /path/to/trip-folder
    ```

  - The trip folder contract (`docs/contracts/trip-folder-contract.md`) accepts `route.gpx` (or `timeline.json`), `photos/`, and optional `photos.jsonl` sidecars.
  - `felicia-cli` performs candidate generation, content-addressed media ingestion into the blob store, and transactional SQLite drafting in a single step with full authorship protection.
  - Visual review and essay authoring are handled interactively in the Desktop Studio (`make desktop`) or Web Admin (`make admin`).

### Current layout

```text
apps/{felicia-core,felicia-runtime,felicia-providers,felicia-publication,
felicia-server,felicia-cli,felicia-desktop,felicia-admin,felicia-public-site}/
packages/{felicia-model,felicia-reader}/  contracts/  ops/  scripts/  docs/
```

The ownership map and dependency direction are defined in
[`docs/development/layout.md`](docs/development/layout.md) and
[ADR 0007](docs/adr/0007-application-and-package-layout.md).
`felicia-core` is the pure domain and port layer (no I/O). `felicia-runtime`
owns use cases, `felicia-providers` owns persistence implementations,
`felicia-publication` owns the public contract, and apps/felicia-server/CLI adapters compose
runtime and publication ports. `felicia-reader` owns the public reader facade,
Atlas theme, and memento components. `felicia-model` owns reader
data/public contracts and i18n catalogs. The admin studio and public site remain separate hosts.
The root Go module has been retired; all Go code is built through `go.work`.

## Build, Run & Test

All daily operations go through `make <target>`; use `make help` for the current target list.
mise selects Node LTS and stable/latest channels for the other managed tools;
`mise.lock` records the resolved versions. Use `mise install --locked` after checkout.
Bun installs the frozen frontend workspace lockfile and executes compatible
scripts with `--bun`; source uses portable Node-style APIs. Node LTS is the
fallback for demonstrated runtime capability failures, not a second package
manager. Vitest owns unit tests and Playwright owns browser checks. A runtime
probe checks that unit tests execute under the requested runtime.
SQLite uses a single `apps/felicia-providers/sqlite/schema.sql`, without a migration framework.

| Target          | Does                                                                             |
| --------------- | -------------------------------------------------------------------------------- |
| `make fmt`      | format Go, frontend code, and Markdown                                            |
| `make vet`      | `go vet ./...`                                                                   |
| `make lint`     | `golangci-lint run` (mise)                                                       |
| `make test`     | `go test -race -cover ./...`                                                     |
| `make check`    | formatting checks + vet + lint + tests + feature contracts — **before commit**  |
| `make build`    | build all binaries                                                               |
| `make validate` | check + build + public/Admin frontend checks — **before PR**                     |
| `make admin`    | local admin GUI: authoring API + felicia-admin on `0.0.0.0` for Tailscale access |
| `make desktop`  | native desktop studio: package and run local macOS app (`FeliciaStudio.app`)     |
| `make e2e-install` | install Chromium/WebKit for project-pinned Playwright                         |
| `make e2e`      | integrated Bun/Playwright desktop-composition tests on isolated synthetic state |

`make e2e` builds a test-only headless transport (`production,e2e`) using the real
embedded admin and desktop handler. It does not drive the Wails window or native
dialogs. Normal desktop builds have no test flags or authoring TCP listener. Evidence
lives under workstation `.tmp/felicia-desktop-redesign/`; native visual acceptance
remains a separate gate. `make test-admin-e2e` retains the existing web-host authoring
closed loop. The ordered redesign plan is
[`docs/roadmap/desktop-studio-redesign.md`](docs/roadmap/desktop-studio-redesign.md).

## Coding Conventions

- Conventional commits (`feat:`, `fix:`, `chore:`, `deploy:`) — no emojis.
- Go: standard `gofmt`/`goimports`; errors wrapped with context; small interfaces at seams.
- 2-space indent for config files (YAML/TOML/JSON).
- Test-first for the importer core; pure functions + fixtures, no network in unit tests.
- Keep it simple — avoid speculative abstraction.
- **UI library first (hard rule):** use the existing shared UI components and
  Bits UI primitives for studio controls before native or custom alternatives.
  Extend shared components rather than styling one-off controls in a view.
  A fallback requires a demonstrated library limitation or failure, recorded
  with the attempted component, reproducer, and verification evidence. Convenience
  or incomplete wiring is not a reason to bypass the library.

## Key Files & Entry Points

- `docs/direction.md` — research-stage north star: the idea + _personal-now / product-ready_ direction.
- `docs/research/` — exploration trail (workflows, liuaaron teardown, product-vs-personal,
  mementos-not-tickets, notion-prototype, notion-to-stack, source-connectors, transit-tickets,
  authoring-publish-flow, ux-restyle, memento-arrangement, reader-admin-surfaces,
  adventurelog teardown). Backend core: `backend-stack.md` (stack + decisions D1–D9),
  `data-model.md` (stable schema), `memento-templates.md` (declarative kind-template registry).
- `docs/archive/` — parked design/spec/plan drafts (premature lock-in); detail, not binding.
- asobi graph `felicia:*` — decisions (ADRs), session state. Run `asobi` commands.

## Quality Standards

`make check` must pass before every commit; `make validate` before every PR (both
hook-enforced). No `--no-verify`. Don't commit or push without explicit confirmation.

## Development-Flow Constraints

These are cheap rules that would have caught defects this repo actually shipped.
Each one names the failure it prevents, so it can be retired if the failure
stops being possible.

1. **Provider intent is explicit, and mis-selection fails loudly.**
   [ADR 0005](docs/adr/0005-sqlite-storage-and-content-addressed-media.md) defines
   SQLite as the sole v1 engine; any unselected or misconfigured DSN fails loudly at startup.

2. **A v1 schema change lands in SQLite only (`schema.sql`).**
   [ADR 0005](docs/adr/0005-sqlite-storage-and-content-addressed-media.md) establishes
   SQLite as the single persistence engine; breaking changes are acceptable while unreleased.

3. **Every user-facing surface has exactly one documented `make` target.**
   The admin GUI — the primary authoring surface — had no launcher, so the
   documented flow could not be started from the documentation. If a person is
   meant to open it, `make help` names it.

4. **The local authoring stack may bind the tailnet, and packaging must not publish it.**
   `make admin` binds the API, site preview, and admin GUI to `0.0.0.0` so the
   author can use them from a Tailscale client; use `FELICIA_HOST=127.0.0.1`
   for a host-only session. The admin API remains unauthenticated by design,
   so the host firewall/Tailscale policy must be the access boundary. Deployment
   packaging never publishes the admin port.

5. **Superseding an ADR answers the costs the superseded one enumerated.**
   [ADR 0001](docs/adr/0001-personal-now-product-ready.md) and
   [ADR 0005](docs/adr/0005-sqlite-storage-and-content-addressed-media.md)
   consolidated the architecture onto a single SQLite engine. Historical ADRs
   are preserved in `docs/archive/adr/`.

6. **Private authoring data never sits on a committable path.**
   The original journal is the artifact ADR-0025 says must not leave the
   machine. Private workspaces, databases, and originals live under `.felicia/`.
   Sanitized, explicitly published journey inputs may be committed only under
   `publication/journeys/`; `.gitignore` covers every SQLite spelling the
   tooling can emit.

## Docs-Sync Discipline (per PR)

Every PR that changes system behavior, capability, or delivery progress must update
the matching status docs **in English, in the same PR, before merge**:

- the README "Status & roadmap" summary, if the overall picture changed;
- [`docs/roadmap.md`](docs/roadmap.md) / the active epic doc, if milestone or epic
  progress changed;
- [`docs/roadmap/user-journey.md`](docs/roadmap/user-journey.md) — the per-stage
  status of the selected end-to-end journey (collection → intake → authoring →
  publish → deploy).

This is checked at the PR gate alongside `make validate`; individual commits are not
required to carry doc updates. GitHub is the single ledger for issue state — do not
recreate local issue mirrors (the old drafts are archived under
`docs/archive/github-issues/`); scripted issue lookups use a `GITHUB_TOKEN`
environment variable, not interactive `gh auth login`.

## Issue Convention

One classification standard for the open ledger. Three competing title schemes
(`M0 —`, `[FELICIA-PAGES-01.x]`, `P0:`) and an unlabelled backlog previously made
"what is next" unanswerable without reading every issue.

- **Title:** `R<n>: <lowercase summary>`, where `R0`–`R5` is the roadmap milestone
  from [`docs/roadmap.md`](docs/roadmap.md). No other prefix. Priority and type live
  in labels, never in the title.
- **Milestone:** every issue carries its `R<n>` milestone — this is the coarse
  ordering, and it is what makes milestone completion percentages true.
- **Type label** (exactly one): `type:epic` (milestone umbrella), `type:feature`,
  `type:defect` (shipped behavior contradicts a binding contract), `type:decision`
  (needs an ADR or a joint call first).
- **Priority label** (exactly one): `prio:P0` breaks a stated invariant or destroys
  data · `prio:P1` blocks the documented end-to-end journey · `prio:P2` real gap with
  a workaround · `prio:P3` correctness/polish, nobody blocked.
- **Work order:** milestone ascending, then priority ascending. Query it, don't guess:

  ```text
  gh issue list --state open --milestone "R4 — Ingestion and route enrichment" --label prio:P0
  ```

- Closed issues keep their historical titles: `[FELICIA-PAGES-01.x]` is the
  doc↔issue trace in [`pages-v1-epic.md`](docs/roadmap/pages-v1-epic.md). They carry
  milestones but are not retitled. Epic-local `M1`–`M4` numbering stays scoped to its
  epic doc and never appears in an issue title.
- A `type:defect` body cites the contract it violates (ADR, `docs/contracts/*`, or
  AGENTS.md) plus `file:line` evidence. Without that citation it is a `type:feature`.
