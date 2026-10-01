---
id: "0004"
title: "Authorship Protection and the Journey Ingest Seam"
status: "accepted"
date: "2026-10-01"
related:
  - "0001"
  - "0002"
---

# ADR 0004: Authorship Protection and the Journey Ingest Seam

## Context

An automated intake pipeline seeds trips by importing tracks from Dawarich and photos from Immich. However, an author spends significant time writing essays, fine-tuning captions, selecting representative photos, and correcting place names.

Early importer implementations risked clobbering manual edits whenever a trip was refreshed or re-imported.

## Decision

**Enforce a strict architectural boundary between ingest writes and authoring writes.**

1. **Auto-Ingest Seeds; It Never Clobbers:**
   - The initial import creates records and populates automated fields (timestamps, raw route, photo dimensions).
   - Re-import updates only unauthored fields and never overwrites fields claimed by the author.
2. **Server Owns the Authored Mask:**
   - The server maintains `authored_fields` masks on journeys, mementos, and stop candidates.
   - When an author edits a field in the Admin GUI, the server automatically adds that field to the authored mask; clients cannot blank or clear the mask.
3. **Atomic Concurrency & Revisions:**
   - Mementos and stop candidates use revision numbers (`expected_revision`) to prevent write conflicts.
   - Journey ingest is atomic, preventing race conditions between concurrent imports and manual edits.

## Consequences

### Positive

- Re-importing a journey to bring in late photos or refined GPS points is completely safe.
- Author time and creative investment are permanently protected.

### Negative / Trade-offs

- Ingest and authoring logic must be maintained as distinct write paths in the domain and persistence layers.
