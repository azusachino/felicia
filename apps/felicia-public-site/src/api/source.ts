import { adaptJourney, type ApiJourney, type ApiJourneyListItem, type ApiMemento, type ApiSiteSettings, type Journey } from "@felicia/reader"

function mediaURL(value: string): string {
  if (/^(?:https?:)?\//.test(value)) return value
  return `${import.meta.env.BASE_URL || "/"}${value}`
}

function endpoint(path: string): string {
  const apiBase = import.meta.env.VITE_API_BASE || ""
  const baseURL = apiBase || import.meta.env.BASE_URL || "/"
  return `${baseURL.replace(/\/$/, "")}${path}.json`
}

export async function loadSiteSettings(): Promise<ApiSiteSettings> {
  const url = endpoint("/api/v1/site")
  const res = await fetch(url)
  if (!res.ok) {
    throw new Error(`Failed to load site settings: ${res.statusText}`)
  }

  return (await res.json()) as ApiSiteSettings
}

export async function loadJourney(id: string): Promise<Journey> {
  const journeyUrl = endpoint(`/api/v1/journeys/${id}`)
  const mementosUrl = endpoint(`/api/v1/journeys/${id}/mementos`)

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

  return adaptJourney(apiJourney, apiMementos, mediaURL)
}

export async function loadJourneys(): Promise<Journey[]> {
  const url = endpoint("/api/v1/journeys")
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
