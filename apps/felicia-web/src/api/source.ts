import { adaptJourney, type ApiJourney, type ApiJourneyListItem, type ApiMemento, type ApiSiteSettings, type Journey } from "@felicia/reader"

function endpoint(path: string): string {
  const base = import.meta.env.VITE_API_BASE || ""
  return `${base.replace(/\/$/, "")}${path}`
}

function mediaURL(value: string): string {
  if (/^(?:https?:)?\//.test(value)) return value
  const base = import.meta.env.VITE_MEDIA_BASE || "/media/"
  return `${base.replace(/\/$/, "")}/${value.replace(/^\//, "")}`
}

export async function loadSiteSettings(): Promise<ApiSiteSettings> {
  const res = await fetch(endpoint("/api/v1/site"))
  if (!res.ok) throw new Error(`Failed to load site settings: ${res.statusText}`)
  return (await res.json()) as ApiSiteSettings
}

export async function loadJourney(id: string): Promise<Journey> {
  const [journeyRes, mementosRes] = await Promise.all([fetch(endpoint(`/api/v1/journeys/${id}`)), fetch(endpoint(`/api/v1/journeys/${id}/mementos`))])
  if (!journeyRes.ok) throw new Error(`Failed to load journey "${id}": ${journeyRes.statusText}`)
  if (!mementosRes.ok) throw new Error(`Failed to load mementos for "${id}": ${mementosRes.statusText}`)
  const apiJourney = (await journeyRes.json()) as ApiJourney
  const apiMementos = ((await mementosRes.json()) as ApiMemento[] | null) ?? []
  return adaptJourney(apiJourney, apiMementos, mediaURL)
}

export async function loadJourneys(): Promise<Journey[]> {
  const res = await fetch(endpoint("/api/v1/journeys"))
  if (!res.ok) throw new Error(`Failed to load journeys: ${res.statusText}`)
  const items = (await res.json()) as ApiJourneyListItem[]
  return Promise.all(
    items.map(async (item) => {
      const journey = await loadJourney(item.id)
      journey.representativeDots = item.representative_dots ?? []
      return journey
    }),
  )
}
