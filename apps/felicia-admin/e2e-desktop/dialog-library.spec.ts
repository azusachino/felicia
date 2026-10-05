import { test, expect } from "./fixtures"

test("scan uses a library dialog, focuses its field and preserves the pending import guard", async ({ page }) => {
  await page.goto("/")
  const opener = page.getByRole("button", { name: "Scan trip folder", exact: true })
  await opener.click()
  const dialog = page.getByRole("dialog", { name: "Scan a trip folder", exact: true })
  const path = dialog.getByLabel("Folder path", { exact: true })
  await expect(path).toBeFocused()
  await page.keyboard.press("Escape")
  await expect(dialog).toBeHidden()
  await expect(opener).toBeFocused()
  await opener.click()
  await page.route("**/api/admin/local-journeys/scan", (route) =>
    route.fulfill({ json: { journey_id: "0190cbde-f300-7000-8000-000000000002", plan: { routes: [], stops: [], mementos: [], issues: [] } } }),
  )
  await path.fill("/synthetic/trip")
  await dialog.getByLabel("Slug", { exact: true }).fill("synthetic-trip")
  await dialog.getByLabel("Title", { exact: true }).fill("Synthetic trip")
  await dialog.getByRole("button", { name: "Scan and preview", exact: true }).click()
  await expect(dialog.getByRole("button", { name: "Confirm import", exact: true })).toBeEnabled()
  let release!: () => void
  const held = new Promise<void>((resolve) => {
    release = resolve
  })
  await page.route("**/api/admin/local-journeys/import", async (route) => {
    await held
    await route.fulfill({ status: 503, json: { error: "Synthetic retry" } })
  })
  await dialog.getByRole("button", { name: "Confirm import", exact: true }).click()
  await expect(dialog.getByRole("button", { name: "Cancel", exact: true })).toBeDisabled()
  await expect(path).toBeDisabled()
  await expect(dialog.getByRole("button", { name: "Scan and preview", exact: true })).toBeDisabled()
  await page.keyboard.press("Escape")
  await expect(dialog).toBeVisible()
  release()
  await expect(dialog.getByRole("alert")).toContainText("Synthetic retry")
  await page.keyboard.press("Escape")
  await expect(dialog).toBeHidden()
  await expect(opener).toBeFocused()
  await expect(page.locator("dialog")).toHaveCount(0)
})

test.describe("preview", () => {
  test.use({ sampleMode: true, userAgent: "Mozilla/5.0 wails.io" })
  test("library preview closes with Escape from its reader and returns focus without losing the draft", async ({ page }) => {
    await page.route("**/wails/runtime.js", (route) => route.fulfill({ contentType: "application/javascript", body: "" }))
    await page.goto("/#/site")
    const title = page.getByRole("textbox", { name: "Title", exact: true })
    await title.fill("Unsaved synthetic identity")
    const opener = page.getByRole("button", { name: "Preview last built site", exact: true })
    await opener.click()
    const dialog = page.getByRole("dialog", { name: "Last built site", exact: true })
    await expect(dialog.locator("iframe")).toBeVisible()
    await page.evaluate(() => {
      const frame = document.querySelector("iframe")!
      window.dispatchEvent(new MessageEvent("message", { origin: "https://untrusted.invalid", source: frame.contentWindow, data: "felicia:preview:escape" }))
      window.dispatchEvent(new MessageEvent("message", { origin: new URL(frame.src).origin, source: window, data: "felicia:preview:escape" }))
    })
    await expect(dialog).toBeVisible()
    const reader = page.frameLocator('iframe[title="Last built site"]')
    await reader.getByRole("button", { name: /^(Light|ライト|浅色)$/ }).click()
    await page.keyboard.press("Escape")
    await expect(dialog).toBeHidden()
    await expect(opener).toBeFocused()
    await expect(title).toHaveValue("Unsaved synthetic identity")
    await expect(page.locator("dialog")).toHaveCount(0)
  })
})
