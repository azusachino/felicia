# AGENTS.md — felicia

Single source of truth for humans and agents working in this repo.

## Project Overview

**felicia** is a map-based travel journal (modeled on [liuaaron.com](https://liuaaron.com/),
"Aaron's Waypoints"). Each **journey** is drawn on a dark world map as an orange route line;
along it sit **ticket stubs** (receipts, transit passes, admission tickets). Clicking a ticket
animates it open into an **essay** and a **photo gallery**. The map is the index; the tickets
are the stories.

North star: [`docs/direction.md`](docs/direction.md) (direction: *personal now,
product-ready*). Earlier design/spec drafts are parked in [`docs/archive/`](docs/archive/).
Status: **research stage** — flow is research → spec → TDD → implementation, unhurried
(~6-month horizon).

## Tech Stack & Architecture

- **Backend:** Go 1.26 — a `waypoints` ingestion CLI + an HTTP API server.
- **DB:** Postgres + PostGIS (relational + geo; canonical source of truth).
- **Object storage:** S3-compatible interface; **R2** backend (MinIO/B2 swappable by config).
- **Frontend:** Vite + Mapbox GL SPAs — public site + admin authoring app (bun workspace).
- **Host:** Raspberry Pi (docker-compose) behind a **Cloudflare Tunnel** (no open ports).
- **Ingestion sources (self-hosted):** Immich (photos/ticket stubs, via API) + Dawarich
  (passive iPhone GPS track, via API); joined on timestamp. Vision-LLM (Claude) pre-fills
  ticket metadata for confirmation.

**Authoring model (A+E):** an auto-ingest pipeline seeds *ingested* fields; an admin UI is
where you author *essays / photo curation / animation*. The importer is **field-scoped** and
**never overwrites authored fields** — re-import is always safe (see design §5).

### Planned layout

```
cmd/{waypoints,api}        internal/{domain,geo,exif,gpx,immich,dawarich,ocr,
migrations/  content/trips/   importer,store/{pg,memrepo},objectstore,config,api}
web/{public,admin}  deploy/  docs/
```
`internal/domain` is the pure TDD core (no I/O). Every external source is an interface impl.

## Build, Run & Test

All daily operations go through `make <target>`. **Tools:** runtimes (go, bun) from **mise**
(`mise install`); system tools (golangci-lint, goose, postgres/postgis) from the **nix flake**
(`nix develop`, or `make` wraps them via `NIX_RUN`).

| Target | Does |
| --- | --- |
| `make fmt` | format Go |
| `make vet` | `go vet ./...` |
| `make lint` | `golangci-lint run` (nix) |
| `make test` | `go test -race -cover ./...` |
| `make check` | fmt + vet + lint + test — **before commit** |
| `make build` | build all binaries |
| `make validate` | check + build — **before PR** (frontend + migration smoke join once those exist) |
| `make migrate` | `goose up` (needs `DATABASE_DSN`) |

## Coding Conventions

- Conventional commits (`feat:`, `fix:`, `chore:`, `deploy:`) — no emojis.
- Go: standard `gofmt`/`goimports`; errors wrapped with context; small interfaces at seams.
- 2-space indent for config files (YAML/TOML/JSON).
- Test-first for the importer core; pure functions + fixtures, no network in unit tests.
- Keep it simple — avoid speculative abstraction.

## Key Files & Entry Points

- `docs/direction.md` — research-stage north star: the idea + *personal-now / product-ready* direction.
- `docs/research/` — exploration trail (workflows, liuaaron teardown, product-vs-personal).
- `docs/archive/` — parked design/spec/plan drafts (premature lock-in); detail, not binding.
- rosemary graph `felicia:*` — decisions (ADRs), session state. Run `/rosemary start`.

## Quality Standards

`make check` must pass before every commit; `make validate` before every PR (both
hook-enforced). No `--no-verify`. Don't commit or push without explicit confirmation.
