import { test, expect } from "./fixtures"

test.use({ sampleMode: true, timezoneId: "Asia/Tokyo" })
const journeyId = "0190cbde-f300-7000-8000-000000000002"

test("sample source actions produce data and persisted idempotent intake, and snap works", async ({ page }) => {
  await page.goto(`/#/journey/${journeyId}`)
  for (const [name, suffix] of [
    ["Sync route", "sync-route"],
    ["Preview visits", "visits"],
    ["Preview photo tray", "tray"],
    ["Plan intake", "intake/plan"],
  ]) {
    const completed = page.waitForResponse((response) => response.url().endsWith(`/${journeyId}/${suffix}`))
    await page.getByRole("button", { name, exact: true }).click()
    expect((await completed).status()).toBe(200)
    await expect(page.locator('[role="status"]').filter({ hasText: /\d/ }).first()).toBeVisible()
    await expect(page.locator('[role="alert"]')).toHaveCount(0)
  }
  await expect(page.locator(".preview-list")).toHaveCount(2)
  const stops = await (await page.request.get(`/api/admin/journeys/${journeyId}/stop-candidates`)).json()
  expect(stops.length).toBeGreaterThan(0)
  await expect(page.locator(".candidate-list li")).toHaveCount(stops.length)
  const replanned = page.waitForResponse((response) => response.url().endsWith(`/${journeyId}/intake/plan`))
  await page.getByRole("button", { name: "Plan intake", exact: true }).click()
  expect((await replanned).status()).toBe(200)
  const repeated = await (await page.request.get(`/api/admin/journeys/${journeyId}/stop-candidates`)).json()
  expect(repeated.map((stop: { id: string }) => stop.id).sort()).toEqual(stops.map((stop: { id: string }) => stop.id).sort())
  await page.getByRole("link", { name: /A little paper keepsake/ }).click()
  const snapResponse = page.waitForResponse((response) => response.url().endsWith(`/${journeyId}/snap`))
  await page.getByRole("button", { name: "Snap to route", exact: true }).click()
  expect((await snapResponse).status()).toBe(200)
  await expect(page.locator('[role="alert"]')).toHaveCount(0)
})

for (const locale of ["en", "ja", "zh"]) {
  test(`journey library dates render yyyy-MM-DD and create untouched in ${locale}`, async ({ page }) => {
    await page.addInitScript((locale) => localStorage.setItem("felicia.admin.locale", locale), locale)
    await page.clock.setFixedTime(new Date("2026-01-02T12:30:00Z"))
    await page.goto("/#/journey/new")
    const fields = page.locator("[data-date-field-input]")
    await expect(fields).toHaveCount(2)
    for (const field of await fields.all()) {
      expect(
        await field.evaluate((element) =>
          Array.from(element.children)
            .map((child) => child.textContent)
            .join(""),
        ),
      ).toBe("2026-01-02")
      expect(await field.locator("[data-segment]").evaluateAll((segments) => segments.map((segment) => segment.getAttribute("data-segment")))).toEqual(["year", "month", "day"])
    }
    await expect(page.locator('[name="date_start"]')).toHaveValue("2026-01-02")
    await expect(page.locator('[name="date_end"]')).toHaveValue("2026-01-02")
    await page.locator("#new-title").fill("Library date defaults")
    await page.locator("#new-place").fill("Fictional place")
    await page.locator('button[type="submit"]').click()
    await expect(page.getByRole("heading", { name: "Library date defaults", exact: true })).toBeVisible()
  })
}
