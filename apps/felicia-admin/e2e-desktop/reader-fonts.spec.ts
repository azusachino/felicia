import { test, expect } from "./fixtures"

test.use({ sampleMode: true, userAgent: "Mozilla/5.0 (Macintosh; Intel Mac OS X) AppleWebKit/605.1.15 wails.io" })

test("last-built sample reader loads its original Latin and Japanese typefaces without external font requests", async ({ page, context }) => {
  const external: string[] = []
  const localFonts: string[] = []
  await context.route("**/*", async (route) => {
    const url = new URL(route.request().url())
    if (url.pathname === "/wails/runtime.js") {
      await route.fulfill({ contentType: "application/javascript", body: "" })
      return
    }
    if (url.protocol === "http:" || url.protocol === "https:") {
      if (url.hostname !== "127.0.0.1" && url.hostname !== "localhost") {
        external.push(url.toString())
        await route.abort()
        return
      }
      if (url.pathname.endsWith(".woff2")) localFonts.push(url.toString())
    }
    await route.continue()
  })
  await page.goto("/#/site")
  await page.getByRole("button", { name: "Preview last built site", exact: true }).click()
  const frame = page.frameLocator('iframe[title="Last built site"]')
  await expect(frame.locator("body")).toContainText("A Kyoto afternoon")
  const loaded = await frame.locator("body").evaluate(async () => {
    const requests = [
      ["Inter", "Kyoto"],
      ["Outfit", "Kyoto"],
      ["Share Tech Mono", "KYOTO"],
      ["Spectral", "Kyoto"],
      ["Zen Old Mincho", "京都の旅"],
    ]
    return Promise.all(requests.map(async ([family, text]) => (await document.fonts.load(`400 16px "${family}"`, text)).length))
  })
  expect(loaded.every((count) => count > 0)).toBe(true)
  expect(localFonts.length).toBeGreaterThanOrEqual(5)
  // Existing map tiles remain an online reader dependency. They were blocked
  // above: typography and published stories must still load without them.
  expect(external.filter((url) => /fonts\.(googleapis|gstatic)\.com/.test(url))).toEqual([])
  expect(external.every((url) => new URL(url).hostname === "server.arcgisonline.com")).toBe(true)
})
