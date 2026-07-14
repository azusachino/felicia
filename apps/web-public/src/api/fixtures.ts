import type { ApiJourney, ApiMemento } from "./types"

type Scenario = {
  journeys: Array<{
    id: string
    slug: string
    title: string
    place: string
    country?: string
    region?: string
    date_start: string
    date_end: string
    gps_route: number[][][]
    mementos: Array<{
      id: string
      kind: string
      seq: number
      occurred_at: string
      occurred_tz?: string
      title: string
      place: string
      geom: number[]
      kind_data: Record<string, unknown>
      photos?: ApiMemento["photos"]
    }>
  }>
}

export async function loadGoldenRouteFixture(): Promise<{
  journey: ApiJourney
  mementos: ApiMemento[]
}> {
  const scenario = (await Bun.file(
    new URL("../../../../scripts/data.json", import.meta.url),
  ).json()) as Scenario
  const source = scenario.journeys[0]
  const journey: ApiJourney = {
    id: source.id,
    journal_id: "0190cbde-f300-7000-8000-000000000000",
    slug: source.slug,
    title: source.title,
    place: source.place,
    country: source.country,
    region: source.region,
    date_start: source.date_start,
    date_end: source.date_end,
    gps_route: { type: "MultiLineString", coordinates: source.gps_route },
    authored_fields: [],
  }
  const mementos: ApiMemento[] = source.mementos.map((memento) => ({
    id: memento.id,
    kind: memento.kind,
    seq: memento.seq,
    occurred_at: memento.occurred_at,
    journey_id: source.id,
    occurred_tz: memento.occurred_tz ?? "Asia/Tokyo",
    title: memento.title,
    place: memento.place,
    geom: { type: "Point", coordinates: memento.geom },
    kind_data: memento.kind_data,
    photos: memento.photos,
    authored_fields: [],
  }))
  return { journey, mementos }
}
