import { test, expect } from "./fixtures"
import { writeFile } from "node:fs/promises"

test("creates a journey through labeled fields and persists it after reload", async ({ page }, info) => {
  const errors: string[] = []
  page.on("pageerror", (error) => errors.push(error.message))
  page.on("console", (message) => {
    if (message.type() === "error") errors.push(message.text())
  })
  await page.goto("/")
  await expect(page.getByRole("heading", { name: "Journeys", exact: true })).toBeVisible()
  await expect(page.getByText("No journeys yet.", { exact: true })).toBeVisible()
  const libraryImage = info.outputPath("library-baseline.png")
  await page.screenshot({ path: libraryImage })
  await info.attach("library-baseline", { path: libraryImage, contentType: "image/png" })
  const ariaPath = info.outputPath("library-aria.txt")
  await writeFile(ariaPath, await page.getByRole("main").ariaSnapshot())
  await info.attach("library-aria", { path: ariaPath, contentType: "text/plain" })

  await page.getByRole("button", { name: "New journey", exact: true }).click()
  const create = page.getByRole("button", { name: "Create journey", exact: true })
  await expect(create).toBeDisabled()
  await page.getByLabel("Title", { exact: true }).fill("Synthetic Kyoto Journey")
  await page.getByLabel("Place", { exact: true }).fill("Kyoto")
  await page.getByText("More options", { exact: true }).click()
  await page.getByLabel("Slug", { exact: true }).fill("synthetic-kyoto")
  await page.getByLabel("Start date", { exact: true }).fill("2026-05-01")
  await page.getByLabel("End date", { exact: true }).fill("2026-05-03")
  await expect(create).toBeEnabled()
  const createImage = info.outputPath("create-baseline.png")
  await page.screenshot({ path: createImage })
  await info.attach("create-baseline", { path: createImage, contentType: "image/png" })
  const response = page.waitForResponse((r) => r.url().endsWith("/api/admin/journeys") && r.request().method() === "POST")
  await create.click()
  expect((await response).status()).toBe(200)
  await expect(page).toHaveURL(/#\/journey\/[0-9a-f-]+$/)
  await expect(page.getByRole("heading", { name: "Synthetic Kyoto Journey", exact: true })).toBeVisible()
  await page.reload()
  await expect(page.getByRole("heading", { name: "Synthetic Kyoto Journey", exact: true })).toBeVisible()
  await page.goto("/")
  await expect(page.getByRole("link", { name: /Synthetic Kyoto Journey/ })).toBeVisible()
  expect(errors).toEqual([])
})

test("cancels creation without persisting a row", async ({ page, request }) => {
  await page.goto("/")
  await page.getByRole("button", { name: "New journey", exact: true }).click()
  await page.getByLabel("Title", { exact: true }).fill("Cancelled synthetic journey")
  await page.getByRole("button", { name: "Close", exact: true }).click()
  await expect(page.getByText("No journeys yet.", { exact: true })).toBeVisible()
  const rows = await request.get("/api/admin/journeys")
  expect(await rows.json()).toEqual([])
})

test("rejects unauthenticated and cross-origin authoring requests", async ({ studio }) => {
  const unauthenticated = await fetch(`${studio.url}/api/admin/journeys`)
  expect(unauthenticated.status).toBe(403)
  const crossOrigin = await fetch(`${studio.url}/api/admin/journeys`, {
    method: "POST",
    headers: { Cookie: `felicia-e2e=${studio.token}`, Origin: "https://untrusted.example" },
  })
  expect(crossOrigin.status).toBe(403)
})
