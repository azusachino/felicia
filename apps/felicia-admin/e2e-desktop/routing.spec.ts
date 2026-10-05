import { randomUUID } from "node:crypto"
import type { Page } from "@playwright/test"
import type { AdminJourney } from "../src/api"
import { test, expect } from "./fixtures"
import { chooseSelect } from "./select-field"
import { decideDiscard } from "./discard-dialog"

async function createJourney(page: Page, slug: string, title: string): Promise<string> {
  // page.goto('/') returns before the client-only framework has initialized.
  // Establish UI readiness before a subsequent same-document hash navigation.
  await expect(page.getByRole("heading", { name: "Journeys", exact: true })).toBeVisible()
  const response = await page.request.post("/api/admin/journeys", {
    data: { slug, title, place: "Synthetic city", date_start: "2026-03-20", date_end: "2026-03-21" },
  })
  expect(response.ok(), await response.text()).toBe(true)
  const created: Pick<AdminJourney, "id"> = await response.json()
  return created.id
}

function encodedId(id: string): string {
  return `%${id.charCodeAt(0).toString(16)}${id.slice(1)}`
}

test("empty, unknown and malformed URLs recover to the library", async ({ page }) => {
  for (const hash of ["", "#", "#/", "#/unknown", "#/journey", "#/journey/", "#/journey/a/memento/", "#/journey/%ZZ", "#/journey/a/memento/%ZZ"]) {
    await page.goto(`/${hash}`)
    await expect(page.getByRole("heading", { name: "Journeys", exact: true })).toBeVisible()
    await expect(page.getByRole("link", { name: "Journeys", exact: true })).toHaveAttribute("aria-current", "page")
  }
})

test("deep links decode IDs, accept trailing slashes and preserve history", async ({ page }) => {
  await page.goto("/")
  const first = await createJourney(page, "routing-first", "Routing first")
  const second = await createJourney(page, "routing-second", "Routing second")
  await page.goto(`/#/journey/${encodedId(first)}/?tab=overview`)
  await expect(page.getByRole("heading", { name: "Routing first", exact: true })).toBeVisible()
  await page.getByRole("link", { name: "Site & Deploy", exact: true }).click()
  await expect(page.getByRole("heading", { name: "Site & Deploy", exact: true })).toBeVisible()
  await expect(page.getByRole("link", { name: "Site & Deploy", exact: true })).toHaveAttribute("aria-current", "page")
  await page.goBack()
  await expect(page.getByRole("heading", { name: "Routing first", exact: true })).toBeVisible()
  await page.goForward()
  await expect(page.getByRole("heading", { name: "Site & Deploy", exact: true })).toBeVisible()
  await page.goto("/#/site/?tab=output")
  await expect(page.getByRole("heading", { name: "Site & Deploy", exact: true })).toBeVisible()
  await page.goto(`/#/journey/${first}`)
  await expect(page.getByRole("heading", { name: "Routing first", exact: true })).toBeVisible()
  await page.evaluate((id) => {
    location.hash = `#/journey/${id}`
  }, second)
  await expect(page.getByRole("heading", { name: "Routing second", exact: true })).toBeVisible()
  await expect(page.getByRole("heading", { name: "Routing first", exact: true })).toHaveCount(0)
  await page.reload()
  await expect(page.getByRole("heading", { name: "Routing second", exact: true })).toBeVisible()
})

