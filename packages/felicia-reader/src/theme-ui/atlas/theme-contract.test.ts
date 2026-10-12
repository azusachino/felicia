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

  test("Atlas passes onToggleTheme to FloatingToolbar as the canonical single theme control", () => {
    const file = fileURLToPath(new URL("./Atlas.svelte", import.meta.url))
    const content = readFileSync(file, "utf8")
    expect(content).toContain("onToggleTheme=")
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

  test("AtlasDetail supports light/dark theme and respects Miku teal accent", () => {
    const file = fileURLToPath(new URL("./AtlasDetail.svelte", import.meta.url))
    const content = readFileSync(file, "utf8")
    expect(content).toContain('theme = "dark"')
    expect(content).toContain('class:theme-light={theme === "light"}')
    expect(content).toContain('#39C5BB')
    expect(content).toContain('top: 4.5rem')
  })

  test("TripPlaybackCard stabilizes mobile height and contains scrolling inside card", () => {
    const file = fileURLToPath(new URL("../../playback/TripPlaybackCard.svelte", import.meta.url))
    const content = readFileSync(file, "utf8")
    expect(content).toContain('overscroll-contain')
    expect(content).toContain('touch-action:pan-y')
    expect(content).toContain('h-[15.5rem]')
  })
})
