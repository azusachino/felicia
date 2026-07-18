# Architecture Decision Records (ADRs) — felicia

This directory stores the immutable records of design and architecture choices made during the development of **felicia**. Each record is numbered, dated, and follows a standard format (Context, Decision, Consequences) to serve as a shared source of truth across development sessions and AI agents.

## Index of Decisions

| ID                                                           | Title                                                  | Date       | Status   |
| ------------------------------------------------------------ | ------------------------------------------------------ | ---------- | -------- |
| [0001](0001-personal-now-product-ready.md)                   | Personal-Now, Product-Ready Direction                  | 2026-06-12 | Accepted |
| [0002](0002-mementos-not-tickets.md)                         | Mementos as the Core Unit                              | 2026-06-14 | Accepted |
| [0003](0003-presentation-agnostic-contract.md)               | Presentation-Agnostic Contract                         | 2026-07-08 | Accepted |
| [0004](0004-jp-first-i18n.md)                                | Japanese-First i18n                                    | 2026-07-01 | Accepted |
| [0005](0005-place-as-derived-visit.md)                       | Places as Derived Visits                               | 2026-07-09 | Accepted |
| [0006](0006-memento-template-registry.md)                    | Declarative Memento Template Registry                  | 2026-07-09 | Accepted |
| [0007](0007-backend-stack.md)                                | Backend Stack (PostgreSQL 18 & Go)                     | 2026-07-08 | Accepted |
| [0008](0008-single-user-local-first-ssg.md)                  | Single-User Local-First & SSG Compiler Model           | 2026-07-09 | Accepted |
| [0016](0016-four-module-go-workspace.md)                     | Four-Module Go Workspace Boundaries                    | 2026-07-14 | Accepted |
| [0017](0017-sqlite-first-storage.md)                         | SQLite-First Storage with Optional PostgreSQL          | 2026-07-14 | Accepted |
| [0018](0018-api-runtime-separation.md)                       | API Transport and Runtime Separation                   | 2026-07-14 | Accepted |
| [0019](0019-authored-content-and-system-locales.md)          | Authored Content and System Locales                    | 2026-07-14 | Accepted |
| [0020](0020-root-module-retirement.md)                       | Retire the Transitional Root Go Module                 | 2026-07-14 | Accepted |
| [0021](0021-runtime-configuration-and-database-modes.md)     | Runtime Configuration and Database Modes               | 2026-07-14 | Accepted |
| [0022](0022-unified-intake-and-draft-pipeline.md)            | Unified Intake and Draft Pipeline                      | 2026-07-16 | Accepted |
| [0023](0023-portable-journey-package.md)                     | Portable Journey Package for Import and Agents         | 2026-07-16 | Accepted |
| [0024](0024-optional-ai-enrichment.md)                       | Optional OCR and AI Enrichment                         | 2026-07-16 | Accepted |
| [0025](0025-static-and-self-hosted-modes.md)                 | Static and Self-Hosted Product Modes                   | 2026-07-17 | Accepted |
| [0026](0026-local-first-media-and-blob-storage.md)           | Local-First Media with Pluggable Blob Storage          | 2026-07-17 | Accepted |
| [0027](0027-provider-matrix-and-application-composition.md)  | Provider Matrix and Application Composition            | 2026-07-17 | Accepted |
| [0028](0028-cli-compiler-and-shared-publication-boundary.md) | CLI Compiler and Shared Publication Boundary           | 2026-07-17 | Proposed |
| [0029](0029-community-go-workspace-layout.md)                | Community-Shaped Go Workspace Layout                   | 2026-07-17 | Accepted |
| [0030](0030-intake-planning-contract.md)                     | Intake Planning Contract and Candidate Review Boundary | 2026-07-18 | Proposed |
