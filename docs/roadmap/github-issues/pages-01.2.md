Replace the fixture-only Pages builder with a SQLite-backed compiler.

## Scope

- Read through provider-neutral runtime ports.
- Generate the existing `.json` projection and SPA artifact.
- Exclude unpublished/private records.
- Remove direct PostgreSQL construction from the static path.

## Acceptance

A clean SQLite fixture produces a complete artifact without JSON seed input, API, PostgreSQL, or Valkey.
