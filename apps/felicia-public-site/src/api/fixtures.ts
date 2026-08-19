import type { ApiJourney, ApiMemento } from "@felicia/shared"

type LocalJourney = {
  id: string
  journal_id: string
  slug: string
  title: string
  place: string
  country?: string
  region?: string
  date_start: string
  date_end: string
  source_ref?: string
}

type LocalMemento = {
  id: string
  kind: string
  seq: number
  occurred_at: string
  occurred_tz?: string
  title: string
  place: string
  geom: number[]
  kind_data: Record<string, unknown>
  essay?: string
  media?: Array<{ path: string; caption?: string }>
}

export async function loadGoldenRouteFixture(): Promise<{
  journey: ApiJourney
  mementos: ApiMemento[]
}> {
  const root = new URL("../../../../examples/preview/local-journey/", import.meta.url)
  const source = (await Bun.file(new URL("journey.json", root)).json()) as LocalJourney
  const mementoSource = (await Bun.file(new URL("mementos.json", root)).json()) as { mementos: LocalMemento[] }
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
    gps_route: {
      type: "MultiLineString",
      coordinates: [
        [
          [139.7074, 35.8163],
          [138.9807, 34.6667],
          [139.0975, 34.9071],
          [139.1394, 34.8918],
        ],
      ],
    },
    authored_fields: [],
  }
  const mementos: ApiMemento[] = mementoSource.mementos.map((memento) => ({
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
    essay: memento.essay,
    photos: (memento.media ?? []).map((photo, index) => ({
      id: `${memento.id}-photo-${index + 1}`,
      memento_id: memento.id,
      object_key: photo.path,
      content_hash: "",
      caption: photo.caption,
      seq: index + 1,
    })),
  }))
  return { journey, mementos }
}
