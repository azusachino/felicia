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
- **DB:** SQLite is the local-first provider; PostgreSQL remains supported for deployments that need it.
- **Object storage:** S3-compatible interface; **R2** backend (MinIO/B2 swappable by config).
- **Frontend:** Vite + MapLibre GL SPAs — public site, private reader, and admin authoring app (bun workspace).
- **Locales:** static system UI catalogs support Japanese, English, and Chinese. Authored content
  has no translation sidecar and is rendered exactly as entered.
- **Host:** self-hosted container deployment; Cloudflare Tunnel is an optional ingress.
- **Ingestion sources (self-hosted):** Immich (photos/ticket stubs, via API) + Dawarich
  (passive iPhone GPS track, via API); joined on timestamp. Vision-LLM (Claude) pre-fills
  ticket metadata for confirmation.

**Authoring model (A+E):** an auto-ingest pipeline seeds _ingested_ fields; an admin UI is
where you author _essays / photo curation / animation_. The importer is **field-scoped** and
**never overwrites authored fields** — re-import is always safe (see design §5).

### Current layout

```
apps/{felicia-core,felicia-runtime,felicia-providers,felicia-publication,
felicia-server,felicia-cli,felicia-admin,felicia-web,felicia-public-site}/
packages/{felicia-model,felicia-runtime,felicia-components,felicia-renderers,
felicia-reader}/  contracts/  ops/  scripts/  docs/
```

The ownership map and dependency direction are defined in
[`docs/development/layout.md`](docs/development/layout.md) and
[ADR-0034](docs/adr/0034-application-and-shared-package-layout.md).
`felicia-core` is the pure domain and port layer (no I/O). `felicia-runtime`
owns use cases, `felicia-providers` owns persistence implementations,
`felicia-publication` owns the public contract, and apps/felicia-server/CLI adapters compose
runtime and publication ports. `felicia-reader` owns the public reader facade,
named design registry, and concrete compositions. `felicia-model` owns reader
data/public contracts; `felicia-runtime`, `felicia-components`, and
`felicia-renderers` are reusable package boundaries. The admin, private reader,
and public site remain separate hosts.
The root Go module has been retired; all Go code is built through `go.work`.

## Build, Run & Test

All daily operations go through `make <target>`. **Tools:** Go, Bun, uv, Prettier,
golangci-lint, goose, and sqlc come from the checked-in **mise** configuration. PostgreSQL
18 + PostGIS remain disposable container infrastructure.

| Target          | Does                                                                             |
| --------------- | -------------------------------------------------------------------------------- |
| `make fmt`      | format Go                                                                        |
| `make vet`      | `go vet ./...`                                                                   |
| `make lint`     | `golangci-lint run` (mise)                                                       |
| `make test`     | `go test -race -cover ./...`                                                     |
| `make check`    | fmt + vet + lint + test + feature contracts — **before commit**                  |
| `make build`    | build all binaries                                                               |
| `make validate` | check + build + public/admin/private frontend checks — **before PR**             |
| `make migrate`  | `goose up` (needs `DATABASE_DSN`)                                                |
| `make admin`    | local admin GUI: authoring API + felicia-admin on `0.0.0.0` for Tailscale access |

## Coding Conventions

- Conventional commits (`feat:`, `fix:`, `chore:`, `deploy:`) — no emojis.
- Go: standard `gofmt`/`goimports`; errors wrapped with context; small interfaces at seams.
- 2-space indent for config files (YAML/TOML/JSON).
- Test-first for the importer core; pure functions + fixtures, no network in unit tests.
- Keep it simple — avoid speculative abstraction.

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
   [ADR-0021](docs/adr/0021-runtime-configuration-and-database-modes.md) already
   forbids implicit provider changes, but the config contract has a hole: a
   PostgreSQL DSN with no `DATABASE_DRIVER` silently starts SQLite.
   `ops/compose.yaml` does exactly this, so its API ran on a throwaway
   in-container SQLite file while Postgres, PostGIS, and every migration sat
   unused — with nothing in the logs to say so. Configuring a DSN for a provider
   you did not select must be a startup error, never a silent default.

2. **A dual-provider schema change ships a parity check.**
   [ADR-0017](docs/adr/0017-sqlite-first-storage.md) required conformance tests
   "to prevent SQLite and PostgreSQL behavior from drifting", and
   `apps/felicia-providers/contract` delivers that — for _behavior_. Schema shape is
   unguarded, and the two DDLs have already diverged (`tb_journal` in
   `apps/felicia-server/migrations/`, `tb_journals` in `apps/felicia-providers/sqlite/schema.sql`). Any change
   touching both providers asserts shape parity in a test, not in review.

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
   [ADR-0008](docs/adr/0008-single-user-local-first-ssg.md) rejected a second
   database engine and named the three costs it was avoiding: duplicate DDL
   migrations, repository translation layers, and custom Go-memory spatial
   logic. [ADR-0017](docs/adr/0017-sqlite-first-storage.md) reversed it five days
   later on local-setup ergonomics without disputing those costs — and all three
   arrived (a second DDL that has drifted, ~1.3k lines of second provider,
   `SnapToRoute`/`GetDisplayRoute` reimplemented in Go). A superseding ADR states
   how each named cost will be contained, or records that it is accepted.

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

  ```
  gh issue list --state open --milestone "R4 — Ingestion and route enrichment" --label prio:P0
  ```

- Closed issues keep their historical titles: `[FELICIA-PAGES-01.x]` is the
  doc↔issue trace in [`pages-v1-epic.md`](docs/roadmap/pages-v1-epic.md). They carry
  milestones but are not retitled. Epic-local `M1`–`M4` numbering stays scoped to its
  epic doc and never appears in an issue title.
- A `type:defect` body cites the contract it violates (ADR, `docs/contracts/*`, or
  AGENTS.md) plus `file:line` evidence. Without that citation it is a `type:feature`.
