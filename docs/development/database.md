# Database development

Felicia supports two persistence providers behind the same repository contract:

- SQLite is the default for local development and single-user operation.
- PostgreSQL/PostGIS is available for deployment and provider-parity coverage.

## Configuration precedence

Runtime configuration is merged in this order:

1. Built-in defaults.
2. Optional `felicia.toml`.
3. `FELICIA_*` environment variables.
4. Legacy unprefixed environment variables, when the prefixed variable is absent.

Use `FELICIA_CONFIG` to select a different TOML file. A missing default
`felicia.toml` is allowed; a missing explicitly selected file is an error.
Secrets such as DSNs and upstream API keys belong in the environment.

Example local configuration:

```toml
[database]
driver = "sqlite"
path = "felicia.db"

[server]
port = "8080"
```

Environment values override file values:

```bash
FELICIA_DATABASE_DRIVER=postgres \
FELICIA_DATABASE_DSN='postgres://postgres:password@localhost:5432/felicia?sslmode=disable' \
go run ./apps/apiserver/cmd/api
```

The runtime never silently changes providers when the selected provider is
missing required configuration.

## Local commands

```bash
make dev                 # API with SQLite at felicia.db
make dev-sqlite          # explicit SQLite API
make dev-postgres        # PostgreSQL/PostGIS stack, migration, seed, web app
make migrate             # PostgreSQL migrations only
```

SQLite applies its embedded schema when the provider opens the database. The
file provider enables foreign keys, WAL mode, a busy timeout, and a bounded
single-writer connection configuration. Existing `felicia.sqlite` files can
still be opened by setting `DATABASE_PATH` explicitly.

## Tests

```bash
make test-sqlite
make test-workflow

FELICIA_TEST_DATABASE_DSN='postgres://postgres:password@localhost:5432/felicia?sslmode=disable' \
  make test-postgres

FELICIA_TEST_DATABASE_DSN='postgres://postgres:password@localhost:5432/felicia?sslmode=disable' \
  make test-workflow-postgres
```

PostgreSQL tests require `FELICIA_TEST_DATABASE_DSN`, not ordinary
`DATABASE_DSN`. The test database must be disposable: integration tests clean
tables before exercising provider behavior. Tests also verify that Goose
migrations have reached version 8 or later.

The repository contract suite runs against both providers. Provider-specific
features, such as PostGIS route acceleration, remain in PostgreSQL-specific
tests.
