import { readFileSync } from "node:fs"
import { fileURLToPath } from "node:url"
import { describe, expect, test } from "vitest"

describe("Atlas theme and container contract", () => {
  test("Atlas.svelte main element includes .waypoints and toggles .light class", () => {
    const file = fileURLToPath(new URL("./Atlas.svelte", import.meta.url))
    const content = readFileSync(file, "utf8")

    // Playwright e2e (sample.spec.ts:76) expects:
    // await expect(reader.locator(".waypoints")).toHaveClass(/light/)
    // and .not.toHaveClass(/light/) on dark theme.
    expect(content).toMatch(/<main[^>]*class="[^"]*\bwaypoints\b[^"]*"/)
    expect(content).toContain('class:light={theme === "light"}')
  })

  test("Atlas does not pass onToggleTheme to FloatingToolbar, avoiding double theme switchers", () => {
    const file = fileURLToPath(new URL("./Atlas.svelte", import.meta.url))
    const content = readFileSync(file, "utf8")
    expect(content).not.toContain("onToggleTheme=")
  })

  test("FloatingToolbar includes felicia-mark brand icon", () => {
    const file = fileURLToPath(new URL("../../components/FloatingToolbar.svelte", import.meta.url))
    const content = readFileSync(file, "utf8")
    expect(content).toContain('/felicia-mark.svg')
    expect(content).not.toContain('bg-orange-400')
  })

  test("DAY_PALETTE adopts Hatsune Miku signature teal as Day 1", () => {
    const file = fileURLToPath(new URL("../../playback/timeline.ts", import.meta.url))
    const content = readFileSync(file, "utf8")
    expect(content).toContain('#39C5BB')
    expect(content).toContain('#FF5E98')
    expect(content).not.toContain('#ff7e47')
  })
})
