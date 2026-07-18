Make the Pages workflow safe and portable for forks.

## Scope

- Derive repository name and project-site base path from the GitHub context.
- Remove hardcoded owner, repository, asset, and local absolute paths.
- Document Pages source/build settings for forks.
- Keep workflow permissions minimal and avoid requiring application secrets.
- Add a fork smoke fixture and verify rebuilds after cloning to a different path.

## Acceptance

A fork can run the documented CLI and workflow without editing source paths or leaking private data.
