Add the development local-filesystem media path before S3/R2 integration.

## Scope

- Read originals from a configured media root.
- Generate bounded derivatives and strip EXIF/GPS metadata.
- Copy only referenced derivatives into `dist/media/`.
- Reject traversal and keep originals outside the artifact.

## Acceptance

A fixture with public, unsupported, and private media produces only safe deterministic public assets.
