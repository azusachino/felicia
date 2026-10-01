---
id: "0029"
title: "Community-Shaped Go Workspace Layout"
status: "accepted"
date: "2026-07-17"
decisions:
  - "Keep the multi-module go.work strategy while removing the apps/ prefix from backend module roots."
  - "Use core, runtime, providers, publication, server, and cli as backend module names."
  - "Keep frontend applications under apps/ because they are separate Bun applications."
  - "Use cli/cmd/felicia as the future user-facing command composition root."
related:
  - "0016"
  - "0028"
supersedes:
  - "0016"
---

# ADR 0029: Community-Shaped Go Workspace Layout

## Context

The backend modules were initially placed under `apps/`, producing import paths
such as `github.com/azusachino/felicia/apps/core`. That naming is understandable
inside the repository but is unusual for Go modules and makes the future public
CLI and reusable packages look like frontend applications.

Felicia already uses a committed `go.work` workspace. The goal is to improve the
module names without collapsing provider boundaries or moving the unrelated Bun
applications.

## Decision

Backend module roots are named directly:

```text
core/         domain and provider ports
runtime/      application use cases
providers/    database, source, and blob adapters
publication/  public projections and compiler contracts
server/       HTTP delivery and server composition
cli/          user-facing CLI and CLI composition
apps/         web-admin and web-public only
```

The workspace remains multi-module. The future binary is built from
`cli/cmd/felicia`, and the command name is `felicia`.

## Consequences

- Go import paths are shorter and communicate backend package ownership.
- The existing provider/runtime seams remain intact.
- The server can be postponed without making the CLI depend on HTTP packages.
- The repository still has more than one `go.mod`; that is intentional and is
  managed by `go.work`.
- Existing external imports require a migration because module paths changed.
