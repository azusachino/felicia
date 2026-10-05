import { test, expect } from "./fixtures"

test.use({ sampleMode: true, userAgent: "Mozilla/5.0 (Macintosh; Intel Mac OS X) wails.io" })

const journey = "0190cbde-f300-7000-8000-000000000002"

test.beforeEach(async ({ page }) => {
  await page.route("**/wails/runtime.js", (route) => route.fulfill({ contentType: "application/javascript", body: "" }))
})

test("dirty Add memento Return uses a visible library confirmation without native confirm", async ({ page }) => {
  await page.addInitScript(() => {
    window.confirm = () => false
  })
  await page.goto(`/#/journey/${journey}/memento/new`)
  await page.getByRole("textbox", { name: "Title", exact: true }).fill("Unsaved synthetic memento")
  await page.getByRole("button", { name: "Return", exact: true }).click()
  const prompt = page.getByRole("alertdialog")
  await expect(prompt).toBeVisible()
  await prompt.getByRole("button", { name: "Keep editing", exact: true }).click()
  await expect(page.getByRole("textbox", { name: "Title", exact: true })).toHaveValue("Unsaved synthetic memento")
  await page.getByRole("button", { name: "Return", exact: true }).click()
  await prompt.getByRole("button", { name: "Discard changes", exact: true }).click()
  await expect(page.getByRole("heading", { name: /A Kyoto afternoon/ })).toBeVisible()
})

test("sample output location does not invoke an unavailable browse API", async ({ page }) => {
  const browseRequests: string[] = []
  page.on("request", (request) => {
    if (new URL(request.url()).pathname === "/api/admin/browse") browseRequests.push(request.url())
  })
  await page.goto("/#/site")
  await expect(page.getByText("Temporary workspace output is managed automatically.", { exact: true })).toBeVisible()
  await expect(page.getByRole("button", { name: "Change location…", exact: true })).toHaveCount(0)
  expect(browseRequests).toEqual([])
  await expect(page.getByRole("alert")).toHaveCount(0)
})

test("temporary memento editor exposes generated photo addition without a file picker", async ({ page }) => {
  await page.goto(`/#/journey/${journey}/memento/0190cbde-f300-7000-8000-000000000010`)
  await expect(page.getByRole("button", { name: "Add sample photo", exact: true })).toBeVisible()
  await expect(page.locator('input[type="file"]')).toHaveCount(0)
})
