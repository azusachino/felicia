import { test, expect } from "./fixtures"
import { message, type Locale } from "../src/i18n"

test.use({ sampleMode: true, userAgent: "Mozilla/5.0 (Macintosh; Intel Mac OS X) wails.io" })

for (const locale of ["en", "ja", "zh"] as Locale[]) {
  test(`standalone Close stays in an empty isolated studio and sample reopens fresh in ${locale}`, async ({ page }) => {
    await page.route("**/wails/runtime.js", (route) => route.fulfill({ contentType: "application/javascript", body: "" }))
    await page.addInitScript((locale) => localStorage.setItem("felicia.admin.locale", locale), locale)
    await page.goto("/#/")
    const originalURL = page.url()
    if (locale === "en") {
      await page.getByRole("link", { name: /A Kyoto afternoon/ }).click()
      await page.getByRole("link", { name: /A little paper keepsake/ }).click()
      const essay = page.getByLabel("Essay", { exact: true })
      await essay.fill("Unsaved synthetic essay")
      page.once("dialog", (dialog) => dialog.dismiss())
      await page.getByRole("button", { name: "Close", exact: true }).click()
      await expect(essay).toHaveValue("Unsaved synthetic essay")
      expect((await (await page.request.get("/api/desktop/workspace")).json()).sample).toBe(true)
      page.once("dialog", (dialog) => dialog.accept())
      await page.getByRole("link", { name: "Journeys", exact: true }).click()
    }
    const workspace = await (await page.request.get("/api/desktop/workspace")).json()
    expect(workspace).toEqual({ sample: true, return_available: false, isolated: true })
    await page.getByRole("button", { name: message(locale, "admin.sample.close"), exact: true }).click()
    await expect(page.getByText(message(locale, "admin.sample.empty_studio"), { exact: true })).toBeVisible()
    await expect(page.getByText(message(locale, "admin.journeys.empty"), { exact: true })).toBeVisible()
    expect(page.url()).toBe(originalURL)
    expect(await (await page.request.get("/api/admin/journeys")).json()).toEqual([])
    expect(await (await page.request.get("/api/desktop/workspace")).json()).toEqual({ sample: false, return_available: false, isolated: true })
    await expect(page.getByRole("button", { name: message(locale, "admin.connectors.scan_trip_folder"), exact: true })).toHaveCount(0)
    for (const endpoint of ["/api/desktop/pick-folder", "/api/admin/local-journeys/import", "/api/admin/site/output-dir"]) {
      expect((await page.request.post(endpoint, { data: {} })).status()).toBe(403)
    }
    await page.getByRole("button", { name: message(locale, "admin.sample.open"), exact: true }).click()
    await expect(page.getByRole("link", { name: /A Kyoto afternoon/ })).toBeVisible()
    expect((await (await page.request.get("/api/admin/journeys")).json()).length).toBe(1)
    await page.getByRole("button", { name: message(locale, "admin.sample.close"), exact: true }).click()
    await expect(page.getByText(message(locale, "admin.journeys.empty"), { exact: true })).toBeVisible()
  })
}
