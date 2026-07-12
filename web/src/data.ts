// Shared presentation types and labels. Journey content comes from the API
// adapter; this module intentionally contains no demo-owned mock dataset.

export type Coordinates = [number, number]
export type MementoKind = 'goods' | 'transit' | 'stamp' | 'receipt' | 'souvenir'
export type Lang = 'ja' | 'en' | 'zh'
export type Theme = 'dark' | 'light'

export interface L {
  ja: string
  en: string
  zh: string
}

export interface Station {
  name: string
  ja: string
  coords: Coordinates
}

export interface Memento {
  id: string
  kind: MementoKind
  visitId: string
  title: L
  date: L
  place: L
  vendor: L
  price: string
  coords: Coordinates
  essay: L
  kindData?: Record<string, unknown>
  photos: { src: string; caption: L }[]
  transit?: {
    operator: L
    line: L
    from: Station
    to: Station
    fare: string
  }
}

export interface Visit {
  id: string
  label: L
  coords: Coordinates
}

export interface Journey {
  id: string
  title: L
  dates: L
  place: L
  route: Coordinates[]
  routeSegments?: Coordinates[][]
  visits: Visit[]
  mementos: Memento[]
  representativeDots?: { coord: Coordinates; label: string }[]
}

export const kindLabel: Record<MementoKind, L> = {
  transit: { ja: '交通', en: 'Transit', zh: '交通' },
  stamp: { ja: '御朱印', en: 'Stamp', zh: '御朱印' },
  goods: { ja: 'グッズ', en: 'Goods', zh: '周边' },
  receipt: { ja: 'レシート', en: 'Receipt', zh: '收据' },
  souvenir: { ja: 'おみやげ', en: 'Souvenir', zh: '纪念品' },
}

export const uiText = {
  journeys: { ja: '旅の記録', en: 'Journeys', zh: '旅程' },
  all: { ja: 'すべて表示', en: 'View all', zh: '查看全部' },
  story: { ja: '物語', en: 'The Story', zh: '故事' },
}

export interface MementoCard {
  memento: Memento
  journey: Journey
}
