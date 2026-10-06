import { afterEach, describe, expect, test } from "vitest"
import { createMemento, isConflict, ApiError, type CreatedMemento } from "./api"

// These tests pin the frontend side of POST /api/admin/mementos/create
// (docs/contracts/automatic-sequence-allocation.md): the request goes to the
// dedicated create endpoint, carries the caller-chosen id and authored
// values, and can never carry seq or expected_revision — the server allocates
// the position and the caller navigates from the persisted row it returns.

const originalFetch = globalThis.fetch
afterEach(() => {
  globalThis.fetch = originalFetch
})

// recordedFetch captures every request (URL + parsed body) and replies with
// the queued responses in order; an empty queue answers 500.
let recorded: Array<{ url: string; body: Record<string, unknown> | undefined }> = []
function recordedFetch(responses: Array<{ status: number; body: unknown }>) {
  recorded = []
  let call = 0
  globalThis.fetch = ((_url: string | URL, init?: RequestInit) => {
    recorded.push({ url: String(_url), body: init?.body ? (JSON.parse(String(init.body)) as Record<string, unknown>) : undefined })
    const response = responses[call] ?? { status: 500, body: { error: "unstubbed" } }
    call += 1
    return Promise.resolve(Response.json(response.body, { status: response.status }))
  }) as unknown as typeof fetch
}

const createdRow = (overrides: Partial<CreatedMemento> = {}): CreatedMemento => ({
  id: "0190cbde-f300-7000-8000-000000000005",
  journey_id: "journey-1",
  kind: "goods",
  seq: 5,
  revision: 1,
  ...overrides,
})

const createPayload = () => ({
  id: "0190cbde-f300-7000-8000-000000000005",
  journey_id: "journey-1",
  kind: "goods",
  title: "Tenugui",
  place: "Kyoto",
  occurred_at: "2026-04-01T00:00:00Z",
  occurred_tz: "UTC",
  kind_data: {},
  state: "draft" as const,
})

describe("createMemento", () => {
  test("posts the authored payload without seq or expected_revision to the create endpoint", async () => {
    recordedFetch([{ status: 200, body: createdRow() }])
    const row = await createMemento(createPayload())
    expect(recorded).toHaveLength(1)
    expect(recorded[0].url).toContain("/api/admin/mementos/create")
    expect(recorded[0].body).toMatchObject({
      id: "0190cbde-f300-7000-8000-000000000005",
      journey_id: "journey-1",
      kind: "goods",
      title: "Tenugui",
      state: "draft",
      kind_data: {},
    })
    expect(recorded[0].body).not.toHaveProperty("seq")
    expect(recorded[0].body).not.toHaveProperty("expected_revision")
    // The authorship mask is server-derived; the client never sends one.
    expect(recorded[0].body).not.toHaveProperty("authored_fields")
    expect(row).toEqual(createdRow())
  })

  test("returns the persisted row identity the caller navigates with", async () => {
    recordedFetch([{ status: 200, body: createdRow({ seq: 6, revision: 1 }) }])
    const row = await createMemento(createPayload())
    expect(row.id).toBe("0190cbde-f300-7000-8000-000000000005")
    expect(row.seq).toBe(6)
    expect(row.revision).toBe(1)
  })

  test("surfaces an incompatible same-ID retry as a 409 ApiError", async () => {
    recordedFetch([{ status: 409, body: { error: "memento already exists with different creation values" } }])
    const cause = await createMemento(createPayload()).then(
      () => null,
      (error: unknown) => error,
    )
    expect(cause).toBeInstanceOf(ApiError)
    expect(isConflict(cause)).toBe(true)
  })

  test("propagates validation failures with their status", async () => {
    recordedFetch([{ status: 400, body: { error: "validation failed" } }])
    const cause = await createMemento(createPayload()).then(
      () => null,
      (error: unknown) => error,
    )
    expect(cause).toBeInstanceOf(ApiError)
    expect((cause as ApiError).status).toBe(400)
  })
})
