# ADR 0021: Runtime Configuration and Database Modes

- Status: Accepted
- Date: 2026-07-14

## Context

Felicia supports SQLite for local-first, single-user operation and PostgreSQL
for deployment and provider-parity coverage. The runtime needs a predictable
configuration contract that works for a developer starting the API directly,
for scripts, and for CI. Database selection must be explicit when PostgreSQL
is requested; an invalid PostgreSQL configuration must not silently fall back
to SQLite.

## Decision

Felicia uses the following configuration precedence, from lowest to highest:

1. Built-in defaults.
2. An optional `felicia.toml` file.
3. `FELICIA_*` environment variables.
4. Legacy unprefixed environment variables, only when the corresponding
   `FELICIA_*` variable is absent.

`FELICIA_CONFIG` selects an alternate configuration file. A missing default
`felicia.toml` is allowed; a configured file that exists but is malformed is
an error. Secrets such as database DSNs and upstream API keys are supplied by
the environment and are not committed to configuration fixtures.

SQLite is the default API and local-development provider, using `felicia.db`
as its default path. PostgreSQL is selected explicitly with
`DATABASE_DRIVER=postgres` or `FELICIA_DATABASE_DRIVER=postgres` and requires
a configured DSN. The runtime never changes providers implicitly when a
provider's required configuration is missing.

The normal local commands are SQLite-first:

- `make dev` and `make test-workflow` use SQLite.
- `make dev-postgres` and `make test-postgres` explicitly exercise PostgreSQL.
- `make migrate` remains PostgreSQL-only because the current Goose migration
  set targets PostgreSQL/PostGIS.

Both providers implement the same repository contract. Provider-specific
features, including PostGIS acceleration, remain outside that contract.

## Consequences

New local installations require no database service. Configuration can be
checked in without embedding secrets, while environment variables can safely
override local defaults and file values in CI or deployment. Existing
`felicia.sqlite` databases remain usable when their path is explicitly set;
the default filename change does not rename or copy user data automatically.

The Makefile must expose separate SQLite and PostgreSQL targets so the active
backend is visible from the command. Tests must cover configuration merging
and run backend conformance coverage independently for both providers.
