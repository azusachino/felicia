---
title: "Felicia Architecture Experiments"
status: "exploratory"
date: "2026-07-17"
---

# Felicia Architecture Experiments

This directory records executable experiments used to choose Felicia's data model,
static publication workflow, and self-hosted storage providers. Nothing here is a
final architecture decision. A result may reject the current implementation and
may lead to an ADR revision.

## Working hypotheses

1. Felicia's canonical model should be relational SQLite/PostgreSQL data, not a
   hand-edited JSON document.
2. GPX/XML should be preserved as source material and normalized into route
   segments and timestamped points.
3. Static JSON/GeoJSON should be generated as a read projection for GitHub Pages.
4. A static build should work from SQLite with local media and no API server,
   PostgreSQL, Valkey, or cloud credentials.
5. The same canonical fixture should exercise SQLite, PostgreSQL, server reads,
   and static publication.

## Experiment matrix

| ID  | Question                                                                                | Setup                                                   | Evidence                                                | Exit condition                                                          |
| --- | --------------------------------------------------------------------------------------- | ------------------------------------------------------- | ------------------------------------------------------- | ----------------------------------------------------------------------- |
| E1  | Can the current SQLite workflow preserve a journey, memento, route, and media metadata? | Existing `scripts/test_journey_workflow.py` with SQLite | API assertions and database inspection                  | Baseline behavior recorded before schema changes                        |
| E2  | Can the current PostgreSQL provider satisfy the same contract?                          | Podman Compose PostgreSQL/PostGIS and existing workflow | Provider contract and workflow results                  | No provider-specific domain behavior is required                        |
| E3  | Is route-point storage better than one `gps_route` JSON column?                         | Same GPX fixture imported into both schemas             | queryability, round-trip fidelity, size, and build time | Keep timestamps/elevation and rebuild public geometry deterministically |
| E4  | Can a SQLite database produce the complete static reader artifact?                      | SQLite fixture, local media root, static publisher      | `dist/` inspection and browser smoke                    | no API, PostgreSQL, Valkey, or cloud dependency                         |
| E5  | Can GitHub Actions build and publish that artifact?                                     | disposable runner job with no secrets                   | Pages artifact and deployed URL                         | project-site base path and media URLs work                              |
| E6  | Do 100 journeys remain practical?                                                       | generated relational fixture with 100 journeys          | DB size, build time, JSON projection size, browser load | index stays lightweight; journey data loads on demand                   |
| E7  | Does local media behave like the future blob provider?                                  | local originals/derivatives and content hashes          | upload, derivative, publish, rebuild                    | public paths are stable and private originals never enter `dist/`       |

## Run the current baselines

```bash
make test-workflow
make fmt-check
make web-check
```

PostgreSQL requires a disposable DSN:

```bash
make db-up
DATABASE_DSN='postgres://postgres:password@localhost:5432/felicia?sslmode=disable' make migrate
FELICIA_TEST_DATABASE_DSN='postgres://postgres:password@localhost:5432/felicia?sslmode=disable' \
  make test-workflow-postgres
```

The experiments should prefer `uv run` scripts for orchestration and keep fixture
data under version control. Podman/Compose is only the disposable infrastructure
boundary; it must not become part of the core model.

## Evidence to persist

Each experiment should record:

- command and environment profile;
- input fixture identity and checksum;
- schema/provider version;
- result and failure output;
- measured database size and build duration when relevant;
- a conclusion: retain, revise, or reject the hypothesis.

Do not call a hypothesis an accepted ADR until its experiment has a reproducible
result or the product owner explicitly chooses a direction without experimentation.

Current evidence:

- [2026-07-17 provider baseline](2026-07-17-provider-baseline.md)
- [2026-07-17 GitHub Pages design demo](2026-07-17-pages-design-demo.md)
- [2026-07-17 public read contract](2026-07-17-public-read-contract.md)

## Existing experiment materials

The repository already contains the first input set and orchestration examples:

- `scripts/test_journey_workflow.py` — `uv`-runnable HTTP workflow case;
- `scripts/data.json` and `scripts/data.schema.json` — current seed input and its
  validation shape;
- `scripts/tracks/*.gpx` — source GPX fixtures;
- `deploy/compose.yaml` — disposable PostgreSQL, Valkey, API, Caddy, and tunnel
  integration;
- `deploy/Caddyfile` — shared-demo public-origin routing, including the deliberate
  rule that the admin API is not publicly proxied.

## Current known implementation gaps

- `apps/providers/sqlite/schema.sql` stores `tb_journeys.gps_route` as JSON text;
- `apps/apiserver/cmd/build/main.go` constructs PostgreSQL directly;
- the static publisher does not yet copy local public media derivatives;
- the GitHub Pages workflow and static design demo exist, but deployment and
  browser review are still outstanding;
- the existing `scripts/data.json` is a useful fixture, but it is not evidence that
  JSON should be Felicia's canonical model.
