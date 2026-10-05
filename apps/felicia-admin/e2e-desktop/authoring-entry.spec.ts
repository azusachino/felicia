import { test, expect } from "./fixtures"

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
  const original = await (await page.request.get(`/api/admin/journeys/${id}`)).json()
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
  page.once("dialog", (dialog) => dialog.dismiss())
  await page.getByRole("button", { name: "Back to journey", exact: true }).click()
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
  page.once("dialog", (dialog) => dialog.dismiss())
  await page.getByRole("button", { name: "Return", exact: true }).click()
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
