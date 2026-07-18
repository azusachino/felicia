export interface AdminJourney {
  id: string
  journal_id: string
  slug: string
  title: string
  place: string
  country?: string
  region?: string
  date_start: string
  date_end: string
  authored_fields: string[]
}

export interface AdminMemento {
  id: string
  journey_id: string
  kind: string
  seq: number
  title: string
  place: string
  essay?: string
  vendor?: string
  price_amount?: number
  price_currency?: string
  state: "candidate" | "draft" | "authored" | "published" | "archived" | string
}

export interface AdminOverview {
  journeys: AdminJourney[]
  mementos: AdminMemento[]
}

function apiURL(path: string): string {
  const base = (import.meta.env.VITE_API_BASE as string | undefined) ?? ""
  return `${base}${path}`
}

async function getJSON<T>(path: string): Promise<T> {
  const response = await fetch(apiURL(path), { headers: { Accept: "application/json" } })
  if (!response.ok) throw new Error(`Felicia API returned ${response.status}`)
  return (await response.json()) as T
}

export async function loadAdminOverview(): Promise<AdminOverview> {
  const journeys = await getJSON<AdminJourney[]>("/api/admin/journeys")
  const mementos = (await Promise.all(journeys.map((journey) => getJSON<AdminMemento[]>(`/api/admin/journeys/${journey.id}/mementos`)))).flat()
  return { journeys, mementos }
}
