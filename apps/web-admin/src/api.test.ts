import { afterEach, beforeAll, describe, expect, test } from "bun:test"
import {
  ApiError,
  browseDirectories,
  compileSite,
  countMementosByState,
  countPendingStopCandidates,
  deleteMemento,
  getBuildStatus,
  getJourney,
  getJourneyBuildStatus,
  getMemento,
  getSiteInfo,
  getTemplates,
  isConflict,
  listJourneys,
  listMementoPhotos,
  listMementos,
  listStopCandidates,
  loadJourneySummaries,
  photoTray,
  planIntake,
  promoteStopCandidate,
  reviewStopCandidate,
  routePointCount,
  snapToRoute,
  sortMementosBySeq,
  syncRoute,
  syncVisits,
  updateSiteOutDir,
  upsertMemento,
  upsertPhoto,
  type AdminJourney,
  type AdminMemento,
  type AdminMementoDetail,
  type AdminStopCandidate,
  type UpsertMementoRequest,
} from "./api"

// bun test runs outside Vite, so import.meta.env isn't populated the way it
// is in the browser/dev-server build. Seed it once, the same way
// web-public's source.test.ts does for its fetch boundary tests.
const importMeta = import.meta as unknown as { env: { VITE_API_BASE?: string } }
beforeAll(() => {
  if (!importMeta.env) importMeta.env = {}
  importMeta.env.VITE_API_BASE = "http://localhost:8080"
})

const originalFetch = globalThis.fetch
afterEach(() => {
  globalThis.fetch = originalFetch
})

function mockFetchOnce(status: number, body: unknown) {
  globalThis.fetch = (() => Promise.resolve(Response.json(body, { status }))) as unknown as typeof fetch
}

function mockFetchByUrl(handlers: Record<string, { status?: number; body: unknown }>) {
  globalThis.fetch = ((url: string | URL) => {
    const key = Object.keys(handlers).find((pattern) => url.toString().includes(pattern))
    if (!key) return Promise.resolve(new Response("Not Found", { status: 404 }))
    const { status = 200, body } = handlers[key]
    return Promise.resolve(Response.json(body, { status }))
  }) as unknown as typeof fetch
}

const journey = (overrides: Partial<AdminJourney> = {}): AdminJourney => ({
  id: "journey-1",
  journal_id: "journal-1",
  slug: "japan-spring-2026",
  title: "Japan Spring 2026",
  place: "Tokyo & Kyoto",
  date_start: "2026-03-20",
  date_end: "2026-04-05",
  authored_fields: [],
  ...overrides,
})

const memento = (overrides: Partial<AdminMemento> = {}): AdminMemento => ({
  id: "memento-1",
  journey_id: "journey-1",
  kind: "receipt",
  seq: 1,
  title: "Coffee receipt",
  place: "Kyoto",
  state: "draft",
  ...overrides,
})

const mementoDetail = (overrides: Partial<AdminMementoDetail> = {}): AdminMementoDetail => ({
  id: "memento-1",
  journey_id: "journey-1",
  kind: "goods",
  seq: 1,
  occurred_at: "2026-03-21T09:00:00Z",
  occurred_tz: "Asia/Tokyo",
  geom: [139.7, 35.6],
  title: "Tenugui",
  place: "Kyoto",
  kind_data: { name: "Tenugui" },
  authored_fields: [],
  state: "draft",
  revision: 2,
  created_at: "2026-03-20T00:00:00Z",
  updated_at: "2026-03-20T00:00:00Z",
  ...overrides,
})

const upsertRequest = (overrides: Partial<UpsertMementoRequest> = {}): UpsertMementoRequest => ({
  id: "memento-1",
  journey_id: "journey-1",
  kind: "goods",
  seq: 1,
  title: "Tenugui",
  place: "Kyoto",
  kind_data: { name: "Tenugui" },
  state: "draft",
  expected_revision: 2,
  ...overrides,
})

