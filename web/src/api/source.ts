import type { Journey } from '../data'
import { adaptJourney } from './adapt'
import type { ApiJourney, ApiJourneyListItem, ApiMemento } from './types'

export async function loadJourney(slug: string): Promise<Journey> {
  const apiBase = import.meta.env.VITE_API_BASE || ''
  const isProd = import.meta.env.PROD

  const journeyUrl = isProd ? `/api/v1/journeys/${slug}.json` : `${apiBase}/api/v1/journeys/${slug}`
  const mementosUrl = isProd
    ? `/api/v1/journeys/${slug}/mementos.json`
    : `${apiBase}/api/v1/journeys/${slug}/mementos`

  const [journeyRes, mementosRes] = await Promise.all([fetch(journeyUrl), fetch(mementosUrl)])

  if (!journeyRes.ok) {
    throw new Error(`Failed to load journey details for slug "${slug}": ${journeyRes.statusText}`)
  }
  if (!mementosRes.ok) {
    throw new Error(`Failed to load mementos for slug "${slug}": ${mementosRes.statusText}`)
  }

  const apiJourney = (await journeyRes.json()) as ApiJourney
  const apiMementos = (await mementosRes.json()) as ApiMemento[]

  return adaptJourney(apiJourney, apiMementos)
}

export async function loadJourneys(): Promise<Journey[]> {
  const apiBase = import.meta.env.VITE_API_BASE || ''
  const isProd = import.meta.env.PROD

  const url = isProd ? '/api/v1/journeys.json' : `${apiBase}/api/v1/journeys`
  const res = await fetch(url)
  if (!res.ok) {
    throw new Error(`Failed to load journeys list: ${res.statusText}`)
  }

  const items = (await res.json()) as ApiJourneyListItem[]

  // Fetch details and mementos in parallel to build full Journey objects
  return Promise.all(items.map((item) => loadJourney(item.slug)))
}
