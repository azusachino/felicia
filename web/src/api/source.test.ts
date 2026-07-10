import { describe, expect, test, mock, beforeAll, afterAll } from 'bun:test'
import { loadJourney, loadJourneys } from './source'
import type { ApiJourney, ApiMemento } from './types'

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

describe('source API', () => {
  let japanSpringJourney: ApiJourney
  let japanSpringMementos: ApiMemento[]

  beforeAll(async () => {
    japanSpringJourney = (await Bun.file(
      new URL('./__fixtures__/journeys/japan-spring-2026.json', import.meta.url),
    ).json()) as ApiJourney

    japanSpringMementos = (await Bun.file(
      new URL('./__fixtures__/journeys/japan-spring-2026/mementos.json', import.meta.url),
    ).json()) as ApiMemento[]
  })

  afterAll(() => {
    globalThis.fetch = originalFetch
  })

  test('loadJourney fetches detail and mementos, then adapts them (dev mode)', async () => {
    globalThis.fetch = mock((url: string | URL) => {
      const urlStr = url.toString()
      expect(urlStr).not.toContain('.json') // should not have .json in dev mode
      if (urlStr.endsWith('/api/v1/journeys/japan-spring-2026')) {
        return Promise.resolve(Response.json(japanSpringJourney))
      }
      if (urlStr.endsWith('/api/v1/journeys/japan-spring-2026/mementos')) {
        return Promise.resolve(Response.json(japanSpringMementos))
      }
      return Promise.resolve(new Response('Not Found', { status: 404 }))
    }) as unknown as typeof fetch

    const journey = await loadJourney('japan-spring-2026')

    expect(journey.id).toBe('0190cbde-f300-7000-8000-111111111111')
    expect(journey.title.ja).toBe('日本春旅 2026')
    expect(journey.mementos).toHaveLength(2)
  })

  test('loadJourney fetches detail and mementos (prod/static mode)', async () => {
    importMeta.env.PROD = true

    globalThis.fetch = mock((url: string | URL) => {
      const urlStr = url.toString()
      expect(urlStr).toContain('.json') // should have .json in static mode
      expect(urlStr).not.toContain('http://localhost:8080')
      if (urlStr.endsWith('/api/v1/journeys/japan-spring-2026.json')) {
        return Promise.resolve(Response.json(japanSpringJourney))
      }
      if (urlStr.endsWith('/api/v1/journeys/japan-spring-2026/mementos.json')) {
        return Promise.resolve(Response.json(japanSpringMementos))
      }
      return Promise.resolve(new Response('Not Found', { status: 404 }))
    }) as unknown as typeof fetch

    try {
      const journey = await loadJourney('japan-spring-2026')
      expect(journey.id).toBe('0190cbde-f300-7000-8000-111111111111')
    } finally {
      importMeta.env.PROD = false
    }
  })

  test('loadJourneys fetches list, then detail/mementos for each, and adapts (dev mode)', async () => {
    const listFixture = [
      {
        slug: 'japan-spring-2026',
        title: '日本春旅 2026',
        memento_count: 2,
        representative_dots: [],
      },
    ]

    globalThis.fetch = mock((url: string | URL) => {
      const urlStr = url.toString()
      expect(urlStr).not.toContain('.json') // should not have .json in dev mode
      if (urlStr.endsWith('/api/v1/journeys')) {
        return Promise.resolve(Response.json(listFixture))
      }
      if (urlStr.endsWith('/api/v1/journeys/japan-spring-2026')) {
        return Promise.resolve(Response.json(japanSpringJourney))
      }
      if (urlStr.endsWith('/api/v1/journeys/japan-spring-2026/mementos')) {
        return Promise.resolve(Response.json(japanSpringMementos))
      }
      return Promise.resolve(new Response('Not Found', { status: 404 }))
    }) as unknown as typeof fetch

    const journeys = await loadJourneys()

    expect(journeys).toHaveLength(1)
    expect(journeys[0].id).toBe('0190cbde-f300-7000-8000-111111111111')
  })

  test('loadJourneys fetches list and detail/mementos (prod/static mode)', async () => {
    importMeta.env.PROD = true

    const listFixture = [
      {
        slug: 'japan-spring-2026',
        title: '日本春旅 2026',
        memento_count: 2,
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
      if (urlStr.endsWith('/api/v1/journeys/japan-spring-2026.json')) {
        return Promise.resolve(Response.json(japanSpringJourney))
      }
      if (urlStr.endsWith('/api/v1/journeys/japan-spring-2026/mementos.json')) {
        return Promise.resolve(Response.json(japanSpringMementos))
      }
      return Promise.resolve(new Response('Not Found', { status: 404 }))
    }) as unknown as typeof fetch

    try {
      const journeys = await loadJourneys()
      expect(journeys).toHaveLength(1)
      expect(journeys[0].id).toBe('0190cbde-f300-7000-8000-111111111111')
    } finally {
      importMeta.env.PROD = false
    }
  })

  test('loadJourney throws on non-ok response', async () => {
    globalThis.fetch = mock(() =>
      Promise.resolve(new Response('Error', { status: 500 })),
    ) as unknown as typeof fetch

    expect(loadJourney('japan-spring-2026')).rejects.toThrow()
  })

  test('loadJourneys throws on non-ok response', async () => {
    globalThis.fetch = mock(() =>
      Promise.resolve(new Response('Error', { status: 500 })),
    ) as unknown as typeof fetch

    expect(loadJourneys()).rejects.toThrow()
  })
})
