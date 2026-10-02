import en from "./locales/en.json"
import ja from "./locales/ja.json"
import zh from "./locales/zh.json"
import { MESSAGE_KEYS, type AdminMessageKey } from "./keys"

export type Locale = "ja" | "en" | "zh"
export { MESSAGE_KEYS, type AdminMessageKey }

export const LOCALE_STORAGE_KEY = "felicia.admin.locale"

type Catalog = Record<AdminMessageKey, string>

export const catalogs = {
  ja,
  en,
  zh,
} satisfies Record<Locale, Catalog>

export function message(locale: Locale, key: AdminMessageKey, values: Record<string, string | number> = {}): string {
  return Object.entries(values).reduce<string>((text, [name, value]) => text.replaceAll(`{${name}}`, String(value)), catalogs[locale][key])
}

const stateKeys: Record<string, AdminMessageKey> = {
  candidate: "admin.lifecycle.state.candidate",
  draft: "admin.lifecycle.state.draft",
  authored: "admin.lifecycle.state.authored",
  published: "admin.lifecycle.state.published",
  archived: "admin.lifecycle.state.archived",
  proposed: "admin.lifecycle.state.proposed",
  ignored: "admin.lifecycle.state.ignored",
  merged: "admin.lifecycle.state.merged",
}

export function statusMessage(locale: Locale, status: string): string {
  const key = stateKeys[status]
  return key ? message(locale, key) : status
}

export function resolveLocale(preference: string | null, browserLanguage: string | null): Locale {
  const preferred = asLocale(preference)
  if (preferred) return preferred
  const browser = asLocale(browserLanguage?.split("-")[0] ?? null)
  return browser ?? "ja"
}

function asLocale(value: string | null): Locale | null {
  return value === "ja" || value === "en" || value === "zh" ? value : null
}

export function loadLocale(): Locale {
  const browserLanguage = typeof navigator === "undefined" ? null : navigator.language
  try {
    return resolveLocale(localStorage.getItem(LOCALE_STORAGE_KEY), browserLanguage)
  } catch {
    return resolveLocale(null, browserLanguage)
  }
}

export function saveLocale(locale: Locale): void {
  try {
    localStorage.setItem(LOCALE_STORAGE_KEY, locale)
  } catch {
    // The selected locale still applies for this session when storage is unavailable.
  }
}
