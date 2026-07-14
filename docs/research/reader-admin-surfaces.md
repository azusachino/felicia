# Research — the reader vs. admin surfaces (the "management page")

> 2026-07-02. "What about the management page — public user vs. logged-in user?" This note
> answers it by consolidating what's already decided into one picture: felicia is **two
> surfaces**, not one page with a login toggle. Mostly derivable from existing ADRs; the one
> genuinely open question (a third _viewer_ tier) is called out. Outcome ADR:
> `felicia:decision:reader-admin-surfaces`. Research-stage. Sits beside
> [`authoring-publish-flow.md`](authoring-publish-flow.md) (the E-half flow) and
> [`ux-restyle.md`](ux-restyle.md) (scaffolding out of the reader).

## Two surfaces, not one page

The planned layout is already `web/{public,admin}` — two SPAs, deliberately:

|          | **Public reader** (`web/public`)                                                       | **Admin / management** (`web/admin`)                                     |
| -------- | -------------------------------------------------------------------------------------- | ------------------------------------------------------------------------ |
| **Who**  | anonymous — anyone with the URL                                                        | the author only                                                          |
| **Auth** | none                                                                                   | **Cloudflare Access** (`admin-auth-cf-access`)                           |
| **API**  | read-only public endpoints                                                             | `/api/admin/*`, JWT-verified                                             |
| **Sees** | published subset: dark map + amber route + designed stubs + essays + curated galleries | everything — drafts, unpublished mementos, ingest candidates, raw fields |
| **Does** | read, navigate, share                                                                  | ingest review, curate, author essays, set visibility, publish            |

This is why the demo's three tabs (Artifact / **Review Queue** / **Transit Creator**) are a
_demo convenience_, not the product: Review Queue and Transit Creator are **admin** surfaces.
`ux-restyle.md` already calls for pulling that scaffolding out of the reader; this note says
where it goes — the admin app.

## The management page = the authoring-publish flow, given a home

`authoring-publish-flow.md` defines the motions; the admin app is where they live:

```
INGEST   → candidates auto-proposed (Immich × Dawarich on timestamp)
CURATE   → confirm / merge / reject   ← "Review Queue" belongs here
AUTHOR   → pick kind, attach photos, write the essay; Transit Creator lives here
PUBLISH  → flip a bounded subset public (per-journey / per-memento / per-field)
```

Publishing is a **boundary, not a switch** (that doc's §3): public gets essay + curated
gallery + route _shape_; private keeps originals, raw GPS precision, EXIF, exact timestamps
(the `.claude/rules/config.md` privacy invariant). So "logged-in vs. public" is really
**author-with-full-access vs. anonymous-with-the-published-subset**.

## Viewer tiers — the one open question

Everything above assumes **two tiers**: anonymous reader + author-admin. A _third_ tier — a
logged-in **viewer** (friends who sign in to see private-but-shared mementos) — is **not**
decided. It is explicitly out of scope for now: `personal-now-product-ready` defers real
**user auth** ("Cloudflare Access suffices"), and a viewer tier needs accounts + per-memento
share ACLs — a product feature, not a clean seam.

**Position (recommended): binary tiers.** Anonymous public + Cloudflare-Access author. A
viewer tier is a later product addition behind the same publish/visibility seam, additive not
reshaping — consistent with personal-now-product-ready. Unpublished simply means author-only.

## What this locks vs. leaves open

- **Locked (this ADR):** two surfaces (`web/public` read-only + `web/admin` behind CF Access);
  authoring/curation/creator scaffolding lives in admin, never in the reader chrome; binary
  viewer tiers for now.
- **Open:** whether/when to add a logged-in viewer tier; publish granularity UX (journey-at-
  once vs. memento drip — from `authoring-publish-flow.md`); where authoring lives _during
  research_ (Notion vs. in-stack admin — from `notion-to-stack.md`).

## Reference to steal from

**AdventureLog** — same stack (MapLibre + PostGIS + Svelte MapLibre) and it has a real
public/private model already; worth studying how it draws the shared-vs-private line and
whether it runs one app with roles or split surfaces, before we commit the admin app's shape.
