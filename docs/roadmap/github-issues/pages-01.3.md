Make GPX a real input to static publication.

## Scope

- Import a checked-in GPX fixture into SQLite.
- Preserve source filename/checksum and import-run identity.
- Preserve timestamps and elevation where available.
- Generate deterministic public geometry and malformed-GPX cases.

## Acceptance

Two builds from the same SQLite data produce equivalent route JSON without publishing private source files.
