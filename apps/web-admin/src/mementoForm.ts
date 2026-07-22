// Pure form-mapping/validation helpers for the memento editor (ADMIN-01.4 /
// ADMIN-01.5). Kept dependency-free from Svelte/DOM so they're testable under
// bun:test without a browser, the same split api.ts already uses for its
// fetch boundary vs. the view components.
//
// Server field-name/shape notes this module has to respect (see
// server/api/server.go + core/domain/{entity,template,validate}.go):
//   - GET /api/admin/mementos/{id} returns `geom` in orb's raw array form
//     (Point -> [lng, lat], LineString -> [[lng, lat], ...], or null) — NOT
//     GeoJSON. POST /api/admin/mementos expects the opposite: a GeoJSON-ish
//     {type, coordinates} object. geomToLatLngInputs / geomRequestFromPoints
//     bridge that asymmetry.
//   - kind_data `money` fields are {amount, currency}; `station`/`venue`
//     fields are {name, coords: [lng, lat]} (see checkType/hasCoords in
//     core/domain/validate.go). buildKindData/parseKindData mirror that.
//   - Validation issues come back as domain.Issue{Field, Code} — capitalized,
//     no json tags, same as AdminTemplateField in api.ts.

import type { AdminIssue, AdminTemplateField, MementoGeom, UpsertMementoGeom, UpsertMementoRequest, UpsertPhotoRequest } from "./api"

// --- Date/time -------------------------------------------------------------

// occurred_at is an RFC3339 instant server-side. The editor uses a plain
// <input type="datetime-local"> (no seconds/offset), so we treat its value as
// UTC — the simplest thing that works without pulling in a timezone library;
// occurred_tz (an IANA name) carries the display zone separately.
export function toRFC3339(localValue: string): string {
  if (!localValue) return ""
  if (localValue.endsWith("Z") || /[+-]\d{2}:\d{2}$/.test(localValue)) return localValue
  return localValue.length === 16 ? `${localValue}:00Z` : `${localValue}Z`
}

export function fromRFC3339(value: string | undefined): string {
  if (!value) return ""
  return value.slice(0, 16)
}

// --- Price (the memento's own price_amount/price_currency, not kind_data) --

export interface PriceFormFields {
  amount: string
  currency: string
}

// price_amount is an int64 minor-units field server-side (see
// docs/research/data-model.md); a non-integer value there fails the whole
// request's JSON decode rather than surfacing a friendly per-field issue, so
// truncate client-side instead of passing a fractional number through.
export function parsePriceAmount(amount: string): number | undefined {
  const trimmed = amount.trim()
  if (!trimmed) return undefined
  const parsed = Number(trimmed)
  return Number.isFinite(parsed) ? Math.trunc(parsed) : undefined
}

export function buildPriceFields(fields: PriceFormFields): { price_amount?: number; price_currency?: string } {
  const amount = parsePriceAmount(fields.amount)
  const currency = fields.currency.trim().toUpperCase()
  if (amount === undefined && !currency) return {}
  return { price_amount: amount, price_currency: currency || undefined }
}

export function priceFormFieldsFromMemento(amount: number | undefined, currency: string | undefined): PriceFormFields {
  return { amount: amount !== undefined ? String(amount) : "", currency: currency ?? "" }
}

// --- Location (common "geom" field, driven by the kind's anchor) ----------

export interface LatLngInput {
  lat: string
  lng: string
}

export function parseLatLng(input: LatLngInput): [number, number] | null {
  const lat = Number(input.lat)
  const lng = Number(input.lng)
  if (!Number.isFinite(lat) || !Number.isFinite(lng)) return null
  if (lat < -90 || lat > 90 || lng < -180 || lng > 180) return null
  return [lng, lat]
}

// A point anchor needs one coordinate pair, an edge anchor needs two (the
// journey's display route composes from→to). Anything else defaults to a
// single point so an unrecognized/missing template doesn't crash the form.
export function pointCountForAnchor(anchor: string): number {
  return anchor === "edge" ? 2 : 1
}

export function emptyLatLngInputs(anchor: string): LatLngInput[] {
  return Array.from({ length: pointCountForAnchor(anchor) }, () => ({ lat: "", lng: "" }))
}

// Maps a GET response's raw orb-form geom into the editor's per-point inputs.
export function geomToLatLngInputs(geom: MementoGeom | undefined, anchor: string): LatLngInput[] {
  if (!geom || geom.length === 0) return emptyLatLngInputs(anchor)
  const points: [number, number][] = typeof geom[0] === "number" ? [geom as [number, number]] : (geom as [number, number][])
  const inputs = points.map(([lng, lat]) => ({ lat: String(lat), lng: String(lng) }))
  // Pad out to the anchor's expected point count so the form always has a
  // stable number of rows to render/edit.
  while (inputs.length < pointCountForAnchor(anchor)) inputs.push({ lat: "", lng: "" })
  return inputs
}

