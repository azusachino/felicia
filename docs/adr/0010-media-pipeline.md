# ADR 0010: Media Pipeline — Immich Ingest, Object Storage, Public Serving

* **Status:** Accepted
* **Date:** 2026-07-10
* **Decisions:** `felicia:decision:media-pipeline`

## Context
Photos originate in a self-hosted Immich library. The public journal is a static site that cannot
call Immich (it needs a secret API key, and Immich may be offline or private). We need a durable
way to store the *curated* photos felicia actually publishes, serve them without exposing Immich,
and keep re-import safe. Near-term the product is **private/single-author**; public serving is a
product-ready seam, not a near-term need.

## Decision
Two planes, one seam.

1. **Immich is a private *ingest* source, never the public *serving* path.** Only the admin app
   holds the `x-api-key`; nothing public touches Immich.

2. **Copy-on-attach, not whole-library mirror.** When the author attaches an Immich photo to a
   memento, felicia:
   * downloads the original (`GET /assets/{id}/original`),
   * generates web derivatives (a ~2048px display image + a ~400px thumbnail, webp/avif) and
     **strips EXIF — especially GPS** (coordinates already live in the DB for the map; the served
     image must not leak them),
   * content-addresses by `content_hash` (Immich `checksum`) → `object_key =
     photos/{hash}/{variant}.webp`; upload is idempotent (skip if present), so re-import is free
     and de-duped,
   * records `memento_photos{ object_key, content_hash, … }` (columns already exist).

3. **Object store = S3-compatible R2** (MinIO/B2 swappable by config; `internal/objectstore`),
   fronted by a CDN. Content-addressed keys are immutable → long-lived caching. The SSG bakes the
   public image URLs into the static output.

4. **Near-term (private-only):** do **not** build the R2 copy + static serving yet. The private
   admin/reader serves images by proxying Immich through the **keyed backend** (key stays
   server-side, never in the browser). The copy-to-R2 step is added at publish time.

5. **Deferred defaults (revisit at publish):** store **derivatives only** (original stays in
   Immich, re-derivable); serve from a **public bucket + CDN** (signed URLs only if per-journey
   privacy is introduced). Cold-archiving originals to a private prefix is an additive option.

## Consequences
* **Private-first, cheap now:** no object-storage build in the near term; images flow through the
  keyed backend. The `memento_photos.object_key` / `content_hash` columns already accommodate the
  future R2 keys, so the seam is in the schema, not a rewrite.
* **Immich never in the public path:** the static site depends only on R2/CDN; Immich uptime and
  the API key never reach a public visitor.
* **Privacy by construction:** stripping EXIF/GPS on copy prevents the served images from leaking
  locations the map already renders deliberately.
* **Re-import safe:** content-addressing de-dupes and makes re-attach a no-op, consistent with the
  field-scoped importer (design §5).
