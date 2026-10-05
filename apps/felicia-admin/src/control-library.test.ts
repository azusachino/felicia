import { readdirSync, readFileSync } from "node:fs"
import { join } from "node:path"
import { fileURLToPath } from "node:url"
import { parse } from "svelte/compiler"
import { expect, test } from "vitest"

test("studio widgets use shared/library components, not view-level native substitutes", () => {
  const root = fileURLToPath(new URL("./", import.meta.url))
  const violations: string[] = []
  for (const file of readdirSync(root, { recursive: true, encoding: "utf8" })) {
    if (!file.endsWith(".svelte") || file.startsWith("lib/components/ui/")) continue
    const ast = parse(readFileSync(join(root, file), "utf8"), { modern: true })
    JSON.stringify(ast, (_key, node) => {
      if (node?.type === "RegularElement" && ["input", "textarea", "select", "button", "details", "summary", "dialog"].includes(node.name)) violations.push(`${file}: ${node.name}`)
      if (node?.type === "CallExpression") {
        const callee = node.callee
        const global = callee.type === "MemberExpression" && ["window", "globalThis"].includes(callee.object?.name)
        const name = callee.type === "Identifier" ? callee.name : global ? (callee.property?.name ?? callee.property?.value) : null
        if (["alert", "confirm", "prompt"].includes(name)) violations.push(`${file}: native ${name}()`)
      }
      return node
    })
  }
  expect(violations).toEqual([])
})
