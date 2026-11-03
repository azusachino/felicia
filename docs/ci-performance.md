# CI performance round

Baseline: latest 50 runs sampled via `gh`; 157 successful jobs.
Successful-job medians by lane: backend (make check-ci) 306s,
desktop composition (macOS, Chromium + WebKit) 550.5s, frontend
(make web-check) 41s. The workflow had no caches of any kind: Go
modules/build state, Bun dependencies and Playwright browsers were
re-fetched on every run.

## Change ([PR #180](https://github.com/azusachino/felicia/pull/180))

Added `actions/cache` for:

- Go build and module caches, located through mise's `go env` (the
  pattern iroha already uses), keyed on `go.work*`/`go.mod`/`go.sum`
  with SHA-pinned key and prefix restore keys — backend and desktop
  jobs;
- Bun's install cache keyed on `bun.lock` — all three jobs;
- the Playwright browser cache (`~/Library/Caches/ms-playwright`)
  keyed on the admin app's pinned `@playwright/test` version — desktop
  job.

Also replaced the shared-ref concurrency group (whose
`cancel-in-progress` could cancel a running main check) with
PR-number/run-id groups that cancel only superseded PR runs and
preserve every main run. No test, lint, build or browser-install
command changed; `make check` passed locally in an isolated worktree
before push.

## Measurements

Revision `705cf1a`, [PR run 41620816397](https://github.com/azusachino/felicia/actions/runs/41620816397):

| Lane | Baseline median | Cold (attempt 1) | Warm (attempt 2) |
| --- | --- | --- | --- |
| Backend | 306s | 240s | **114s** |
| Desktop composition | 550.5s | 589s | 532s |
| Frontend | 41s | 34s | 44s |

Warm verdict: the backend Go cache is the clear win — **62.7% below
the baseline median** (240s cold, 114s warm), consistent with race
-test recompilation moving to cached artifacts. Desktop and frontend
moves are within hosted-runner noise (−3.4% and +7%); the Playwright
and Bun caches remove real download work but that work was a small
fraction of those lanes. One cold/warm pair; source and runner
variation apply. The caches are correct-by-construction (keyed on the
inputs they serve) and cost little when they hit, so they are kept for
all three lanes, but only the backend claim is a measured speedup.
