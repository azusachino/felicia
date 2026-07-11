> Project-local fallback lessons captured when asobi was unavailable. Migrate into asobi when possible.

## 2026-07-12 — public-api-slug-identifiers

Status: resolved

Tried: Keep public journey detail endpoints and frontend requests keyed by the
human-readable journey slug while treating UUIDs as an internal implementation
detail.

Why it failed: Slugs are mutable content metadata and are not safe resource
identifiers. The running API logs still showed requests such as
`/api/v1/journeys/bangkok`, despite the intended UUID-based API design.

Do instead: Public and static detail endpoints must use journey UUIDs:
`/api/v1/journeys/{uuid}` and `/api/v1/journeys/{uuid}/mementos`. Include the
UUID in list projections, make the client use `item.id`, and test the exact
request paths in both live and static modes. Keep slugs for authoring, display,
and seed reconciliation only.

## 2026-07-12 — verify-runtime-contract

Status: active

Tried: Rely on a code change and unit tests without checking the request path
emitted by the running stack.

Why it failed: The implementation and the observed runtime contract diverged;
the browser continued to request slug URLs.

Do instead: After API route changes, restart the stack and inspect one browser
request/log line. Require tests for the live URL shape, static generated file
shape, and list-to-detail identifier handoff before declaring the change done.

## 2026-07-12 — validate-every-journey-source

Status: active

Finding: Bangkok is a single-country journey (`THA`); its mementos move from
Bangkok to Ayutthaya, both within Thailand. The remaining Bangkok problem is
that its route is still a synthetic three-point polyline with no GPX source,
while only the Japan routes have local GPX provenance.

Do instead: Audit every journey independently. For each one, verify country,
memento places, route coordinates, dates, and route provenance. Never claim the
seed uses real routes globally when one journey still uses fallback geometry.

## 2026-07-12 — seed-stale-memento-rows

Status: resolved

Finding: The canonical Bangkok fixture contains only Thailand records, but a
running Bangkok page can show Japan records because seed reconciliation reuses
an existing journey UUID by slug and only upserts the current five mementos.
It does not remove old mock mementos already attached to that UUID.

Do instead: Treat a seed rerun as a replacement of the mock projection for each
mock journey. Delete or replace stale rows under the journey's `mock:<slug>:`
source namespace before inserting the canonical mementos, while preserving
non-mock/authored rows. Verify the rendered journey payload after reseeding.

Resolved 2026-07-12: local seed now resets the dedicated mock journal before
rebuilding it, and all nine live UUID payloads were verified clean.
