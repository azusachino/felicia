# Incremental local checks

Use `make local-check` during iteration, following Iroha's grouped local-check
workflow. Successful groups are content-keyed under ignored
`.cache/local-checks/`, with `RUN` or `HIT` printed for each group.

```sh
make local-check                         # run changed groups, reuse unchanged ones
make local-check ARGS='--groups scripts' # Python/local-workflow change
make local-check ARGS='--groups admin'   # admin frontend change
make local-check ARGS='--groups docs'    # prose-only change
make local-check ARGS='--force'          # rerun all groups
```

| Group | Checks | Source inputs |
| --- | --- | --- |
| `go` | Go formatting, vet, lint, race/coverage tests | All Go apps and `go.work*` |
| `scripts` | Ruff and Python feature-contract tests | Scripts/tests, Go apps used by CLI fixtures, frontend/package layout, contracts and publication fixtures |
| `admin` | Admin formatting, type/lint checks and Vitest tests | Admin app and shared packages |
| `public` | Public frontend checks and public/shared Vitest tests | Public app and shared packages |
| `docs` | Markdown checks | Markdown and documentation files |

Shared packages invalidate both frontend groups and scripts. Check runners, root
configuration, lockfiles outside app/package trees, contracts, publication fixtures
and unknown paths conservatively invalidate all groups. Local-journey helper/test
changes invalidate scripts only; Go changes also invalidate scripts because those
tests seed and compile through the CLI. Explicit `--groups` selects checks, not a
claim that every dependent group was verified.

Keys include sorted tracked/nonignored untracked file names and bytes, deletions,
checkout-contained symlink targets, ignored root/app/package `.env*` inputs, tool
versions and resolved paths, Python/platform, commands and relevant environment.
Environment and dotenv contents are hashed, never stored in cache records. Outer
Make control flags/selectors and terminal/session metadata do not affect keys.
Missing tools or runtime versions that differ from the project's resolved mise
selection fail closed; use the normal Make target to enter its environment.
The verifier asks mise to resolve LTS/latest selectors and excludes ancestor
config selections. Both Bun and Node identities participate in keys so a change
to either the primary runtime or its compatibility fallback invalidates hits.

Locked frontend installation and the cheap layout guard always run, including on
all-hit runs. The layout guard catches ignored leftovers at retired paths, which
are not in the source snapshot. A failed/interrupted run removes prior success;
inputs changing during checks prevent publication; corrupt records are misses.
After manually altering ignored dependencies/build state, use `--force`.

This cache is local iteration evidence, not acceptance evidence. `make check`,
`make validate`, CI, browser suites, builds, packaging and native acceptance remain
uncached and unchanged. Browser checks still use `make e2e` and
`make test-admin-e2e`. No per-package Go cache, dependency-integrity cache or
browser-result cache is added; the native Go build/test cache remains in use.

The runner is adapted from Iroha's `scripts/local_checks.py` at
`11fd490c348149e9a854dbe9b3b416d2f49c45a9`; the groups and guard are Felicia-owned.

## Verification checkpoint

Fresh independent Herdr peer `felicia-local-check-review` (`wV:p4J`), Pi
`openai-codex/gpt-6-luna` medium (startup argv and live footer confirmed by the
lead), reviewed the working diff at `main` /
`9c38309074645a425294fbdef9bc19c2c76d925a`. The peer ran `make local-check`
(all five groups `RUN`), repeated it (all five `HIT`), forced only docs, ran
`make validate` once, and ran `make fmt-docs-check docs-build`; all exited 0.
Locked installation and layout checks ran even on hits.

The peer found missing invalidation when a dangling symlink's target changed.
A regression failed before the one-line correction and passed afterward. The
final independent recheck used only `make local-check ARGS='--groups scripts'`
twice: first `RUN` with 70 Python tests passing, then `HIT`, both exit 0. The
six-test layout guard passed both times. No finding remains; full gates above
predate this focused correction and were not rerun per line. The final cache
suite has 13 tests for boundaries, keys, tools, failure/force/corruption, symlinks,
preflight and CI refusal. Docs-only closeout uses the docs group and docs build.

This is verification of local tooling, not native S1 acceptance. The original
checkpoint was committed in `1d08e3e` after owner approval; Iroha was read-only. Raw logs under
workstation `.tmp/felicia-local-checks/review/` are disposable; this page preserves
the results.
