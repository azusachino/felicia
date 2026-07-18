import { describe, expect, test } from "bun:test"
import { journeyDetailHash, listHash, parseRoute } from "./router"

describe("parseRoute", () => {
  test("treats an empty hash as the journey list", () => {
    expect(parseRoute("")).toEqual({ name: "list" })
  })

  test("treats a bare '#' as the journey list", () => {
    expect(parseRoute("#")).toEqual({ name: "list" })
  })

  test("treats '#/' as the journey list", () => {
    expect(parseRoute("#/")).toEqual({ name: "list" })
  })

  test("parses a journey detail hash", () => {
    expect(parseRoute("#/journey/abc-123")).toEqual({ name: "detail", id: "abc-123" })
  })

  test("tolerates a trailing slash on a detail hash", () => {
    expect(parseRoute("#/journey/abc-123/")).toEqual({ name: "detail", id: "abc-123" })
  })

  test("decodes a URL-encoded id", () => {
    expect(parseRoute("#/journey/abc%20123")).toEqual({ name: "detail", id: "abc 123" })
  })

  test("falls back to the journey list for an unrecognized route", () => {
    expect(parseRoute("#/unknown")).toEqual({ name: "list" })
    expect(parseRoute("#/journey")).toEqual({ name: "list" })
    expect(parseRoute("#/journey/")).toEqual({ name: "list" })
  })
})

describe("journeyDetailHash", () => {
  test("builds a deep-linkable detail hash", () => {
    expect(journeyDetailHash("abc-123")).toBe("#/journey/abc-123")
  })

  test("round-trips through parseRoute", () => {
    const hash = journeyDetailHash("j/needs encoding")
    expect(parseRoute(hash)).toEqual({ name: "detail", id: "j/needs encoding" })
  })

  test("listHash resolves back to the list route", () => {
    expect(parseRoute(listHash)).toEqual({ name: "list" })
  })
})
