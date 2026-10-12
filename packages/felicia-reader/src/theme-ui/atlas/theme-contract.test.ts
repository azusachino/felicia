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
})
