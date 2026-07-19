import { describe, expect, test } from "bun:test"
import type { AdminTemplateField } from "./api"
import {
  buildKindData,
  buildPhotoPayload,
  buildPriceFields,
  buildUpsertPayload,
  emptyKindFormState,
  emptyLatLngInputs,
  fromRFC3339,
  geomRequestFromPoints,
  geomToLatLngInputs,
  groupIssuesByField,
  lifecycleActionLabel,
  nextLifecycleState,
  parseKindData,
  parseLatLng,
  parsePriceAmount,
  parseSeq,
  previousLifecycleState,
  toRFC3339,
  unpublishActionLabel,
  FORM_LEVEL_ISSUE_KEY,
} from "./mementoForm"

const transitFields: AdminTemplateField[] = [
  { Name: "operator", Type: "text", Required: true, Values: null },
  { Name: "line", Type: "text", Required: false, Values: null },
  { Name: "from", Type: "station", Required: true, Values: null },
  { Name: "to", Type: "station", Required: true, Values: null },
  { Name: "fare", Type: "money", Required: false, Values: null },
]

const goodsFields: AdminTemplateField[] = [
  { Name: "name", Type: "text", Required: true, Values: null },
  { Name: "shop", Type: "text", Required: false, Values: null },
  { Name: "price", Type: "money", Required: false, Values: null },
  { Name: "manufacturer", Type: "text", Required: false, Values: null },
]

describe("toRFC3339 / fromRFC3339", () => {
  test("appends seconds and Z to a bare datetime-local value", () => {
    expect(toRFC3339("2026-03-21T09:00")).toBe("2026-03-21T09:00:00Z")
  })

  test("leaves an already-zoned value alone", () => {
    expect(toRFC3339("2026-03-21T09:00:00Z")).toBe("2026-03-21T09:00:00Z")
    expect(toRFC3339("2026-03-21T09:00:00+09:00")).toBe("2026-03-21T09:00:00+09:00")
  })

  test("empty input round-trips to empty", () => {
    expect(toRFC3339("")).toBe("")
    expect(fromRFC3339(undefined)).toBe("")
    expect(fromRFC3339("")).toBe("")
  })

  test("fromRFC3339 truncates to datetime-local precision", () => {
    expect(fromRFC3339("2026-03-21T09:00:00Z")).toBe("2026-03-21T09:00")
  })
})

describe("price fields", () => {
  test("parsePriceAmount truncates a fractional value", () => {
    expect(parsePriceAmount("210.9")).toBe(210)
  })

  test("parsePriceAmount treats blank as unset", () => {
    expect(parsePriceAmount("  ")).toBeUndefined()
  })

  test("buildPriceFields omits both when blank", () => {
    expect(buildPriceFields({ amount: "", currency: "" })).toEqual({})
  })

  test("buildPriceFields uppercases the currency", () => {
    expect(buildPriceFields({ amount: "210", currency: "jpy" })).toEqual({ price_amount: 210, price_currency: "JPY" })
  })
})

describe("location (geom) mapping", () => {
  test("emptyLatLngInputs sizes to the anchor", () => {
    expect(emptyLatLngInputs("point")).toEqual([{ lat: "", lng: "" }])
    expect(emptyLatLngInputs("edge")).toEqual([
      { lat: "", lng: "" },
      { lat: "", lng: "" },
    ])
  })

  test("geomToLatLngInputs maps a Point (orb array form)", () => {
    expect(geomToLatLngInputs([139.7, 35.6], "point")).toEqual([{ lat: "35.6", lng: "139.7" }])
  })

  test("geomToLatLngInputs maps a LineString (orb array form)", () => {
    expect(
      geomToLatLngInputs(
        [
          [139.7, 35.6],
          [135.7, 34.9],
        ],
        "edge",
      ),
    ).toEqual([
      { lat: "35.6", lng: "139.7" },
      { lat: "34.9", lng: "135.7" },
    ])
  })

  test("geomToLatLngInputs falls back to empty inputs for null geom", () => {
    expect(geomToLatLngInputs(null, "point")).toEqual([{ lat: "", lng: "" }])
  })

  test("parseLatLng rejects out-of-range or non-numeric input", () => {
    expect(parseLatLng({ lat: "35.6", lng: "139.7" })).toEqual([139.7, 35.6])
    expect(parseLatLng({ lat: "999", lng: "139.7" })).toBeNull()
    expect(parseLatLng({ lat: "", lng: "" })).toBeNull()
  })

  test("geomRequestFromPoints builds a Point for a point anchor", () => {
    expect(geomRequestFromPoints("point", [{ lat: "35.6", lng: "139.7" }])).toEqual({ type: "Point", coordinates: [139.7, 35.6] })
  })

  test("geomRequestFromPoints builds a LineString for an edge anchor with two valid points", () => {
    expect(
      geomRequestFromPoints("edge", [
        { lat: "35.6", lng: "139.7" },
        { lat: "34.9", lng: "135.7" },
      ]),
    ).toEqual({
      type: "LineString",
      coordinates: [
        [139.7, 35.6],
        [135.7, 34.9],
      ],
    })
  })

  test("geomRequestFromPoints returns null when an edge anchor doesn't have two valid points yet", () => {
    expect(geomRequestFromPoints("edge", [{ lat: "35.6", lng: "139.7" }])).toBeNull()
  })

  test("geomRequestFromPoints returns null for a point anchor with no valid point", () => {
    expect(geomRequestFromPoints("point", [{ lat: "", lng: "" }])).toBeNull()
  })
})

