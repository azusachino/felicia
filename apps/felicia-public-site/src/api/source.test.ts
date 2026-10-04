import { describe, expect, test, vi, beforeAll, afterEach } from "vitest"
import { loadJourney, loadJourneys } from "./source"
import type { ApiJourney, ApiMemento } from "@felicia/reader"
import { loadGoldenRouteFixture } from "../../tests/fixtures"

const originalFetch = globalThis.fetch
let journeyID = ""

describe("source API", () => {
  let japanSpringJourney: ApiJourney
  let japanSpringMementos: ApiMemento[]

  beforeAll(async () => {
    const fixture = await loadGoldenRouteFixture()
    japanSpringJourney = fixture.journey
    japanSpringMementos = fixture.mementos
    journeyID = japanSpringJourney.id
  })

  afterEach(() => {
    globalThis.fetch = originalFetch
  })

  test("loadJourney fetches detail and mementos, then adapts them (dev mode)", async () => {
    globalThis.fetch = vi.fn((url: string | URL) => {
      const urlStr = url.toString()
      if (urlStr.endsWith(`/api/v1/journeys/${journeyID}.json`)) {
        return Promise.resolve(Response.json(japanSpringJourney))
      }
      if (urlStr.endsWith(`/api/v1/journeys/${journeyID}/mementos.json`)) {
        return Promise.resolve(Response.json(japanSpringMementos))
      }
      return Promise.resolve(new Response("Not Found", { status: 404 }))
    }) as unknown as typeof fetch

    const journey = await loadJourney(journeyID)

    expect(journey.id).toBe(journeyID)
    expect(journey.title.en).toBe(japanSpringJourney.title)
    expect(journey.mementos).toHaveLength(japanSpringMementos.length)
  })

  test("loadJourneys fetches list, then detail/mementos for each, and adapts (dev mode)", async () => {
    const listFixture = [
      {
        id: journeyID,
        slug: japanSpringJourney.slug,
        title: japanSpringJourney.title,
        memento_count: japanSpringMementos.length,
        representative_dots: [],
      },
    ]

    globalThis.fetch = vi.fn((url: string | URL) => {
      const urlStr = url.toString()
      if (urlStr.endsWith("/api/v1/journeys.json")) {
        return Promise.resolve(Response.json(listFixture))
      }
      if (urlStr.endsWith(`/api/v1/journeys/${journeyID}.json`)) {
        return Promise.resolve(Response.json(japanSpringJourney))
      }
      if (urlStr.endsWith(`/api/v1/journeys/${journeyID}/mementos.json`)) {
        return Promise.resolve(Response.json(japanSpringMementos))
      }
      return Promise.resolve(new Response("Not Found", { status: 404 }))
    }) as unknown as typeof fetch

    const journeys = await loadJourneys()

    expect(journeys).toHaveLength(1)
    expect(journeys[0].id).toBe(journeyID)
    expect(journeys[0].representativeDots).toEqual([])
  })

  test("loadJourneys tolerates null mementos and representative_dots (empty journey)", async () => {
    const bareID = "f02ed764-5a4a-41c1-8553-3a283832c7d7"
    const listFixture = [{ id: bareID, slug: "bare-2026", title: "空路", memento_count: 0, representative_dots: null }]

    globalThis.fetch = vi.fn((url: string | URL) => {
      const urlStr = url.toString()
      if (urlStr.endsWith("/api/v1/journeys.json")) {
        return Promise.resolve(Response.json(listFixture))
      }
      if (urlStr.endsWith(`/api/v1/journeys/${bareID}/mementos.json`)) {
        return Promise.resolve(Response.json(null))
      }
      if (urlStr.endsWith(`/api/v1/journeys/${bareID}.json`)) {
        return Promise.resolve(Response.json({ id: bareID, slug: "bare-2026", gps_route: null }))
      }
      return Promise.resolve(new Response("Not Found", { status: 404 }))
    }) as unknown as typeof fetch

    const journeys = await loadJourneys()

    expect(journeys).toHaveLength(1)
    expect(journeys[0].mementos).toEqual([])
    expect(journeys[0].representativeDots).toEqual([])
  })

  test("loadJourney throws on non-ok response", async () => {
    globalThis.fetch = vi.fn(() => Promise.resolve(new Response("Error", { status: 500 }))) as unknown as typeof fetch

    await expect(loadJourney(journeyID)).rejects.toThrow()
  })

  test("loadJourneys throws on non-ok response", async () => {
    globalThis.fetch = vi.fn(() => Promise.resolve(new Response("Error", { status: 500 }))) as unknown as typeof fetch

    await expect(loadJourneys()).rejects.toThrow()
  })
})
