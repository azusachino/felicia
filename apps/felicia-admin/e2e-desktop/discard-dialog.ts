import { expect, type Page } from "@playwright/test"

export async function decideDiscard(page: Page, discard: boolean): Promise<void> {
  const prompt = page.getByRole("alertdialog")
  await expect(prompt).toBeVisible()
  await prompt.getByRole("button", { name: discard ? "Discard changes" : "Keep editing", exact: true }).click()
  await expect(prompt).toHaveCount(0)
}