describe("kind_data mapping (transit/goods forms)", () => {
  test("emptyKindFormState seeds text/money/station shapes per field type", () => {
    expect(emptyKindFormState(transitFields)).toEqual({
      operator: "",
      line: "",
      from: { name: "", lat: "", lng: "" },
      to: { name: "", lat: "", lng: "" },
      fare: { amount: "", currency: "" },
    })
  })

  test("parseKindData maps a fetched kind_data blob into form state", () => {
    const state = parseKindData(transitFields, {
      operator: "JR East",
      from: { name: "Shibuya", coords: [139.701, 35.659] },
      to: { name: "Shinjuku", coords: [139.699, 35.69] },
      fare: { amount: 210, currency: "JPY" },
    })
    expect(state.operator).toBe("JR East")
    expect(state.from).toEqual({ name: "Shibuya", lat: "35.659", lng: "139.701" })
    expect(state.fare).toEqual({ amount: "210", currency: "JPY" })
    expect(state.line).toBe("")
  })

  test("parseKindData tolerates missing kind_data", () => {
    expect(parseKindData(goodsFields, undefined)).toEqual(emptyKindFormState(goodsFields))
    expect(parseKindData(goodsFields, null)).toEqual(emptyKindFormState(goodsFields))
  })

  test("buildKindData emits only fields with content, following the template's closed field set", () => {
    const state = emptyKindFormState(goodsFields)
    state.name = "Tenugui"
    state.price = { amount: "1200", currency: "jpy" }
    // shop/manufacturer left blank.
    expect(buildKindData(goodsFields, state)).toEqual({
      name: "Tenugui",
      price: { amount: 1200, currency: "JPY" },
    })
  })

  test("buildKindData round-trips a station field", () => {
    const state = emptyKindFormState(transitFields)
    state.operator = "JR East"
    state.from = { name: "Shibuya", lat: "35.659", lng: "139.701" }
    state.to = { name: "Shinjuku", lat: "35.69", lng: "139.699" }
    const data = buildKindData(transitFields, state)
    expect(data.from).toEqual({ name: "Shibuya", coords: [139.701, 35.659] })
    expect(data.to).toEqual({ name: "Shinjuku", coords: [139.699, 35.69] })
    expect(data.line).toBeUndefined()
  })

  test("buildKindData omits a money field with no amount or currency", () => {
    const state = emptyKindFormState(goodsFields)
    state.name = "Tenugui"
    expect(buildKindData(goodsFields, state).price).toBeUndefined()
  })
})

