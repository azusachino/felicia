import { describe, expect, test } from "bun:test"
import { catalogs, message, resolveLocale, type MessageKey } from "./catalog"

describe("system locale catalogs", () => {
  test("cover every system key in every supported locale", () => {
    const keys = Object.keys(catalogs.ja) as MessageKey[]
    for (const locale of ["ja", "en", "zh"] as const) {
      expect(Object.keys(catalogs[locale]).sort()).toEqual(keys.sort())
      for (const key of keys) expect(catalogs[locale][key]).not.toBe("")
    }
  })

  test("normalizes browser language tags and falls back to Japanese", () => {
    expect(resolveLocale("zh-CN")).toBe("zh")
    expect(resolveLocale("en-US")).toBe("en")
    expect(resolveLocale("fr-FR")).toBe("ja")
    expect(message("en", "design.map")).toBe("Map")
  })
})