const candidate = (overrides: Partial<AdminStopCandidate> = {}): AdminStopCandidate => ({
  id: "candidate-1",
  journey_id: "journey-1",
  label: "Fushimi Inari",
  state: "proposed",
  confidence: 0.87,
  arrive: "2026-03-21T09:00:00Z",
  depart: "2026-03-21T10:30:00Z",
  revision: 0,
  ...overrides,
})

describe("listJourneys / getJourney / listMementos / listStopCandidates", () => {
  test("listJourneys maps a successful response", async () => {
    mockFetchOnce(200, [journey()])
    const result = await listJourneys()
    expect(result).toHaveLength(1)
    expect(result[0].slug).toBe("japan-spring-2026")
  })

  test("getJourney throws the server's error message on a non-ok response", async () => {
    mockFetchOnce(404, { error: "journey not found" })
    await expect(getJourney("missing")).rejects.toThrow("journey not found")
  })

  test("getJourney falls back to a generic message when the body isn't JSON", async () => {
    globalThis.fetch = (() => Promise.resolve(new Response("boom", { status: 500 }))) as unknown as typeof fetch
    await expect(getJourney("j1")).rejects.toThrow("Felicia API returned 500")
  })

  test("listMementos maps a successful response", async () => {
    mockFetchOnce(200, [memento(), memento({ id: "memento-2", seq: 2 })])
    const result = await listMementos("journey-1")
    expect(result).toHaveLength(2)
  })

  test("listStopCandidates surfaces a 503 as an error", async () => {
    mockFetchOnce(503, { error: "intake candidate storage is unavailable" })
    await expect(listStopCandidates("journey-1")).rejects.toThrow("intake candidate storage is unavailable")
  })
})

describe("countMementosByState / countPendingStopCandidates / sortMementosBySeq", () => {
  test("tallies mementos per state", () => {
    const counts = countMementosByState([memento({ state: "draft" }), memento({ id: "m2", state: "draft" }), memento({ id: "m3", state: "published" })])
    expect(counts).toEqual({ draft: 2, published: 1 })
  })

  test("counts only proposed stop candidates", () => {
    const candidates: AdminStopCandidate[] = [
      { id: "c1", journey_id: "journey-1", label: "Cafe", state: "proposed", confidence: 0.8, arrive: "", depart: "", revision: 0 },
      { id: "c2", journey_id: "journey-1", label: "Station", state: "kept", confidence: 0.9, arrive: "", depart: "", revision: 1 },
      { id: "c3", journey_id: "journey-1", label: "Alley", state: "proposed", confidence: 0.5, arrive: "", depart: "", revision: 0 },
    ]
    expect(countPendingStopCandidates(candidates)).toBe(2)
  })

  test("sorts mementos by seq without mutating the input", () => {
    const input = [memento({ id: "m3", seq: 3 }), memento({ id: "m1", seq: 1 }), memento({ id: "m2", seq: 2 })]
    const sorted = sortMementosBySeq(input)
    expect(sorted.map((m) => m.id)).toEqual(["m1", "m2", "m3"])
    expect(input.map((m) => m.id)).toEqual(["m3", "m1", "m2"])
  })
})

describe("loadJourneySummaries", () => {
  test("aggregates memento counts, state counts, and pending stop candidates per journey", async () => {
    mockFetchByUrl({
      "/api/admin/journeys/journey-1/mementos": { body: [memento({ state: "draft" }), memento({ id: "m2", state: "published" })] },
      "/api/admin/journeys/journey-1/stop-candidates": {
        body: [{ id: "c1", journey_id: "journey-1", label: "Cafe", state: "proposed", confidence: 0.8, arrive: "", depart: "" }],
      },
      "/api/admin/journeys": { body: [journey()] },
    })

    const summaries = await loadJourneySummaries()
    expect(summaries).toHaveLength(1)
    expect(summaries[0].mementoCount).toBe(2)
    expect(summaries[0].stateCounts).toEqual({ draft: 1, published: 1 })
    expect(summaries[0].stopCandidateCount).toBe(1)
  })

  test("degrades to a null stop-candidate count instead of failing the whole list", async () => {
    globalThis.fetch = ((url: string | URL) => {
      const key = url.toString()
      if (key.includes("/stop-candidates")) return Promise.resolve(Response.json({ error: "intake candidate storage is unavailable" }, { status: 503 }))
      if (key.includes("/mementos")) return Promise.resolve(Response.json([memento()]))
      return Promise.resolve(Response.json([journey()]))
    }) as unknown as typeof fetch

    const summaries = await loadJourneySummaries()
    expect(summaries[0].stopCandidateCount).toBeNull()
    expect(summaries[0].mementoCount).toBe(1)
  })
})

