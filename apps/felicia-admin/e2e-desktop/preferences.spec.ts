import { test, expect } from "./fixtures"

for (const [locale, settingsName, journeysName, newName, createName, titleName] of [
  ["ja", "設定", "旅程", "新しい旅程", "旅程を作成", "タイトル"],
  ["zh", "设置", "旅程", "新建旅程", "创建旅程", "标题"],
] as const) {
  test(`settings persist ${locale} and the creation sheet reflows`, async ({ page }, info) => {
    await page.setViewportSize({ width: 900, height: 600 })
    await page.goto("/")
    await page.getByRole("button", { name: "Settings", exact: true }).click()
    await page.getByRole("combobox").selectOption(locale)
    await page.keyboard.press("Escape")
    await expect(page.getByRole("button", { name: settingsName, exact: true })).toBeFocused()
    await page.reload()
    await expect(page.getByRole("heading", { name: journeysName, exact: true })).toBeVisible()
    await page.getByRole("button", { name: newName, exact: true }).click()
    const sheet = page.getByRole("dialog")
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

test("creation actions remain reachable with 200% CSS zoom", async ({ page }, info) => {
  await page.setViewportSize({ width: 1100, height: 760 })
  await page.goto("/")
  await page.evaluate(() => {
    document.documentElement.style.zoom = "2"
  })
  await page.getByRole("button", { name: "New journey", exact: true }).click()
  const sheet = page.getByRole("dialog")
  const box = await sheet.getByRole("button", { name: "Create journey", exact: true }).boundingBox()
  expect(box).not.toBeNull()
  expect(box!.y).toBeGreaterThanOrEqual(0)
  expect(box!.y + box!.height).toBeLessThanOrEqual(760)
  await page.screenshot({ path: info.outputPath("creation-zoom-200.png") })
})
