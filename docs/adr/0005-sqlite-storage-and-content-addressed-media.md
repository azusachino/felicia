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
   - Felicia is an unreleased, single-author studio; breaking changes during active development are acceptable without migration framework overhead. The database initializes its clean v1 relational schema directly from embedded `schema.sql`.
   - WAL mode and foreign key enforcement are enabled by default.
2. **Content-Addressed Media Storage:**
   - Original media bytes live behind the core `BlobStore` port. The local filesystem provider stores keys beneath the configured private media root with owner-only file permissions.
   - Uploads derive their key from the SHA-256 digest (`media/<sha256>/original.<ext>`); the client filename never controls a filesystem path. Database references retain the content hash, eliminating filename collisions across trips.
   - The admin upload accepts JPEG, PNG, and WebP images up to 20 MiB. HEIC/HEIF remains an explicit conversion step.
   - Public compilation reads the private original and emits a resized, re-encoded derivative with image metadata stripped. Original bytes are never copied into the public artifact.
3. **Recovery by Copy:**
   - In a single-file SQLite database with immutable blob storage, backup and recovery is simply copying the `.felicia/` directory.

## Consequences

### Positive

- Subtracted 3,341 lines of PostgreSQL code and all external Goose migration dependencies.
- Zero external database services needed to run the studio or run tests.
- Two trips with identical photo names never collide or overwrite each other.
- The same upload and preview path works against a provider-neutral blob port; the local provider remains the v1 implementation.
- Public compilation continues to enforce the privacy boundary even when originals are uploaded directly from the admin UI.

### Negative / Trade-offs

- Multi-user concurrent writes over network connections are not supported; Felicia is local-first.
