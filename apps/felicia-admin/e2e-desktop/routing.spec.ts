import { randomUUID } from "node:crypto"
import type { Page } from "@playwright/test"
import type { AdminJourney } from "../src/api"
import { test, expect } from "./fixtures"

async function createJourney(page: Page, slug: string, title: string): Promise<string> {
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
  await page.getByLabel("Essay", { exact: true }).fill("Unsaved synthetic draft")
  await page.getByRole("button", { name: "Settings", exact: true }).click()
  await page.getByRole("combobox", { name: "Language", exact: true }).selectOption("ja")
  await expect(page.getByLabel("文章", { exact: true })).toHaveValue("Unsaved synthetic draft")
  await page.getByRole("combobox", { name: "言語", exact: true }).selectOption("en")
  await page.keyboard.press("Escape")
  await expect(page.getByLabel("Essay", { exact: true })).toHaveValue("Unsaved synthetic draft")
  await page.getByRole("link", { name: /Back to journey/ }).click()
  await expect(page.getByRole("heading", { name: "Routing editor journey", exact: true })).toBeVisible()
})
