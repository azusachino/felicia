// Shared presentation types and labels. Journey content comes from the API
// adapter; this module intentionally contains no demo-owned mock dataset.
import type { MessageKey } from "./i18n/catalog"

export type Coordinates = [number, number]
export type MementoKind = "goods" | "transit" | "stamp" | "receipt" | "souvenir" | "live"
export type Lang = "ja" | "en" | "zh"
export type { Theme } from "./theme-ui/themes"

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

export const kindLabel: Record<MementoKind, MessageKey> = {
  transit: "kind.transit",
  stamp: "kind.stamp",
  goods: "kind.goods",
  receipt: "kind.receipt",
  souvenir: "kind.souvenir",
  live: "kind.live",
}

export const uiText = {
  journeys: "ui.journeys",
  all: "ui.all",
  story: "ui.story",
  close: "ui.close",
} satisfies Record<string, MessageKey>

export interface MementoCard {
  memento: Memento
  journey: Journey
}
