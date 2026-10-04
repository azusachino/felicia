import { test, expect } from "./fixtures"

for (const viewport of [
  { width: 1100, height: 760 },
  { width: 900, height: 600 },
  { width: 720, height: 600 },
]) {
  test(`compact shell and full creation page at ${viewport.width}x${viewport.height}`, async ({ page }, info) => {
    await page.setViewportSize(viewport)
    await page.goto("/")
    await expect(page.getByText("YP", { exact: true })).toHaveCount(0)
    await expect(page.getByText("Local workspace", { exact: true })).toHaveCount(0)
    await expect(page.getByRole("button", { name: "Refresh", exact: true })).toHaveCount(0)
    const mark = page.getByRole("img", { name: "Felicia", exact: true })
    await expect(mark).toBeVisible()
    if (viewport.width > 820) {
      const brand = await mark.boundingBox()
      const title = await page.getByRole("heading", { name: "Journeys", exact: true }).boundingBox()
      const action = await page.getByRole("button", { name: "New journey", exact: true }).boundingBox()
      expect(brand).not.toBeNull()
      expect(title).not.toBeNull()
      expect(action).not.toBeNull()
      expect(Math.abs(title!.y + title!.height / 2 - (brand!.y + brand!.height / 2))).toBeLessThanOrEqual(1)
      expect(Math.abs(action!.y + action!.height / 2 - (brand!.y + brand!.height / 2))).toBeLessThanOrEqual(1)
    }
    await expect(page.getByRole("combobox", { name: "Language", exact: true })).toBeHidden()
    const settingsTrigger = page.getByRole("button", { name: "Settings", exact: true })
    await settingsTrigger.click()
    const settings = page.getByRole("dialog", { name: "Settings", exact: true })
    await expect(settings.getByRole("combobox", { name: "Language", exact: true })).toBeVisible()
    await expect(settingsTrigger).toHaveAttribute("aria-expanded", "true")
    await page.keyboard.press("Escape")
    await expect(settings).toBeHidden()
    await expect(settingsTrigger).toBeFocused()
    await settingsTrigger.click()
    await page.getByRole("heading", { name: "Journeys", exact: true }).click()
    await expect(settings).toBeHidden()
    await page.getByRole("button", { name: "New journey", exact: true }).click()
    await expect(page).toHaveURL(/#\/journey\/new$/)
    await expect(page.getByRole("dialog")).toHaveCount(0)
    const sheet = page.getByRole("region", { name: "Create a new journey", exact: true })
    await expect(sheet).toBeVisible()
    await expect(sheet.getByLabel("Title", { exact: true })).toBeFocused()
    const create = sheet.getByRole("button", { name: "Create journey", exact: true })
    const box = await create.boundingBox()
    expect(box).not.toBeNull()
    expect(box!.y + box!.height).toBeLessThanOrEqual(viewport.height)
    expect(await page.evaluate(() => document.documentElement.scrollWidth)).toBeLessThanOrEqual(viewport.width)
    await page.screenshot({ path: info.outputPath("creation-page.png") })
    await sheet.getByRole("button", { name: "Cancel", exact: true }).click()
    await expect(sheet).toBeHidden()
    await expect(page.getByRole("button", { name: "New journey", exact: true })).toBeVisible()
  })
}

test("popover and dialog motion honor reduced-motion preference", async ({ page }) => {
  await page.goto("/")
  const settingsTrigger = page.getByRole("button", { name: "Settings", exact: true })
  await settingsTrigger.click()
  const settings = page.getByRole("dialog", { name: "Settings", exact: true })
  await expect(settings).toBeVisible()
  await settingsTrigger.click()
  await expect(settings).toBeHidden()

  await page.getByRole("button", { name: "Scan trip folder", exact: true }).click()
  const creation = page.getByRole("dialog", { name: "Scan a trip folder", exact: true })
  expect(await creation.evaluate((element) => getComputedStyle(element).transitionDuration)).toContain("0.16s")
  await page.emulateMedia({ reducedMotion: "reduce" })
  expect(await creation.evaluate((element) => getComputedStyle(element).transitionDuration)).toBe("0s")
  await page.keyboard.press("Escape")
  await expect(creation).toBeHidden()

  await settingsTrigger.click()
  await expect(settings).toHaveCSS("transition-duration", "0s")
  await expect(settings).toHaveCSS("animation-name", "none")
})
