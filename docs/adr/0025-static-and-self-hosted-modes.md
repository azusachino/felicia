---
id: "0025"
title: "Static and Self-Hosted Product Modes"
status: "accepted"
date: "2026-07-17"
decisions:
  - "Treat the static GitHub Pages publication and self-hosted server as first-class Felicia modes."
  - "Share one canonical domain model, runtime, importer, template registry, and publication contract between modes."
  - "Keep administration, imports, drafts, and uploads in the server mode; publish only static projections to GitHub Pages."
related:
  - "0022"
  - "0023"
supersedes: []
---

# ADR 0025: Static and Self-Hosted Product Modes

## Context

Felicia has two legitimate product destinations:

1. a simple public journal deployed to GitHub Pages; and
2. a self-hosted server that provides the private admin surface, imports, drafts,
   uploads, previews, and publishing.

GitHub Pages cannot run Felicia's Go API, database, admin application, or upload
endpoint. A self-hosted installation should not require GitHub, a managed database,
or a companion mobile app. Treating one as a fallback for the other would either
make the static release impossible or make the server needlessly dependent on a
hosting platform.

## Decision

Felicia supports two first-class modes:

```text
server mode → author, import, review, preview, publish
static mode → build and serve the published reader
```

Both modes share the same:

- canonical domain model and validation;
- runtime use cases;
- source adapters and package import;
- memento template registry;
- public projection and publication boundary.

The server is the authoring system. Static mode consumes a published projection
and has no write capability. A static build contains HTML, JavaScript, CSS, JSON
read projections, and public media derivatives.

The primary release workflow is:

```text
Felicia server → publish/build → static artifact → GitHub Pages or any static host
```

Self-hosting remains a complete alternative:

```text
Felicia server → private admin + public reader
```

The static artifact must not contain drafts, private originals, raw source records,
database files, secrets, or unrounded private geometry.

## Consequences

- GitHub Pages is a supported product target, not merely a demo deployment.
- Server mode remains useful without GitHub or an external network service.
- The public reader can be tested from a static artifact independently of the API.
- The publisher must consume provider-neutral repository and media ports.
- Admin and import APIs are server-only and must never be assumed available in the
  static reader.
- A future static host, object CDN, or embedded desktop distribution can reuse the
  same publisher.

## Deferred

- Automated GitHub Actions synchronization from private source providers.
- A separate mobile application.
- Immutable multi-version public snapshots beyond the current published projection.