test("memento parameters and live locale updates keep unsaved input", async ({ page }) => {
  // The S1 desktop handler lacks the per-memento photo-list endpoint.
  // Supply only that response to isolate routing/locale behavior here;
  // the web-host closed loop covers authoring with the real photo API.
  await page.route("**/api/admin/mementos/*/photos", (route) => route.fulfill({ json: [] }))
  await page.goto("/")
  const journeyId = await createJourney(page, "routing-editor", "Routing editor journey")
  const id = randomUUID()
  const response = await page.request.post("/api/admin/mementos", {
    data: { id, journey_id: journeyId, kind: "goods", seq: 1, title: "Routing memento", place: "Synthetic city", state: "draft", kind_data: { name: "Synthetic postcard" } },
  })
  expect(response.ok(), await response.text()).toBe(true)
  await page.goto(`/#/journey/${encodedId(journeyId)}/memento/${encodedId(id)}/`)
  await expect(page.getByLabel("Title", { exact: true })).toHaveValue("Routing memento")
  const back = page.getByRole("button", { name: "Back to journey", exact: true })
  await expect(back).toHaveText("Return")
  await expect(back.locator('svg[aria-hidden="true"]')).toHaveCount(1)
  await page.getByLabel("Essay", { exact: true }).fill("Unsaved synthetic draft")
  await page.getByRole("button", { name: "Settings", exact: true }).click()
  await chooseSelect(page, page.getByRole("button", { name: "Language", exact: true }), "ja")
  await expect(page.getByLabel("文章", { exact: true })).toHaveValue("Unsaved synthetic draft")
  await chooseSelect(page, page.getByRole("button", { name: "言語", exact: true }), "en")
  await page.keyboard.press("Escape")
  await expect(page.getByLabel("Essay", { exact: true })).toHaveValue("Unsaved synthetic draft")
  await page.getByRole("button", { name: /Back to journey/ }).click()
  await decideDiscard(page, false)
  await expect(page.getByLabel("Essay", { exact: true })).toHaveValue("Unsaved synthetic draft")
  const reloadDialog = page.waitForEvent("dialog")
  const reloadAttempt = page.evaluate(() => location.reload())
  const unload = await reloadDialog
  expect(unload.type()).toBe("beforeunload")
  await unload.dismiss()
  await reloadAttempt
  await expect(page.getByLabel("Essay", { exact: true })).toHaveValue("Unsaved synthetic draft")
  await page.getByRole("button", { name: /Back to journey/ }).click()
  await decideDiscard(page, true)
  await expect(page.getByRole("heading", { name: "Routing editor journey", exact: true })).toBeVisible()
  const stored = await page.request.get(`/api/admin/mementos/${id}`)
  expect(stored.ok()).toBe(true)
  expect((await stored.json()).essay ?? "").toBe("")
})

test("keyboard save preserves edits typed while the submitted draft is in flight", async ({ page }) => {
  await page.route("**/api/admin/mementos/*/photos", (route) => route.fulfill({ json: [] }))
  await page.goto("/")
  const journeyId = await createJourney(page, "routing-save", "Keyboard save journey")
  const id = randomUUID()
  const created = await page.request.post("/api/admin/mementos", {
    data: {
      id,
      journey_id: journeyId,
      kind: "goods",
      seq: 1,
      title: "Keyboard memento",
      place: "Synthetic city",
      state: "draft",
      kind_data: { name: "Synthetic postcard" },
      geom: { type: "Point", coordinates: [139.7, 35.6] },
      occurred_at: "2026-03-20T12:00:00Z",
      occurred_tz: "UTC",
    },
  })
  expect(created.ok(), await created.text()).toBe(true)
  await page.goto(`/#/journey/${journeyId}/memento/${id}`)
  await expect(page.getByLabel("Title", { exact: true })).toHaveValue("Keyboard memento")
  const saveButton = page
    .getByRole("region", { name: "Save and lifecycle actions" })
    .getByRole("button", { name: /^(Save|Saving…)$/ })
    .first()
  let releaseResponse!: () => void
  const held = new Promise<void>((resolve) => {
    releaseResponse = resolve
  })
  await page.route("**/api/admin/mementos", async (route) => {
    const response = await route.fetch()
    await held
    await route.fulfill({ response })
  })
  await page.getByLabel("Essay", { exact: true }).fill("Submitted keyboard draft")
  await page.keyboard.press("Control+s")
  await expect(saveButton).toBeDisabled()
  await page.getByLabel("Essay", { exact: true }).fill("Later unsaved edit")
  releaseResponse()
  await expect(saveButton).toBeEnabled()
  await expect(page.getByLabel("Essay", { exact: true })).toHaveValue("Later unsaved edit")
  const submitted = await page.request.get(`/api/admin/mementos/${id}`)
  expect((await submitted.json()).essay).toBe("Submitted keyboard draft")
  await page.getByRole("button", { name: /Back to journey/ }).click()
  await decideDiscard(page, false)
  await expect(page.getByLabel("Essay", { exact: true })).toHaveValue("Later unsaved edit")
  await page.unroute("**/api/admin/mementos")
  const saved = page.waitForResponse((response) => response.url().endsWith("/api/admin/mementos") && response.request().method() === "POST")
  await page.keyboard.press("Meta+s")
  expect((await saved).ok()).toBe(true)
  await expect(saveButton).toBeEnabled()
  const stored = await page.request.get(`/api/admin/mementos/${id}`)
  expect((await stored.json()).essay).toBe("Later unsaved edit")
  await page.getByRole("button", { name: /Back to journey/ }).click()
  await expect(page.getByRole("heading", { name: "Keyboard save journey", exact: true })).toBeVisible()
})
