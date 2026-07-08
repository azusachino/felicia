# AdventureLog teardown — architecture, not aesthetics

> Desk research on [AdventureLog](https://github.com/seanmorley15/AdventureLog)
> (seanmorley15), 2026-07-02, via GitHub repo, official docs
> ([adventurelog.app/docs](https://adventurelog.app/docs)), and a third-party
> writeup ([BrightCoding, 2026-02-16](https://www.blog.brightcoding.dev/2026/02/16/adventurelog-the-revolutionary-self-hosted-travel-tracker)).
> No local checkout, no headless-browser pass — this is desk research on
> published sources, not a live-site teardown like `liuaaron-teardown.md`.
> Flagged below wherever a claim rests on a single secondary source rather
> than primary docs/code.
>
> Why this one: AdventureLog is a near stack-twin — MapLibre + PostGIS +
> Svelte, self-hosted, GPL v3, ~2.1k stars, actively developed (v0.7 → v0.12
> across 2024–2026). Where liuaaron.com gave us front-of-house taste, this
> is a back-of-house reference: how does a real, shipping app in our exact
> stack corner structure its data model, its public/private line, and its
> entry-list/map pairing.

## Method

Read the GitHub repo README/structure, the official docs site (usage guide,
admin panel, changelogs — notably `development_timeline.html` and the
`v0-10-0.html` release notes), and the BrightCoding writeup. Cross-checked
claims across at least two sources where possible; single-source claims are
marked. Did not clone the repo or read Django model source directly — treat
field-level claims (e.g. exact model field names) as informed inference from
docs/changelog prose, not verified code.

## Findings

### 1. App topology — one app, not two surfaces

AdventureLog is a **single unified SvelteKit app** with one login, not a
split public/admin pair. Every user (owner or invited collaborator) hits the
same UI; there's no separate "authoring" surface. Public exposure is handled
by **per-object sharing**, not a separate deployment:

- Individual **locations** or whole **collections** (itineraries) can be
  flipped to a public link (unauthenticated read access to that one object).
- **Collaborative editing** is invite-based: v0.11 replaced looser sharing
  with explicit send/accept/manage invites per collection, giving
  collaborators write access to that collection specifically.
- Auth is **allauth**-based (Django), supporting email/password and
  OIDC/social login (evolved from an earlier Lucia-based scheme when the
  frontend was still doing its own auth, pre-Django-backend pivot).
- An admin panel exists but is for **instance administration** (user quotas,
  instance settings) — not a second content-authoring surface. It's closer
  to Django's built-in `/admin/` than to a bespoke curation UI.

This is a materially different shape from felicia's decision. felicia keeps
**two physically separate apps** (`web/public` anon read-only, `web/admin`
behind Cloudflare Access) with no user accounts on the public side at all —
public visitors never authenticate, and there is no per-object sharing
model because there's only one author (personal journal, not a multi-user
SaaS). AdventureLog's ACL model exists to solve a problem felicia doesn't
have (multiple accounts, collaborative trip planning). Confirms our
two-surface + zero-accounts choice is the right cut for a single-author
journal, not a gap.

### 2. Entry-list + map pairing — filterable list, not a chronological rail

AdventureLog's primary browsing pattern is **locations-as-markers on a map**
with a **filterable list alongside it** (filter by visit status — visited/
planned/wishlist — by category, or by date range), not the liuaaron-style
strict chronological timeline rail felicia is building. Per the docs and the
BrightCoding writeup:

- Markers carry status-based icons (visited/planned/wishlist); clicking a
  marker opens a popup with a summary and a link to the detail page (client
  routed, not an inline animate-open).
- Users can **click empty map area to create a new location** — an authoring
  affordance baked into the same surface as browsing, which only makes sense
  because there's no separate admin app.
- The map **auto-fits bounds** to show all currently-filtered markers.
- Collections (itineraries) with start/end dates get a dedicated **trip map
  + chronological timeline view** — v0.10 specifically added "chronologically
  accurate map and timeline view showing adventures in the right order,"
  i.e. only inside a dated collection does browsing become time-ordered; the
  top-level view is filter-driven, not date-driven.
- Three view modes are offered for a collection: itinerary list, map view,
  calendar view — not one fused index+map+detail like liuaaron.

So AdventureLog's default mental model is "map of everywhere I've been /
want to go," filtered; felicia's is "map of one journey," walked
chronologically. felicia's index-rail-drives-camera pairing (click entry →
map flies to it, ordered by date) is closer to AdventureLog's *per-collection
itinerary view* than to its top-level location browser. Worth noting for
later: AdventureLog treats "browse everything" and "walk one trip
chronologically" as two different views: felicia's public site is
single-journey-at-a-time by design (each journey is its own route), so this
distinction mostly resolves itself, but it's a reminder that a future
"all journeys" overview page would need its own (filtered, not
chronological) map pattern if we ever add one.

### 3. Data model — Location (with Visits) → Collection, PostGIS points not lines

Model shape (per docs/changelog; not verified against Django source):

- **Location** — the base entity (renamed from "Adventure" in v0.11 to
  clarify semantics). Fields include name, description, category, rating
  (1–5), a PostGIS point geometry, and visit status. A `distance_to()` method
  is mentioned doing PostGIS proximity math.
- **Visit** — a location can have **multiple visits** (repeat trips to the
  same place get their own date/notes each), a one-to-many that felicia's
  current model (one memento = one moment) doesn't need but is a clean
  pattern if we ever wanted a "revisited" case.
- **Collection** — groups locations for trip planning. Undated, it's just a
  folder; add start/end dates and it becomes a full itinerary with ordering,
  a route/transportation layer between locations, budgeting, checklists,
  accommodation notes.
- **Categories** — user-defined tags on locations, not a fixed enum. This is
  the closest AdventureLog analog to felicia's `kind` tag, but it's
  free-form organizational metadata (for filtering), not a
  rendering-template selector. AdventureLog does not appear to have anything
  like felicia's `kind`-tagged stub-rendering concept — locations don't have
  visually distinct "kinds" of physical artifact; they're uniform
  pin+card records regardless of what they represent (a museum, a
  restaurant, a hike).
- **Trails/Activities** (added v0.11) — GPX-uploadable trails and
  Strava/Wanderer-linked activities, geometrically closer to LineStrings,
  but this is bolt-on outdoor-recreation tracking, separate from the core
  Location model, not the same "route line" concept as a journey path.
- **Transportation** (collection-exclusive) shows "the route between
  locations and mode of transportation" — this is the nearest thing to
  felicia's route-line-per-journey, but it reads as point-to-point hop
  segments per transportation leg (flight, train, etc.), not one continuous
  hand/algorithmically-drawn track. Could not verify whether these are
  drawn as PostGIS LineStrings or just straight great-circle/point-to-point
  UI connectors — not stated in any source consulted.
- Primary keys are **UUIDs**, switched from auto-increment ints in v0.5.1
  specifically to avoid painful migrations once public sharing existed
  (sequential IDs leak object count / enable enumeration on public share
  links). Directly relevant to felicia's public API: worth checking our
  public-facing IDs aren't sequential either, if they aren't already opaque.

### 4. Stack specifics

- **Frontend:** SvelteKit + TailwindCSS + DaisyUI, `svelte-maplibre` for
  declarative map bindings (same library felicia is evaluating/using).
- **Map tiles:** defaults to free OSM raster/vector tiles via MapLibre;
  self-hosted tile server or Mapbox is opt-in via `.env` config. No forced
  vendor lock-in — matches felicia's self-hosted-first instinct.
- **Backend:** Django + Django REST Framework, PostGIS via `postgis/postgis`
  Docker image. This is the one big stack divergence from felicia (Go, not
  Django) — the pivot story is instructive though: the project *started*
  frontend-only with SvelteKit doing its own persistence (browser
  localStorage), then hit a wall on file handling and durable multi-device
  storage and pivoted to a real backend (July 2024) specifically for
  "admin, built-in auth, and mature file handling." felicia skipped this
  detour by starting with a Go+Postgres backend from day one — the
  AdventureLog history reads as validation that a backend-first call was
  right for anything beyond a toy.
- **Auth:** `django-allauth`, session/token-based; REST API uses DRF token
  auth. No mention of a headless/edge-auth pattern like felicia's Cloudflare
  Access gate — AdventureLog's admin surface is inside the same Django app,
  gated by Django's own permission system, not an edge proxy.
- **API shape:** REST (DRF viewsets), not GraphQL. One documented pattern:
  `get_queryset()` on the Collection viewset filters to "owned or shared"
  itineraries server-side, with `@action`-decorated custom endpoints (e.g.
  GPX export, invite accept) layered onto standard CRUD routes. Reasonable
  precedent for felicia's own API-shape decisions if/when the public API
  needs anything beyond plain REST reads.
- **Deployment:** Docker Compose (default), Traefik variant, and a
  Kubernetes/kustomize path — all documented, all containers-first, no bare
  metal Raspberry Pi story evident (unlike felicia's target host). Minimum
  specs quoted at 2GB RAM / 10GB storage, which suggests it would probably
  run acceptably on a Pi-class host even though that's not their documented
  path — untested claim, flagged as such.

### 5. What to steal vs reject

**Steal:**

- **Opaque public IDs (UUID, not sequential int)** for anything exposed on
  the public site — AdventureLog learned this the hard way (retrofitted in
  v0.5.1) after already having public share links; felicia should make sure
  every publicly-routable ID (journey slug, memento ID) is non-enumerable
  from day one. Worth a quick audit of current schema/design docs.
- **Undated-vs-dated collection duality** (folder vs. itinerary) is a clean
  idea in the abstract, but felicia's domain already resolves this — every
  journey has dates by construction, so we don't need the toggle. Noting it
  only because it's a tidy pattern if felicia ever adds an "unsorted /
  draft" journey state pre-dates-being-set.
- **`get_queryset()`-style server-side owned-or-shared filtering** as the
  shape for scoped API reads is a reasonable REST precedent if felicia's
  admin API ever needs multi-actor visibility rules — currently moot
  (single author), logged for later if felicia ever adds a second
  contributor.
- **Backend-first validation** — AdventureLog's frontend-only-then-pivot
  history is a real-world data point that our Go+Postgres-from-day-one
  choice avoids a known failure mode (file handling / durable state in a
  frontend-only app). No action needed, just confirms the existing call.

**Reject / already diverged on purpose:**

- **Single unified app with roles.** felicia deliberately keeps public and
  admin as two separate apps with no shared login and no public-side
  accounts at all. AdventureLog's per-object sharing + collaborator invites
  solve a multi-user problem felicia doesn't have. Don't import this
  complexity.
- **Free-form user categories as the closest thing to `kind`.** AdventureLog
  categories are just filter tags; they don't drive rendering. felicia's
  `kind`-tagged stub templates (design decision, see
  `docs/research/mementos-not-tickets.md`) are a stricter, purpose-built
  concept — nothing here suggests changing that.
- **Click-map-to-create-location as a browsing-surface affordance.** Only
  works because AdventureLog has one app. felicia's authoring happens in
  `web/admin` via the A+E pipeline, not by clicking the public map.
  Don't blur that line even superficially.
- **Django/DRF backend.** Not a felicia rejection of Django per se — it's
  moot, felicia is Go, already decided, no reason to reconsider given
  AdventureLog's experience doesn't surface a Go-specific gap.

## Open questions

- Is AdventureLog's "Transportation" route between locations actually a
  PostGIS `LineString`, or just a UI line connecting two points with no
  persisted geometry? Not resolved by any source consulted — would need a
  repo clone / Django model read to confirm. Matters for felicia only as a
  sanity check on our own route-line storage choice (already decided
  per-day `LineString` segments, per the liuaaron teardown), not as an open
  design question on our side.
- Does AdventureLog do any image processing / EXIF handling on uploaded
  photos before serving them publicly? No source consulted mentions this at
  all — worth flagging as a gap since it's exactly the privacy invariant
  felicia has hard-committed to (EXIF-strip before public serve). If
  AdventureLog doesn't strip EXIF on public share links, that's a real
  privacy footgun in a comparable app — not verified either way here, but
  seems worth a follow-up check since it would be a concrete cautionary
  example if confirmed.
- This teardown didn't check actual response payload shapes (REST JSON) or
  bundle-inspect the SvelteKit frontend the way the liuaaron teardown did
  with playwright — everything here is docs/changelog-sourced. A follow-up
  pass cloning the repo (it's GPL, self-hostable, Docker Compose one-liner
  per the docs) would let us verify the DRF viewset/model claims directly
  and settle the two questions above.
