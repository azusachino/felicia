import { test, expect } from "./fixtures"

test.use({ sampleMode: true })

for (const locale of ["en", "ja", "zh"]) {
  test(`library date/time field renders fixed order and edits seconds in ${locale}`, async ({ page }) => {
    await page.addInitScript((locale) => localStorage.setItem("felicia.admin.locale", locale), locale)
    await page.goto("/#/journey/0190cbde-f300-7000-8000-000000000002/memento/0190cbde-f300-7000-8000-000000000010")
    const field = page.locator("[data-date-field-input]")
    await expect(field).toBeVisible()
    const displayed = () =>
      field.evaluate((element) =>
        Array.from(element.children)
          .map((child) => child.textContent)
          .join("")
          .replace(/\u00a0/g, " "),
      )
    expect(await displayed()).toBe("2026-05-03 09:00:00")
    expect(await field.locator("[data-segment]").evaluateAll((items) => items.map((item) => item.getAttribute("data-segment")))).toEqual(["year", "month", "day", "hour", "minute", "second"])
    const second = field.locator('[data-segment="second"]')
    await second.focus()
    await page.keyboard.press("ArrowUp")
    await expect(second).toHaveText("01")
    const save = page.locator(".actions button").first()
    const saved = page.waitForResponse((response) => response.url().endsWith("/api/admin/mementos") && response.request().method() === "POST")
    await save.click()
    expect((await saved).status()).toBe(200)
    await expect(page.locator(".save-feedback")).toBeVisible()
    const memento = await (await page.request.get("/api/admin/mementos/0190cbde-f300-7000-8000-000000000010")).json()
    expect(memento.occurred_at).toBe("2026-05-03T09:00:01Z")
    expect(memento.occurred_tz).toBe("Asia/Tokyo")
    await page.reload()
    await expect(field.locator('[data-segment="second"]')).toHaveText("01")
    expect(await displayed()).toBe("2026-05-03 09:00:01")
    const back = page.locator(".back-link button")
    await expect(back).toContainText(locale === "en" ? "Return" : locale === "ja" ? "戻る" : "返回")
    expect((await back.boundingBox())!.height).toBeGreaterThanOrEqual(36)
    await back.click()
    await expect(page).toHaveURL(/#\/journey\/0190cbde-f300-7000-8000-000000000002$/)
    await expect(page.locator(".back-link button")).toBeVisible()
  })
}
