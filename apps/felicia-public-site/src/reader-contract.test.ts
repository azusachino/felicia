import { readFileSync } from "node:fs"
import { fileURLToPath } from "node:url"
import { describe, expect, test } from "vitest"

describe("Public reader appearance controls contract", () => {
  test("App.svelte binds theme to Reader directly and does not render redundant outer appearance buttons", () => {
    const file = fileURLToPath(new URL("./App.svelte", import.meta.url))
    const content = readFileSync(file, "utf8")

    // The unified theme toggle lives inside Reader's FloatingToolbar;
    // App.svelte must bind theme directly to Reader and not define a redundant .reader-appearance toolbar
    expect(content).not.toContain('class="reader-appearance"')
    expect(content).toContain("<Reader bind:lang bind:theme {loadJourneys} />")
  })
})
