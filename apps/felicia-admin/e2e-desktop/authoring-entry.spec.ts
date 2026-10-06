import { test, expect } from "./fixtures"
import { decideDiscard } from "./discard-dialog"
import { setDate } from "./date-field"

async function createEmptyJourney(page: import("@playwright/test").Page) {
  await page.goto("/#/journey/new")
  await page.getByLabel("Title", { exact: true }).fill("My new journey")
  await page.getByLabel("Place", { exact: true }).fill("Fictional city")
  await page.getByRole("button", { name: "Create journey", exact: true }).click()
  await expect(page.getByRole("heading", { name: "My new journey", exact: true })).toBeVisible()
}

test("a freshly created journey can be edited and its first memento authored", async ({ page }) => {
  await createEmptyJourney(page)
  const id = page.url().split("/").pop()!
  const created = await (await page.request.get(`/api/admin/journeys/${id}`)).json()
  // Seed a synthetic source reference through the normal journey API; the
  // edit form passes source_ref through unchanged and the handler must
  // retain it across a metadata-only edit.
  const seededSource = await page.request.post("/api/admin/journeys", {
    data: { ...created, source_ref: "synthetic:trip-42", date_start: created.date_start.slice(0, 10), date_end: created.date_end.slice(0, 10), expected_revision: created.revision },
  })
  expect(seededSource.status()).toBe(200)
  const original = await (await page.request.get(`/api/admin/journeys/${id}`)).json()
  expect(original.source_ref).toBe("synthetic:trip-42")
  await page.getByRole("button", { name: "Edit journey", exact: true }).click()
  await expect(page.getByLabel("Title", { exact: true })).toHaveValue("My new journey")
  await page.getByLabel("Title", { exact: true }).fill("My edited journey")
  await page.getByLabel("Place", { exact: true }).fill("Another fictional city")
  await page.getByRole("button", { name: "Save journey", exact: true }).click()
  await expect(page.getByRole("heading", { name: "My edited journey", exact: true })).toBeVisible()
  const edited = await (await page.request.get(`/api/admin/journeys/${id}`)).json()
  expect(edited.id).toBe(original.id)
  expect(edited.journal_id).toBe(original.journal_id)
  expect(edited.slug).toBe(original.slug)
  expect(edited.date_start).toBe(original.date_start)
  expect(edited.source_ref).toBe("synthetic:trip-42")
  expect(edited.revision).toBeGreaterThan(original.revision)
  await page.getByRole("button", { name: "Add memento", exact: true }).click()
  const kind = page.getByRole("button", { name: "Kind", exact: true })
  await expect(kind).toHaveAttribute("data-select-trigger", "")
  expect((await kind.boundingBox())!.height).toBeGreaterThanOrEqual(40)
  await kind.click()
  await page.getByRole("option", { name: "goods", exact: true }).click()
  await page.getByLabel("Title", { exact: true }).fill("First keepsake")
  await page.getByRole("button", { name: "Create", exact: true }).click()
  await expect(page.getByLabel("Title", { exact: true })).toHaveValue("First keepsake")
  await expect(page).toHaveURL(/#\/journey\/[^/]+\/memento\/[0-9a-f-]+$/)
  await page.getByLabel("Essay", { exact: true }).fill("A story written without importing anything.")
  const saved = page.waitForResponse((response) => response.url().endsWith("/api/admin/mementos") && response.request().method() === "POST")
  await page.locator(".actions").getByRole("button", { name: "Save", exact: true }).click()
  expect((await saved).status()).toBe(200)
  await page.reload()
  await expect(page.getByLabel("Essay", { exact: true })).toHaveValue("A story written without importing anything.")
  const rows = await (await page.request.get(`/api/admin/journeys/${id}/mementos`)).json()
  expect(rows).toHaveLength(1)
  expect(rows[0].state).toBe("draft")
  expect(rows[0].seq).toBe(0)
})

test("journey edit cancellation and conflicting save preserve input and the other writer", async ({ page }) => {
  await createEmptyJourney(page)
  const id = page.url().split("/").pop()!
  const original = await (await page.request.get(`/api/admin/journeys/${id}`)).json()
  await page.getByRole("button", { name: "Edit journey", exact: true }).click()
  await page.getByLabel("Title", { exact: true }).fill("Unsaved owner title")
  await page.getByRole("button", { name: "Back to journey", exact: true }).click()
  await decideDiscard(page, false)
  await expect(page.getByLabel("Title", { exact: true })).toHaveValue("Unsaved owner title")
  const competing = await page.request.post("/api/admin/journeys", {
    data: { ...original, title: "Other writer title", date_start: original.date_start.slice(0, 10), date_end: original.date_end.slice(0, 10), expected_revision: original.revision },
  })
  expect(competing.status()).toBe(200)
  const rejected = page.waitForResponse((response) => response.url().endsWith("/api/admin/journeys") && response.request().method() === "POST")
  await page.getByRole("button", { name: "Save journey", exact: true }).click()
  expect((await rejected).status()).toBe(409)
  await expect(page.getByRole("alert")).toContainText("another writer")
  await expect(page.getByLabel("Title", { exact: true })).toHaveValue("Unsaved owner title")
  expect((await (await page.request.get(`/api/admin/journeys/${id}`)).json()).title).toBe("Other writer title")

  // The explicit recovery path: discard the stale edit, re-enter the form
  // (which reloads the observed revision), and save on top of the other
  // writer's change. The stale input is not silently retried.
  const competingSaved = await competing.json()
  await page.getByRole("button", { name: "Back to journey", exact: true }).click()
  await decideDiscard(page, true)
  await expect(page.getByRole("heading", { name: "Other writer title", exact: true })).toBeVisible()
  await page.getByRole("button", { name: "Edit journey", exact: true }).click()
  await expect(page.getByLabel("Title", { exact: true })).toHaveValue("Other writer title")
  await page.getByLabel("Title", { exact: true }).fill("Reconciled title")
  await page.getByRole("button", { name: "Save journey", exact: true }).click()
  await expect(page.getByRole("heading", { name: "Reconciled title", exact: true })).toBeVisible()
  const reconciled = await (await page.request.get(`/api/admin/journeys/${id}`)).json()
  expect(reconciled.revision).toBe(competingSaved.revision + 1)
  expect(reconciled.title).toBe("Reconciled title")
})

test("journey edit rejects an inverted date range without submitting, then a valid correction saves once", async ({ page }) => {
  await createEmptyJourney(page)
  const id = page.url().split("/").pop()!
  const original = await (await page.request.get(`/api/admin/journeys/${id}`)).json()
  await page.getByRole("button", { name: "Edit journey", exact: true }).click()
  await setDate(page.getByRole("group", { name: "Start date", exact: true }), "2026-05-01")
  await setDate(page.getByRole("group", { name: "End date", exact: true }), "2026-05-03")

  let journeyPosts = 0
  page.on("request", (request) => {
    if (request.method() === "POST" && new URL(request.url()).pathname === "/api/admin/journeys") journeyPosts += 1
  })

  await setDate(page.getByRole("group", { name: "End date", exact: true }), "2026-04-01")
  await expect(page.locator("#date-range-error")).toContainText("End date must be on or after the start date.")
  await expect(page.locator('[name="date_start"]')).toHaveValue("2026-05-01")
  await expect(page.locator('[name="date_end"]')).toHaveValue("2026-04-01")
  await expect(page.getByLabel("Title", { exact: true })).toHaveValue("My new journey")
  await page.getByRole("button", { name: "Save journey", exact: true }).click()
  // Deterministic no-submit evidence: the click leaves the editor open with
  // the range error still shown (submit() returns before any request).
  await expect(page).toHaveURL(/#\/journey\/[^/]+\/edit$/)
  await expect(page.locator("#date-range-error")).toContainText("End date must be on or after the start date.")

  await setDate(page.getByRole("group", { name: "End date", exact: true }), "2026-05-10")
  await expect(page.locator("#date-range-error")).toHaveCount(0)
  const corrected = page.waitForResponse((response) => response.url().endsWith("/api/admin/journeys") && response.request().method() === "POST")
  await page.getByRole("button", { name: "Save journey", exact: true }).click()
  expect((await corrected).status()).toBe(200)
  // The listener spanned the invalid click too: exactly one journey POST in
  // total proves the rejected click never reached the network.
  expect(journeyPosts).toBe(1)
  const saved = await (await page.request.get(`/api/admin/journeys/${id}`)).json()
  expect(saved.revision).toBe(original.revision + 1)
  expect(saved.date_start.slice(0, 10)).toBe("2026-05-01")
  expect(saved.date_end.slice(0, 10)).toBe("2026-05-10")
  expect(saved.slug).toBe(original.slug)
})

test("journey edit slug collision retains inputs and both rows, then a corrected retry saves", async ({ page }) => {
  await createEmptyJourney(page)
  const id = page.url().split("/").pop()!
  const seeded = await page.request.post("/api/admin/journeys", {
    data: { title: "Collision neighbour", place: "Nara", slug: "taken-slug", date_start: "2026-05-01", date_end: "2026-05-02" },
  })
  expect(seeded.status()).toBe(200)
  const original = await (await page.request.get(`/api/admin/journeys/${id}`)).json()

  await page.getByRole("button", { name: "Edit journey", exact: true }).click()
  await page.getByText("More options", { exact: true }).click()
  await page.getByLabel("Slug", { exact: true }).fill("taken-slug")
  const rejected = page.waitForResponse((response) => response.url().endsWith("/api/admin/journeys") && response.request().method() === "POST")
  await page.getByRole("button", { name: "Save journey", exact: true }).click()
  expect((await rejected).status()).toBe(409)
  await expect(page.getByRole("alert")).toContainText("Choose another slug")
  await expect(page.getByLabel("Title", { exact: true })).toHaveValue("My new journey")
  await expect(page.getByLabel("Place", { exact: true })).toHaveValue("Fictional city")
  await expect(page.getByLabel("Slug", { exact: true })).toHaveValue("taken-slug")

  const rows = await (await page.request.get("/api/admin/journeys")).json()
  expect(rows).toHaveLength(2)
  const editedRow = rows.find((row: { id: string }) => row.id === id)
  const neighbourRow = rows.find((row: { id: string }) => row.id !== id)
  expect(editedRow.title).toBe("My new journey")
  expect(editedRow.slug).toBe(original.slug)
  expect(editedRow.revision).toBe(original.revision)
  expect(neighbourRow.slug).toBe("taken-slug")
  expect(neighbourRow.title).toBe("Collision neighbour")

  await page.getByLabel("Slug", { exact: true }).fill("unique-edited-slug")
  await page.getByRole("button", { name: "Save journey", exact: true }).click()
  await expect(page.getByRole("heading", { name: "My new journey", exact: true })).toBeVisible()
  const saved = await (await page.request.get(`/api/admin/journeys/${id}`)).json()
  expect(saved.slug).toBe("unique-edited-slug")
  expect(saved.revision).toBe(original.revision + 1)
  expect(saved.id).toBe(original.id)
  expect(saved.journal_id).toBe(original.journal_id)
  const neighbourAfter = await (await page.request.get(`/api/admin/journeys/${neighbourRow.id}`)).json()
  expect(neighbourAfter.slug).toBe("taken-slug")
  expect(neighbourAfter.title).toBe("Collision neighbour")
})

test("new memento keyboard choice, dirty return, pending navigation and retry retain one draft", async ({ page }) => {
  await createEmptyJourney(page)
  const journeyId = page.url().split("/").pop()!
  await page.getByRole("button", { name: "Add memento", exact: true }).click()
  const kind = page.getByRole("button", { name: "Kind", exact: true })
  await kind.focus()
  await kind.press("Enter")
  await expect(page.getByRole("listbox")).toBeVisible()
  await page.keyboard.type("transit")
  await page.keyboard.press("Enter")
  await expect(kind).toHaveText("transit")
  const title = page.getByLabel("Title", { exact: true })
  await title.fill("A remembered ride")
  expect((await title.boundingBox())!.height).toBeGreaterThanOrEqual(40)
  await page.getByRole("button", { name: "Return", exact: true }).click()
  await decideDiscard(page, false)
  await expect(title).toHaveValue("A remembered ride")
  let release!: () => void
  const held = new Promise<void>((resolve) => {
    release = resolve
  })
  const ids: string[] = []
  await page.route("**/api/admin/mementos", async (route) => {
    ids.push(route.request().postDataJSON().id)
    if (ids.length === 1) {
      await held
      await route.fulfill({ status: 503, contentType: "application/json", body: JSON.stringify({ error: "Synthetic temporary failure" }) })
    } else await route.continue()
  })
  const create = page.locator("form button[type=submit]")
  expect((await create.boundingBox())!.height).toBeGreaterThanOrEqual(40)
  try {
    await create.click()
    await expect(create).toBeDisabled()
    await expect(kind).toBeDisabled()
    await expect(title).toBeDisabled()
    await page.getByRole("link", { name: "Site & Deploy", exact: true }).click()
    await expect(page).toHaveURL(/\/memento\/new$/)
    expect(ids).toHaveLength(1)
  } finally {
    release()
  }
  await expect(page.getByRole("alert")).toContainText("Synthetic temporary failure")
  await expect(title).toHaveValue("A remembered ride")
  await create.click()
  await expect(page).toHaveURL(/\/memento\/[0-9a-f-]+$/)
  expect(ids).toHaveLength(2)
  expect(ids[0]).toBe(ids[1])
  expect(await (await page.request.get(`/api/admin/journeys/${journeyId}/mementos`)).json()).toHaveLength(1)
})
