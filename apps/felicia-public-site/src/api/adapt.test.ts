import { describe, expect, test } from "bun:test"
import type { ApiJourney, ApiMemento } from "@felicia/shared"
import { adaptJourney } from "@felicia/shared"
import { loadGoldenRouteFixture } from "./fixtures"

const journey = (overrides: Partial<ApiJourney> = {}): ApiJourney => ({
  id: "journey-1",
  journal_id: "journal-1",
  slug: "japan-spring-2026",
  title: "日本春旅 2026",
  place: "東京 & 京都",
  date_start: "2026-03-20",
  date_end: "2026-04-05",
  authored_fields: [],
  ...overrides,
})

const memento = (overrides: Partial<ApiMemento> = {}): ApiMemento => ({
  id: "memento-1",
  journey_id: "journey-1",
  kind: "receipt",
  seq: 1,
  occurred_at: "2026-03-21T15:30:00+09:00",
  occurred_tz: "Asia/Tokyo",
  title: "スマートコーヒー レシート",
  place: "京都",
  price_amount: 1200,
  price_currency: "JPY",
  geom: { type: "Point", coordinates: [135.7583, 34.9859] },
  ...overrides,
})

describe("adaptJourney", () => {
  test("adapts the captured canonical compiler fixture", async () => {
    const { journey: apiJourney, mementos: apiMementos } = await loadGoldenRouteFixture()

    const result = adaptJourney(apiJourney, apiMementos)
    expect(result.id).toBe("44724c10-9202-5ba2-8550-cf6f94ad7998")
    expect(result.route).toHaveLength(4)
    expect(result.visits).toHaveLength(3)
    expect(result.mementos).toHaveLength(3)
    expect(result.mementos[0].photos).toHaveLength(1)
    expect(result.mementos[1].kind).toBe("ticket")
  })

  test("keeps authored content unchanged across system locales and groups mementos by visit", () => {
    const first = memento({
      id: "memento-1",
    })
    const second = memento({ id: "memento-2", title: "別の記憶", place: "京都" })
    const result = adaptJourney(journey(), [first, second])

    expect(result.title).toEqual({ ja: "日本春旅 2026", en: "日本春旅 2026", zh: "日本春旅 2026" })
    expect(result.mementos[0].title).toEqual({
      ja: "スマートコーヒー レシート",
      en: "スマートコーヒー レシート",
      zh: "スマートコーヒー レシート",
    })
    expect(result.visits).toHaveLength(1)
    expect(result.mementos.map((item) => item.visitId)).toEqual([result.visits[0].id, result.visits[0].id])
    expect(result.mementos[0].price).toBe("JPY 1,200")
  })

  test("normalizes string transit stations and vendor kind data", () => {
    const result = adaptJourney(journey(), [
      memento({
        id: "transit-1",
        kind: "transit",
        kind_data: { operator: "JR", line: "Kansai local", from: "Osaka", to: "Kobe", vendor: "Harbor Kitchen" },
      }),
    ])

    expect(result.mementos[0].transit?.from.name).toBe("Osaka")
    expect(result.mementos[0].transit?.to.name).toBe("Kobe")
    expect(result.mementos[0].vendor.en).toBe("Harbor Kitchen")
  })

  test("flattens a multi-line route and tolerates an empty route", () => {
    const result = adaptJourney(
      journey({
        gps_route: {
          type: "MultiLineString",
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
    expect(result.routeSegments).toEqual([
      [
        [139.7, 35.6],
        [139.8, 35.7],
      ],
      [[135.7, 35.0]],
    ])
    expect(adaptJourney(journey(), []).route).toEqual([])
    expect(adaptJourney(journey(), []).routeSegments).toEqual([])
    expect(result.visits).toEqual([])
  })

  test("keeps a LineString as one route segment", () => {
    const result = adaptJourney(
      journey({
        gps_route: {
          type: "LineString",
          coordinates: [
            [139.7, 35.6],
            [139.8, 35.7],
          ],
        },
      }),
      [],
    )
    expect(result.routeSegments).toEqual([
      [
        [139.7, 35.6],
        [139.8, 35.7],
      ],
    ])
  })

  test("maps transit kind data into the existing station view model", () => {
    const result = adaptJourney(journey(), [
      memento({
        kind: "transit",
        kind_data: {
          operator: "JR East",
          line: "Tokaido Shinkansen",
          from: { name: "Tokyo", coords: [139.7671, 35.6812] },
          to: { name: "Kyoto", coords: [135.7583, 34.9859] },
        },
      }),
    ])
    expect(result.mementos[0].kind).toBe("transit")
    expect(result.mementos[0].transit?.from.coords).toEqual([139.7671, 35.6812])
    expect(result.mementos[0].transit?.to.name).toBe("Kyoto")
  })

  test("preserves ticket mementos for the shared ticket template", () => {
    const result = adaptJourney(journey(), [
      memento({
        kind: "ticket",
        title: "大室山リフト",
        kind_data: {
          name: "Omuroyama Climbing Lift",
          ticket_type: "おとな往復（中学生以上）",
          price: { amount: 1000, currency: "JPY" },
          venue: { name: "Omuroyama Lift", coords: [139.0975, 34.9071] },
        },
      }),
    ])

    expect(result.mementos[0].kind).toBe("ticket")
    expect(result.mementos[0].kindData?.ticket_type).toBe("おとな往復（中学生以上）")
  })
})
