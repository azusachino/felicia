import { randomUUID } from "node:crypto"
import { test, expect } from "./fixtures"

test.use({ sampleMode: false, userAgent: "Mozilla/5.0 (Macintosh; Intel Mac OS X) wails.io" })
const picture = Buffer.from("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR4nGP43+DwHwAHAAK/K9fH4gAAAABJRU5ErkJggg==", "base64")

test("ordinary desktop uploads images, blocks pending navigation and preserves photo curation", async ({ page }) => {
  await page.route("**/wails/runtime.js", (route) => route.fulfill({ contentType: "application/javascript", body: "" }))
  await page.goto("/#/")
  const journeyResponse = await page.request.post("/api/admin/journeys", {
    data: { slug: "synthetic-photo-upload", title: "Synthetic photo journey", place: "Kyoto", date_start: "2026-05-03", date_end: "2026-05-03" },
  })
  expect(journeyResponse.ok(), await journeyResponse.text()).toBe(true)
  const journey = await journeyResponse.json()
  const id = randomUUID()
  const created = await page.request.post("/api/admin/mementos", {
    data: {
      id,
      journey_id: journey.id,
      kind: "goods",
      seq: 1,
      title: "Synthetic image authoring",
      place: "Kyoto",
      state: "draft",
      kind_data: { name: "Postcard" },
      geom: { type: "Point", coordinates: [135.7, 35.0] },
      occurred_at: "2026-05-03T12:00:00Z",
      occurred_tz: "UTC",
    },
  })
  expect(created.ok(), await created.text()).toBe(true)
  await page.reload()
  await page.getByRole("link", { name: /Synthetic photo journey/ }).click()
  await page.getByRole("link", { name: /Synthetic image authoring/ }).click()
  await expect(page.getByRole("button", { name: "Add photos", exact: true })).toBeVisible()
  await expect(page.locator('input[type="file"]')).toHaveCount(1)
  let release!: () => void
  let observed!: () => void
  const held = new Promise<void>((resolve) => {
    release = resolve
  })
  const started = new Promise<void>((resolve) => {
    observed = resolve
  })
  await page.route("**/photos/upload", async (route) => {
    observed()
    await held
    // Keep the original binary multipart body; only delay forwarding.
    await route.continue()
  })
  try {
    const uploaded = page.waitForResponse((response) => response.url().endsWith("/photos/upload") && response.request().method() === "POST")
    await page.getByLabel("Add photos", { exact: true }).setInputFiles({ name: "private-source-sentinel.png", mimeType: "image/png", buffer: picture })
    await started
    await expect(page.getByRole("button", { name: "Add photos", exact: true })).toBeDisabled()
    await expect(page.getByRole("button", { name: "Back to journey", exact: true })).toBeDisabled()
    await page.getByRole("link", { name: "Journeys", exact: true }).click()
    await expect(page).toHaveURL(new RegExp(`/memento/${id}$`))
    release()
    const response = await uploaded
    expect(response.status()).toBe(201)
    const photo = await response.json()
    expect(photo.object_key).toMatch(/\/original\.png$/)
    expect(photo.content_hash).toMatch(/^sha256:/)
    expect(JSON.stringify(photo)).not.toContain("private-source-sentinel")
    await expect(page.locator(".photo-row")).toHaveCount(1)
    await expect(page.getByLabel("Title", { exact: true })).toHaveValue("Synthetic image authoring")
    await page.locator(".photo-row").scrollIntoViewIfNeeded()
    await expect(page.locator(".photo-row").getByRole("img")).toHaveJSProperty("naturalWidth", 1)
    const caption = page.locator(".photo-row").getByLabel("Caption", { exact: true })
    await caption.fill("Synthetic uploaded caption")
    const saved = page.waitForResponse((response) => response.url().endsWith("/api/admin/photos") && response.request().method() === "POST")
    await page.getByRole("button", { name: "Save caption", exact: true }).click()
    expect((await saved).status()).toBe(200)
    await page.reload()
    await expect(caption).toHaveValue("Synthetic uploaded caption")
  } finally {
    release()
  }
})
