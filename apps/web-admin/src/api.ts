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

// geom as GET /api/admin/mementos/{id} actually returns it: paulmach/orb's
// Point/LineString marshal to plain JSON arrays (no custom MarshalJSON), so
// this is NOT GeoJSON — a point is [lng, lat], an edge is [[lng, lat], ...].
// (POST /api/admin/mementos expects the opposite shape — see
// UpsertMementoGeom below.) null when the memento has no geometry yet.
export type MementoGeom = [number, number] | [number, number][] | null

// The full memento record (ADMIN-01.4 editor), as returned by
// GET /api/admin/mementos/{id} and GET /api/admin/journeys/{id}/mementos.
// AdminMemento above stays the trimmed shape the journey list/detail views
// already depend on; this is additive, not a replacement.
export interface AdminMementoDetail {
  id: string
  journey_id: string
  kind: string
  seq: number
  occurred_at: string
  occurred_tz: string
  geom: MementoGeom
  title: string
  place: string
  vendor?: string
  essay?: string
  price_amount?: number
  price_currency?: string
  kind_data: Record<string, unknown>
  source_ref?: string
  authored_fields: string[]
  orphaned_at?: string
  state: MementoState | string
  revision: number
  created_at: string
  updated_at: string
}

// Request shape for POST /api/admin/mementos's `geom` — the manual-patch
// upsert path decodes this GeoJSON-ish {type, coordinates} form (see
// mementoGeom/handleUpsertMemento in server/api/server.go), which is why it
// doesn't match AdminMementoDetail.geom above.
export type UpsertMementoGeom = { type: "Point"; coordinates: [number, number] } | { type: "LineString"; coordinates: [number, number][] }

// Request body for POST /api/admin/mementos (ADMIN-01.4/01.5). This is a
// full-payload upsert: every field is sent every time, and expected_revision
// is the optimistic-concurrency token from the last-loaded state. A stale
// value (or none, against an existing record) surfaces as a 409 ApiError.
export interface UpsertMementoRequest {
  id: string
  journey_id: string
  kind: string
  seq: number
  occurred_at?: string
  occurred_tz?: string
  geom?: UpsertMementoGeom | null
  title: string
  place: string
  vendor?: string
  essay?: string
  price_amount?: number
  price_currency?: string
  kind_data: Record<string, unknown>
  source_ref?: string
  authored_fields?: string[]
  orphaned_at?: string
  state: MementoState | string
  expected_revision?: number
}

// Server-side validation issue (ADMIN-01.4 inline errors). Mirrors
// domain.Issue verbatim — no json tags, so capitalized field names, same
// deal as AdminTemplateField below.
export interface AdminIssue {
  Field: string
  Code: string
}

// Body of a snap-to-route response for a single point (as opposed to
// SyncRouteResult's whole-route form). Unlike AdminMementoDetail.geom, this
// one *is* GeoJSON — handleSnapToRoute responds via the shared toGeoJSON
// helper server-side.
export interface SnapPointResult {
  point: { type: "Point"; coordinates: [number, number] }
}

// Request body for POST /api/admin/photos (ADMIN-01.4 photo metadata
// editing). Metadata-only — object_key/content_hash identify bytes that
// already exist in the media store; this editor never uploads them.
export interface UpsertPhotoRequest {
  id: string
  memento_id: string
  object_key: string
  content_hash: string
  caption?: string
  seq: number
  taken_at?: string
  source_ref?: string
}

// A memento photo as returned by GET /api/admin/mementos/{id}/photos —
// domain.MementoPhoto, which (unlike the templates payload) carries json
// tags, so fields are snake_case. Photos arrive in sequence order.
export interface AdminMementoPhoto {
  id: string
  memento_id: string
  object_key: string
  content_hash: string
  caption?: string
  seq: number
  taken_at?: string
  source_ref?: string
  created_at: string
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
  // Optimistic-concurrency token: every review/promote call should echo this
  // back as expected_revision so a concurrent reviewer's write conflicts
  // instead of silently overwriting (ADMIN-01.3b conflict handling).
  revision: number
}

// intake/plan response (ADMIN-01.3a): the planner's diagnostics alongside the
// proposed stops, so the "Plan intake" trigger can report both counts.
export interface AdminIntakeIssue {
  severity: "info" | "warning" | "error"
  code: string
  message: string
}

export interface PlanIntakeResult {
  journey_id: string
  stops: AdminStopCandidate[]
  issues: AdminIntakeIssue[]
}

// GET /api/admin/templates response shape (ADMIN-01.3b kind picker). This
// mirrors core/domain.Template/Field verbatim, including the capitalized
// field names — those types have no `json` tags, so Go's default
// marshaling (exported field name as-is) is what actually goes over the
// wire.
export interface AdminTemplateField {
  Name: string
  Type: string
  Required: boolean
  Values: string[] | null
}

export interface AdminTemplate {
  Kind: string
  Anchor: "point" | "edge"
  Stub: string
  Fields: AdminTemplateField[]
}

export type AdminTemplateRegistry = Record<string, AdminTemplate>

