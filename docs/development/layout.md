# Repository layout

This is the canonical current-tree guide. The decision record for the
architecture is [ADR 0007](../adr/0007-application-and-package-layout.md).

```text
apps/
  felicia-core/          # pure domain and ports
  felicia-runtime/       # application use cases
  felicia-providers/     # persistence/source/blob adapters
  felicia-publication/   # public projections and static compiler
  felicia-server/        # HTTP composition and admin API
  felicia-cli/           # CLI and compiler composition
  felicia-admin/         # private authoring/admin studio
  felicia-public-site/   # public reader host
packages/
  felicia-model/         # reader data, public contracts, locale/theme settings
  felicia-reader/        # reader facade, Atlas theme, memento components, styles
contracts/               # canonical cross-language contract source
publication/journeys/    # sanitized production journey catalog
ops/                     # deployment-owned files
scripts/                 # repository automation
docs/                    # documentation and ADRs
```

## Boundaries

The Go modules point inward toward `felicia-core`; server and CLI composition
depend on publication, never the other way around. No application module is a
library dependency of another application module.

The frontend packages have explicit ownership: `felicia-model` owns the reader
view models and public transport projections; `felicia-runtime` owns
renderer-neutral scene meaning; `felicia-components` owns reusable visual
building blocks; `felicia-renderers` owns renderer seams; and `felicia-reader`
owns the named design registry (`cartography`, `cabinet`, `techo`, and `atlas`),
concrete compositions, localization, and styles. The admin host owns authoring
and may preview the reader facade, but remains an admin application. The private
reader host owns the live/private API shell. The public-site host owns browser
bootstrapping and transport/static-artifact adaptation.

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
