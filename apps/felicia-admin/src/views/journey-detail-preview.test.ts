import { readFileSync } from "node:fs"
import { join } from "node:path"
import { fileURLToPath } from "node:url"
import { parse } from "svelte/compiler"
import { describe, expect, test } from "vitest"

describe("JourneyDetail preview button uniqueness", () => {
  test("contains exactly one studio.openPreview button to prevent Playwright strict mode conflicts", () => {
    const file = join(fileURLToPath(new URL(".", import.meta.url)), "JourneyDetail.svelte")
    const content = readFileSync(file, "utf8")
    const ast = parse(content, { modern: true })

    const previewCalls: string[] = []
    JSON.stringify(ast, (_key, node) => {
      if (node?.type === "CallExpression") {
        const callee = node.callee
        if (callee?.type === "MemberExpression" && callee.object?.name === "studio" && callee.property?.name === "openPreview") {
          previewCalls.push("studio.openPreview()")
        }
      }
      return node
    })

    // Regression: Exactly one preview button in the header actions.
    // Duplicate in .build-shortcut caused Playwright strict mode violation in sample.spec.ts:42.
    expect(previewCalls).toHaveLength(1)
    expect(content).toContain("onclick={() => studio.openPreview()}")
  })
})
