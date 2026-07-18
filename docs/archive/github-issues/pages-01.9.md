Provide an agent-friendly CLI for arranging a fork's journey data and publishing it.

## Scope

- Define repository-local layout for source packages, SQLite state, generated projections, and public media.
- Add deterministic commands for `validate`, `import`, `diff`, `build`, and `publish`.
- Make dry-run output reviewable and machine-readable.
- Keep generated files limited to the configured public paths.
- Never require an agent to edit generated JSON by hand or expose secrets in logs.

## Acceptance

An agent or user can validate a package, review a diff, build the public artifact, and arrange the result for a Git commit using documented CLI commands.
