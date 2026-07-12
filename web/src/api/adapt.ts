import type { Coordinates, Journey, L, Memento, MementoKind, Station, Visit } from '../data'
import type { ApiGeoJSONGeometry, ApiJourney, ApiMemento, ApiTranslationMap } from './types'

type Language = 'ja' | 'en' | 'zh'

function translation(translations: ApiTranslationMap | undefined, lang: Language, field: string) {
  const value = translations?.[lang]?.[field]
  return typeof value === 'string' ? value : undefined
}

function localized(
  canonical: string | undefined,
  translations: ApiTranslationMap | undefined,
  field: string,
): L {
  const ja = translation(translations, 'ja', field) ?? canonical ?? ''
  return {
    ja,
    en: translation(translations, 'en', field) ?? ja,
    zh: translation(translations, 'zh', field) ?? ja,
  }
}

function coordinate(value: unknown): Coordinates | undefined {
  if (
    Array.isArray(value) &&
    value.length >= 2 &&
    typeof value[0] === 'number' &&
    typeof value[1] === 'number'
  ) {
    return [value[0], value[1]]
  }
  return undefined
}

function geometryPoint(geometry: ApiGeoJSONGeometry | undefined): Coordinates {
  if (!geometry) return [0, 0]
  if (geometry.type === 'Point') return coordinate(geometry.coordinates) ?? [0, 0]
  if (geometry.type === 'LineString' && Array.isArray(geometry.coordinates)) {
    return coordinate(geometry.coordinates[0]) ?? [0, 0]
  }
  return [0, 0]
}

function routeCoordinates(geometry: ApiGeoJSONGeometry | undefined): Coordinates[] {
  if (!geometry || !Array.isArray(geometry.coordinates)) return []
  if (geometry.type === 'Point') return []
  if (geometry.type === 'LineString') {
    return geometry.coordinates
      .map(coordinate)
      .filter((value): value is Coordinates => value !== undefined)
  }
  return geometry.coordinates.flatMap((segment) =>
    Array.isArray(segment)
      ? segment.map(coordinate).filter((value): value is Coordinates => value !== undefined)
      : [],
  )
}

function routeSegments(geometry: ApiGeoJSONGeometry | undefined): Coordinates[][] {
  if (!geometry || !Array.isArray(geometry.coordinates)) return []
  if (geometry.type === 'Point') return []
  if (geometry.type === 'LineString') {
    const segment = geometry.coordinates
      .map(coordinate)
      .filter((value): value is Coordinates => value !== undefined)
    return segment.length ? [segment] : []
  }
  return geometry.coordinates
    .filter((segment): segment is unknown[] => Array.isArray(segment))
    .map((segment) =>
      segment.map(coordinate).filter((value): value is Coordinates => value !== undefined),
    )
    .filter((segment) => segment.length > 0)
}

function datePart(value: string | undefined): string {
  return value?.slice(0, 10) ?? ''
}

function price(amount: number | undefined, currency: string | undefined): string {
  if (amount === undefined || amount === null) return ''
  return `${currency ?? ''} ${amount.toLocaleString('en-US')}`.trim()
}

function kind(value: string): MementoKind {
  const known: MementoKind[] = ['goods', 'transit', 'stamp', 'receipt', 'souvenir']
  return known.includes(value as MementoKind) ? (value as MementoKind) : 'goods'
}

function objectValue(value: unknown): Record<string, unknown> | undefined {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
    ? (value as Record<string, unknown>)
    : undefined
}

function stringValue(value: unknown): string | undefined {
  return typeof value === 'string' ? value : undefined
}

function station(value: unknown, fallback: Coordinates): Station {
  const data = objectValue(value)
  const coords = coordinate(data?.coords) ?? fallback
  const name = stringValue(data?.name) ?? ''
  return { name, ja: name, coords }
}

function transitData(
  apiMemento: ApiMemento,
  title: L,
  fallback: Coordinates,
): Memento['transit'] | undefined {
  if (apiMemento.kind !== 'transit') return undefined
  const data = objectValue(apiMemento.kind_data)
  if (!data) return undefined
  return {
    operator: localized(stringValue(data.operator), apiMemento.translations, 'operator'),
    line: localized(stringValue(data.line), apiMemento.translations, 'line'),
    from: station(data.from, fallback),
    to: station(data.to, fallback),
    fare: title.ja ? price(apiMemento.price_amount, apiMemento.price_currency) : '',
  }
}

function adaptMemento(apiMemento: ApiMemento, visitId: string, visitCoords: Coordinates): Memento {
  const title = localized(apiMemento.title, apiMemento.translations, 'title')
  const place = localized(apiMemento.place, apiMemento.translations, 'place')
  const vendor = localized(apiMemento.vendor, apiMemento.translations, 'vendor')
  const essay = localized(apiMemento.essay, apiMemento.translations, 'essay')
  const coords = geometryPoint(apiMemento.geom)
  return {
    id: apiMemento.id,
    kind: kind(apiMemento.kind),
    visitId,
    title,
    date: localized(datePart(apiMemento.occurred_at), apiMemento.translations, 'date'),
    place,
    vendor,
    price: price(apiMemento.price_amount, apiMemento.price_currency),
    coords: coords[0] === 0 && coords[1] === 0 ? visitCoords : coords,
    essay,
    kindData: apiMemento.kind_data,
    photos: (apiMemento.photos ?? []).map((photo) => ({
      src: photo.object_key,
      caption: localized(photo.caption, undefined, 'caption'),
    })),
    transit: transitData(apiMemento, title, visitCoords),
  }
}

export function adaptJourney(apiJourney: ApiJourney, apiMementos: ApiMemento[]): Journey {
  const visits: Visit[] = []
  const mementos: Memento[] = []
  const visitByKey = new Map<string, Visit>()
  const journeyTranslations = apiJourney.translations

  for (const apiMemento of apiMementos) {
    const coords = geometryPoint(apiMemento.geom)
    const visitKey = `${apiMemento.place}:${coords[0]},${coords[1]}`
    let visit = visitByKey.get(visitKey)
    if (!visit) {
      visit = {
        id: `visit:${apiJourney.slug}:${visitKey}`,
        label: localized(apiMemento.place, apiMemento.translations, 'place'),
        coords,
      }
      visitByKey.set(visitKey, visit)
      visits.push(visit)
    }
    mementos.push(adaptMemento(apiMemento, visit.id, visit.coords))
  }

  return {
    id: apiJourney.id,
    title: localized(apiJourney.title, journeyTranslations, 'title'),
    dates: localized(
      `${datePart(apiJourney.date_start)} — ${datePart(apiJourney.date_end)}`,
      journeyTranslations,
      'dates',
    ),
    place: localized(apiJourney.place, journeyTranslations, 'place'),
    route: routeCoordinates(apiJourney.gps_route),
    routeSegments: routeSegments(apiJourney.gps_route),
    visits,
    mementos,
  }
}