// Builds the POST /api/admin/mementos geom payload from the form's inputs.
// Returns null when there aren't enough valid points yet — the caller omits
// `geom` (or sends null) and lets the server's geometry validation surface an
// inline "location is required" issue rather than guessing.
export function geomRequestFromPoints(anchor: string, points: LatLngInput[]): UpsertMementoGeom | null {
  const parsed = points.map(parseLatLng).filter((p): p is [number, number] => p !== null)
  if (anchor === "edge") {
    if (parsed.length < 2) return null
    return { type: "LineString", coordinates: parsed }
  }
  if (parsed.length === 0) return null
  return { type: "Point", coordinates: parsed[0] }
}

// --- kind_data (transit/goods hardcoded forms, driven by the registry) -----

export type KindFieldValue = string | { amount: string; currency: string } | { name: string; lat: string; lng: string } | undefined

export type KindFormState = Record<string, KindFieldValue>

export function emptyKindFormState(fields: AdminTemplateField[]): KindFormState {
  const state: KindFormState = {}
  for (const field of fields) {
    state[field.Name] = field.Type === "money" ? { amount: "", currency: "" } : field.Type === "station" || field.Type === "venue" ? { name: "", lat: "", lng: "" } : ""
  }
  return state
}

// Maps a fetched memento's kind_data blob into the form's per-field state,
// following each field's declared type from the registry template.
export function parseKindData(fields: AdminTemplateField[], data: Record<string, unknown> | null | undefined): KindFormState {
  const state = emptyKindFormState(fields)
  if (!data) return state
  for (const field of fields) {
    const value = data[field.Name]
    if (value === undefined || value === null) continue
    if (field.Type === "money") {
      const money = value as { amount?: number; currency?: string }
      state[field.Name] = { amount: money.amount !== undefined ? String(money.amount) : "", currency: money.currency ?? "" }
    } else if (field.Type === "station" || field.Type === "venue") {
      const place = value as { name?: string; coords?: [number, number] }
      state[field.Name] = {
        name: place.name ?? "",
        lng: place.coords?.[0] !== undefined ? String(place.coords[0]) : "",
        lat: place.coords?.[1] !== undefined ? String(place.coords[1]) : "",
      }
    } else {
      state[field.Name] = String(value)
    }
  }
  return state
}

// Builds the kind_data object to submit from the form's state, following the
// registry's field list so the closed-field-set contract (core/domain
// validateTemplate) is respected — only known fields are ever emitted, and a
// field is only included once it has some content (an all-blank optional
// field stays entirely absent, not an empty-string/zero placeholder).
export function buildKindData(fields: AdminTemplateField[], state: KindFormState): Record<string, unknown> {
  const data: Record<string, unknown> = {}
  for (const field of fields) {
    const raw = state[field.Name]
    if (raw === undefined) continue
    if (field.Type === "money") {
      const money = raw as { amount: string; currency: string }
      const amount = money.amount?.trim() ?? ""
      const currency = money.currency?.trim() ?? ""
      if (!amount && !currency) continue
      const parsedAmount = Number(amount)
      data[field.Name] = { amount: amount ? parsedAmount : null, currency: currency.toUpperCase() || null }
    } else if (field.Type === "station" || field.Type === "venue") {
      const place = raw as { name: string; lat: string; lng: string }
      const name = place.name?.trim() ?? ""
      const lat = Number(place.lat)
      const lng = Number(place.lng)
      const hasCoords = Number.isFinite(lat) && Number.isFinite(lng)
      if (!name && !hasCoords) continue
      data[field.Name] = { name, coords: hasCoords ? [lng, lat] : null }
    } else {
      const text = (raw as string)?.trim() ?? ""
      if (!text) continue
      data[field.Name] = text
    }
  }
  return data
}

// --- Assembling the full upsert payload ------------------------------------

export interface CommonFormFields {
  title: string
  place: string
  occurredAtLocal: string // <input type="datetime-local"> value, or ""
  occurredTz: string
  essay: string
  vendor: string
  price: PriceFormFields
}

// The subset of a fetched memento that the editor carries through unchanged
// on every save (identity, ordering, provenance) rather than re-deriving.
export interface MementoIdentity {
  id: string
  journey_id: string
  kind: string
  seq: number
  source_ref?: string
  authored_fields: string[]
  orphaned_at?: string
}

// Composes the full POST /api/admin/mementos body from the editor's form
// state. Pure and independent of how kindData was produced, so it works for
// both the hardcoded transit/goods forms and the read-only-JSON kinds (which
// pass their unmodified kind_data straight through).
export function buildUpsertPayload(params: {
  identity: MementoIdentity
  common: CommonFormFields
  anchor: string
  points: LatLngInput[]
  kindData: Record<string, unknown>
  state: string
  expectedRevision: number
}): UpsertMementoRequest {
  const price = buildPriceFields(params.common.price)
  return {
    id: params.identity.id,
    journey_id: params.identity.journey_id,
    kind: params.identity.kind,
    seq: params.identity.seq,
    occurred_at: params.common.occurredAtLocal ? toRFC3339(params.common.occurredAtLocal) : undefined,
    occurred_tz: params.common.occurredTz.trim() || undefined,
    geom: geomRequestFromPoints(params.anchor, params.points),
    title: params.common.title,
    place: params.common.place,
    vendor: params.common.vendor.trim() || undefined,
    essay: params.common.essay.trim() || undefined,
    price_amount: price.price_amount,
    price_currency: price.price_currency,
    kind_data: params.kindData,
    source_ref: params.identity.source_ref,
    authored_fields: params.identity.authored_fields,
    orphaned_at: params.identity.orphaned_at,
    state: params.state,
    expected_revision: params.expectedRevision,
  }
}

