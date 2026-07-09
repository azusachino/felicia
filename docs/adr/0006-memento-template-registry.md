# ADR 0006: Declarative Memento Template Registry

* **Status:** Accepted
* **Date:** 2026-07-09
* **Decisions:** `felicia:decision:memento-template-registry`

## Context
As we expand *felicia*, we will need to support a wide range of travel mementos (Japanese train tickets, goshuin stamps, restaurant receipts, event tickets, concert wristbands, goods). If we hardcode form widgets, backend validation schemas, and database columns for every new kind of memento, the codebase will suffer from code duplication, db migrations, and frontend drift.

## Decision
We decided to make memento kinds **declarative templates declared as data** (YAML files loaded at runtime) rather than writing custom code for each kind.

Properties of the registry:
1. **Unified Schema Declaration:** Each kind is declared in a YAML file (e.g. `kinds/transit.yaml`, `kinds/live.yaml`) containing:
   * `kind`: Name (e.g. `transit`).
   * `anchor`: Geometry binding (`point` which requires 1 coordinate field, or `edge` which requires 2 coordinate fields).
   * `stub`: Frontend renderer component identifier.
   * `fields`: Array of properties `{ name, type, required, translatable }`.
2. **Three-Consumer Drive:** A single YAML declaration drives:
   * **The Admin Form:** Generated dynamically from the field types (no custom form code per kind).
   * **Domain Validation:** A pure Go function (`Validate(tpl, kind_data)`) that validates incoming JSON payloads (enforces closed field sets, type correctness, currency formats, and coordinate requirements).
   * **Stub Rendering:** The frontend loads the `stub` component and resolves translatable keys.
3. **Flexible JSONB Column:** All kind-specific non-translatable fields are stored in a single Postgres `kind_data jsonb` column (or SQLite `kind_data TEXT` column). The database schema does not change when new kinds are added.

## Consequences
* Adding a new travel stub requires writing a simple YAML file and a Svelte rendering component. No database DDL migrations, Go compilation updates, or custom editor forms are required.
* Ensures strict validation: incoming ingest payloads must match the template exactly before saving.
* Avoids the "Generic ETL DSL" trap by restricting the template to fields, anchors, and refs. Custom ETL processing logic remains in Go.
