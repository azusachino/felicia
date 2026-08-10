---
id: "0032"
title: "SQLite-First v1, PostgreSQL Follow-up"
status: "proposed"
date: "2026-08-10"
decisions:
  - "SQLite is the only supported and acceptance-tested persistence provider for v1.0."
  - "PostgreSQL/PostGIS is deferred to a v1.1/v1.2 follow-up milestone."
related: ["0017", "0021", "0027", "0028"]
supersedes: []
---

# ADR 0032: SQLite-First v1, PostgreSQL Follow-up

## Intent

Ship the first useful Felicia release around the path that already proves the
product's value:

```text
local authoring → SQLite → static compile → published artifact → Pages
```

The first release should not make the author install, configure, migrate, or
verify a second database engine. PostgreSQL/PostGIS remains a deliberate future
provider, not an accidental part of the v1 contract.

V1 supports two usage profiles over that same SQLite boundary:

- **CLI/Pages:** a lightweight workflow that uses the CLI locally and publishes
  a static artifact to GitHub Pages without a long-running server;
- **single-owner self-hosting:** a SQLite-backed server and Admin GUI for the
  owner, with the public reader exposed through the configured ingress and the
  admin surface reachable only from the owner's private network.

## Context

ADR-0008 initially selected PostgreSQL as the sole database. ADR-0017 then
reversed that choice: SQLite became the default local provider and PostgreSQL
remained optional. ADR-0021 and ADR-0027 established the runtime configuration
and provider-neutral module boundaries.

Issue #68 identifies the cost that remains when both providers are treated as
active at the same time: independently maintained DDL, provider-specific
composition, incomplete deployment selection, and a growing parity surface.

The v1 product does not need PostgreSQL to complete its local authoring,
publication, or Pages workflow. Carrying both schema surfaces as equal v1
commitments adds maintenance without advancing the v1 acceptance outcome.

## Proposed decision

1. **SQLite is the v1 persistence contract.** The supported v1 API, admin GUI,
   CLI static compiler, workflow tests, and acceptance journey use SQLite.
2. **PostgreSQL/PostGIS is deferred.** It is removed from the v1 support and
   acceptance promise and re-entered as a separately scoped v1.1/v1.2 milestone.
3. **The provider-neutral seams stay.** `core`, `runtime`, and `publication`
   remain independent of concrete storage. This proposal does not justify a
   layering rewrite or SQL translation layer.
4. **SQLite schema is the only evolving v1 schema.** New v1 schema changes are
   not duplicated into PostgreSQL while PostgreSQL is deferred. The PostgreSQL
   provider may remain in the repository as frozen, explicitly non-v1 work, but
   it must not be presented as parity-supported. A policy boundary does not
   erase the existing physical schema drift; it makes that drift an explicit
   deferred-provider snapshot instead of an active v1 obligation.
5. **V1 entrypoints fail clearly.** A supported v1 executable must reject an
   explicit PostgreSQL selection rather than start an unsupported provider. Any
   retained PostgreSQL command is named and documented as experimental, with no
   implication that it is part of the v1 product.
6. **V1 gates are honest.** The v1 quality and release gates must not require a
   PostgreSQL service, PostgreSQL migrations, or PostgreSQL behavioral parity.
   PostgreSQL-specific commands and documentation are clearly marked as
   deferred development work, and CI keeps any PostgreSQL check in a separate
   non-v1 lane or removes it until re-entry work begins.
7. **Single-owner self-hosting remains in v1.** The CLI/Pages profile does not
   require a long-running server. The self-host profile uses SQLite plus the
   server/Admin GUI. Public ingress exposes the reader only; the owner admin
   surface binds to a configured private network interface or separate private
   listener. Loopback remains a safe development default, not a product
   restriction, and unauthenticated admin access is never public.
8. **Deployment boundaries are separate.** The Compose DSN/driver mismatch and
   admin exposure rule are fixed or explicitly removed from the v1 supported
   deployment shape; neither is postponed as a PostgreSQL design question.

This removes the active two-schema commitment from the supported v1 product
surface without deleting the future provider or changing the runtime
abstraction that will support it later. It does not claim that the repository's
two DDL files are already equivalent.

## Work required before v1 acceptance

The scope decision is only truthful once the repository reflects it:

- supported v1 configuration rejects `postgres` and the v1 docs stop
  advertising `make dev-postgres`/`make test-postgres` as release paths;
- the v1 build and CI gates no longer depend on a PostgreSQL service or its
  migrations, while any retained experimental lane is visibly separate;
- the Compose/share path either becomes explicitly non-v1 or selects its
  provider without ambiguity;
- the self-host deployment separates public reader ingress from the owner admin
  surface using a private listener or an equivalent network ACL; loopback is
  documented as the development default;
- one reproducible real-journey fixture exercises admin authoring → SQLite →
  static compile → Pages artifact, rather than relying only on separate
  synthetic workflows.

## V1 acceptance boundary

The v1 path is complete when all of these work without PostgreSQL:

- local admin authoring and API startup with the default SQLite database;
- intake, authored-field preservation, lifecycle transitions, and publication;
- static compilation with published-only JSON and safe media derivatives;
- live/static contract checks and the Pages workflow;
- single-owner self-hosting with private-network admin access and public-reader
  separation;
- the selected real journey's reader → author → publish acceptance pass.

SQLite-specific behavior may be improved for v1. PostgreSQL support is not a
release blocker or a parity claim during this phase.

## PostgreSQL re-entry gate

Before advertising PostgreSQL support in v1.1/v1.2, make one explicit decision
about each item below:

1. schema ownership and migration strategy, including a baseline for the frozen
   PostgreSQL snapshot;
2. automated schema-shape parity, or a documented provider-specific schema
   contract;
3. fail-loud provider selection when DSN and driver disagree;
4. composition of every supported executable, including static compilation;
5. PostgreSQL migration, contract, workflow, and deployment smoke coverage;
6. SQLite-to-PostgreSQL data portability for an existing local journal and a
   policy for any pre-v1 PostgreSQL users;
7. network binding and authentication boundaries for the admin API.

The follow-up is not complete until `make test-postgres` and the PostgreSQL
workflow prove the same user-facing contract that v1 proves with SQLite. The
follow-up must also state the compatibility target, issue breakdown, migration
owner, and acceptance fixture before PostgreSQL is advertised again.

## Non-goals

- Delete the PostgreSQL provider or its history now.
- Rewrite provider-neutral interfaces.
- Translate SQLite SQL into PostgreSQL SQL, or vice versa.
- Keep two independently evolving schemas merely to preserve an informal parity
  promise before PostgreSQL has a release milestone.

## Trade-off

Keeping the PostgreSQL code and its history preserves reversibility, but it
also preserves maintenance cost and visible schema drift. That cost is
accepted only as a bounded, frozen follow-up surface; it is not a reason to
keep two active v1 commitments.

## Status

Proposed pending the decision and follow-up issue breakdown in GitHub issue
#68. Once accepted, update the roadmap and database-development instructions
to reflect SQLite-only v1 support.
