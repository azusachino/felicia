import { describe, expect, test, mock, beforeAll, afterAll } from 'bun:test'
import { loadJourney, loadJourneys } from './source'
import type { ApiJourney, ApiMemento } from './types'
import { loadGoldenRouteFixture } from './testdata'

// Access import.meta.env in a type-safe way that is extensible and avoids ESLint any/ignore rules
const importMeta = import.meta as unknown as {
  env: {
    VITE_API_BASE?: string
    PROD?: boolean
    DEV?: boolean
  }
}

if (!importMeta.env) {
  importMeta.env = {
    VITE_API_BASE: 'http://localhost:8080',
    PROD: false,
    DEV: true,
  }
}

const originalFetch = globalThis.fetch
const journeyID = '0190cbde-f300-7000-8000-111111111111'

describe('source API', () => {
  let japanSpringJourney: ApiJourney
  let japanSpringMementos: ApiMemento[]

  beforeAll(async () => {
    const fixture = await loadGoldenRouteFixture()
    japanSpringJourney = fixture.journey
    japanSpringMementos = fixture.mementos
  })

  afterAll(() => {
    globalThis.fetch = originalFetch
  })

  test('loadJourney fetches detail and mementos, then adapts them (dev mode)', async () => {
    globalThis.fetch = mock((url: string | URL) => {
      const urlStr = url.toString()
      expect(urlStr).not.toContain('.json') // should not have .json in dev mode
      if (urlStr.endsWith(`/api/v1/journeys/${journeyID}`)) {
        return Promise.resolve(Response.json(japanSpringJourney))
      }
      if (urlStr.endsWith(`/api/v1/journeys/${journeyID}/mementos`)) {
        return Promise.resolve(Response.json(japanSpringMementos))
      }
      return Promise.resolve(new Response('Not Found', { status: 404 }))
    }) as unknown as typeof fetch

    const journey = await loadJourney(journeyID)

    expect(journey.id).toBe('0190cbde-f300-7000-8000-111111111111')
    expect(journey.title.en).toBe('Narita Express Day Trip')
    expect(journey.mementos).toHaveLength(5)
  })

  test('loadJourney fetches detail and mementos (prod/static mode)', async () => {
    importMeta.env.PROD = true

    globalThis.fetch = mock((url: string | URL) => {
      const urlStr = url.toString()
      expect(urlStr).toContain('.json') // should have .json in static mode
      expect(urlStr).not.toContain('http://localhost:8080')
      if (urlStr.endsWith(`/api/v1/journeys/${journeyID}.json`)) {
        return Promise.resolve(Response.json(japanSpringJourney))
      }
      if (urlStr.endsWith(`/api/v1/journeys/${journeyID}/mementos.json`)) {
        return Promise.resolve(Response.json(japanSpringMementos))
      }
      return Promise.resolve(new Response('Not Found', { status: 404 }))
    }) as unknown as typeof fetch

    try {
      const journey = await loadJourney(journeyID)
      expect(journey.id).toBe('0190cbde-f300-7000-8000-111111111111')
    } finally {
      importMeta.env.PROD = false
    }
  })

  test('loadJourneys fetches list, then detail/mementos for each, and adapts (dev mode)', async () => {
    const listFixture = [
      {
        id: journeyID,
        slug: 'golden-route',
        title: '日本ゴールデンルート',
        memento_count: 11,
        representative_dots: [],
      },
    ]

    globalThis.fetch = mock((url: string | URL) => {
      const urlStr = url.toString()
      expect(urlStr).not.toContain('.json') // should not have .json in dev mode
      if (urlStr.endsWith('/api/v1/journeys')) {
        return Promise.resolve(Response.json(listFixture))
      }
      if (urlStr.endsWith(`/api/v1/journeys/${journeyID}`)) {
        return Promise.resolve(Response.json(japanSpringJourney))
      }
      if (urlStr.endsWith(`/api/v1/journeys/${journeyID}/mementos`)) {
        return Promise.resolve(Response.json(japanSpringMementos))
      }
      return Promise.resolve(new Response('Not Found', { status: 404 }))
    }) as unknown as typeof fetch

    const journeys = await loadJourneys()

    expect(journeys).toHaveLength(1)
    expect(journeys[0].id).toBe('0190cbde-f300-7000-8000-111111111111')
    expect(journeys[0].representativeDots).toEqual([])
  })

  test('loadJourneys fetches list and detail/mementos (prod/static mode)', async () => {
    importMeta.env.PROD = true

    const listFixture = [
      {
        id: journeyID,
        slug: 'golden-route',
        title: '日本ゴールデンルート',
        memento_count: 11,
        representative_dots: [],
      },
    ]

    globalThis.fetch = mock((url: string | URL) => {
      const urlStr = url.toString()
      expect(urlStr).toContain('.json') // should have .json in static mode
      expect(urlStr).not.toContain('http://localhost:8080')
      if (urlStr.endsWith('/api/v1/journeys.json')) {
        return Promise.resolve(Response.json(listFixture))
      }
      if (urlStr.endsWith(`/api/v1/journeys/${journeyID}.json`)) {
        return Promise.resolve(Response.json(japanSpringJourney))
      }
      if (urlStr.endsWith(`/api/v1/journeys/${journeyID}/mementos.json`)) {
        return Promise.resolve(Response.json(japanSpringMementos))
      }
      return Promise.resolve(new Response('Not Found', { status: 404 }))
    }) as unknown as typeof fetch

    try {
      const journeys = await loadJourneys()
      expect(journeys).toHaveLength(1)
      expect(journeys[0].id).toBe('0190cbde-f300-7000-8000-111111111111')
      expect(journeys[0].representativeDots).toEqual([])
    } finally {
      importMeta.env.PROD = false
    }
  })

  test('loadJourney throws on non-ok response', async () => {
    globalThis.fetch = mock(() =>
      Promise.resolve(new Response('Error', { status: 500 })),
    ) as unknown as typeof fetch

    expect(loadJourney(journeyID)).rejects.toThrow()
  })

  test('loadJourneys throws on non-ok response', async () => {
    globalThis.fetch = mock(() =>
      Promise.resolve(new Response('Error', { status: 500 })),
    ) as unknown as typeof fetch

    expect(loadJourneys()).rejects.toThrow()
  })
})
