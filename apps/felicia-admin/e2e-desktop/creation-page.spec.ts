import { test, expect } from "./fixtures"

async function openDraft(page: import("@playwright/test").Page) {
  await page.goto("/#/journey/new")
  await expect(page.getByRole("heading", { name: "Create a new journey", exact: true })).toBeVisible()
  await expect(page.getByRole("dialog")).toHaveCount(0)
  await expect(page.getByLabel("Title", { exact: true })).toBeFocused()
}

async function fillDraft(page: import("@playwright/test").Page) {
  await page.getByLabel("Title", { exact: true }).fill("Synthetic page journey")
  await page.getByLabel("Place", { exact: true }).fill("Kyoto")
  await page.getByLabel("Start date", { exact: true }).fill("2026-05-03")
  await page.getByLabel("End date", { exact: true }).fill("2026-05-03")
}

test("direct creation route and explicit Cancel discard without a write", async ({ page }) => {
  await openDraft(page)
  await fillDraft(page)
  await page.getByRole("button", { name: "Cancel", exact: true }).click()
  await expect(page).toHaveURL(/#\/$/)
  expect(await (await page.request.get("/api/admin/journeys")).json()).toEqual([])
  await page.getByRole("button", { name: "New journey", exact: true }).click()
  await expect(page.getByLabel("Title", { exact: true })).toHaveValue("")
  await expect(page.getByLabel("Start date", { exact: true })).toHaveValue("")
})

test("Back is a native icon link and protects an unsaved draft", async ({ page }) => {
  await openDraft(page)
  await page.getByLabel("Title", { exact: true }).fill("Keep this draft")
  const back = page.getByRole("link", { name: "Back to journeys", exact: true })
  await expect(back).toHaveAttribute("href", "#/")
  await expect(back).toHaveText("")
  await back.focus()
  await expect(page.getByRole("tooltip")).toHaveText("Back to journeys")
  const dismissed = new Promise<void>((resolve) => {
    page.once("dialog", async (dialog) => {
      expect(dialog.message()).toContain("unsaved journey")
      await dialog.dismiss()
      resolve()
    })
  })
  await page.keyboard.press("Enter")
  await dismissed
  await expect(page).toHaveURL(/#\/journey\/new$/)
  await expect(page.getByLabel("Title", { exact: true })).toHaveValue("Keep this draft")
  page.once("dialog", (dialog) => dialog.accept())
  await back.click()
  await expect(page.getByRole("heading", { name: "Journeys", exact: true })).toBeVisible()
  expect(await (await page.request.get("/api/admin/journeys")).json()).toEqual([])
})

test("reload cancellation preserves the unsaved creation draft", async ({ page }) => {
  await openDraft(page)
  await fillDraft(page)
  const dialog = page.waitForEvent("dialog")
  const reloadAttempt = page.evaluate(() => location.reload())
  const unload = await dialog
  expect(unload.type()).toBe("beforeunload")
  await unload.dismiss()
  await reloadAttempt
  await expect(page.getByLabel("Title", { exact: true })).toHaveValue("Synthetic page journey")
  await expect(page.getByLabel("End date", { exact: true })).toHaveValue("2026-05-03")
  expect(await (await page.request.get("/api/admin/journeys")).json()).toEqual([])
})

test("invalid date range is explained without a write; equal dates create once", async ({ page }) => {
  await openDraft(page)
  await fillDraft(page)
  await page.getByLabel("End date", { exact: true }).fill("2026-05-02")
  await expect(page.getByRole("alert")).toHaveText("End date must be on or after the start date.")
  await expect(page.getByLabel("End date", { exact: true })).toHaveAttribute("aria-invalid", "true")
  await page.getByRole("button", { name: "Create journey", exact: true }).click()
  expect(await (await page.request.get("/api/admin/journeys")).json()).toEqual([])
  await expect(page.getByLabel("Title", { exact: true })).toHaveValue("Synthetic page journey")
  await page.getByLabel("End date", { exact: true }).fill("2026-05-03")
  await expect(page.getByRole("alert")).toHaveCount(0)
  await page.getByRole("button", { name: "Create journey", exact: true }).click()
  await expect(page.getByRole("heading", { name: "Synthetic page journey", exact: true })).toBeVisible()
  expect(await (await page.request.get("/api/admin/journeys")).json()).toHaveLength(1)
})

test("derived slug stays editable and optional metadata is submitted", async ({ page }) => {
  await openDraft(page)
  await fillDraft(page)
  await page.getByText("More options", { exact: true }).click()
  await expect(page.getByLabel("Slug", { exact: true })).toHaveValue("synthetic-page-journey")
  await page.getByLabel("Slug", { exact: true }).fill("my-owned-slug")
  await page.getByLabel("Title", { exact: true }).fill("Edited synthetic journey")
  await expect(page.getByLabel("Slug", { exact: true })).toHaveValue("my-owned-slug")
  await page.getByLabel("Country (optional)", { exact: true }).fill(" JP ")
  await page.getByLabel("Region (optional)", { exact: true }).fill(" Kansai ")
  const submitted = page.waitForRequest((request) => request.url().endsWith("/api/admin/journeys") && request.method() === "POST")
  await page.getByRole("button", { name: "Create journey", exact: true }).click()
  expect((await submitted).postDataJSON()).toMatchObject({ title: "Edited synthetic journey", slug: "my-owned-slug", country: "JP", region: "Kansai" })
  await expect(page.getByRole("heading", { name: "Edited synthetic journey", exact: true })).toBeVisible()
  await page.reload()
  await expect(page.getByRole("heading", { name: "Edited synthetic journey", exact: true })).toBeVisible()
})

test("pending creation blocks navigation and repeated submit without duplicating writes", async ({ page }) => {
  await openDraft(page)
  await fillDraft(page)
  let release!: () => void
  let observed!: () => void
  const held = new Promise<void>((resolve) => (release = resolve))
  const started = new Promise<void>((resolve) => (observed = resolve))
  let writes = 0
  await page.route("**/api/admin/journeys", async (route) => {
    if (route.request().method() !== "POST") return route.continue()
    writes++
    const response = await route.fetch()
    observed()
    await held
    await route.fulfill({ response })
  })
  try {
    await page.getByRole("button", { name: "Create journey", exact: true }).click()
    await started
    await expect(page.getByLabel("Title", { exact: true })).toBeDisabled()
    await expect(page.getByRole("button", { name: "Creating…", exact: true })).toBeDisabled()
    await expect(page.getByRole("button", { name: "Cancel", exact: true })).toBeDisabled()
    await page.getByRole("link", { name: "Site & Deploy", exact: true }).click()
    await expect(page).toHaveURL(/#\/journey\/new$/)
    await page.evaluate(() => (document.getElementById("journey-create-form") as HTMLFormElement).requestSubmit())
    expect(writes).toBe(1)
  } finally {
    release()
  }
  await expect(page.getByRole("heading", { name: "Synthetic page journey", exact: true })).toBeVisible()
  expect(writes).toBe(1)
  expect(await (await page.request.get("/api/admin/journeys")).json()).toHaveLength(1)
})
