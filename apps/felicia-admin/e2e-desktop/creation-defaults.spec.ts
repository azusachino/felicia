import { test, expect } from "./fixtures"

test.use({ sampleMode: true, timezoneId: "America/Los_Angeles", userAgent: "Mozilla/5.0 (Macintosh; Intel Mac OS X) AppleWebKit/605.1.15 wails.io" })

test("Create succeeds without touching the actual local-date defaults", async ({ page }) => {
  await page.route("**/wails/runtime.js", (route) => route.fulfill({ contentType: "application/javascript", body: "" }))
  await page.clock.setFixedTime(new Date("2026-01-02T00:30:00Z"))
  await page.goto("/#/journey/new")
  await expect(page.locator('[name="date_start"]')).toHaveValue("2026-01-01")
  await expect(page.locator('[name="date_end"]')).toHaveValue("2026-01-01")
  await page.getByLabel("Title", { exact: true }).fill("Untouched date defaults")
  await page.getByLabel("Place", { exact: true }).fill("Fictional place")
  await page.getByRole("button", { name: "Create journey", exact: true }).click()
  await expect(page.getByRole("heading", { name: "Untouched date defaults", exact: true })).toBeVisible()
  const journeys = await (await page.request.get("/api/admin/journeys")).json()
  const journey = journeys.find((item: { title: string }) => item.title === "Untouched date defaults")
  expect(journey.date_start.slice(0, 10)).toBe("2026-01-01")
  expect(journey.date_end.slice(0, 10)).toBe("2026-01-01")
  await page.reload()
  await expect(page.getByRole("heading", { name: "Untouched date defaults", exact: true })).toBeVisible()
})

test("untouched default dates do not make a new journey dirty", async ({ page }) => {
  await page.route("**/wails/runtime.js", (route) => route.fulfill({ contentType: "application/javascript", body: "" }))
  let dialogs = 0
  page.on("dialog", async (dialog) => {
    dialogs++
    await dialog.dismiss()
  })
  await page.goto("/#/journey/new")
  await page.getByRole("button", { name: "Back to journeys", exact: true }).click()
  await expect(page.getByRole("heading", { name: "Journeys", exact: true })).toBeVisible()
  expect(dialogs).toBe(0)
  expect((await (await page.request.get("/api/admin/journeys")).json()).length).toBe(1)
})
