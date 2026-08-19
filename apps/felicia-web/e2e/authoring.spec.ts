// ADMIN-01.8 closed-loop E2E: import -> review candidate -> author ->
// publish -> compile -> the compiled artifact contains the authored essay.
//
// Process lifecycle (disposable API server, `bun run dev`, the mock
// Dawarich/Immich upstream, the seeded journey) is entirely owned by
// scripts/e2e_admin_gui.py, which invokes this spec via `bunx playwright
// test` and passes the context it already set up through env vars:
//   - E2E_BASE_URL  the web-admin dev server (playwright.config.ts baseURL)
//   - E2E_API_BASE  the disposable admin API server (for the live
//                   public-API assertion below)
//   - E2E_JOURNEY_ID the UUID of the journey seeded via the CLI import path
//   - E2E_OUT_DIR    informational only from this spec's point of view: the
//                    server compiles into SITE_OUT_DIR (set on the server
//                    process, not here), which scripts/e2e_admin_gui.py sets
//                    equal to this same directory and re-reads off disk
//                    after this spec exits, as a second, independent check
//
// The intake candidate this spec reviews comes from scripts/mock_upstream.py's
// fixed Dawarich visits fixture (fetched live over HTTP by the real intake
// planner) rather than from anything this spec or the Python harness
// authors directly — so which of the two candidates ("明治神宮" or "道頓堀")
// ends up promoted is not assumed; the spec reads back whichever it clicked.
import { test, expect, type Page } from "@playwright/test"

function requireEnv(name: string): string {
  const value = process.env[name]
  if (!value) throw new Error(`${name} is required (set by scripts/e2e_admin_gui.py)`)
  return value
}

const JOURNEY_ID = requireEnv("E2E_JOURNEY_ID")
const API_BASE = requireEnv("E2E_API_BASE")

// Kept identical to the constants of the same name in scripts/e2e_admin_gui.py
// — that script's filesystem-side check on the compiled artifact looks for
// these exact strings, so a drift between the two would fail loudly rather
// than the check silently no-op'ing.
const JOURNEY_TITLE = "Admin GUI E2E Journey"
const MEMENTO_TITLE = "Admin GUI E2E Memento"
const ESSAY_SENTINEL = "Felicia admin GUI E2E authored essay -- sentinel 9f3c2b1a"
const GOODS_NAME = "E2E Souvenir"

// ADMIN-02 M2: site identity constants, kept identical to the constants of
// the same name in scripts/e2e_admin_gui.py for the same reason as above —
// that script's filesystem-side check on the compiled api/v1/site.json
// looks for these exact values.
const SITE_TITLE = "Admin GUI E2E Site"
const SITE_DESIGN = "v4"
const SITE_DESIGN_LABEL = "Atlas"
const SITE_ACCENT = "#336699"

