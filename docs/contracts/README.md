# Felicia contracts

Felicia's contracts are owned here before implementation details. A contract
must define the stable meaning of data or behavior; a provider, frontend, CLI,
or local file format is an adapter of that contract.

## Contract layers

| Layer | Authority | Purpose |
| --- | --- | --- |
| Canonical | [`contracts/canonical/v1/schema.json`](../../contracts/canonical/v1/schema.json) | Felicia-owned records and media semantics |
| Workspace | `schemas/local-authoring-v1.schema.json` | Human/agent-editable local files |
| CLI | `felicia-cli` commands and JSON/JSONL output | Offline planning, import, review, and publication |
| Admin API | `server/api` transport projection | Server-side authoring and source synchronization |
| Public API | `publication` projection | Published reader data and static JSON |

The workspace schema is not the canonical model. It is intentionally convenient
for editing and may contain review controls that never enter the public API.

## Versioning rules

- Canonical identifiers use `felicia.canonical.vN`.
- A breaking field, meaning, lifecycle, identity, or media change requires a new
  version.
- Additive optional fields may remain within a version only when old readers can
  safely ignore them.
- Projections must document their mapping to canonical fields.
- Provider-specific fields stay in provenance or adapter payloads; they do not
  become canonical by accident.
- Every write path must define idempotency, revision conflict, and error behavior.

## Contract-first gate

Before implementing a new entity or endpoint, add or update:

1. canonical schema and examples;
2. capability/trait semantics;
3. projection mapping;
4. compatibility and migration rules;
5. contract tests for CLI, HTTP, providers, and static output.

The Go interfaces in `core/ports` remain implementation seams. The versioned
behavioral traits are declared in `core/contracts`; neither replaces the JSON
contract or its projection tests.
