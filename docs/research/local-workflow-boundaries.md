# Local workflow boundaries

The local journey workflow is an orchestration path, not a second backend. Its
files are deliberately split by responsibility:

```text
raw GPX/photos/sidecar
        │
        ▼
felicia-cli journey plan       runtime/intake + providers/local
        │ plan.json
        ▼
local_journey_author.py        human/agent stop and memento decisions
        │ workspace.json + journey.json, stops.json, mementos.json
        ▼
local_journey_package.py       portable package serialization + media hashing
        │ journey.zip
        ▼
felicia-cli import/static      importer, SQLite provider, publication compiler
        │
        ▼
apps/web-public                API adaptation + presentation designs
```

The UV entrypoint, [`scripts/local_journey.py`](../../scripts/local_journey.py),
only coordinates these seams. It should not grow provider logic, package
normalization, or frontend rendering logic.

The backend remains responsible for canonical domain validation and persistence;
the frontend remains responsible for adapting the public contract and rendering
map/collection/journal designs. The UV scripts are disposable local orchestration
and fixture tooling, so they may prepare files and call the real CLI but must not
reimplement backend rules.

When a new concern appears, place it at the narrowest seam:

- source extraction or candidate derivation → `runtime/intake` or a provider;
- authored workspace interaction → `scripts/local_journey_author.py`;
- package/media transport → `scripts/local_journey_package.py` and `core/journeypackage`;
- public shape or duplicate-safe UI keys → `publication` or `apps/web-public/src/api`;
- visual behavior → the relevant design component, not the API loader.
