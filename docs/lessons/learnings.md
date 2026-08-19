> Project-local fallback lessons captured when asobi was unavailable. Migrate into asobi when possible.

## 2026-07-12 — stable-journey-identifiers

Type: finding

Lesson: A journey UUID is the stable API identity; a slug is user/content data.
The seed may resolve an existing database UUID by slug for idempotent imports,
but no public web request or generated API asset should depend on that slug.

## 2026-07-12 — bangkok-audit

Type: finding

Lesson: The Bangkok journey is not cross-country: `THA` covers both Bangkok and
Ayutthaya. Its real defect is inconsistent route provenance—the route remains
synthetic while other journeys use local GPX files. Country validation and route
provenance are separate checks.

## 2026-07-12 — route-and-content-provenance

Type: work-experience

Lesson: Mock route geometry must either come from a local GPX source with
provenance or be clearly labeled as synthetic. Do not attach a real GPX track
to a journey whose dates and places describe a different trip.

## 2026-07-12 — canonical-vs-runtime-data

Type: wrong-approach

Lesson: A clean publication workspace does not prove the webpage is clean. The
database is a persistent projection and can retain stale mock mementos after a
slug-to-existing-UUID seed rerun. Always compare the browser/API payload with
the canonical fixture, then clean the mock source namespace before reseeding.

Resolved 2026-07-12: the seed now resets the dedicated mock journal and the
Bangkok UUID deep link was verified in both the API and Playwright.
