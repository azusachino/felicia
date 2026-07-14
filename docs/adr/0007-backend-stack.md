# ADR 0007: Backend Stack (PostgreSQL 18 & Go)

- **Status:** Accepted
- **Date:** 2026-07-08
- **Decisions:** `felicia:decision:backend-stack`

## Context

Choosing the backend stack for _felicia_ requires balancing standard, stable library ecosystems with performance and build portability. Furthermore, the database choice needs to align with modern standards for spatial geometry and JSON manipulation.

## Decision

We decided to build the backend in **Go** targeting **PostgreSQL 18** as our primary relational database, supported by specific production-grade libraries.

The libraries and tools selected:

1. **HTTP Router (chi):** A lightweight, stdlib-compatible router that keeps handlers as standard `http.HandlerFunc`, easing unit testing and permitting Edge JWT verification middlewares.
2. **Postgres Driver (pgx v5):** The modern Go standard for PostgreSQL, supporting native `jsonb`, `timestamptz`, and performance optimizations like binary copy.
3. **Query Layer (sqlc):** Generates compile-time type-safe Go code from raw SQL queries. Ensures query correctness during compilation and provides explicit, reviewable SQL in pull requests.
4. **Go Geometry (paulmach/orb):** A pure-Go geometry library used for light geometry manipulation, Douglas–Peucker simplification, and WKB/GeoJSON encoding on the application side. All heavy spatial computations stay in PostGIS SQL.
5. **Object Storage (minio-go v7):** A lightweight, S3-compatible client that allows pointing at Cloudflare R2, MinIO, or Backblaze B2 via simple configuration.
6. **Migrations (goose):** Simple, SQL-based database migrations.

### PostgreSQL 18 Exploitation:

We explicitly target PostgreSQL 18 to leverage its modern features:

- **SQL/JSON Standard:** Queries utilize standard functions (`JSON_VALUE`, `JSON_EXISTS`, `JSON_QUERY`) over legacy PG operators (`#>`, `->>`), aligning our codebase with standard SQL.
- **Advanced MERGE:** Employs the `MERGE` statement for field-scoped upserts (implementing the "no-clobber" rule in a single atomic transaction).
- **Sequential UUIDv7:** Native primary keys use sequential UUIDv7 to optimize B-Tree index layout and avoid fragmentation.
- **Parallel Index Scanning:** Parallel execution plans speed up heavy spatial queries (`ST_Collect`, `ST_DWithin`).

## Consequences

- **CGO-Free builds:** By using pure-Go libraries (like `orb` and avoiding C-based GEOS or libvips), the backend builds into tiny, secure distroless/scratch Docker containers.
- **High query safety:** Type-safe Go structs from sqlc prevent runtime database scanner errors.
- **Database performance:** Transactions are atomic and fast, shifting sorting and proximity evaluations to the PostGIS query planner.