describe("import/preview triggers", () => {
  test("syncRoute POSTs and the result maps to a route point count", async () => {
    mockFetchOnce(200, {
      status: "ok",
      gps_route: {
        type: "LineString",
        coordinates: [
          [139.7, 35.6],
          [139.8, 35.7],
        ],
      },
    })
    const result = await syncRoute("journey-1")
    expect(result.status).toBe("ok")
    expect(routePointCount(result)).toBe(2)
  })

  test("routePointCount flattens a MultiLineString", () => {
    expect(
      routePointCount({
        status: "ok",
        gps_route: {
          type: "MultiLineString",
          coordinates: [
            [
              [139.7, 35.6],
              [139.8, 35.7],
            ],
            [[135.7, 35.0]],
          ],
        },
      }),
    ).toBe(3)
  })

  test("routePointCount treats a null route as zero points", () => {
    expect(routePointCount({ status: "ok", gps_route: null })).toBe(0)
  })

  test("syncRoute surfaces a service-unavailable error when ingest isn't configured", async () => {
    mockFetchOnce(503, { error: "ingest sources not configured" })
    await expect(syncRoute("journey-1")).rejects.toThrow("ingest sources not configured")
  })

  test("syncVisits maps the visit preview list", async () => {
    mockFetchOnce(200, [{ coord: [139.7, 35.6], label: "Station", arrive: "2026-03-20T09:00:00Z", depart: "2026-03-20T09:30:00Z", confidence: 0.92, source_ref: "dawarich:1" }])
    const visits = await syncVisits("journey-1")
    expect(visits).toHaveLength(1)
    expect(visits[0].label).toBe("Station")
  })

  test("photoTray maps the tray item list, including items without GPS", async () => {
    mockFetchOnce(200, [
      { id: "asset-1", at: "2026-03-20T09:00:00Z", coord: [139.7, 35.6], checksum: "abc123", source_ref: "immich:1" },
      { id: "asset-2", at: "2026-03-20T10:00:00Z", checksum: "def456", source_ref: "immich:2" },
    ])
    const assets = await photoTray("journey-1")
    expect(assets).toHaveLength(2)
    expect(assets[1].coord).toBeUndefined()
  })
})