// --- Server validation issues -----------------------------------------------

const issueMessages: Record<string, string> = {
  required_missing: "This field is required.",
  unknown_field: "This field isn't part of the kind template.",
  type_mismatch: "This value doesn't match the expected type.",
  anchor_mismatch: "The location doesn't match this kind's shape (a single point vs. a from/to route).",
  bad_currency: "Currency must be a 3-letter code, e.g. JPY.",
  invalid_state: "That state transition isn't valid.",
  invalid_timezone: "Enter a recognized IANA timezone, e.g. Asia/Tokyo.",
  invalid_geometry: "A location is required before saving.",
  invalid_coordinate: "Coordinates must be valid longitude/latitude values.",
  invalid_transition: "That state change isn't allowed — publish and unpublish move one step at a time.",
  delete_requires_unpublish: "Unpublish this memento before deleting it.",
}

export function issueMessage(issue: AdminIssue): string {
  return issueMessages[issue.Code] ?? issue.Code
}

// Groups issues by field name so the editor can render them next to the
// field they apply to. Template-level issues (an anchor mismatch has no
// single field) carry Field: "" server-side and land in the "__form__"
// bucket for a form-level banner instead.
export const FORM_LEVEL_ISSUE_KEY = "__form__"

export function groupIssuesByField(issues: AdminIssue[]): Record<string, string[]> {
  const grouped: Record<string, string[]> = {}
  for (const issue of issues) {
    const key = issue.Field || FORM_LEVEL_ISSUE_KEY
    grouped[key] = [...(grouped[key] ?? []), issueMessage(issue)]
  }
  return grouped
}

// --- Lifecycle (draft -> authored -> published) -----------------------------

// The server itself doesn't enforce a strict sequence beyond validating the
// target state's completeness (ValidateForState), but the editor only ever
// exposes the single "next" step so an author can't skip straight past
// review states by accident.
export function nextLifecycleState(state: string): string | null {
  switch (state) {
    case "draft":
      return "authored"
    case "authored":
      return "published"
    default:
      return null
  }
}

export function lifecycleActionLabel(state: string): string | null {
  switch (state) {
    case "draft":
      return "Mark authored"
    case "authored":
      return "Publish"
    default:
      return null
  }
}

// The one backward step the editor exposes (ADMIN-02 M1 02.1a): a published
// memento can be withdrawn to authored. The server's upsert already accepts
// any target state (it only validates the target's completeness), so this is
// purely a GUI-side affordance — no new endpoint, no new save path. Nothing
// else steps backward (draft has nothing earlier to return to, and archived
// isn't part of this lifecycle's forward/backward pair).
export function previousLifecycleState(state: string): string | null {
  switch (state) {
    case "published":
      return "authored"
    default:
      return null
  }
}

export function unpublishActionLabel(state: string): string | null {
  switch (state) {
    case "published":
      return "Unpublish"
    default:
      return null
  }
}

// --- Photos (metadata-only caption/seq editing) -----------------------------

export interface PhotoFormFields {
  objectKey: string
  contentHash: string
  caption: string
  seq: string
  takenAt: string
  sourceRef: string
}

export function parseSeq(value: string, fallback = 0): number {
  const parsed = Number.parseInt(value, 10)
  return Number.isInteger(parsed) ? parsed : fallback
}

export function photoFormFieldsFromRequest(fields: Partial<PhotoFormFields>): PhotoFormFields {
  return {
    objectKey: fields.objectKey ?? "",
    contentHash: fields.contentHash ?? "",
    caption: fields.caption ?? "",
    seq: fields.seq ?? "0",
    takenAt: fields.takenAt ?? "",
    sourceRef: fields.sourceRef ?? "",
  }
}

// Builds the POST /api/admin/photos payload for one photo row. object_key and
// content_hash are carried through as entered/loaded rather than derived —
// this editor is metadata-only and never uploads bytes, so those identify an
// object that already exists in the media store.
export function buildPhotoPayload(id: string, mementoId: string, fields: PhotoFormFields): UpsertPhotoRequest {
  return {
    id,
    memento_id: mementoId,
    object_key: fields.objectKey.trim(),
    content_hash: fields.contentHash.trim(),
    caption: fields.caption.trim() || undefined,
    seq: parseSeq(fields.seq),
    taken_at: fields.takenAt ? toRFC3339(fields.takenAt) : undefined,
    source_ref: fields.sourceRef.trim() || undefined,
  }
}
