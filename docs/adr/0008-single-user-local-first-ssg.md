# ADR 0008: Single-User Local-First & SSG Compiler Model

* **Status:** Accepted
* **Date:** 2026-07-09
* **Decisions:** `felicia:decision:single-user-local-first-ssg`

## Context
Running a persistent server and a production-grade database (Go + PostgreSQL 18 + PostGIS + Traefik) on a cloud server/VPS requires continuous hosting costs, backup monitoring, and exposes write APIs to the public web. For personal journals where there is exactly one author and the public audience is read-only, this operational overhead is unnecessary.

## Decision
We decided to support a **Single-User Local-First & Static-Site Generation (SSG)** compiler mode as our primary personal deployment model, modeled on the pattern of `yihong0618/running_page`.

Implementation details:
1. **CGO-Free SQLite Database:**
   * In local mode, the persistent PostgreSQL database is replaced by a local SQLite file (`felicia.db`).
   * The system utilizes a pure-Go SQLite driver (`ncruces/go-sqlite3` or `modernc.org/sqlite`), keeping the compilation CGO-free.
   * Spatial processing (Dawarich track simplification, coordinate snaps) is executed in Go memory using the `paulmach/orb` library and saved as standard WKB BLOBs in SQLite.
2. **Local Admin Dashboard (`localhost`):**
   * The author runs the Go CLI locally (`felicia admin`). This starts a local web server on `localhost:8080` talking to the SQLite file.
   * The author opens their local browser to curate trips, write essays, and configure stubs.
3. **Static compiler (`felicia build`):**
   * A build script queries the SQLite database and exports the public-facing view as a fully static website in `dist/`:
     * Generates static JSON files representing the API route trees (e.g. `/api/v1/journeys.json`).
     * Exports resized and EXIF-stripped image derivatives.
     * Emits the compiled Vite Svelte SPA.
4. **Automated CI/CD (GitHub Actions / Cron):**
   * Ingestion runs automatically via GitHub Actions (on a schedule). It triggers the Go CLI to fetch tracks from Dawarich API and photos from Immich API, runs the static build, commits metadata back to Git, and deploys it to a static host (GitHub Pages or Cloudflare Pages) for free.

## Consequences
* **Zero Hosting Cost:** The public-facing site is fully static and hosted on free CDN tiers (Cloudflare Pages, GitHub Pages).
* **Absolute Security:** The public site is read-only. Write capabilities and the admin panel are only accessible locally on the author's machine.
* **Git Versioning:** All authored essays and metadata are checked into a private Git repository, serving as a clean, versioned backup.
* **Code Reusability:** The Svelte SPA frontend remains completely unchanged; it fetches `/api/v1/...` static JSON files instead of a live API server.
