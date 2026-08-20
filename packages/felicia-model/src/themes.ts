import type { MessageKey } from "./i18n/catalog"

export type ThemeMode = "dark" | "light"
export type Theme = ThemeMode

export interface ThemeDefinition {
  id: ThemeMode
  labelKey: MessageKey
  className: string
}

export const themeModes: ThemeDefinition[] = [
  { id: "dark", labelKey: "theme.dark", className: "theme-dark" },
  { id: "light", labelKey: "theme.light", className: "theme-light" },
]

export function themeFromId(id: string | undefined): ThemeDefinition {
  return themeModes.find((theme) => theme.id === id) ?? themeModes[0]
}
