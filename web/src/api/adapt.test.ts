import { describe, expect, test } from 'bun:test'
import type { ApiJourney, ApiMemento } from './types'
import { adaptJourney } from './adapt'

const journey = (overrides: Partial<ApiJourney> = {}): ApiJourney => ({
  id: 'journey-1',
  journal_id: 'journal-1',
  slug: 'japan-spring-2026',
  title: '日本春旅 2026',
  place: '東京 & 京都',
  date_start: '2026-03-20',
  date_end: '2026-04-05',
  authored_fields: [],
  ...overrides,
})

const memento = (overrides: Partial<ApiMemento> = {}): ApiMemento => ({
  id: 'memento-1',
  journey_id: 'journey-1',
  kind: 'receipt',
  seq: 1,
  occurred_at: '2026-03-21T15:30:00+09:00',
  occurred_tz: 'Asia/Tokyo',
  title: 'スマートコーヒー レシート',
  place: '京都',
  price_amount: 1200,
  price_currency: 'JPY',
  geom: { type: 'Point', coordinates: [135.7583, 34.9859] },
  ...overrides,
})

describe('adaptJourney', () => {
  test('adapts the captured canonical compiler fixture', async () => {
    const apiJourney = (await Bun.file(
      new URL('./__fixtures__/journeys/japan-spring-2026.json', import.meta.url),
    ).json()) as ApiJourney
    const apiMementos = (await Bun.file(
      new URL('./__fixtures__/journeys/japan-spring-2026/mementos.json', import.meta.url),
    ).json()) as ApiMemento[]

    const result = adaptJourney(apiJourney, apiMementos)
    expect(result.id).toBe('0190cbde-f300-7000-8000-111111111111')
    expect(result.route).toHaveLength(3)
    expect(result.visits).toHaveLength(2)
    expect(result.mementos).toHaveLength(3)
    expect(result.mementos[0].photos[0].src).toBe('media/photos/tokyo_ticket.jpg')
    expect(result.mementos[1].kind).toBe('receipt')
  })

  test('maps translations with Japanese fallback and groups mementos by visit', () => {
    const first = memento({
      id: 'memento-1',
      translations: { en: { title: 'Kyoto Cafe' } },
    })
    const second = memento({ id: 'memento-2', title: '別の記憶', place: '京都' })
    const result = adaptJourney(journey(), [first, second])

    expect(result.title).toEqual({ ja: '日本春旅 2026', en: '日本春旅 2026', zh: '日本春旅 2026' })
    expect(result.mementos[0].title).toEqual({
      ja: 'スマートコーヒー レシート',
      en: 'Kyoto Cafe',
      zh: 'スマートコーヒー レシート',
    })
    expect(result.visits).toHaveLength(1)
    expect(result.mementos.map((item) => item.visitId)).toEqual([
      result.visits[0].id,
      result.visits[0].id,
    ])
    expect(result.mementos[0].price).toBe('JPY 1,200')
  })

  test('flattens a multi-line route and tolerates an empty route', () => {
    const result = adaptJourney(
      journey({
        gps_route: {
          type: 'MultiLineString',
          coordinates: [
            [
              [139.7, 35.6],
              [139.8, 35.7],
            ],
            [[135.7, 35.0]],
          ],
        },
      }),
      [],
    )
    expect(result.route).toEqual([
      [139.7, 35.6],
      [139.8, 35.7],
      [135.7, 35.0],
    ])
    expect(adaptJourney(journey(), []).route).toEqual([])
    expect(result.visits).toEqual([])
  })

  test('maps transit kind data into the existing station view model', () => {
    const result = adaptJourney(journey(), [
      memento({
        kind: 'transit',
        kind_data: {
          operator: 'JR East',
          line: 'Tokaido Shinkansen',
          from: { name: 'Tokyo', coords: [139.7671, 35.6812] },
          to: { name: 'Kyoto', coords: [135.7583, 34.9859] },
        },
      }),
    ])
    expect(result.mementos[0].kind).toBe('transit')
    expect(result.mementos[0].transit?.from.coords).toEqual([139.7671, 35.6812])
    expect(result.mementos[0].transit?.to.name).toBe('Kyoto')
  })
})
