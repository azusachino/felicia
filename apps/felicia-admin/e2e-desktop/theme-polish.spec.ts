import { test, expect } from "./fixtures"

test.use({ sampleMode: true, userAgent: "Mozilla/5.0 (Macintosh; Intel Mac OS X) AppleWebKit/605.1.15 wails.io" })

for (const appearance of ["Light", "Dark"] as const) {
  for (const width of [1100, 720]) {
    test(`${appearance} editor uses coherent surfaces and keeps Save visible at ${width}px`, async ({ page }, info) => {
      await page.route("**/wails/runtime.js", (route) => route.fulfill({ contentType: "application/javascript", body: "" }))
      await page.setViewportSize({ width, height: 760 })
      await page.goto("/")
      await page.getByRole("button", { name: "Settings", exact: true }).click()
      await page.getByRole("button", { name: appearance, exact: true }).click()
      await page.keyboard.press("Escape")
      await page.getByRole("link", { name: /A Kyoto afternoon/ }).click()
      await page.getByRole("link", { name: /A little paper keepsake/ }).click()
      const save = page.getByRole("button", { name: "Save", exact: true })
      await expect(save).toBeInViewport()
      const title = page.getByRole("textbox", { name: "Title", exact: true })
      const ratios = await title.evaluate((input) => {
        const luminance = (color: string) => {
          const rgb = color
            .match(/[\d.]+/g)!
            .slice(0, 3)
            .map(Number)
            .map((v) => {
              const s = v / 255
              return s <= 0.04045 ? s / 12.92 : ((s + 0.055) / 1.055) ** 2.4
            })
          return rgb[0] * 0.2126 + rgb[1] * 0.7152 + rgb[2] * 0.0722
        }
        const contrast = (a: string, b: string) => {
          const x = luminance(a),
            y = luminance(b)
          return (Math.max(x, y) + 0.05) / (Math.min(x, y) + 0.05)
        }
        const label = input.closest("label")!
        const content = document.querySelector(".content")!
        const badge = document.querySelector(".editor-header .badge")!
        const fieldStyle = getComputedStyle(input)
        return [
          contrast(getComputedStyle(label).color, getComputedStyle(content).backgroundColor),
          contrast(fieldStyle.color, fieldStyle.backgroundColor),
          contrast(getComputedStyle(badge).color, getComputedStyle(badge).backgroundColor),
        ]
      })
      for (const ratio of ratios) expect(ratio).toBeGreaterThanOrEqual(4.5)
      await title.fill("Unsaved sample title")
      await expect(page.getByRole("status")).toContainText("Unsaved changes")
      await page.locator(".content").evaluate((element) => {
        element.scrollTop = 1400
      })
      await expect(save).toBeInViewport()
      await page.screenshot({ path: info.outputPath("theme-editor.png") })
    })
  }
}
