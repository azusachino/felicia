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
path = ".felicia/felicia.sqlite"

[server]
port = "8080"
```

Environment values override file values:

```bash
FELICIA_DATABASE_DRIVER=postgres \
FELICIA_DATABASE_DSN='postgres://postgres:password@localhost:5432/felicia?sslmode=disable' \
go run ./apps/felicia-server/cmd/api
```

The runtime never silently changes providers when the selected provider is
missing required configuration.

## Local commands

```bash
make admin               # authoring GUI, same database as make dev
make dev                 # API with SQLite at .felicia/felicia.sqlite
make dev-sqlite          # explicit SQLite API
make dev-postgres        # PostgreSQL/PostGIS stack, migration, seed, web app
make migrate             # PostgreSQL migrations only
```

`make dev` and `make admin` share `.felicia/felicia.sqlite` on purpose: authoring
in the GUI and then serving the public reader should read one journal, not two.
`.felicia/` is gitignored, which the repo-root default it replaced was not — the
authored journal is the one artifact
[ADR-0025](../adr/0025-static-and-self-hosted-modes.md) keeps on the machine, so
it must not sit on a committable path.

As when the default filename last changed
([ADR-0021](../adr/0021-runtime-configuration-and-database-modes.md)), user data
is not renamed or copied automatically. An existing repo-root `felicia.db` is
still opened by setting `DATABASE_PATH=felicia.db`, or move it once:

```bash
mkdir -p .felicia && mv felicia.db .felicia/felicia.sqlite
```

## Local filesystem convention

Felicia's private development state lives below `.felicia/`:

- `.felicia/felicia.sqlite` is the default local database.
- `.felicia/workspaces/<slug>/` contains generated journey workspaces.
- `.felicia/media/` and `.felicia/site/` are reserved for shared local assets and previews.

New local paths must not introduce a `local-*` child prefix. Feature names such as
`local-journey` may remain in historical documentation or test fixtures when they
describe an older contract.

SQLite applies its embedded schema when the provider opens the database. The
file provider enables foreign keys, WAL mode, a busy timeout, and a bounded
single-writer connection configuration.

## Tests

```bash
make test-sqlite
make test-workflow

FELICIA_TEST_DATABASE_DSN='postgres://postgres:password@localhost:5432/felicia?sslmode=disable' \
  make test-postgres

FELICIA_TEST_DATABASE_DSN='postgres://postgres:password@localhost:5432/felicia?sslmode=disable' \
  make test-workflow-postgres

# Preferred: create, migrate, and drop a unique database per run.
FELICIA_TEST_POSTGRES_ADMIN_DSN='postgres://postgres:password@localhost:5432/postgres?sslmode=disable' \
  make test-workflow-postgres
```

PostgreSQL tests require a test DSN rather than ordinary `DATABASE_DSN`. The
preferred workflow uses `FELICIA_TEST_POSTGRES_ADMIN_DSN` to create, migrate,
and drop a unique database per run. The direct `FELICIA_TEST_DATABASE_DSN`
form remains available when CI or another runner owns database lifecycle; that
database must be disposable because integration tests clean tables before
exercising provider behavior. Tests also verify that Goose migrations have
reached version 8 or later.

The repository contract suite runs against both providers. Provider-specific
features, such as PostGIS route acceleration, remain in PostgreSQL-specific
tests.
