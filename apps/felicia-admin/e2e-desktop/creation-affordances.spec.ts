import { test, expect } from "./fixtures"
import { setDate } from "./date-field"

test.use({ userAgent: "Mozilla/5.0 (Macintosh; Intel Mac OS X) AppleWebKit/605.1.15 wails.io" })

test.describe("isolated sample creation", () => {
  test.use({ sampleMode: true })
  test("New journey opens a clickable validated creation page, not folder import", async ({ page }) => {
    await page.route("**/wails/runtime.js", (route) => route.fulfill({ contentType: "application/javascript", body: "" }))
    await page.goto("/")
    await expect(page.getByRole("region", { name: "Sample workspace", exact: true })).toBeVisible()
    await expect(page.getByRole("button", { name: "Scan trip folder", exact: true })).toHaveCount(0)
    await page.getByRole("button", { name: "New journey", exact: true }).click()
    await expect(page.getByRole("heading", { name: "Create a new journey", exact: true })).toBeVisible()
    await expect(page.getByRole("dialog", { name: "Scan a trip folder", exact: true })).toBeHidden()
    const create = page.getByRole("button", { name: "Create journey", exact: true })
    await expect(create).toBeEnabled()
    await create.click()
    await expect(page.getByLabel("Title", { exact: true })).toBeFocused()
    expect((await (await page.request.get("/api/admin/journeys")).json()).length).toBe(1)
    await page.getByLabel("Title", { exact: true }).fill("Created in the sample")
    await page.getByLabel("Place", { exact: true }).fill("Fictional place")
    await setDate(page.getByRole("group", { name: "Start date", exact: true }), "2026-06-01")
    await setDate(page.getByRole("group", { name: "End date", exact: true }), "2026-06-02")
    await create.click()
    await expect(page.getByRole("heading", { name: "Created in the sample", exact: true })).toBeVisible()
    await page.reload()
    await expect(page.getByRole("heading", { name: "Created in the sample", exact: true })).toBeVisible()
    expect((await (await page.request.get("/api/admin/journeys")).json()).length).toBe(2)
    expect((await (await page.request.get("/api/desktop/workspace")).json()).sample).toBe(true)
  })
})

test("folder import explains an empty path instead of leaving Scan silently disabled", async ({ page }) => {
  await page.route("**/wails/runtime.js", (route) => route.fulfill({ contentType: "application/javascript", body: "" }))
  await page.goto("/")
  await page.getByRole("button", { name: "Scan trip folder", exact: true }).click()
  const dialog = page.getByRole("dialog", { name: "Scan a trip folder", exact: true })
  const scan = dialog.getByRole("button", { name: "Scan and preview", exact: true })
  await expect(scan).toBeEnabled()
  await scan.click()
  await expect(dialog.getByRole("alert")).toContainText("Choose or enter a trip folder before scanning")
})
