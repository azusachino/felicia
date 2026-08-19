export type Locale = "ja" | "en" | "zh"

export type MessageKey =
  | "design.cartography"
  | "design.cabinet"
  | "design.techo"
  | "design.atlas"
  | "theme.dark"
  | "theme.light"
  | "system.language"
  | "system.design"
  | "kind.transit"
  | "kind.stamp"
  | "kind.goods"
  | "kind.receipt"
  | "kind.souvenir"
  | "kind.live"
  | "kind.ticket"
  | "ui.journeys"
  | "ui.all"
  | "ui.story"
  | "ui.close"
  | "ui.zoom"

export type Catalog = Record<MessageKey, string>

export const catalogs: Record<Locale, Catalog> = {
  ja: {
    "design.cartography": "地図帳",
    "design.cabinet": "標本箱",
    "design.techo": "手帳",
    "design.atlas": "世界地図",
    "theme.dark": "ダーク",
    "theme.light": "ライト",
    "system.language": "言語",
    "system.design": "デザイン",
    "kind.transit": "交通",
    "kind.stamp": "御朱印",
    "kind.goods": "グッズ",
    "kind.receipt": "レシート",
    "kind.souvenir": "おみやげ",
    "kind.live": "ライブ",
    "kind.ticket": "入場券",
    "ui.journeys": "旅の記録",
    "ui.all": "すべて表示",
    "ui.story": "物語",
    "ui.close": "閉じる",
    "ui.zoom": "写真を拡大",
  },
  en: {
    "design.cartography": "Cartography",
    "design.cabinet": "Cabinet",
    "design.techo": "Techo",
    "design.atlas": "Atlas",
    "theme.dark": "Dark",
    "theme.light": "Light",
    "system.language": "Language",
    "system.design": "Design",
    "kind.transit": "Transit",
    "kind.stamp": "Stamp",
    "kind.goods": "Goods",
    "kind.receipt": "Receipt",
    "kind.souvenir": "Souvenir",
    "kind.live": "Live",
    "kind.ticket": "Ticket",
    "ui.journeys": "Journeys",
    "ui.all": "View all",
    "ui.story": "The Story",
    "ui.close": "Close",
    "ui.zoom": "Open photo",
  },
  zh: {
    "design.cartography": "地图志",
    "design.cabinet": "藏品柜",
    "design.techo": "手帐",
    "design.atlas": "世界图册",
    "theme.dark": "深色",
    "theme.light": "浅色",
    "system.language": "语言",
    "system.design": "设计",
    "kind.transit": "交通",
    "kind.stamp": "印章",
    "kind.goods": "周边",
    "kind.receipt": "收据",
    "kind.souvenir": "纪念品",
    "kind.live": "演出",
    "kind.ticket": "票券",
    "ui.journeys": "旅程",
    "ui.all": "查看全部",
    "ui.story": "故事",
    "ui.close": "关闭",
    "ui.zoom": "放大照片",
  },
}

export function resolveLocale(value: string | null | undefined): Locale {
  if (value === "ja" || value === "en" || value === "zh") return value
  if (value?.startsWith("ja")) return "ja"
  if (value?.startsWith("zh")) return "zh"
  if (value?.startsWith("en")) return "en"
  return "ja"
}

export function message(locale: Locale, key: MessageKey): string {
  return catalogs[locale][key] ?? catalogs.ja[key] ?? key
}
