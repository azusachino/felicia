import { readFileSync } from "node:fs"
import { fileURLToPath } from "node:url"
import { parse } from "svelte/compiler"
import { describe, expect, test } from "vitest"

describe("Public reader appearance controls contract", () => {
  test("App.svelte defines .reader-appearance with Sun/Moon theme toggle buttons", () => {
    const file = fileURLToPath(new URL("./App.svelte", import.meta.url))
    const content = readFileSync(file, "utf8")
    const ast = parse(content, { modern: true })

    // Must define reader-appearance container and two buttons
    expect(content).toContain('class="reader-appearance"')
    expect(content).toContain('chooseTheme("light")')
    expect(content).toContain('chooseTheme("dark")')
    expect(content).toContain("theme.light")
    expect(content).toContain("theme.dark")

    const buttonAriaLabels: string[] = []
    JSON.stringify(ast, (_key, node) => {
      if (node?.type === "RegularElement" && node.name === "button") {
        const ariaLabel = node.attributes?.find((a: { name: string }) => a.name === "aria-label")
        if (ariaLabel) {
          buttonAriaLabels.push(ariaLabel.name)
        }
      }
      return node
    })

    expect(buttonAriaLabels.length).toBeGreaterThanOrEqual(2)
  })
})
