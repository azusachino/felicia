Add the release-quality checks currently missing from the Pages experiment.

## Scope

- Assert no raw GPS, source files, or private originals enter `dist/`.
- Check missing media and malformed JSON diagnostics.
- Add keyboard, reduced-motion, mobile, and basic accessibility checks.
- Compare output manifests across identical rebuilds.

## Acceptance

The publication is safe to share and reproducible from a clean checkout.
