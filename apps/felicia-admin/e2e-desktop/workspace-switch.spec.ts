import { test, expect } from "./fixtures"

test.use({ userAgent: "Mozilla/5.0 (Macintosh; Intel Mac OS X) AppleWebKit/605.1.15 wails.io" })

test("sample opens and returns in the same page without changing the original workspace", async ({ page, context }) => {
  await page.route("**/wails/runtime.js", (route) => route.fulfill({ contentType: "application/javascript", body: "" }))
  const response = await page.request.post("/api/admin/journeys", {
    data: { title: "Synthetic original", place: "Fictional place", slug: "synthetic-original", date_start: "2026-06-01", date_end: "2026-06-02" },
  })
  expect(response.status()).toBe(200)
  await page.goto("/")
  await expect(page.getByRole("link", { name: /Synthetic original/ })).toBeVisible()
  const origin = new URL(page.url()).origin
  let popups = 0
  page.on("popup", () => popups++)
  await page.getByRole("button", { name: "Open sample trip", exact: true }).click()
  await expect(page.getByRole("region", { name: "Sample workspace", exact: true })).toBeVisible()
  await expect(page.getByRole("link", { name: /A Kyoto afternoon/ })).toBeVisible()
  await expect(page.getByRole("link", { name: /Synthetic original/ })).toHaveCount(0)
  await expect(page.getByRole("button", { name: "Scan trip folder", exact: true })).toHaveCount(0)
  expect(context.pages()).toHaveLength(1)
  expect(new URL(page.url()).origin).toBe(origin)
  await page.getByRole("button", { name: "Return", exact: true }).click()
  await expect(page.getByRole("region", { name: "Sample workspace", exact: true })).toHaveCount(0)
  await expect(page.getByRole("link", { name: /Synthetic original/ })).toBeVisible()
  await expect(page.getByRole("link", { name: /A Kyoto afternoon/ })).toHaveCount(0)
  await expect(page.getByRole("button", { name: "Open sample trip", exact: true })).toBeVisible()
  expect(context.pages()).toHaveLength(1)
  expect(popups).toBe(0)
})

test("cancelled dirty-page navigation never switches the backend workspace", async ({ page }) => {
  await page.route("**/wails/runtime.js", (route) => route.fulfill({ contentType: "application/javascript", body: "" }))
  await page.goto("/#/journey/new")
  await page.getByLabel("Title", { exact: true }).fill("Keep this unsaved title")
  await page.getByRole("button", { name: "Settings", exact: true }).click()
  page.once("dialog", (dialog) => dialog.dismiss())
  await page.getByRole("button", { name: "Open sample trip", exact: true }).click()
  await expect(page.getByLabel("Title", { exact: true })).toHaveValue("Keep this unsaved title")
  expect((await (await page.request.get("/api/desktop/workspace")).json()).sample).toBe(false)
})
