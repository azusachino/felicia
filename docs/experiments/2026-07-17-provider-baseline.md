---
title: "Provider Baseline: SQLite and PostgreSQL"
status: "partial"
date: "2026-07-17"
experiment: "E1/E2"
---

# Provider Baseline: SQLite and PostgreSQL

## Inputs and harness

- Workflow script: [`scripts/test_journey_workflow.py`](../../scripts/test_journey_workflow.py)
- JSON fixture: [`scripts/data.json`](../../scripts/data.json)
- JSON schema: [`scripts/data.schema.json`](../../scripts/data.schema.json)
- GPX fixture: [`scripts/tracks/narita-express.gpx`](../../scripts/tracks/narita-express.gpx)
- Compose services: [`ops/compose.yaml`](../../ops/compose.yaml)
- Shared public origin: [`ops/Caddyfile`](../../ops/Caddyfile)

The workflow is runnable through the repository's `uv` wrapper:

```bash
make test-workflow
```

## SQLite result

Command:

```bash
make test-workflow
```

Result on 2026-07-17:

```text
full journey workflow passed
```

This proves the current SQLite API path can create a journal, create a journey,
save a draft memento, publish it with a revision check, attach media metadata,
and expose the published memento through the public read API.

It does not yet prove GPX parsing, local blob storage, static publishing, or a
normalized route-point schema.

## PostgreSQL result

Podman and `podman-compose` were available. `make db-up` started or found the
repository's PostgreSQL/Valkey development services, but the baseline could not
be run because the host port `5432` resolved to an existing PostgreSQL instance
whose `postgres` password did not match the Compose default:

```text
FATAL: password authentication failed for user "postgres"
```

This is an environment collision, not evidence against the PostgreSQL provider.
The next run must use an isolated Compose project and an explicit host port, or a
known disposable DSN. Do not change the schema or provider based on this failure.

## Current conclusion

- **SQLite:** retain as the first runnable baseline.
- **PostgreSQL:** experiment blocked by environment isolation; rerun required.
- **JSON fixture:** retain as test input and seed material only, not canonical-model evidence.
- **GPX fixture:** exists and should become the input for E3 route normalization.
- **Compose/Caddy:** useful self-hosted integration harness, but not yet a static
  GitHub Pages publisher.
