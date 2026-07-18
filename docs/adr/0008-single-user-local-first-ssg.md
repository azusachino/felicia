---
id: "0008"
title: "PostgreSQL 18 & SSG Compiler Model"
status: "accepted"
date: "2026-07-09"
decisions:
  - "felicia:decision:pg18-ssg-compiler"
related: ["0017", "0025", "0027"]
supersedes: []
---

# ADR 0008: PostgreSQL 18 & SSG Compiler Model

> **SUPERSEDED (storage): see [ADR-0017](0017-sqlite-first-storage.md).**
> The "PostgreSQL 18 + PostGIS as the sole database engine, no SQLite fallback"
> decision below was reversed on 2026-07-14: SQLite is now the default local
> provider and PostgreSQL/PostGIS is optional ([ADR-0017](0017-sqlite-first-storage.md),
> reaffirmed by [ADR-0027](0027-provider-matrix-and-application-composition.md)).
> The SSG compiler model and local admin workflow in this ADR still stand
> ([ADR-0025](0025-static-and-self-hosted-modes.md)).

## Context

We need to support a single-user workflow where the author uploads data and publishes a personal version of _felicia_ as a data-driven static site (similar to `yihong0618/running_page`). We must choose the canonical database architecture for this system, balancing robust spatial features with local portability.

## Decision

We decided to adopt **PostgreSQL 18 + PostGIS** as the sole database engine of _felicia_ for data storage and spatial operations. There is no fallback database (such as SQLite3). The static site compiler and local admin server connect directly to PostgreSQL 18 + PostGIS.

Implementation details:

1. **Database Engine (PostgreSQL 18 + PostGIS):**
   - PostgreSQL 18 with the PostGIS extension is the sole database engine.
   - All spatial joins, coordinates snapping, and route simplifications leverage native PostGIS database-layer calculations (`ST_Collect`, `ST_Simplify`, `ST_DWithin`).
2. **Static Compiler Output (`felicia build`):**
   - The compiler queries the PostgreSQL 18 + PostGIS database and exports the public-facing view as a fully static website in `dist/`:
     - Generates static JSON files representing the API route trees (e.g. `/api/v1/journeys.json`).
     - Exports resized and EXIF-stripped image derivatives.
     - Emits the compiled Vite Svelte SPA.
3. **Local Admin UI (`localhost`):**
   - The author runs the Go CLI locally (`felicia admin`). This starts a local web server on `localhost:8080` talking to the configured PostgreSQL 18 database.
   - The author opens their local browser to curate trips, write essays, and configure stubs.

## Consequences

- **Database Alignment:** The system utilizes PostgreSQL 18 + PostGIS exclusively, ensuring full compatibility with production spatial engines and standard SQL/JSON features.
- **Zero Hosting Cost for Reader:** The public-facing site is fully static and hosted on free CDN tiers (Cloudflare Pages, GitHub Pages).
- **Simplified Core:** By avoiding a secondary database engine (SQLite3), we eliminate duplicate DDL migrations, repository translation layers, and custom Go-memory spatial logic.
