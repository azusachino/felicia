import { test, expect } from "./fixtures"
import { chooseSelect } from "./select-field"
import { message, type Locale } from "../src/i18n"

for (const locale of ["en", "ja", "zh"] as Locale[]) {
  for (const width of [1100, 720]) {
    test(`shared Site controls save and remain usable in ${locale} at ${width}px`, async ({ page }) => {
      await page.emulateMedia({ colorScheme: width === 720 ? "dark" : "light" })
      await page.setViewportSize({ width, height: 760 })
      await page.goto("/")
      await expect(page.getByRole("heading", { name: "Journeys", exact: true })).toBeVisible()
      if (locale !== "en") {
        await page.getByRole("button", { name: "Settings", exact: true }).click()
        await chooseSelect(page, page.getByRole("button", { name: "Language", exact: true }), locale)
        await page.keyboard.press("Escape")
      }
      await page.goto("/#/site")
      const title = page.getByLabel(message(locale, "admin.site.title"), { exact: true })
      await expect(title).toBeVisible()
      await title.fill(`Synthetic studio ${locale}`)
      const language = page.getByRole("button", { name: message(locale, "admin.site.default_language"), exact: true })
      const theme = page.getByRole("button", { name: message(locale, "admin.site.default_theme"), exact: true })
      const accent = page.getByLabel(message(locale, "admin.site.accent_color"), { exact: true })
      for (const control of [title, language, theme, accent]) {
        const box = (await control.boundingBox())!
        expect(box.height).toBeGreaterThanOrEqual(40)
        expect(box.x).toBeGreaterThanOrEqual(0)
        expect(box.x + box.width).toBeLessThanOrEqual(width)
      }
      await chooseSelect(page, language, "zh")
      await chooseSelect(page, theme, "light")
      await expect(page.locator(".design-card")).toHaveAttribute("aria-pressed", "true")
      const saved = page.waitForResponse((response) => response.url().endsWith("/api/admin/site-settings") && response.request().method() === "PUT")
      await page.getByRole("button", { name: message(locale, "admin.site.save_settings"), exact: true }).click()
      expect((await saved).status()).toBe(200)
      await page.reload()
      await expect(title).toHaveValue(`Synthetic studio ${locale}`)
      await expect(page.locator("html")).toHaveAttribute("lang", locale)
      const settings = await (await page.request.get("/api/admin/site-settings")).json()
      expect(settings.default_language).toBe("zh")
      expect(settings.default_theme).toBe("light")
    })
  }
}
