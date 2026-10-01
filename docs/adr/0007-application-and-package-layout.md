---
id: "0007"
title: "Application and Package Layout"
status: "accepted"
date: "2026-10-01"
related:
  - "0001"
  - "0003"
  - "0005"
---

# ADR 0007: Application and Package Layout

## Context

As Felicia evolved, the codebase accumulated speculative layers, resulting in 6 Go modules, 3 frontend applications, and 5 monorepo packages.

Micro-packages like `packages/felicia-renderers` (25 LOC) and `packages/felicia-runtime` (80 LOC) added monorepo linking friction with no genuine consumer separation. Similarly, maintaining two nearly identical reader apps (`felicia-web` and `felicia-public-site`) was unnecessary.

## Decision

**Consolidate the codebase into focused, high-cohesion applications and packages.**

1. **Go Workspace (`go.work`):**
   - `apps/felicia-core`: Pure domain models, value objects, and repository ports (no I/O).
   - `apps/felicia-runtime`: Use-case workflows, intake coordination, and business logic.
   - `apps/felicia-providers`: Concrete I/O adapters (SQLite persistence, Dawarich, Immich, local filesystem).
   - `apps/felicia-publication`: Public projection compiler, sanitization, and artifact writer.
   - `apps/felicia-server`: Local HTTP REST API server powering the admin studio.
   - `apps/felicia-cli`: Canonical command-line tool for packaging, importing, and static compilation.

2. **Frontend Applications:**
   - `apps/felicia-admin`: The authoring studio (Svelte 5 SPA).
   - `apps/felicia-public-site`: The public reader shell (Svelte 5 + MapLibre). Can connect to live `felicia-server` via `VITE_API_BASE` or load static JSON artifacts.

3. **Frontend Packages:**
   - `packages/felicia-model`: Headless domain contracts, DTO types, and static i18n catalogs (pure TypeScript, zero Svelte/DOM dependencies).
   - `packages/felicia-reader`: The complete reader UI library, including Atlas theme and memento components.

## Consequences

### Positive

- Strict dependency flow: pure domain at the bottom, ports implemented by providers, consumed by server and CLI.
- Frontends collapsed from 3 apps and 5 packages down to 2 apps and 2 packages.
- Zero cyclic dependencies, fast builds, and clear conceptual boundaries.

### Negative / Trade-offs

- Reorganizing historical code requires updating import paths and boundary conformance tests.
