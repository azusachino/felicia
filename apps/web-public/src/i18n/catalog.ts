export type Locale = "ja" | "en" | "zh"

export type MessageKey =
  | "design.map"
  | "design.collection"
  | "design.journal"
  | "design.atlas"
  | "system.language"
  | "system.design"
  | "kind.transit"
  | "kind.stamp"
  | "kind.goods"
  | "kind.receipt"
  | "kind.souvenir"
  | "ui.journeys"
  | "ui.all"
  | "ui.story"

export type Catalog = Record<MessageKey, string>

export const catalogs: Record<Locale, Catalog> = {
  ja: {
    "design.map": "地図",
    "design.collection": "コレクション",
    "design.journal": "手帳",
    "design.atlas": "世界地図",
    "system.language": "言語",
    "system.design": "デザイン",
    "kind.transit": "交通",
    "kind.stamp": "御朱印",
    "kind.goods": "グッズ",
    "kind.receipt": "レシート",
    "kind.souvenir": "おみやげ",
    "ui.journeys": "旅の記録",
    "ui.all": "すべて表示",
    "ui.story": "物語",
  },
  en: {
    "design.map": "Map",
    "design.collection": "Collection",
    "design.journal": "Journal",
    "design.atlas": "Atlas",
    "system.language": "Language",
    "system.design": "Design",
    "kind.transit": "Transit",
    "kind.stamp": "Stamp",
    "kind.goods": "Goods",
    "kind.receipt": "Receipt",
    "kind.souvenir": "Souvenir",
    "ui.journeys": "Journeys",
    "ui.all": "View all",
    "ui.story": "The Story",
  },
  zh: {
    "design.map": "地图",
    "design.collection": "藏品",
    "design.journal": "手帐",
    "design.atlas": "世界图册",
    "system.language": "语言",
    "system.design": "设计",
    "kind.transit": "交通",
    "kind.stamp": "印章",
    "kind.goods": "周边",
    "kind.receipt": "收据",
    "kind.souvenir": "纪念品",
    "ui.journeys": "旅程",
    "ui.all": "查看全部",
    "ui.story": "故事",
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