describe("intake inbox (ADMIN-01.3b)", () => {
  test("planIntake maps the plan response, including issues", async () => {
    mockFetchOnce(200, {
      journey_id: "journey-1",
      stops: [candidate()],
      issues: [{ severity: "warning", code: "stop_label_missing", message: "stop x has no source label" }],
    })
    const result = await planIntake("journey-1")
    expect(result.stops).toHaveLength(1)
    expect(result.issues).toEqual([{ severity: "warning", code: "stop_label_missing", message: "stop x has no source label" }])
  })

  test("planIntake surfaces a service-unavailable error when the track source isn't configured", async () => {
    mockFetchOnce(503, { error: "no track source configured" })
    await expect(planIntake("journey-1")).rejects.toThrow("no track source configured")
  })

  test("getTemplates maps the kind registry, keyed by kind", async () => {
    mockFetchOnce(200, {
      transit: { Kind: "transit", Anchor: "edge", Stub: "transit-stub", Fields: [{ Name: "operator", Type: "text", Required: true, Values: null }] },
      goods: { Kind: "goods", Anchor: "point", Stub: "goods-stub", Fields: [] },
    })
    const templates = await getTemplates()
    expect(Object.keys(templates)).toEqual(["transit", "goods"])
    expect(templates.transit.Fields[0].Name).toBe("operator")
  })

  test("promoteStopCandidate POSTs the kind and expected revision, returning the draft memento", async () => {
    let capturedBody: unknown
    globalThis.fetch = ((_url: string | URL, init?: RequestInit) => {
      capturedBody = init?.body ? JSON.parse(init.body as string) : undefined
      return Promise.resolve(Response.json(memento({ id: "memento-9", kind: "goods", state: "draft" })))
    }) as unknown as typeof fetch

    const result = await promoteStopCandidate("candidate-1", "goods", 0)
    expect(capturedBody).toEqual({ kind: "goods", expected_revision: 0 })
    expect(result.id).toBe("memento-9")
    expect(result.state).toBe("draft")
  })

  test("promoteStopCandidate omits expected_revision when not given", async () => {
    let capturedBody: unknown
    globalThis.fetch = ((_url: string | URL, init?: RequestInit) => {
      capturedBody = init?.body ? JSON.parse(init.body as string) : undefined
      return Promise.resolve(Response.json(memento()))
    }) as unknown as typeof fetch

    await promoteStopCandidate("candidate-1", "goods")
    expect(capturedBody).toEqual({ kind: "goods" })
  })

  test("promoteStopCandidate surfaces a 409 as an ApiError conflict", async () => {
    mockFetchOnce(409, { error: "stop candidate was already reviewed" })
    let caught: unknown
    try {
      await promoteStopCandidate("candidate-1", "goods", 0)
    } catch (cause) {
      caught = cause
    }
    expect(caught).toBeInstanceOf(ApiError)
    expect(isConflict(caught)).toBe(true)
    expect((caught as ApiError).message).toBe("stop candidate was already reviewed")
  })

  test("reviewStopCandidate POSTs the ignore patch and returns the updated candidate", async () => {
    let capturedBody: unknown
    globalThis.fetch = ((_url: string | URL, init?: RequestInit) => {
      capturedBody = init?.body ? JSON.parse(init.body as string) : undefined
      return Promise.resolve(Response.json(candidate({ state: "ignored", revision: 1 })))
    }) as unknown as typeof fetch

    const result = await reviewStopCandidate("candidate-1", { state: "ignored", expectedRevision: 0 })
    expect(capturedBody).toEqual({ state: "ignored", expected_revision: 0 })
    expect(result.state).toBe("ignored")
    expect(result.revision).toBe(1)
  })

  test("reviewStopCandidate POSTs the merge patch with merged_into", async () => {
    let capturedBody: unknown
    globalThis.fetch = ((_url: string | URL, init?: RequestInit) => {
      capturedBody = init?.body ? JSON.parse(init.body as string) : undefined
      return Promise.resolve(Response.json(candidate({ id: "candidate-1", state: "merged", revision: 1 })))
    }) as unknown as typeof fetch

    const result = await reviewStopCandidate("candidate-1", { state: "merged", mergedInto: "candidate-2", expectedRevision: 0 })
    expect(capturedBody).toEqual({ state: "merged", merged_into: "candidate-2", expected_revision: 0 })
    expect(result.state).toBe("merged")
  })

  test("reviewStopCandidate surfaces a 409 revision conflict", async () => {
    mockFetchOnce(409, { error: "candidate revision conflict" })
    const failure = reviewStopCandidate("candidate-1", { state: "ignored", expectedRevision: 0 })
    await expect(failure).rejects.toThrow("candidate revision conflict")
  })
})

