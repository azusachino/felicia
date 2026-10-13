---
id: "0009"
title: "Retire Live Photo Video Publishing (Stills Everywhere)"
status: "accepted"
date: "2026-10-13"
related:
  - "0005"
  - "0008"
---

# ADR 0009: Retire Live Photo Video Publishing (Stills Everywhere)

## Context

[ADR 0008](0008-pivot-to-hyoban-spatial-journal.md) section 5 required pairing Apple
Live Photos during ingestion, and (in its consequences) a video transcoding pipeline so
the reader could silently hover-play motion alongside each still. That pipeline shipped
in PR #174: linked MOV/MP4 companions are installed at intake, and publication transcodes
them through `ffmpeg` into bounded, metadata-stripped H.264/AAC MP4 derivatives.

In practice the pipeline was over-built for its payoff:

- It made `ffmpeg` (with `libx264` and AAC) a hard runtime dependency of every
  publication, and a CI dependency for the test suite — a heavyweight requirement
  in exchange for a single feature.
- The motion effect it bought (silent hover/loop playback) was cosmetic on top of
  the still, which already carries the story.
- The owner reviewed the merged result on 2026-10-13 and judged the trade not worth
  the maintenance surface.

## Decision

**Publication is stills-only. Live Photo video motion publishing is retired.**

1. **Companion storage at intake is retained.** Ingestion still detects a photo's
   MOV/MP4 companion, installs it into the content-addressed blob store, and records
   the `video_key` linkage in the database ([ADR 0005](0005-sqlite-storage-and-content-addressed-media.md)).
   Companions stay linked private originals — nothing is deleted and re-ingestion is
   unchanged.
2. **Companions are never published.** The static compiler no longer opens, transcodes,
   or writes companion videos. `video_key` is removed from the `StaticPhoto` public
   projection and from the reader contract (`felicia-model` `contracts/public.ts`,
   `data.ts`). Already-published JSON that still contains a `video_key` is simply
   ignored by the reader; this is a breaking change to the unreleased public contract
   and is acceptable while Felicia is unreleased.
3. **The reader renders stills everywhere.** The video rendering branches (Live Photo
   hover/loop playback, live badges, lightbox motion toggle) are removed from the
   playback card, stage, journal stream, lightbox, and adapter.
4. **`ffmpeg` leaves the product and CI.** The transcode path and its test suite are
   deleted; the backend CI lane no longer installs `ffmpeg`. Ingest-linking tests
   revert to plain fake bytes — playable fixtures are no longer needed.

This decision supersedes the derivative requirement of [ADR 0008](0008-pivot-to-hyoban-spatial-journal.md)
section 5 and its "ingestion and derivative pipelines must now handle video transcoding"
consequence. Ingestion pairing itself (storage and linkage) remains as ADR 0008
specified.

Two related owner calls, confirmed 2026-10-13 and left intact:

- **Leg modes remain an invested direction** (ADR 0008 section 5 `leg_mode`); no
  retirement.
- **The single-point departure-area weather approximation stays** (trip-folder
  contract); per-day weather locations remain rejected.

## Consequences

### Positive

- Stills render everywhere, from one image-only pipeline: no video branches in the
  reader, no projection split.
- `ffmpeg` is no longer a publication runtime or CI dependency; compiles are faster
  and environment-independent.
- The public contract shrinks to what the reader actually renders.

### Negative / Trade-offs

- Published sites never show Live Photo motion; the still is the whole picture.
- Old published artifact JSON containing `video_key` is ignored by the reader
  (breaking change while unreleased — noted, no migration).
- Reinstating video motion publishing later — as a new ADR with its own encoding
  trade-offs — is required before any companion is published again.
