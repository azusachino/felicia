import { test, expect } from "./fixtures"

// Regression for docs/contracts/automatic-sequence-allocation.md
// ("Automatic new-memento creation obtains distinct positions within its
// journey even when two forms were opened from the same list snapshot").
//
// Two independent new-memento forms are opened from the same snapshot
// (seeded explicit memento at seq 4). Each form posts its caller-chosen id
// and authored values WITHOUT a seq to the dedicated create endpoint
// POST /api/admin/mementos/create, which allocates the journey-local
// position atomically server-side. The positions are therefore exactly
// {5, 6} regardless of which snapshot the forms were opened from.
//
// The tests do not assert on any client-side seq computation — only on the
// persisted result and on the fact that the create requests carry no seq.

const SEED_TITLE = "Anchor stub seq four"

async function submitNewMemento(page: import("@playwright/test").Page, title: string) {
  const requests: Array<{ id: string; body: Record<string, unknown> }> = []
  page.on("request", (request) => {
    if (request.method() === "POST" && /^\/api\/admin\/mementos\/create$/.test(new URL(request.url()).pathname)) {
      const body = request.postDataJSON() as Record<string, unknown>
      requests.push({ id: String(body.id), body })
      console.log(`[memento-sequence] POST /api/admin/mementos/create body=${JSON.stringify(body)}`)
    }
  })
  await page.getByLabel("Title", { exact: true }).fill(title)
  const created = page.waitForResponse((response) => new URL(response.url()).pathname === "/api/admin/mementos/create" && response.request().method() === "POST")
  await page.getByRole("button", { name: "Create", exact: true }).click()
  const response = await created
  // Anything other than a successful creation is an unrelated setup failure
  // (auth/transport/validation), not the regression under test.
  expect(response.status(), `memento creation for "${title}" must succeed; a 4xx/5xx is a setup blocker`).toBe(200)
  const body = response.request().postDataJSON()
  expect(body).not.toHaveProperty("seq")
  expect(body).not.toHaveProperty("expected_revision")
  expect(body).not.toHaveProperty("authored_fields")
  await expect(page).toHaveURL(/#\/journey\/[^/]+\/memento\/[0-9a-f-]+$/)
  return requests
}

async function seedJourneyWithAnchor(page: import("@playwright/test").Page) {
  await page.goto("/#/journey/new")
  await page.getByLabel("Title", { exact: true }).fill("Sequence concurrency journey")
  await page.getByLabel("Place", { exact: true }).fill("Fictional harbour")
  await page.getByRole("button", { name: "Create journey", exact: true }).click()
  await expect(page.getByRole("heading", { name: "Sequence concurrency journey", exact: true })).toBeVisible()
  const journeyId = page.url().split("/").pop()!

  // Seed one explicit memento at seq 4 through the existing explicit upsert
  // endpoint so the forms' snapshot maximum is 4 and the expected new
  // positions are 5 and 6.
  const seedId = crypto.randomUUID()
  const seeded = await page.request.post("/api/admin/mementos", {
    data: {
      id: seedId,
      journey_id: journeyId,
      kind: "goods",
      seq: 4,
      title: SEED_TITLE,
      place: "Fictional harbour",
      occurred_at: "2026-05-01T00:00:00Z",
      occurred_tz: "UTC",
      kind_data: {},
      state: "draft",
    },
  })
  expect(seeded.status()).toBe(200)
  return { journeyId, seedId }
}

test("two new-memento forms opened from the same snapshot persist distinct positions {5,6}", async ({ page }) => {
  test.setTimeout(60_000)
  const { journeyId, seedId } = await seedJourneyWithAnchor(page)

  // Two tabs, same context, same journey, both on the new-memento form.
  const other = await page.context().newPage()
  for (const tab of [page, other]) {
    await tab.goto(`/#/journey/${journeyId}/memento/new`)
  }
  // Both forms must have rendered before either submit happens.
  await expect(page.getByLabel("Title", { exact: true })).toBeVisible()
  await expect(other.getByLabel("Title", { exact: true })).toBeVisible()

  const requestsA = await submitNewMemento(page, "Stale form alpha")
  const requestsB = await submitNewMemento(other, "Stale form beta")

  const rows: Array<{ id: string; seq: number; title: string; authored_fields: string[] }> = await (await page.request.get(`/api/admin/journeys/${journeyId}/mementos`)).json()
  console.log(`[memento-sequence] persisted rows: ${JSON.stringify(rows.map((row) => ({ id: row.id, seq: row.seq, title: row.title })))}`)
  console.log(`[memento-sequence] request order: alpha=${JSON.stringify(requestsA)} beta=${JSON.stringify(requestsB)}`)

  expect(rows).toHaveLength(3)
  const seededRow = rows.find((row) => row.id === seedId)
  expect(seededRow).toBeDefined()
  expect(seededRow!.title).toBe(SEED_TITLE)
  expect(seededRow!.seq).toBe(4)

  const seededIds = new Set([seededRow!.id])
  const newRows = rows.filter((row) => !seededIds.has(row.id))
  expect(newRows.map((row) => row.title).sort()).toEqual(["Stale form alpha", "Stale form beta"])
  expect(new Set(newRows.map((row) => row.id)).size).toBe(2)

  // Regression: server-side atomic allocation gives the two stale forms
  // distinct positions exactly {5, 6}; a duplicate would mean the forms
  // allocated from their own snapshot again.
  const newPositions = newRows.map((row) => row.seq).sort((a, b) => a - b)
  expect(newPositions, `automatic sequence allocation: two new mementos must occupy distinct positions {5, 6}, got ${JSON.stringify(newPositions)}`).toEqual([5, 6])
  expect(rows.map((row) => row.seq).sort((a, b) => a - b)).toEqual([4, 5, 6])

  // Each manually created row carries the server-derived authorship mask:
  // the new form sends no authored_fields, so the mask proves the create
  // boundary derived it (the old manual upsert ownership contract).
  for (const row of newRows) {
    for (const field of ["journey_id", "kind", "seq", "occurred_at", "occurred_tz", "title", "place", "kind_data"]) {
      expect(row.authored_fields, `manually created row "${row.title}" must claim "${field}" in its server-derived authored_fields`).toContain(field)
    }
  }
})

test("committed-but-lost create response retries with the same id and payload without a second row", async ({ page }) => {
  test.setTimeout(60_000)
  const { journeyId } = await seedJourneyWithAnchor(page)
  await page.goto(`/#/journey/${journeyId}/memento/new`)
  await expect(page.getByLabel("Title", { exact: true })).toBeVisible()

  // Intercept ONLY the create endpoint. The first submission really reaches
  // the server (route.fetch), the response is then aborted so the browser
  // sees a network failure after the row was committed — the
  // committed-but-lost-response case. The retry passes through unchanged
  // (fallback), so the server's stable-retry behavior is exercised for real,
  // with no mocked allocator and no fake 200.
  let createCalls = 0
  const seenBodies: Array<Record<string, unknown>> = []
  await page.route("**/api/admin/mementos/create", async (route) => {
    createCalls += 1
    seenBodies.push(route.request().postDataJSON() as Record<string, unknown>)
    console.log(`[memento-sequence] intercepted create call ${createCalls}: ${route.request().postData()}`)
    if (createCalls === 1) {
      await route.fetch() // real server write commits
      await route.abort("connectionreset")
      return
    }
    await route.fallback()
  })

  await page.getByLabel("Title", { exact: true }).fill("Lost response stub")
  await page.getByRole("button", { name: "Create", exact: true }).click()

  // The submission failed visibly; the form and its one id are retained for
  // the retry, and the URL is still the new-memento form.
  await expect(page.getByRole("alert")).toBeVisible()
  expect(page.url()).toMatch(/#\/journey\/[^/]+\/memento\/new$/)
  expect(seenBodies[0]).toMatchObject({ journey_id: journeyId, title: "Lost response stub", kind: "goods", state: "draft" })
  expect(seenBodies[0]).not.toHaveProperty("seq")
  expect(seenBodies[0]).not.toHaveProperty("expected_revision")
  const formId = seenBodies[0].id as string

  // The lost-response write DID commit: exactly one new row, position 5
  // above the seeded anchor, revision 1.
  const rowsAfterLoss: Array<{ id: string; seq: number; revision: number; title: string }> = await (await page.request.get(`/api/admin/journeys/${journeyId}/mementos`)).json()
  expect(rowsAfterLoss).toHaveLength(2)
  const committed = rowsAfterLoss.find((row) => row.id === formId)
  expect(committed).toBeDefined()
  expect(committed!.seq).toBe(5)
  expect(committed!.revision).toBe(1)
  expect(committed!.title).toBe("Lost response stub")

  // The retry submits the SAME id and payload — still no seq — and succeeds.
  await page.getByRole("button", { name: "Create", exact: true }).click()
  await expect(page).toHaveURL(new RegExp(`#\\/journey\\/${journeyId}\\/memento\\/${formId}$`))
  expect(createCalls).toBe(2)
  expect(seenBodies[1].id).toBe(formId)
  expect(seenBodies[1]).not.toHaveProperty("seq")
  expect(seenBodies[1]).toEqual(seenBodies[0])

  // The retry neither created a second row nor bumped the stored revision:
  // it returned the stored row, and navigation landed on its editor.
  const rowsAfterRetry: Array<{ id: string; seq: number; revision: number }> = await (await page.request.get(`/api/admin/journeys/${journeyId}/mementos`)).json()
  expect(rowsAfterRetry).toHaveLength(2)
  const afterRetry = rowsAfterRetry.find((row) => row.id === formId)
  expect(afterRetry!.seq).toBe(5)
  expect(afterRetry!.revision).toBe(1)
  await expect(page.getByLabel("Title", { exact: true })).toHaveValue("Lost response stub")
})