describe("memento editor (ADMIN-01.4 / ADMIN-01.5)", () => {
  test("getMemento maps the full record, including revision and raw-array geom", async () => {
    mockFetchOnce(200, mementoDetail())
    const result = await getMemento("memento-1")
    expect(result.revision).toBe(2)
    expect(result.geom).toEqual([139.7, 35.6])
    expect(result.kind_data).toEqual({ name: "Tenugui" })
  })

  test("upsertMemento POSTs the full payload including expected_revision", async () => {
    let capturedUrl: string | undefined
    let capturedBody: unknown
    globalThis.fetch = ((url: string | URL, init?: RequestInit) => {
      capturedUrl = url.toString()
      capturedBody = init?.body ? JSON.parse(init.body as string) : undefined
      return Promise.resolve(Response.json({ status: "ok" }))
    }) as unknown as typeof fetch

    const payload = upsertRequest({ geom: { type: "Point", coordinates: [135.7, 35.0] } })
    const result = await upsertMemento(payload)

    expect(capturedUrl).toContain("/api/admin/mementos")
    expect(capturedBody).toEqual(payload)
    expect(result).toEqual({ status: "ok" })
  })

  test("upsertMemento surfaces a 409 conflict with no memento payload (the caller must re-fetch)", async () => {
    mockFetchOnce(409, { error: "memento was modified; reload before saving" })
    let caught: unknown
    try {
      await upsertMemento(upsertRequest())
    } catch (cause) {
      caught = cause
    }
    expect(isConflict(caught)).toBe(true)
    expect((caught as ApiError).message).toBe("memento was modified; reload before saving")
  })

  test("upsertMemento surfaces validation issues on the thrown ApiError for inline rendering", async () => {
    mockFetchOnce(400, {
      error: "validation failed",
      issues: [
        { Field: "operator", Code: "required_missing" },
        { Field: "", Code: "anchor_mismatch" },
      ],
    })
    let caught: unknown
    try {
      await upsertMemento(upsertRequest())
    } catch (cause) {
      caught = cause
    }
    expect(caught).toBeInstanceOf(ApiError)
    expect((caught as ApiError).issues).toEqual([
      { Field: "operator", Code: "required_missing" },
      { Field: "", Code: "anchor_mismatch" },
    ])
  })

  test("a plain error response has no issues on the ApiError", async () => {
    mockFetchOnce(500, { error: "boom" })
    let caught: unknown
    try {
      await upsertMemento(upsertRequest())
    } catch (cause) {
      caught = cause
    }
    expect((caught as ApiError).issues).toBeUndefined()
  })

  test("snapToRoute POSTs the point and maps the snapped GeoJSON point", async () => {
    let capturedBody: unknown
    globalThis.fetch = ((_url: string | URL, init?: RequestInit) => {
      capturedBody = init?.body ? JSON.parse(init.body as string) : undefined
      return Promise.resolve(Response.json({ point: { type: "Point", coordinates: [139.7, 35.6] } }))
    }) as unknown as typeof fetch

    const result = await snapToRoute("journey-1", [139.65, 35.55])
    expect(capturedBody).toEqual({ point: [139.65, 35.55] })
    expect(result.point.coordinates).toEqual([139.7, 35.6])
  })

  test("snapToRoute surfaces a 422 when the journey has no route to snap to", async () => {
    mockFetchOnce(422, { error: "journey has no route to snap to" })
    await expect(snapToRoute("journey-1", [139.7, 35.6])).rejects.toThrow("journey has no route to snap to")
  })

  test("upsertPhoto POSTs the photo payload", async () => {
    let capturedBody: unknown
    globalThis.fetch = ((_url: string | URL, init?: RequestInit) => {
      capturedBody = init?.body ? JSON.parse(init.body as string) : undefined
      return Promise.resolve(Response.json({ status: "ok" }))
    }) as unknown as typeof fetch

    const payload = { id: "photo-1", memento_id: "memento-1", object_key: "media/abc.jpg", content_hash: "sha256:abc", caption: "At the platform", seq: 0 }
    const result = await upsertPhoto(payload)
    expect(capturedBody).toEqual(payload)
    expect(result).toEqual({ status: "ok" })
  })

  test("listMementoPhotos GETs the memento's photo list", async () => {
    let capturedUrl = ""
    globalThis.fetch = ((url: string | URL) => {
      capturedUrl = String(url)
      return Promise.resolve(
        Response.json([
          {
            id: "photo-1",
            memento_id: "memento-1",
            object_key: "media/abc.jpg",
            content_hash: "sha256:abc",
            caption: "At the platform",
            seq: 0,
            taken_at: "2026-05-01T09:30:00Z",
            created_at: "2026-05-02T00:00:00Z",
          },
        ]),
      )
    }) as unknown as typeof fetch

    const photos = await listMementoPhotos("memento-1")
    expect(capturedUrl).toBe("http://localhost:8080/api/admin/mementos/memento-1/photos")
    expect(photos).toHaveLength(1)
    expect(photos[0].object_key).toBe("media/abc.jpg")
    expect(photos[0].seq).toBe(0)
  })

  test("deleteMemento DELETEs and returns the deleted status", async () => {
    let capturedUrl: string | undefined
    let capturedMethod: string | undefined
    globalThis.fetch = ((url: string | URL, init?: RequestInit) => {
      capturedUrl = url.toString()
      capturedMethod = init?.method
      return Promise.resolve(Response.json({ status: "deleted" }))
    }) as unknown as typeof fetch

    const result = await deleteMemento("memento-1")
    expect(capturedUrl).toBe("http://localhost:8080/api/admin/mementos/memento-1")
    expect(capturedMethod).toBe("DELETE")
    expect(result).toEqual({ status: "deleted" })
  })

  test("deleteMemento surfaces a 404 as an ApiError", async () => {
    mockFetchOnce(404, { error: "memento not found" })
    let caught: unknown
    try {
      await deleteMemento("missing")
    } catch (cause) {
      caught = cause
    }
    expect(caught).toBeInstanceOf(ApiError)
    expect((caught as ApiError).status).toBe(404)
    expect((caught as ApiError).message).toBe("memento not found")
  })

  test("list endpoints coerce a null JSON body (Go nil slice) to an empty array", async () => {
    globalThis.fetch = (() => Promise.resolve(Response.json(null))) as unknown as typeof fetch
    expect(await listMementos("journey-1")).toEqual([])
    globalThis.fetch = (() => Promise.resolve(Response.json(null))) as unknown as typeof fetch
    expect(await listStopCandidates("journey-1")).toEqual([])
    globalThis.fetch = (() => Promise.resolve(Response.json(null))) as unknown as typeof fetch
    expect(await listMementoPhotos("memento-1")).toEqual([])
  })
})