// Request shape for the ignore/merge half of stop-candidate review (promote
// has its own dedicated endpoint/function). Mirrors stopReviewRequest in
// server/api/server.go.
export interface ReviewStopCandidatePatch {
  state: "ignored" | "merged"
  mergedInto?: string
  expectedRevision?: number
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

// ApiError carries the HTTP status alongside the server's error message so
// callers can distinguish a write conflict (409 — someone else changed this
// record) from any other failure without re-parsing the message string.
// `issues`, when the server identifies which field(s) failed validation
// (ADMIN-01.4), lets the memento editor render errors inline instead of as a
// single opaque message.
export class ApiError extends Error {
  status: number
  issues?: AdminIssue[]

  constructor(message: string, status: number, issues?: AdminIssue[]) {
    super(message)
    this.name = "ApiError"
    this.status = status
    this.issues = issues
  }
}

export function isConflict(cause: unknown): cause is ApiError {
  return cause instanceof ApiError && cause.status === 409
}

async function apiErrorDetail(response: Response): Promise<{ message: string; issues?: AdminIssue[] }> {
  try {
    const body = (await response.clone().json()) as { error?: string; issues?: AdminIssue[] }
    if (body?.error) return { message: body.error, issues: body.issues }
  } catch {
    // Response body wasn't JSON (or was empty) — fall through to the status line.
  }
  return { message: `Felicia API returned ${response.status}` }
}

async function getJSON<T>(path: string): Promise<T> {
  const response = await fetch(apiURL(path), { headers: { Accept: "application/json" } })
  if (!response.ok) {
    const { message, issues } = await apiErrorDetail(response)
    throw new ApiError(message, response.status, issues)
  }
  return (await response.json()) as T
}

async function postJSON<T>(path: string, body?: unknown): Promise<T> {
  const response = await fetch(apiURL(path), {
    method: "POST",
    headers: { Accept: "application/json", "Content-Type": "application/json" },
    body: body === undefined ? undefined : JSON.stringify(body),
  })
  if (!response.ok) {
    const { message, issues } = await apiErrorDetail(response)
    throw new ApiError(message, response.status, issues)
  }
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

// Intake inbox (ADMIN-01.3b): runs the intake planner over the journey's
// sources and persists the proposed stop candidates. Mirrors the
// import/preview triggers' pending/success/error shape on the caller side.
export async function planIntake(journeyId: string): Promise<PlanIntakeResult> {
  return postJSON<PlanIntakeResult>(`/api/admin/journeys/${journeyId}/intake/plan`)
}

export async function getTemplates(): Promise<AdminTemplateRegistry> {
  return getJSON<AdminTemplateRegistry>("/api/admin/templates")
}

// Marks a proposed stop candidate kept and creates a draft memento from it.
// expectedRevision guards against a concurrent reviewer; a stale value (or a
// candidate that isn't proposed anymore) surfaces as a 409 ApiError.
export async function promoteStopCandidate(candidateId: string, kind: string, expectedRevision?: number): Promise<AdminMemento> {
  return postJSON<AdminMemento>(`/api/admin/stop-candidates/${candidateId}/promote`, {
    kind,
    expected_revision: expectedRevision,
  })
}

// Ignores or merges a stop candidate via the review endpoint (promote has
// its own dedicated endpoint above). Returns the updated candidate so the
// caller can patch it into the inbox list in place, without a reload.
export async function reviewStopCandidate(candidateId: string, patch: ReviewStopCandidatePatch): Promise<AdminStopCandidate> {
  return postJSON<AdminStopCandidate>(`/api/admin/stop-candidates/${candidateId}/review`, {
    state: patch.state,
    merged_into: patch.mergedInto,
    expected_revision: patch.expectedRevision,
  })
}

// Memento editor (ADMIN-01.4 / ADMIN-01.5).

export async function getMemento(id: string): Promise<AdminMementoDetail> {
  return getJSON<AdminMementoDetail>(`/api/admin/mementos/${id}`)
}

// Full-payload upsert: the server never returns the updated record (just
// {"status":"ok"}), so a caller must re-fetch via getMemento to pick up the
// new revision before its next save — ADMIN-01.5's concurrency guard depends
// on that being fresh.
export async function upsertMemento(payload: UpsertMementoRequest): Promise<{ status: string }> {
  return postJSON<{ status: string }>("/api/admin/mementos", payload)
}

// Projects a proposed point onto the journey's composed route (GPS track +
// authored transit legs). Used by the editor's per-point "snap" helper,
// since non-draft saves must pass the kind's anchor geometry validation.
export async function snapToRoute(journeyId: string, point: [number, number]): Promise<SnapPointResult> {
  return postJSON<SnapPointResult>(`/api/admin/journeys/${journeyId}/snap`, { point })
}

export async function upsertPhoto(payload: UpsertPhotoRequest): Promise<{ status: string }> {
  return postJSON<{ status: string }>("/api/admin/photos", payload)
}

export async function listMementoPhotos(mementoId: string): Promise<AdminMementoPhoto[]> {
  return getJSON<AdminMementoPhoto[]>(`/api/admin/mementos/${mementoId}/photos`)
}
