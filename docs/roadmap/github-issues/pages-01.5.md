Verify that server mode and GitHub Pages mode expose the same public read contract.

## Scope

- Test `.json` paths against the Go server backed by SQLite.
- Compare server responses with static files from the same fixture.
- Keep extensionless routes only as tested compatibility aliases.
- Ensure missing JSON is not rewritten to SPA HTML.

## Acceptance

The frontend switches between static preview and server mode without a read-model or JSON parse error.
