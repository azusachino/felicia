# ADR 0003: Presentation-Agnostic Contract

* **Status:** Accepted
* **Date:** 2026-07-08
* **Decisions:** `felicia:decision:presentation-agnostic-contract`

## Context
As we designed different frontends (v1 Map Reader, v2 Memento-First Collection, v3 Techo/Paper layout), we risked polluting the database schema with view-specific metadata (e.g. columns like `carousel_index`, `is_landing_featured`, `is_washed_out`, or specific styling parameters). This couples database models directly to ephemeral frontend designs.

## Decision
We decided to enforce a strict **Presentation-Agnostic Contract** between the database/API and the frontend clients:
1. **No view-specific columns:** The database schema only stores semantic data, temporal ordering (`occurred_at`, `seq`), and geographic geometry coordinates.
2. **Semantic ordering:** Sort operations are restricted to semantic attributes (chronological sequence, kind aggregation, or along-route progression).
3. **Views as projections:** The frontends are independent projections of the same underlying canonical dataset. A frontend layout (like the paper-themed v3 layout) must resolve its visual grouping, place clustering, and decoration using the semantic JSON returned by the API, rather than storing layout rules in the database.
4. **Endpoint stability:** API responses are versioned in the URL path (`/api/v1/journeys`, `/api/v1/mementos`). A frontend update must not require a breaking API schema change; breaking API shapes must deploy as an additive `/api/v2/...` route.

## Consequences
* **Frontend Flexibility:** We can build, test, and swap multiple frontends (e.g. Map Libre GL layouts vs. completely static HTML spreads) without executing database migrations.
* **API Invariant:** The Go API is simplified to clean query serialization, avoiding UI state management.
* **Slight client complexity:** Frontend clients must compute spatial groupings, date headers, and carousel lists on the fly.
