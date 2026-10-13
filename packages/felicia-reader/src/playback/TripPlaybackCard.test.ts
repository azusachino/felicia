import { describe, expect, it } from "vitest"
import { render } from "svelte/server"
import type { Journey } from "@felicia/model"
import { generatePlaybackSteps, PlaybackController, type PlaybackStep } from "./engine"
import TripPlaybackCard from "./TripPlaybackCard.svelte"

const memento = {
  id: "memento-1",
  kind: "souvenir",
  visitId: "visit-1",
  title: { zh: "烟台港", ja: "煙台港", en: "Yantai Port" },
  date: { zh: "2026-10-01", ja: "2026-10-01", en: "2026-10-01" },
  place: { zh: "烟台", ja: "煙台", en: "Yantai" },
  vendor: { zh: "", ja: "", en: "" },
  price: "",
  coords: [121.37, 37.55] as [number, number],
  essay: { zh: "海", ja: "海", en: "Sea" },
  photos: [],
} as unknown as Journey["mementos"][number]

const journey = {
  id: "journey-1",
  slug: "dongbei",
  title: { zh: "东北自驾", ja: "東北", en: "Northeast" },
  dates: { zh: "2026-10-01 — 2026-10-02", ja: "2026-10-01 — 2026-10-02", en: "2026-10-01 — 2026-10-02" },
  place: { zh: "辽宁", ja: "遼寧", en: "Liaoning" },
  route: [
    [121.37, 37.55],
    [121.65, 38.92],
  ],
  legModes: ["car"],
  mementos: [memento],
} as unknown as Journey

function renderWaypoint(step: PlaybackStep, lang: "en" | "ja" | "zh" = "en"): string {
  const controller = new PlaybackController([step])
  return render(TripPlaybackCard, {
    props: { controller, currentStep: step, lang },
  }).body
}

describe("TripPlaybackCard waypoint rendering", () => {
  it("renders no time row when the waypoint has no authored time", () => {
    const steps = generatePlaybackSteps(journey, "en")
    const waypoint = steps.find((s): s is Extract<PlaybackStep, { type: "waypoint" }> => s.type === "waypoint")
    expect(waypoint).toBeDefined()
    expect(waypoint!.time).toBeUndefined()
    const html = renderWaypoint(waypoint!)
    expect(html).not.toContain("08:30")
    expect(html).not.toContain("18:30")
    expect(html).not.toContain("Departure Time")
    expect(html).not.toContain("Arrival Time")
  })

  it("renders the authored time row when a real timestamp exists", () => {
    const steps = generatePlaybackSteps({ ...journey, departureTime: "06:15" } as Journey, "en")
    const waypoint = steps.find((s): s is Extract<PlaybackStep, { type: "waypoint" }> => s.type === "waypoint")
    const html = renderWaypoint(waypoint!)
    expect(html).toContain("Departure Time")
    expect(html).toContain("06:15")
  })

  it("renders the authored journey place, never an invented city name", () => {
    const steps = generatePlaybackSteps(journey, "en")
    const waypoint = steps.find((s): s is Extract<PlaybackStep, { type: "waypoint" }> => s.type === "waypoint")
    const html = renderWaypoint(waypoint!)
    expect(html).toContain("Liaoning (Origin)")
    for (const city of ["Tokyo", "東京", "Osaka", "Kyoto", "Nagoya"]) {
      expect(html).not.toContain(city)
    }
  })
})
