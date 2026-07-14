import type { Journey } from "../data"
import { adaptJourney } from "./adapt"
import type { ApiJourney, ApiJourneyListItem, ApiMemento } from "./types"

export async function loadJourney(id: string): Promise<Journey> {
  const apiBase = import.meta.env.VITE_API_BASE || ""
  const journeyUrl = `${apiBase}/api/v1/journeys/${id}`
  const mementosUrl = `${apiBase}/api/v1/journeys/${id}/mementos`

  const [journeyRes, mementosRes] = await Promise.all([fetch(journeyUrl), fetch(mementosUrl)])

  if (!journeyRes.ok) {
    throw new Error(`Failed to load journey details for id "${id}": ${journeyRes.statusText}`)
  }
  if (!mementosRes.ok) {
    throw new Error(`Failed to load mementos for id "${id}": ${mementosRes.statusText}`)
  }

  const apiJourney = (await journeyRes.json()) as ApiJourney
  // The API marshals empty collections as null (Go nil slices); coalesce so
  // journeys with no mementos don't crash the adapter's iteration.
  const apiMementos = ((await mementosRes.json()) as ApiMemento[] | null) ?? []

  return adaptJourney(apiJourney, apiMementos)
}

export async function loadJourneys(): Promise<Journey[]> {
  const apiBase = import.meta.env.VITE_API_BASE || ""
  const url = `${apiBase}/api/v1/journeys`
  const res = await fetch(url)
  if (!res.ok) {
    throw new Error(`Failed to load journeys list: ${res.statusText}`)
  }

  const items = (await res.json()) as ApiJourneyListItem[]

  // Fetch details and mementos in parallel to build full Journey objects
  const journeys = await Promise.all(
    items.map(async (item) => {
      const journey = await loadJourney(item.id)
      journey.representativeDots = item.representative_dots ?? []
      return journey
    }),
  )
  return journeys
}