describe("site build & preview (ADMIN-02 M0)", () => {
  test("getSiteInfo maps the site info payload", async () => {
    mockFetchOnce(200, {
      out_dir: ".felicia/site",
      preview_port: "8081",
      spa_ready: false,
      artifact_ready: true,
    })
    const info = await getSiteInfo()
    expect(info).toEqual({
      out_dir: ".felicia/site",
      preview_port: "8081",
      spa_ready: false,
      artifact_ready: true,
    })
  })

  test("compileSite POSTs an empty body to /api/admin/compile and returns the capitalized report", async () => {
    let capturedUrl: string | undefined
    let capturedBody: unknown
    globalThis.fetch = ((url: string | URL, init?: RequestInit) => {
      capturedUrl = url.toString()
      capturedBody = init?.body ? JSON.parse(init.body as string) : undefined
      return Promise.resolve(Response.json({ Journeys: 1, Mementos: 3, Media: 2, Removed: 0 }))
    }) as unknown as typeof fetch

    const report = await compileSite()

    expect(capturedUrl).toContain("/api/admin/compile")
    expect(capturedBody).toEqual({})
    expect(report).toEqual({ Journeys: 1, Mementos: 3, Media: 2, Removed: 0 })
  })

  test("compileSite surfaces a non-ok response as an ApiError", async () => {
    mockFetchOnce(500, { error: "compile failed" })
    await expect(compileSite()).rejects.toThrow("compile failed")
  })

  test("updateSiteOutDir PUTs the new out_dir and returns it", async () => {
    let capturedUrl: string | undefined
    let capturedMethod: string | undefined
    let capturedBody: unknown
    globalThis.fetch = ((url: string | URL, init?: RequestInit) => {
      capturedUrl = url.toString()
      capturedMethod = init?.method
      capturedBody = init?.body ? JSON.parse(init.body as string) : undefined
      return Promise.resolve(Response.json({ out_dir: "/abs/new/path" }))
    }) as unknown as typeof fetch

    const result = await updateSiteOutDir("/abs/new/path")

    expect(capturedUrl).toContain("/api/admin/site")
    expect(capturedMethod).toBe("PUT")
    expect(capturedBody).toEqual({ out_dir: "/abs/new/path" })
    expect(result).toEqual({ out_dir: "/abs/new/path" })
  })

  test("updateSiteOutDir surfaces a non-ok response as an ApiError", async () => {
    mockFetchOnce(422, { error: "path is outside the allowed root" })
    await expect(updateSiteOutDir("/etc")).rejects.toThrow("path is outside the allowed root")
  })

  test("browseDirectories GETs the root when no path is given", async () => {
    let capturedUrl: string | undefined
    globalThis.fetch = ((url: string | URL) => {
      capturedUrl = url.toString()
      return Promise.resolve(Response.json({ root: "/home/user", path: "/home/user", parent: "", dirs: [{ name: "Documents", path: "/home/user/Documents" }] }))
    }) as unknown as typeof fetch

    const result = await browseDirectories()

    expect(capturedUrl).toBe("http://localhost:8080/api/admin/browse")
    expect(result).toEqual({ root: "/home/user", path: "/home/user", parent: "", dirs: [{ name: "Documents", path: "/home/user/Documents" }] })
  })

  test("browseDirectories GETs a given path and coerces a null dirs list to empty", async () => {
    let capturedUrl: string | undefined
    globalThis.fetch = ((url: string | URL) => {
      capturedUrl = url.toString()
      return Promise.resolve(Response.json({ root: "/home/user", path: "/home/user/Documents", parent: "/home/user", dirs: null }))
    }) as unknown as typeof fetch

    const result = await browseDirectories("/home/user/Documents")

    expect(capturedUrl).toBe("http://localhost:8080/api/admin/browse?path=%2Fhome%2Fuser%2FDocuments")
    expect(result.dirs).toEqual([])
    expect(result.parent).toBe("/home/user")
  })

  test("browseDirectories surfaces a non-ok response as an ApiError", async () => {
    mockFetchOnce(422, { error: "path is outside the allowed root" })
    await expect(browseDirectories("/etc")).rejects.toThrow("path is outside the allowed root")
  })
})

