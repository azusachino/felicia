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

export type MementoState = "candidate" | "draft" | "authored" | "published" | "archived"

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
  state: MementoState | string
}

export type StopCandidateState = "proposed" | "kept" | "ignored" | "merged"

export interface AdminStopCandidate {
  id: string
  journey_id: string
  label: string
  state: StopCandidateState | string
  confidence: number
  arrive: string
  depart: string
}

// Per-journey rollup for the journey list view (#/). Fetched alongside the
// journey itself so the list can show a memento count, a state badge
// summary, and a stop-candidate count without the caller re-deriving them.
export interface AdminJourneySummary {
  journey: AdminJourney
  mementoCount: number
  stateCounts: Partial<Record<MementoState, number>>
  // null when the stop-candidate store isn't available (e.g. intake sources
  // not configured) rather than failing the whole journey row.
  stopCandidateCount: number | null
}

// Response shapes for the import/preview triggers on the journey detail view
// (ADMIN-01.2). sync-route writes gps_route and returns the resulting
// geometry; visits/tray are read-only previews and persist nothing.

export interface SyncRouteResult {
  status: string
  // GeoJSON LineString or MultiLineString, or null if the journey has no
  // route yet.
  gps_route: { type: "LineString"; coordinates: [number, number][] } | { type: "MultiLineString"; coordinates: [number, number][][] } | null
}

export interface AdminVisitPreview {
  coord: [number, number]
  label: string
  arrive: string
  depart: string
  confidence: number
  source_ref: string
}

export interface AdminPhotoTrayItem {
  id: string
  at: string
  coord?: [number, number]
  checksum: string
  source_ref: string
}

function apiURL(path: string): string {
  const base = (import.meta.env.VITE_API_BASE as string | undefined) ?? ""
  return `${base}${path}`
}

async function apiErrorMessage(response: Response): Promise<string> {
  try {
    const body = (await response.clone().json()) as { error?: string }
    if (body?.error) return body.error
  } catch {
    // Response body wasn't JSON (or was empty) — fall through to the status line.
  }
  return `Felicia API returned ${response.status}`
}

async function getJSON<T>(path: string): Promise<T> {
  const response = await fetch(apiURL(path), { headers: { Accept: "application/json" } })
  if (!response.ok) throw new Error(await apiErrorMessage(response))
  return (await response.json()) as T
}

async function postJSON<T>(path: string, body?: unknown): Promise<T> {
  const response = await fetch(apiURL(path), {
    method: "POST",
    headers: { Accept: "application/json", "Content-Type": "application/json" },
    body: body === undefined ? undefined : JSON.stringify(body),
  })
  if (!response.ok) throw new Error(await apiErrorMessage(response))
  return (await response.json()) as T
}

export async function listJourneys(): Promise<AdminJourney[]> {
  return getJSON<AdminJourney[]>("/api/admin/journeys")
}

export async function getJourney(journeyId: string): Promise<AdminJourney> {
  return getJSON<AdminJourney>(`/api/admin/journeys/${journeyId}`)
}

export async function listMementos(journeyId: string): Promise<AdminMemento[]> {
  return getJSON<AdminMemento[]>(`/api/admin/journeys/${journeyId}/mementos`)
}

export async function listStopCandidates(journeyId: string): Promise<AdminStopCandidate[]> {
  return getJSON<AdminStopCandidate[]>(`/api/admin/journeys/${journeyId}/stop-candidates`)
}

export function countMementosByState(mementos: AdminMemento[]): Partial<Record<MementoState, number>> {
  const counts: Partial<Record<MementoState, number>> = {}
  for (const memento of mementos) {
    const state = memento.state as MementoState
    counts[state] = (counts[state] ?? 0) + 1
  }
  return counts
}

// Only "proposed" candidates are outstanding work for an author; kept/
// ignored/merged are already resolved decisions, so they don't count toward
// the actionable total shown on the journey list.
export function countPendingStopCandidates(candidates: AdminStopCandidate[]): number {
  return candidates.filter((candidate) => candidate.state === "proposed").length
}

export function sortMementosBySeq(mementos: AdminMemento[]): AdminMemento[] {
  return [...mementos].sort((a, b) => a.seq - b.seq)
}

export async function loadJourneySummaries(): Promise<AdminJourneySummary[]> {
  const journeys = await listJourneys()
  return Promise.all(
    journeys.map(async (journey) => {
      const mementos = await listMementos(journey.id)
      let stopCandidateCount: number | null = null
      try {
        stopCandidateCount = countPendingStopCandidates(await listStopCandidates(journey.id))
      } catch {
        // Intake candidate storage may be unavailable (503); don't let that
        // fail the whole list, just leave the count unknown for this journey.
      }
      return {
        journey,
        mementoCount: mementos.length,
        stateCounts: countMementosByState(mementos),
        stopCandidateCount,
      }
    }),
  )
}

export async function syncRoute(journeyId: string): Promise<SyncRouteResult> {
  return postJSON<SyncRouteResult>(`/api/admin/journeys/${journeyId}/sync-route`)
}

export async function syncVisits(journeyId: string): Promise<AdminVisitPreview[]> {
  return getJSON<AdminVisitPreview[]>(`/api/admin/journeys/${journeyId}/visits`)
}

export async function photoTray(journeyId: string): Promise<AdminPhotoTrayItem[]> {
  return getJSON<AdminPhotoTrayItem[]>(`/api/admin/journeys/${journeyId}/tray`)
}

// Flattens a sync-route response's geometry into a point count for the
// trigger's "success (counts)" summary.
export function routePointCount(result: SyncRouteResult): number {
  if (!result.gps_route) return 0
  if (result.gps_route.type === "LineString") return result.gps_route.coordinates.length
  return result.gps_route.coordinates.reduce((total, line) => total + line.length, 0)
}
