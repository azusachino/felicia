import { describe, expect, test } from "bun:test"
import { stubTemplates, templateFor } from "./stubs"

describe("memento stub registry", () => {
  test("registers every supported memento kind", () => {
    expect(Object.keys(stubTemplates).sort()).toEqual(["goods", "live", "receipt", "souvenir", "stamp", "ticket", "transit"])
  })

  test("returns no template for an unknown kind so the caller can use a photo fallback", () => {
    expect(templateFor("unknown-kind")).toBeUndefined()
  })
})
