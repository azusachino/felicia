import { describe, expect, test } from "bun:test"
import { journeyDetailHash, listHash, mementoEditHash, parseRoute, siteHash } from "./router"

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

  test("parses a memento editor hash", () => {
    expect(parseRoute("#/journey/journey-1/memento/memento-1")).toEqual({ name: "memento", journeyId: "journey-1", id: "memento-1" })
  })

  test("tolerates a trailing slash on a memento editor hash", () => {
    expect(parseRoute("#/journey/journey-1/memento/memento-1/")).toEqual({ name: "memento", journeyId: "journey-1", id: "memento-1" })
  })

  test("decodes URL-encoded ids in a memento editor hash", () => {
    expect(parseRoute("#/journey/j%201/memento/m%202")).toEqual({ name: "memento", journeyId: "j 1", id: "m 2" })
  })

  test("falls back to the journey list for a malformed memento hash", () => {
    expect(parseRoute("#/journey/journey-1/memento")).toEqual({ name: "list" })
    expect(parseRoute("#/journey/journey-1/memento/")).toEqual({ name: "list" })
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

describe("mementoEditHash", () => {
  test("builds a deep-linkable memento editor hash", () => {
    expect(mementoEditHash("journey-1", "memento-1")).toBe("#/journey/journey-1/memento/memento-1")
  })

  test("round-trips through parseRoute, encoding both ids", () => {
    const hash = mementoEditHash("j/1", "m/1")
    expect(parseRoute(hash)).toEqual({ name: "memento", journeyId: "j/1", id: "m/1" })
  })
})

describe("site route (ADMIN-02 M0)", () => {
  test("parses '#/site' as the site route", () => {
    expect(parseRoute("#/site")).toEqual({ name: "site" })
  })

  test("tolerates a trailing slash on the site hash", () => {
    expect(parseRoute("#/site/")).toEqual({ name: "site" })
  })

  test("siteHash resolves back to the site route", () => {
    expect(siteHash).toBe("#/site")
    expect(parseRoute(siteHash)).toEqual({ name: "site" })
  })

  test("unknown hashes still fall back to the journey list, not the site route", () => {
    expect(parseRoute("#/sit")).toEqual({ name: "list" })
    expect(parseRoute("#/sites")).toEqual({ name: "list" })
    expect(parseRoute("#/unknown")).toEqual({ name: "list" })
  })
})
