import { test, expect } from "./fixtures"

test.use({ userAgent: "Mozilla/5.0 (Macintosh; Intel Mac OS X) AppleWebKit/605.1.15 wails.io" })

for (const locale of ["en", "ja", "zh"]) {
  for (const width of [1100, 720]) {
    test(`native preview and sample return fit at ${width}px in ${locale}`, async ({ page }, info) => {
      await page.route("**/wails/runtime.js", (route) => route.fulfill({ contentType: "application/javascript", body: "" }))
      await page.addInitScript((locale) => localStorage.setItem("felicia.admin.locale", locale), locale)
      await page.setViewportSize({ width, height: 760 })
      await page.goto("/")
      await page.locator(".sample-action button").click()
      const banner = page.locator(".sample-banner")
      await expect(banner).toBeVisible()
      const card = (await banner.boundingBox())!
      const button = (await page.locator(".sample-return").boundingBox())!
      expect(button.x).toBeGreaterThanOrEqual(card.x)
      expect(button.x + button.width).toBeLessThanOrEqual(card.x + card.width)
      expect(button.y + button.height).toBeLessThanOrEqual(card.y + card.height)
      expect(button.height).toBeGreaterThanOrEqual(32)
      expect(await banner.evaluate((element) => element.scrollWidth <= element.clientWidth)).toBe(true)
      await page.goto("/#/site")
      const previewLabels = { en: "Preview last built site", ja: "ビルド済みサイトを確認", zh: "预览已构建网站" }
      await page.getByRole("button", { name: previewLabels[locale as keyof typeof previewLabels], exact: true }).click()
      const preview = page.locator('[role="dialog"].compiled-preview')
      await expect(preview).toBeVisible()
      const bounds = (await preview.boundingBox())!
      const toolbar = (await page.locator(".window-toolbar").boundingBox())!
      expect(bounds.y).toBeGreaterThanOrEqual(toolbar.y + toolbar.height + 15)
      expect(bounds.y + bounds.height).toBeLessThanOrEqual(760)
      const title = (await preview.getByRole("heading", { level: 2 }).boundingBox())!
      expect(title.y).toBeGreaterThanOrEqual(toolbar.y + toolbar.height)
      await page.screenshot({ path: info.outputPath("native-preview-position.png") })
    })
  }
}
