import { test, expect } from "./fixtures"
import { chooseSelect } from "./select-field"

for (const [locale, settingsName, journeysName, newName, createName, titleName] of [
  ["ja", "設定", "旅程", "新しい旅程", "旅程を作成", "タイトル"],
  ["zh", "设置", "旅程", "新建旅程", "创建旅程", "标题"],
] as const) {
  test(`settings persist ${locale} and the full creation page reflows`, async ({ page }, info) => {
    await page.setViewportSize({ width: 900, height: 600 })
    await page.goto("/")
    await page.getByRole("button", { name: "Settings", exact: true }).click()
    await chooseSelect(page, page.getByRole("button", { name: "Language", exact: true }), locale)
    await page.keyboard.press("Escape")
    await expect(page.getByRole("button", { name: settingsName, exact: true })).toBeFocused()
    await page.reload()
    await expect(page.getByRole("heading", { name: journeysName, exact: true })).toBeVisible()
    await page.getByRole("button", { name: newName, exact: true }).click()
    await expect(page).toHaveURL(/#\/journey\/new$/)
    await expect(page.getByRole("dialog")).toHaveCount(0)
    const sheet = page.getByRole("main")
    await sheet.getByLabel(titleName, { exact: true }).fill("京都の長い旅程タイトル · 很长的旅行标题 · A long synthetic journey title")
    await expect(sheet.getByRole("button", { name: createName, exact: true })).toBeVisible()
    await page.screenshot({ path: info.outputPath(`creation-${locale}.png`) })
  })
}

test("dark library and settings use the system palette", async ({ page }, info) => {
  await page.emulateMedia({ colorScheme: "dark", reducedMotion: "reduce" })
  await page.setViewportSize({ width: 1440, height: 900 })
  await page.goto("/")
  await expect(page.getByRole("heading", { name: "Journeys", exact: true })).toBeVisible()
  const content = page.getByRole("main")
  expect(await content.evaluate((element) => getComputedStyle(element).backgroundColor)).toBe("rgb(32, 34, 38)")
  await page.screenshot({ path: info.outputPath("library-dark.png") })
  await page.getByRole("button", { name: "Settings", exact: true }).click()
  await page.screenshot({ path: info.outputPath("settings-dark.png") })
})

test("studio appearance persists independently of published settings and follows system changes", async ({ page }, info) => {
  await page.emulateMedia({ colorScheme: "dark", reducedMotion: "reduce" })
  await page.goto("/#/site")
  const before = await page.request.get("/api/admin/site-settings")
  expect(before.ok()).toBe(true)
  const published = await before.json()
  const output = page.getByRole("region", { name: "Output directory", exact: true })
  await expect(output).toBeVisible()
  await expect(output).toHaveCSS("background-color", "rgb(41, 44, 49)")
  await page.getByRole("button", { name: "Settings", exact: true }).click()
  const appearance = page.getByRole("group", { name: "Appearance", exact: true })
  await expect(appearance.getByRole("button", { name: "System", exact: true })).toHaveAttribute("aria-pressed", "true")
  for (const name of ["System", "Light", "Dark"]) await expect(appearance.getByRole("button", { name, exact: true }).locator("svg")).toBeVisible()
  await appearance.getByRole("button", { name: "Light", exact: true }).click()
  await expect(output).toHaveCSS("background-color", "rgb(255, 255, 255)")
  await appearance.getByRole("button", { name: "Dark", exact: true }).click()
  await expect(output).toHaveCSS("background-color", "rgb(41, 44, 49)")
  await page.keyboard.press("Escape")
  await expect(page.getByRole("button", { name: "Settings", exact: true })).toBeFocused()
  await page.emulateMedia({ colorScheme: "light" })
  await page.reload()
  await expect(output).toHaveCSS("background-color", "rgb(41, 44, 49)")
  await page.getByRole("button", { name: "Settings", exact: true }).click()
  await expect(appearance.getByRole("button", { name: "Dark", exact: true })).toHaveAttribute("aria-pressed", "true")
  await appearance.getByRole("button", { name: "System", exact: true }).click()
  await expect(output).toHaveCSS("background-color", "rgb(255, 255, 255)")
  await page.emulateMedia({ colorScheme: "dark" })
  await expect(output).toHaveCSS("background-color", "rgb(41, 44, 49)")
  await page.keyboard.press("Escape")
  const after = await page.request.get("/api/admin/site-settings")
  expect(await after.json()).toEqual(published)
  await page.screenshot({ path: info.outputPath("site-dark.png") })
})

for (const width of [1100, 720]) {
  test(`site folder and build controls stay aligned at ${width}px`, async ({ page }, info) => {
    await page.setViewportSize({ width, height: 760 })
    await page.emulateMedia({ colorScheme: "dark" })
    await page.goto("/#/site")
    const output = page.getByRole("region", { name: "Output directory", exact: true })
    await expect(output).toBeVisible()
    const folder = output.getByRole("button", { name: "Change location…", exact: true })
    await expect(folder).toHaveAttribute("data-slot", "button")
    const build = page.getByRole("button", { name: "Build site", exact: true })
    await expect(build).toHaveAttribute("data-slot", "button")
    for (const control of [folder, build]) {
      const box = await control.boundingBox()
      expect(box).not.toBeNull()
      expect(box!.x).toBeGreaterThanOrEqual(0)
      expect(box!.x + box!.width).toBeLessThanOrEqual(width)
    }
    expect(await page.evaluate(() => document.documentElement.scrollWidth)).toBeLessThanOrEqual(width)
    await page.screenshot({ path: info.outputPath(`site-${width}-dark.png`) })
  })
}

test("creation actions remain reachable with 200% CSS zoom", async ({ page }, info) => {
  await page.setViewportSize({ width: 1100, height: 760 })
  await page.goto("/")
  await page.evaluate(() => {
    document.documentElement.style.zoom = "2"
  })
  await page.getByRole("button", { name: "New journey", exact: true }).click()
  await expect(page).toHaveURL(/#\/journey\/new$/)
  const sheet = page.getByRole("main")
  const box = await sheet.getByRole("button", { name: "Create journey", exact: true }).boundingBox()
  expect(box).not.toBeNull()
  expect(box!.y).toBeGreaterThanOrEqual(0)
  expect(box!.y + box!.height).toBeLessThanOrEqual(760)
  await page.screenshot({ path: info.outputPath("creation-zoom-200.png") })
})
