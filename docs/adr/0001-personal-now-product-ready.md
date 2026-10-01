---
id: "0001"
title: "Personal-Now, Product-Ready Direction"
status: "accepted"
date: "2026-10-01"
related: []
---

# ADR 0001: Personal-Now, Product-Ready Direction

## Context

Felicia began with two competing visions: a full-blown multi-tenant travel SaaS platform and a personal travel journal modeled on Aaron's Waypoints (*liuaaron.com*).

Building for an imaginary multi-tenant SaaS audience created speculative complexity: dual-database abstraction layers, complex auth models, and premature workflow engines, before the author had a single trip reliably curated and published.

## Decision

**Build Felicia first as a personal, local-first studio that produces a versioned static public reader.**

1. **Personal Now:** The immediate user is the author. All requirements are grounded in real trips, actual GPS tracks, and physical mementos.
2. **Product Ready:** The architecture maintains clean interfaces, typed domain models, and strict privacy boundaries so it remains robust, verifiable, and open-source friendly.
3. **Studio + Artifact:** The authoring environment is a private local studio (`felicia-admin` backed by `felicia-server`), while the public reader is a static, pre-rendered artifact deployed to static hosting (GitHub Pages / Cloudflare Pages).

## Consequences

### Positive

- Clarifies the product boundary: no user accounts, passwords, or multi-tenant database partitioning.
- Eliminates cloud hosting costs and runtime database exposure on the public internet.
- Maximizes reading speed and resilience: the public site has zero backend runtime dependencies.

### Negative / Trade-offs

- Publishing is an explicit compilation step rather than instant collaborative editing.
