import type { ApiJourney, ApiMemento } from "@felicia/reader"

export async function loadGoldenRouteFixture(): Promise<{
  journey: ApiJourney
  mementos: ApiMemento[]
}> {
  const journey: ApiJourney = {
    id: "test-journey-2026",
    journal_id: "test-journal-2026",
    slug: "test-journey-2026",
    title: "Test journey 2026",
    place: "Japan",
    country: "Japan",
    region: "Shizuoka",
    date_start: "2026-08-01",
    date_end: "2026-08-02",
    gps_route: {
      type: "MultiLineString",
      coordinates: [
        [
          [138.98, 34.66],
          [139.1, 34.91],
          [139.14, 34.89],
          [139.2, 34.8],
        ],
      ],
    },
    authored_fields: [],
  }
  const mementos: ApiMemento[] = [
    {
      id: "test-memento-beach",
      journey_id: journey.id,
      kind: "souvenir",
      seq: 1,
      occurred_at: "2026-08-01T16:12:35+09:00",
      occurred_tz: "Asia/Tokyo",
      title: "Beach stop",
      place: "Beach",
      geom: { type: "Point", coordinates: [138.98, 34.66] },
      kind_data: { name: "Beach stop" },
      essay: "A quiet pause by the water.",
      photos: [
        {
          id: "test-photo-beach",
          memento_id: "test-memento-beach",
          object_key: "fixtures/beach.jpg",
          content_hash: "",
          caption: "Beach stop",
          seq: 1,
        },
      ],
    },
    {
      id: "test-memento-mountain",
      journey_id: journey.id,
      kind: "ticket",
      seq: 2,
      occurred_at: "2026-08-02T11:43:22+09:00",
      occurred_tz: "Asia/Tokyo",
      title: "Mountain lift ticket",
      place: "Mountain",
      geom: { type: "Point", coordinates: [139.1, 34.91] },
      kind_data: { ticket_variant: "mountain" },
      photos: [],
    },
    {
      id: "test-memento-coast",
      journey_id: journey.id,
      kind: "souvenir",
      seq: 3,
      occurred_at: "2026-08-02T15:08:20+09:00",
      occurred_tz: "Asia/Tokyo",
      title: "Coast walk",
      place: "Coast",
      geom: { type: "Point", coordinates: [139.14, 34.89] },
      kind_data: { name: "Coast walk" },
      photos: [],
    },
  ]
  return { journey, mementos }
}
