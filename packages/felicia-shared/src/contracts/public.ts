import type { Coordinates, MementoKind } from "../data"

export interface ApiGeoJSONGeometry {
  type: "Point" | "LineString" | "MultiLineString"
  coordinates: unknown
}

export interface ApiJourneyListItem {
  id: string
  slug: string
  title: string
  memento_count: number
  representative_dots: { coord: Coordinates; label: string }[]
}

export interface ApiJourney {
  id: string
  journal_id: string
  slug: string
  source_ref?: string
  title: string
  place: string
  country?: string
  region?: string
  date_start: string
  date_end: string
  gps_route?: ApiGeoJSONGeometry
  authored_fields: string[]
}

export interface ApiMementoPhoto {
  id: string
  memento_id: string
  object_key: string
  content_hash: string
  caption?: string
  seq: number
  taken_at?: string
  source_ref?: string
}

export interface ApiMemento {
  id: string
  journey_id: string
  kind: string
  seq: number
  occurred_at: string
  occurred_tz: string
  geom?: ApiGeoJSONGeometry
  title: string
  place: string
  vendor?: string
  essay?: string
  price_amount?: number
  price_currency?: string
  kind_data?: Record<string, unknown>
  source_ref?: string
  photos?: ApiMementoPhoto[]
}

export interface ApiJourneyPayload {
  journey: ApiJourney
  mementos: ApiMemento[]
}

export interface ApiSiteSettings {
  title: string
  description: string
  design: "v1" | "v2" | "v3" | "v4"
  default_language: "ja" | "en" | "zh"
  default_theme: "dark" | "light"
  accent: string
}

export type { Coordinates, MementoKind }
