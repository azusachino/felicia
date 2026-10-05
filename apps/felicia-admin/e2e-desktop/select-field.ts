import type { Locator, Page } from "@playwright/test"

export async function chooseSelect(page: Page, trigger: Locator, value: string): Promise<void> {
  await trigger.click()
  await page.locator(`[role="option"][data-option-value="${value}"]`).click()
}
