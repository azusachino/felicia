---
id: "0034"
title: "Application and Shared Package Layout"
status: "accepted"
date: "2026-08-19"
supersedes:
  - "0029"
related:
  - "0028"
  - "0031"
  - "0033"
---

# ADR 0034: Application and Shared Package Layout

## Context

Felicia has reached the point where the current flat Go workspace and
public-only frontend ownership make the product boundary harder to see than it
needs to be. The backend modules are real applications and libraries, but their
names are not visible in the tree. The public site owns its data models,
design registry, compositions, and styles even though the same public artifact
is the thing the admin flow previews.

The next change must be a one-time cut-over. A long-lived half-old,
half-new layout would make imports, build commands, documentation, and agent
handoffs less reliable. This decision therefore records the target tree,
dependency direction, contract ownership, and migration gates before files are
moved.

## Decision

Adopt an Iroha-shaped application/package layout while keeping Felicia's
domain boundaries and behavior unchanged.

### Target tree

```text
apps/
  felicia-core/          # pure domain and ports Go module
  felicia-runtime/       # application use cases Go module
  felicia-providers/     # persistence/source/blob adapters Go module
  felicia-publication/   # public projections and static compiler Go module
  felicia-server/        # HTTP composition and migrations Go module
  felicia-cli/           # user-facing CLI and compiler composition Go module
  felicia-web/           # private authoring host
  felicia-public-site/   # public reader host
packages/
  felicia-shared/        # public view contracts, designs, compositions, styles
contracts/               # canonical cross-language contract source
ops/                     # deployment-owned files
scripts/                 # repository automation
docs/                    # project documentation and ADRs
```

`apps/` names executable or independently buildable application boundaries.
`packages/felicia-shared` is the only frontend package allowed to own the
public reader's view contract, design registry, reader compositions, shared
styles, and reader-facing localization. Hosts remain thin adapters:

- `felicia-public-site` owns browser bootstrapping, environment handling,
  HTTP/static-artifact loading, URL/media adaptation, and deployment wiring.
- `felicia-web` owns authoring screens and admin APIs. It does not become a
  second public renderer.
- Admin preview continues to consume the compiled public artifact served by
  the public host. Therefore preview and the public site exercise the same
  reader renderer and publication contract.

### Dependency direction

The Go workspace keeps one-way dependencies:

```text
felicia-core
    ^
felicia-runtime <- felicia-providers
    ^                  ^
    +------ felicia-server / felicia-cli

felicia-publication -> felicia-core
felicia-server      -> felicia-publication
felicia-cli         -> felicia-publication
```

No Go module may import an application module. The shared frontend package may
not import either frontend host. Host-specific adapters may import the shared
package; the shared package may depend only on package dependencies and its own
source files.

### Contract ownership

`contracts/canonical/v1/schema.json` remains the canonical cross-language
contract. Go publication types remain the server-side public projection and
compiler boundary. `packages/felicia-shared` owns the TypeScript reader view
models and design inputs derived from that public boundary; it does not invent
a competing schema. HTTP and static-artifact adapters translate at the host
edge.

### Cut-over rules

1. Move files once, preserving behavior and public URLs unless a test or
   configuration path must change with the tree.
2. Rewrite module paths, workspace entries, imports, build commands, migration
   paths, container paths, scripts, and active documentation in the same
   change.
3. Do not duplicate old modules, renderers, contracts, or styles as a bridge.
4. Keep historical ADRs and archived notes as historical records; they may
   mention the layout that existed when they were written. Current guides must
   point only at the target tree.
5. Make the repository's checks discover the Go workspace and assert the
   package/application boundaries so a future rename fails loudly.

### Verification gates

The cut-over is complete only when all of these pass from the repository root:

- `make check`
- `make build`
- `make test-features`
- `make web-check`
- `make docs-build`
- `make validate`

The layout tests must also prove that the old Go module roots and old frontend
host roots are absent, that the shared package has no host imports, and that
the canonical contract paths remain present.

## Consequences

- A fresh contributor can find deployable applications under `apps/` and
  reusable reader code under `packages/` without reconstructing ownership from
  import paths.
- Admin preview remains truthful because it is still the compiled public reader
  artifact, rather than a parallel admin implementation of the public design.
- The cut-over touches build, container, migration, script, test, and document
  paths at once. That is deliberate: leaving one active old path would create a
  second layout contract.
- The shared package is intentionally one package for now. Splitting contracts,
  themes, and compositions into more packages is deferred until independent
  release or dependency boundaries actually appear.

## Rejected alternatives

- **Keep the flat Go roots:** smaller immediate diff, but application ownership
  remains implicit and diverges from the repository layout used for the web
  boundary.
- **Keep all public reader code in `felicia-public-site`:** avoids a package
  move, but makes the shared renderer/contract boundary aspirational and keeps
  preview coupling implicit.
- **Render public preview inside admin:** would create two render paths and
  can make preview differ from the published site.
- **Add a second canonical schema for the frontend:** would create contract
  drift; the existing canonical contract and publication projection already
  provide the needed authority.
