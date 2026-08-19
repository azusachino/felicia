# Repository layout

This is the canonical current-tree guide. The decision record for the one-time
cut-over is [ADR-0034](../adr/0034-application-and-shared-package-layout.md).

```text
apps/
  felicia-core/          # pure domain and ports
  felicia-runtime/       # application use cases
  felicia-providers/     # persistence/source/blob adapters
  felicia-publication/   # public projections and static compiler
  felicia-server/        # HTTP composition and migrations
  felicia-cli/           # CLI and compiler composition
  felicia-web/           # private authoring host
  felicia-public-site/   # public reader host
packages/
  felicia-shared/        # public reader contracts, designs, components, styles
contracts/               # canonical cross-language contract source
ops/                     # deployment-owned files
scripts/                 # repository automation
docs/                    # documentation and ADRs
```

## Boundaries

The Go modules point inward toward `felicia-core`; server and CLI composition
depend on publication, never the other way around. No application module is a
library dependency of another application module.

The shared frontend package owns the public reader's view models, design
registry, compositions, styles, and reader localization. The public-site host
owns browser bootstrapping and transport/static-artifact adaptation. The web
host owns authoring. Admin preview is the compiled public artifact served by
the public renderer, so it is the same reader path users receive.

`contracts/canonical/v1/schema.json` remains the canonical cross-language
contract. TypeScript package types are reader-facing projections, not a second
schema authority.

Repository tools remain under `scripts/`, one focused command per file. The
Makefile delegates workspace discovery and orchestration to UV-run scripts; it
does not carry a module list or grow into a second scripting language. New
shared tooling directories are deferred until a real reuse boundary exists.

## Working rule

When a new file does not clearly belong to one of these boundaries, update
ADR-0034 before adding an abstraction. Current build and run commands are
documented in [AGENTS.md](../../AGENTS.md) and the Makefile; this page records
ownership, not a second task runner.