describe("pending-build tracking (memento-lifecycle staged rebuild)", () => {
  test("getJourneyBuildStatus maps the pending memento ids and count", async () => {
    mockFetchOnce(200, { pending_memento_ids: ["memento-1", "memento-2"], pending_count: 2 })
    const result = await getJourneyBuildStatus("journey-1")
    expect(result).toEqual({ pending_memento_ids: ["memento-1", "memento-2"], pending_count: 2 })
  })

  test("getJourneyBuildStatus coerces a null pending_memento_ids list to empty", async () => {
    mockFetchOnce(200, { pending_memento_ids: null, pending_count: 0 })
    const result = await getJourneyBuildStatus("journey-1")
    expect(result.pending_memento_ids).toEqual([])
  })

  test("getJourneyBuildStatus surfaces a non-ok response as an ApiError", async () => {
    mockFetchOnce(404, { error: "journey not found" })
    await expect(getJourneyBuildStatus("missing")).rejects.toThrow("journey not found")
  })

  test("getBuildStatus maps the per-journey pending counts", async () => {
    mockFetchOnce(200, { pending_by_journey: { "journey-1": 2, "journey-2": 1 } })
    const result = await getBuildStatus()
    expect(result).toEqual({ pending_by_journey: { "journey-1": 2, "journey-2": 1 } })
  })

  test("getBuildStatus surfaces a non-ok response as an ApiError", async () => {
    mockFetchOnce(500, { error: "boom" })
    await expect(getBuildStatus()).rejects.toThrow("boom")
  })
})