describe("buildUpsertPayload", () => {
  test("assembles the full request from identity + common fields + geom + kind_data", () => {
    const payload = buildUpsertPayload({
      identity: { id: "memento-1", journey_id: "journey-1", kind: "goods", seq: 3, authored_fields: ["essay"] },
      common: {
        title: "Tenugui",
        place: "Kyoto",
        occurredAtLocal: "2026-03-21T09:00",
        occurredTz: "Asia/Tokyo",
        essay: "Bought at a small shop.",
        vendor: "  ",
        price: { amount: "1200", currency: "jpy" },
      },
      anchor: "point",
      points: [{ lat: "35.0", lng: "135.7" }],
      kindData: { name: "Tenugui" },
      state: "authored",
      expectedRevision: 4,
    })

    expect(payload).toEqual({
      id: "memento-1",
      journey_id: "journey-1",
      kind: "goods",
      seq: 3,
      occurred_at: "2026-03-21T09:00:00Z",
      occurred_tz: "Asia/Tokyo",
      geom: { type: "Point", coordinates: [135.7, 35.0] },
      title: "Tenugui",
      place: "Kyoto",
      vendor: undefined,
      essay: "Bought at a small shop.",
      price_amount: 1200,
      price_currency: "JPY",
      kind_data: { name: "Tenugui" },
      source_ref: undefined,
      authored_fields: ["essay"],
      orphaned_at: undefined,
      state: "authored",
      expected_revision: 4,
    })
  })

  test("carries source_ref/orphaned_at through unchanged when present", () => {
    const payload = buildUpsertPayload({
      identity: { id: "m1", journey_id: "j1", kind: "goods", seq: 1, source_ref: "immich:1", authored_fields: [], orphaned_at: "2026-01-01T00:00:00Z" },
      common: { title: "", place: "", occurredAtLocal: "", occurredTz: "", essay: "", vendor: "", price: { amount: "", currency: "" } },
      anchor: "point",
      points: [{ lat: "", lng: "" }],
      kindData: {},
      state: "draft",
      expectedRevision: 0,
    })
    expect(payload.source_ref).toBe("immich:1")
    expect(payload.orphaned_at).toBe("2026-01-01T00:00:00Z")
    expect(payload.geom).toBeNull()
    expect(payload.occurred_at).toBeUndefined()
  })
})

describe("validation issue grouping", () => {
  test("groups issues by field and translates known codes", () => {
    const grouped = groupIssuesByField([
      { Field: "operator", Code: "required_missing" },
      { Field: "", Code: "anchor_mismatch" },
      { Field: "operator", Code: "type_mismatch" },
    ])
    expect(grouped.operator).toEqual(["This field is required.", "This value doesn't match the expected type."])
    expect(grouped[FORM_LEVEL_ISSUE_KEY]).toEqual(["The location doesn't match this kind's shape (a single point vs. a from/to route)."])
  })

  test("falls back to the raw code for an unrecognized issue", () => {
    const grouped = groupIssuesByField([{ Field: "geom", Code: "something_new" }])
    expect(grouped.geom).toEqual(["something_new"])
  })
})

describe("lifecycle transitions", () => {
  test("draft advances to authored, authored advances to published", () => {
    expect(nextLifecycleState("draft")).toBe("authored")
    expect(nextLifecycleState("authored")).toBe("published")
    expect(nextLifecycleState("published")).toBeNull()
    expect(nextLifecycleState("archived")).toBeNull()
  })

  test("action labels match the next state", () => {
    expect(lifecycleActionLabel("draft")).toBe("Mark authored")
    expect(lifecycleActionLabel("authored")).toBe("Publish")
    expect(lifecycleActionLabel("published")).toBeNull()
  })

  test("published steps back to authored; nothing else has a backward step", () => {
    expect(previousLifecycleState("published")).toBe("authored")
    expect(previousLifecycleState("authored")).toBeNull()
    expect(previousLifecycleState("draft")).toBeNull()
    expect(previousLifecycleState("archived")).toBeNull()
    expect(previousLifecycleState("candidate")).toBeNull()
  })

  test("unpublish label only appears for published", () => {
    expect(unpublishActionLabel("published")).toBe("Unpublish")
    expect(unpublishActionLabel("authored")).toBeNull()
    expect(unpublishActionLabel("draft")).toBeNull()
  })
})

describe("photo payload mapping", () => {
  test("parseSeq falls back on a non-integer value", () => {
    expect(parseSeq("3")).toBe(3)
    expect(parseSeq("abc", 7)).toBe(7)
    expect(parseSeq("")).toBe(0)
  })

  test("buildPhotoPayload trims and omits blank optional fields", () => {
    const payload = buildPhotoPayload("photo-1", "memento-1", {
      objectKey: "  media/abc.jpg  ",
      contentHash: "sha256:abc",
      caption: "  ",
      seq: "2",
      takenAt: "",
      sourceRef: "immich:9",
    })
    expect(payload).toEqual({
      id: "photo-1",
      memento_id: "memento-1",
      object_key: "media/abc.jpg",
      content_hash: "sha256:abc",
      caption: undefined,
      seq: 2,
      taken_at: undefined,
      source_ref: "immich:9",
    })
  })

  test("buildPhotoPayload converts takenAt to RFC3339", () => {
    const payload = buildPhotoPayload("photo-1", "memento-1", {
      objectKey: "media/abc.jpg",
      contentHash: "sha256:abc",
      caption: "At the platform",
      seq: "0",
      takenAt: "2026-03-21T09:00",
      sourceRef: "",
    })
    expect(payload.taken_at).toBe("2026-03-21T09:00:00Z")
    expect(payload.source_ref).toBeUndefined()
  })
})
