Define the provider-neutral static read contract before replacing the fixture builder.

## Scope

- Specify SQLite read ports for journeys, mementos, routes, and media metadata.
- Freeze `/api/v1/journeys.json`, `/api/v1/journeys/<id>.json`, and `/api/v1/journeys/<id>/mementos.json`.
- Decide GeoJSON geometry and provenance fields.
- Add shared golden contract tests.

## Acceptance

Generated files and Go server responses pass the same contract test.