test.describe.serial("admin GUI closed loop (ADMIN-01.8)", () => {
  let page: Page
  // The promoted candidate's label (e.g. "明治神宮"), captured once it's
  // promoted and reused to find the same row/memento across steps — safer
  // than re-deriving a "still has a Promote button" locator, which stops
  // matching this exact row the moment promotion succeeds.
  let candidateLabel = ""

  test.beforeAll(async ({ browser }) => {
    page = await browser.newPage()
  })

  test.afterAll(async () => {
    await page.close()
  })

  test("navigates from the journey list to the seeded journey detail", async () => {
    await page.goto("/")
    await expect(page.getByRole("heading", { name: "Journeys" })).toBeVisible()
    await page.getByRole("link", { name: new RegExp(JOURNEY_TITLE) }).click()
    await expect(page).toHaveURL(new RegExp(`#/journey/${JOURNEY_ID}$`))
  })

  test("plans intake and shows a proposed candidate in the inbox", async () => {
    await page.getByRole("button", { name: "Plan intake" }).click()
    await expect(page.getByText(/\d+ stops? proposed/)).toBeVisible({ timeout: 15_000 })

    const proposedRow = page
      .locator(".candidate-row")
      .filter({ has: page.getByRole("button", { name: "Promote" }) })
      .first()
    await expect(proposedRow).toBeVisible()
    candidateLabel = (await proposedRow.locator(".candidate-main strong").innerText()).trim()
    expect(candidateLabel.length).toBeGreaterThan(0)
  })

  test("promotes a candidate with a kind picker choice", async () => {
    // Anchor on the row's own title element: a sibling row's merge-target
    // dropdown also contains this label as an option text, so a plain
    // hasText filter would match both rows.
    const candidateRow = page.locator(".candidate-row").filter({ has: page.locator(".candidate-main strong", { hasText: candidateLabel }) })
    await candidateRow.getByLabel(/^Kind for/).selectOption("goods")
    await candidateRow.getByRole("button", { name: "Promote" }).click()
    await expect(candidateRow.getByText("Promoted to a draft memento.")).toBeVisible()

    // It leaves the inbox's actionable set (no more Promote button on this
    // row) and a matching draft memento appears in the memento list.
    await expect(candidateRow.getByRole("button", { name: "Promote" })).toHaveCount(0)
    await expect(page.locator(".memento-link").filter({ hasText: candidateLabel })).toBeVisible()
  })

  test("authors the promoted memento with title, essay, and required kind_data", async () => {
    await page.locator(".memento-link").filter({ hasText: candidateLabel }).click()
    await expect(page).toHaveURL(/#\/journey\/[^/]+\/memento\/[^/]+$/)

    // occurred_at/tz and geometry: occurred_at and the single "goods" point
    // already carried over from the promoted candidate; occurred_tz did
    // not (it isn't part of the promote payload) and a non-draft save
    // requires it, same as the kind's one required field ("name").
    await page.getByLabel("Title", { exact: true }).fill(MEMENTO_TITLE)
    await page.getByLabel("Essay", { exact: true }).fill(ESSAY_SENTINEL)
    await page.getByLabel("Timezone", { exact: true }).fill("Asia/Tokyo")
    await page.getByLabel("name *", { exact: true }).fill(GOODS_NAME)

    await page.getByRole("button", { name: "Save", exact: true }).click()
    await expect(page.getByText("Saved.")).toBeVisible()
    await expect(page.locator(".editor-header .badge")).toHaveText("draft")
  })

  test("advances the lifecycle draft -> authored -> published", async () => {
    const badge = page.locator(".editor-header .badge")
    await page.getByRole("button", { name: "Mark authored" }).click()
    await expect(badge).toHaveText("authored")
    await page.getByRole("button", { name: "Publish" }).click()
    await expect(badge).toHaveText("published")
  })

  test("builds the site from the GUI and the artifact reflects the authored essay", async () => {
    // ADMIN-02 M0: the Site & Deploy page now exposes the compile trigger
    // (apps/felicia-web/src/views/SiteDeploy.svelte), so this drives the real
    // GUI flow instead of posting to /api/admin/compile directly. The
    // server compiles into its configured SITE_OUT_DIR (env var), which
    // scripts/e2e_admin_gui.py sets equal to its own `out_dir` — so there is
    // no request body to carry out_dir in anymore, and that script re-reads
    // the same directory off disk after this test run as a second,
    // independent filesystem-side check.
    await page.goto("/#/site")
    await expect(page.getByRole("heading", { name: "Site & Deploy" })).toBeVisible()

    await page.getByRole("button", { name: "Build site" }).click()
    await expect(page.getByText("Build complete.")).toBeVisible({ timeout: 15_000 })

    const journeysCell = page.locator(".report-cell").filter({ has: page.locator("dt", { hasText: "Journeys" }) })
    await expect(journeysCell.locator("dd")).toBeVisible()
    expect(Number(await journeysCell.locator("dd").innerText())).toBeGreaterThanOrEqual(1)

    // The GUI already reflects "published" (previous step's badge assertion);
    // this confirms the live public API — the same surface the compiled
    // static artifact is meant to match — also serves the authored essay.
    const publicResponse = await page.request.get(`${API_BASE}/api/v1/journeys/${JOURNEY_ID}/mementos`)
    expect(publicResponse.ok()).toBeTruthy()
    const publicMementos = (await publicResponse.json()) as Array<{ essay?: string; title?: string }>
    expect(publicMementos.some((memento) => memento.essay === ESSAY_SENTINEL && memento.title === MEMENTO_TITLE)).toBeTruthy()
  })

  // ADMIN-02 M2: the Site & Deploy page's new Site identity section (design
  // picker, title, accent) — still on /#/site from the previous step.
  test("sets and saves the site identity (design, title, accent)", async () => {
    await expect(page.getByRole("heading", { name: "Site identity" })).toBeVisible()

    const designCard = page.getByRole("button", { name: new RegExp(SITE_DESIGN_LABEL) })
    await designCard.click()
    await expect(designCard).toHaveClass(/selected/)

    await page.getByPlaceholder("Site title").fill(SITE_TITLE)
    await page.getByLabel("Accent color").fill(SITE_ACCENT)

    await page.getByRole("button", { name: "Save site settings" }).click()
    await expect(page.getByText("Saved.")).toBeVisible()
  })

  test("site identity survives a reload", async () => {
    await page.reload()
    await expect(page.getByRole("heading", { name: "Site identity" })).toBeVisible()

    const designCard = page.getByRole("button", { name: new RegExp(SITE_DESIGN_LABEL) })
    await expect(designCard).toHaveClass(/selected/)
    await expect(page.getByPlaceholder("Site title")).toHaveValue(SITE_TITLE)
    await expect(page.getByLabel("Accent color")).toHaveValue(SITE_ACCENT)
  })

  test("building reflects the saved site identity in the live public API", async () => {
    await page.getByRole("button", { name: "Build site" }).click()
    await expect(page.getByText("Build complete.")).toBeVisible({ timeout: 15_000 })

    // Same live/static-parity contract the journeys/mementos checks above
    // rely on: the compiled artifact (checked filesystem-side by
    // scripts/e2e_admin_gui.py's assert_site_settings after this spec exits)
    // must match what the live endpoint serves.
    const siteResponse = await page.request.get(`${API_BASE}/api/v1/site`)
    expect(siteResponse.ok()).toBeTruthy()
    const site = (await siteResponse.json()) as { title?: string; design?: string; accent?: string }
    expect(site.title).toBe(SITE_TITLE)
    expect(site.design).toBe(SITE_DESIGN)
    expect(site.accent).toBe(SITE_ACCENT)
  })

  // ADMIN-02 M1 02.1a: the lifecycle now steps backward too. This reuses the
  // same memento the earlier steps authored and published (found by its
  // authored title, since the candidate's original label stopped matching
  // the memento-list row once authoring gave it a real title) rather than
  // adding a fresh memento, keeping this spec's fixture surface unchanged.
  //
  // Staged rebuild (memento-lifecycle contract §6): publish/unpublish no
  // longer auto-rebuild. This step only flips visibility and asserts the
  // journey is left pending-build; the following step drives the explicit
  // Build & preview action and asserts the pending indication clears.
  test("unpublishes the memento back to authored, staging a pending build", async () => {
    await page.goto(`/#/journey/${JOURNEY_ID}`)
    await page.locator(".memento-link").filter({ hasText: MEMENTO_TITLE }).click()
    await expect(page).toHaveURL(/#\/journey\/[^/]+\/memento\/[^/]+$/)

    const badge = page.locator(".editor-header .badge")
    await expect(badge).toHaveText("published")
    await page.getByRole("button", { name: "Unpublish" }).click()
    await expect(badge).toHaveText("authored")

    // No auto-rebuild: back on the journey detail, the toggle above must
    // surface as a pending build rather than already being reflected in a
    // freshly-compiled artifact.
    await page.goto(`/#/journey/${JOURNEY_ID}`)
    await expect(page.getByRole("button", { name: /^Build & preview \(\d+\)$/ })).toBeVisible()
    await expect(page.locator(".memento-row--pending").filter({ hasText: MEMENTO_TITLE })).toBeVisible()
  })

  test("building resolves the pending unpublish", async () => {
    await page.getByRole("button", { name: /^Build & preview \(\d+\)$/ }).click()
    // The journey-detail build surfaces the report line on success (not the
    // Site page's "Build complete." text).
    await expect(page.locator(".build-report")).toBeVisible({ timeout: 15_000 })

    // The build just resolved every visibility toggle since the last one —
    // the count suffix disappears and the memento row's highlight clears.
    await expect(page.getByRole("button", { name: "Build & preview", exact: true })).toBeVisible()
    await expect(page.locator(".memento-row--pending").filter({ hasText: MEMENTO_TITLE })).toHaveCount(0)
  })

  // Re-publish so this spec leaves the fixture in the same "published, built"
  // state the earlier build/artifact assertions already exercised — the
  // unpublish step above is a self-contained round trip, not a lasting
  // change to the journey. Symmetric to the unpublish steps: republishing
  // stages a pending build too, and this spec resolves it explicitly so the
  // artifact ends up matching the published essay again (the Python
  // harness's own filesystem check runs after this spec exits).
  test("re-publishes the memento, staging a pending build", async () => {
    await page.locator(".memento-link").filter({ hasText: MEMENTO_TITLE }).click()
    await expect(page).toHaveURL(/#\/journey\/[^/]+\/memento\/[^/]+$/)

    const badge = page.locator(".editor-header .badge")
    await expect(badge).toHaveText("authored")
    await page.getByRole("button", { name: "Publish" }).click()
    await expect(badge).toHaveText("published")

    await page.goto(`/#/journey/${JOURNEY_ID}`)
    await expect(page.getByRole("button", { name: /^Build & preview \(\d+\)$/ })).toBeVisible()
    await expect(page.locator(".memento-row--pending").filter({ hasText: MEMENTO_TITLE })).toBeVisible()
  })

  test("building resolves the pending republish and the artifact reflects the published essay again", async () => {
    await page.getByRole("button", { name: /^Build & preview \(\d+\)$/ }).click()
    await expect(page.locator(".build-report")).toBeVisible({ timeout: 15_000 })

    await expect(page.getByRole("button", { name: "Build & preview", exact: true })).toBeVisible()
    await expect(page.locator(".memento-row--pending").filter({ hasText: MEMENTO_TITLE })).toHaveCount(0)

    // Same live public-API check as the first build: the compiled artifact
    // (source of truth for what's deployed, per §6) must show the published
    // essay again, not the gap the unpublish step opened up.
    const publicResponse = await page.request.get(`${API_BASE}/api/v1/journeys/${JOURNEY_ID}/mementos`)
    expect(publicResponse.ok()).toBeTruthy()
    const publicMementos = (await publicResponse.json()) as Array<{ essay?: string; title?: string }>
    expect(publicMementos.some((memento) => memento.essay === ESSAY_SENTINEL && memento.title === MEMENTO_TITLE)).toBeTruthy()
  })
})
