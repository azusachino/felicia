import { afterEach, beforeAll, describe, expect, test } from "bun:test"
import {
  ApiError,
  countMementosByState,
  countPendingStopCandidates,
  getJourney,
  getTemplates,
  isConflict,
  listJourneys,
  listMementos,
  listStopCandidates,
  loadJourneySummaries,
  photoTray,
  planIntake,
  promoteStopCandidate,
  reviewStopCandidate,
  routePointCount,
  sortMementosBySeq,
  syncRoute,
  syncVisits,
  type AdminJourney,
  type AdminMemento,
  type AdminStopCandidate,
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
