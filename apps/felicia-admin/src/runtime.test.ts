import process from "node:process"
import { expect, test } from "vitest"

test("unit tests execute in the requested JavaScript runtime", () => {
  expect(process.versions.bun ? "bun" : "node").toBe(process.env.FELICIA_JS_RUNTIME ?? "bun")
})
