# ADR 0008: PostgreSQL 18 & SSG Compiler Model (with SQLite3 Fallback)

* **Status:** Accepted
* **Date:** 2026-07-09
* **Decisions:** `felicia:decision:single-user-local-first-ssg`

## Context
We need to support a single-user workflow where the author uploads data and publishes a personal version of *felicia* as a data-driven static site (similar to `yihong0618/running_page`). We must choose the canonical database architecture for this system, balancing robust spatial features with local portability.

## Decision
We decided to adopt **PostgreSQL 18 + PostGIS** as the canonical database of *felicia* for data storage and spatial operations, utilizing **SQLite3** strictly as a local, offline development and compilation fallback.

Implementation details:
1. **Primary Database (PostgreSQL 18 + PostGIS):**
   * PostgreSQL 18 with the PostGIS extension is the primary storage engine.
   * All spatial joins, coordinates snapping, and route simplifications are designed to leverage native PostGIS database-layer calculations (`ST_Collect`, `ST_Simplify`, `ST_DWithin`).
2. **Local Fallback Database (SQLite3):**
   * SQLite3 is supported strictly as an offline fallback for local development or disconnected builds.
   * To maintain CGO-free portability in SQLite3 mode, spatial processing (simplification and snapping) is executed in Go memory using the `paulmach/orb` library, and saved to SQLite3 as WKB (Well-Known Binary) BLOBs.
3. **Static Compiler Output (`felicia build`):**
   * The compiler queries the database (PostgreSQL 18 by default, SQLite3 if running offline) and exports the public-facing view as a fully static website in `dist/`:
     * Generates static JSON files representing the API route trees (e.g. `/api/v1/journeys.json`).
     * Exports resized and EXIF-stripped image derivatives.
     * Emits the compiled Vite Svelte SPA.
4. **Local Admin UI (`localhost`):**
   * The author runs the Go CLI locally (`felicia admin`). This starts a local web server on `localhost:8080` talking to the configured database (PG18 or local SQLite3 file).
   * The author opens their local browser to curate trips, write essays, and configure stubs.

## Consequences
* **Database Alignment:** The primary architecture utilizes PostgreSQL 18 + PostGIS, ensuring full compatibility with production spatial engines and standard SQL/JSON features.
* **Zero Hosting Cost for Reader:** The public-facing site is fully static and hosted on free CDN tiers (Cloudflare Pages, GitHub Pages).
* **Portability Fallback:** If the user has no running PostgreSQL server, they can fall back to the SQLite3 engine to compile and publish their site locally.
