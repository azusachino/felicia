# ADR 0001: Personal-Now, Product-Ready Direction

* **Status:** Accepted
* **Date:** 2026-06-12
* **Decisions:** `felicia:decision:personal-now-product-ready`

## Context
When launching the *felicia* project, we faced a choice between building a multi-user SaaS travel tracker (competing directly with platforms like Polarsteps) or building a highly tailored, beautiful personal map journal for a single author. Building a full SaaS application immediately introduces heavy overhead (user auth, billing, multi-tenancy, quotas) which can distract from proving the core user experience ("the map is the index, the mementos are the stories").

## Decision
We decided to build the **personal artifact first** — a beautiful, single-author journal tailored for the creator. However, we must implement all software boundaries as **swappable seams** so that a transition to a multi-user or SaaS product in the future is purely additive rather than a code rewrite.

Key boundaries/hedges to enforce:
1. **Sources behind interfaces:** GPS tracks and photo sync libraries are defined as interfaces (e.g. `TrackSource`, `PhotoSource`) rather than coupling the core logic to Dawarich or Immich APIs.
2. **Trigger-agnostic importer:** The ingestion engine is a Go library call, completely isolated from CLI flag parsing, allowing it to be triggered by local CLI, cron jobs, or HTTP webhook handlers later.
3. **Single journal/account root:** All databases reference a parent `journal` entity (even when only one row exists). Adding multi-tenancy later is a matter of adding a `user_id` or `tenant_id` to the `journal` table, without rewriting the child tables (`journeys`, `mementos`, `photos`).
4. **Privacy invariant at the boundary:** Raw high-precision GPS coords are kept in the database; public-facing route lines are simplified, coordinates are rounded, and public images are resized and EXIF-stripped before upload to object storage.

## Consequences
* **Immediate benefit:** We can build a fast, high-quality, zero-cost personal site without spending months on SaaS auth/billing infrastructure.
* **Architecture overhead:** Go structures require clean interface definitions, and the database schema includes a slightly nested layout (via the `journal` root) to preserve future SaaS capability.
* Multi-user concepts (collaborators, teams, private sharing links) are deferred as non-goals for the initial implementation but can be introduced downstream behind the same schemas.
