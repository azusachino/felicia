import { test, expect } from "./fixtures"
import { message, type Locale } from "../src/i18n"

test.use({ sampleMode: true, userAgent: "Mozilla/5.0 (Macintosh; Intel Mac OS X) AppleWebKit/605.1.15 wails.io" })

for (const locale of ["en", "ja", "zh"] as Locale[]) {
  test(`temporary studio adds generated photos without file selection and curation persists in ${locale}`, async ({ page }) => {
    await page.route("**/wails/runtime.js", (route) => route.fulfill({ contentType: "application/javascript", body: "" }))
    await page.addInitScript((locale) => localStorage.setItem("felicia.admin.locale", locale), locale)
    await page.goto("/#/journey/0190cbde-f300-7000-8000-000000000002/memento/0190cbde-f300-7000-8000-000000000010")
    await expect(page.getByText(message(locale, "admin.mementos.photo_upload_sample_note"), { exact: true })).toBeVisible()
    await expect(page.getByRole("button", { name: message(locale, "admin.mementos.add_photos"), exact: true })).toHaveCount(0)
    await expect(page.locator('input[type="file"]')).toHaveCount(0)
    await expect(page.locator(".photo-row")).toHaveCount(2)
    const response = page.waitForResponse((response) => response.url().endsWith("/photos/sample") && response.request().method() === "POST")
    await page.getByRole("button", { name: message(locale, "admin.mementos.add_sample_photo"), exact: true }).click()
    expect((await response).status()).toBe(201)
    await expect(page.locator(".photo-row")).toHaveCount(3)
    const row = page.locator(".photo-row").last()
    await row.scrollIntoViewIfNeeded()
    await expect(row.getByRole("img")).toHaveJSProperty("naturalWidth", 320)
    const caption = row.getByLabel(message(locale, "admin.mementos.photo_caption"), { exact: true })
    await caption.fill(`Synthetic caption ${locale}`)
    const saved = page.waitForResponse((response) => response.url().endsWith("/api/admin/photos") && response.request().method() === "POST")
    await row.getByRole("button", { name: message(locale, "admin.mementos.photo_save_caption"), exact: true }).click()
    expect((await saved).status()).toBe(200)
    await page.reload()
    await expect(page.locator(".photo-row")).toHaveCount(3)
    await expect(caption).toHaveValue(`Synthetic caption ${locale}`)
  })
}
