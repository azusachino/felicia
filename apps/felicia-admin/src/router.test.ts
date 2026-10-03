import { describe, expect, test } from "vitest"
import { journeyDetailHash, listHash, mementoEditHash, siteHash } from "./router"

describe("route URLs", () => {
  test("journey links encode an identifier as one path segment", () => {
    expect(journeyDetailHash("abc-123")).toBe("#/journey/abc-123")
    expect(journeyDetailHash("j/needs encoding")).toBe("#/journey/j%2Fneeds%20encoding")
  })

  test("memento links encode both identifiers", () => {
    expect(mementoEditHash("j/1", "m 2")).toBe("#/journey/j%2F1/memento/m%202")
  })

  test("static links retain the embedded desktop hash scheme", () => {
    expect(listHash).toBe("#/")
    expect(siteHash).toBe("#/site")
  })
})
