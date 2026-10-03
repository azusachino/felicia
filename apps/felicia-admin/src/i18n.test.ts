import { describe, expect, test } from "vitest"
import { catalogs, message, MESSAGE_KEYS, resolveLocale, type AdminMessageKey, type Locale } from "./i18n"

describe("admin locale catalogs", () => {
  test("each JSON file provides exactly the declared message keys", () => {
    const expectedKeys = [...MESSAGE_KEYS].sort()
    const locales: Locale[] = ["ja", "en", "zh"]
    for (const locale of locales) {
      expect(Object.keys(catalogs[locale]).sort()).toEqual(expectedKeys)
      for (const key of MESSAGE_KEYS as readonly AdminMessageKey[]) {
        expect(message(locale, key)).toBeTruthy()
      }
    }
  })

  test("keeps interpolation placeholders consistent across locales", () => {
    const placeholders = (text: string) => [...text.matchAll(/\{([^}]+)\}/g)].map((match) => match[1]).sort()
    for (const key of MESSAGE_KEYS as readonly AdminMessageKey[]) {
      const english = placeholders(catalogs.en[key])
      expect(placeholders(catalogs.ja[key])).toEqual(english)
      expect(placeholders(catalogs.zh[key])).toEqual(english)
    }
  })

  test("formats catalog placeholders in the selected language", () => {
    expect(message("ja", "admin.build.label_pending", { count: 3 })).toContain("3")
    expect(message("zh", "admin.build.label_pending", { count: 3 })).toContain("3")
  })

  test("prefers a valid saved locale, then browser language, then Japanese", () => {
    expect(resolveLocale("zh", "en-US")).toBe("zh")
    expect(resolveLocale("fr", "en-US")).toBe("en")
    expect(resolveLocale(null, "zh-CN")).toBe("zh")
    expect(resolveLocale(null, "fr-FR")).toBe("ja")
    expect(resolveLocale(null, null)).toBe("ja")
  })
})
