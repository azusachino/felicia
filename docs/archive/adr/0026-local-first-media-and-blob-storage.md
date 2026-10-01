---
id: "0026"
title: "Local-First Media with Pluggable Blob Storage"
status: "accepted"
date: "2026-07-17"
decisions:
  - "Require a local filesystem media backend for development and simple self-hosting."
  - "Keep binary media outside the database and expose provider-neutral blob operations."
  - "Publish only content-addressed, privacy-safe derivatives; keep originals private."
related:
  - "0025"
  - "0023"
  - "0024"
supersedes: []
---

# ADR 0026: Local-First Media with Pluggable Blob Storage

## Context

Felicia needs to store photos and eventually videos during local development,
self-hosting, and static publication. Requiring S3/R2 before the first useful
workflow adds operational cost and makes the OSS project harder to try. Storing
binary content in SQLite or PostgreSQL would make backups, static builds, and
provider changes unnecessarily difficult.

The public site and private authoring system also have different media needs:
original uploads must remain private, while the public site needs resized and
metadata-stripped derivatives.

## Decision

Felicia defines a provider-neutral blob-storage port separate from the database
metadata repository:

```text
BlobStore
├── local filesystem
└── S3-compatible storage (R2, MinIO, B2, or equivalent)
```

The local filesystem backend is the default for development and the minimum
supported self-hosted installation. The database stores media metadata and object
keys, never the binary payload.

The local layout is content-addressed:

```text
var/media/
├── originals/<sha256>/<filename>
└── derivatives/<sha256>/
    ├── preview.webp
    ├── medium.webp
    └── video.mp4
```

The exact directory layout is an implementation detail; the stable contract is
the content hash, media metadata, derivative kind, and provider object key.

The media lifecycle is:

```text
upload → private original → validate/process → public derivative → publish
```

Public derivatives must be resized as appropriate and stripped of EXIF metadata,
including embedded GPS. Public object paths are content-addressed so a published
HTML page can reference immutable URLs without cache invalidation.

The publisher resolves media through configuration:

```text
/media/<hash>.webp
https://cdn.example.example/felicia/<hash>.webp
```

The database stores the logical asset and derivative metadata; it must not store a
machine-local filesystem path in a public projection.

## Consequences

- `make dev` and a fresh self-hosted install work without S3 credentials.
- Switching from local files to R2/MinIO/B2 does not change the domain model.
- Static builds can copy public derivatives into `dist/media/` or reference a CDN.
- Upload processing needs explicit status and failure handling before publication.
- Backups must cover both database metadata and the media root or object bucket.
- Video support can begin with validated common files and defer adaptive streaming.
