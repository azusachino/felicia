---
id: "0005"
title: "SQLite Storage and Content-Addressed Media"
status: "accepted"
date: "2026-10-01"
related:
  - "0001"
---

# ADR 0005: SQLite Storage and Content-Addressed Media

## Context

Felicia previously maintained dual persistence backends: a PostgreSQL 18 + PostGIS provider and a local SQLite provider.

This created persistent schema drift, required external database containers, and added maintenance overhead for zero end-user benefit in a single-author local setup. Concurrently, original media storage previously relied on package filenames, creating collisions when two trips used common names like `photo.jpg`.

## Decision

**Standardize on SQLite as the sole persistence engine, and use content-addressed storage for original media.**

1. **Pure SQLite Persistence:**
   - SQLite is the only database engine for Felicia v1. The PostgreSQL provider, Goose tooling, and dual-provider compose setups are retired.
   - Schema migrations are embedded directly into the Go binary and tracked via SQLite's built-in `PRAGMA user_version`.
   - WAL mode and foreign key enforcement are enabled by default.
2. **Content-Addressed Media Storage:**
   - Original media files are stored under `.felicia/media/originals/<sha256>/<filename>`, uniquely identified by the SHA-256 hash of their bytes.
   - Database references link to content hashes, eliminating filename collisions across trips.
3. **Recovery by Copy:**
   - In a single-file SQLite database with immutable blob storage, backup and recovery is simply copying the `.felicia/` directory.

## Consequences

### Positive

- Subtracted 3,341 lines of PostgreSQL code and all external Goose migration dependencies.
- Zero external database services needed to run the studio or run tests.
- Two trips with identical photo names never collide or overwrite each other.

### Negative / Trade-offs

- Multi-user concurrent writes over network connections are not supported; Felicia is local-first.
