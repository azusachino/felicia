import { describe, expect, it, vi } from "vitest"
import type { Journey, Memento } from "@felicia/model"
import {
  formatChapters,
  formatTime,
  generatePlaybackSteps,
  getStepDuration,
  PlaybackController,
  type PlaybackStep,
} from "./engine"

const sampleMemento: Memento = {
  id: "memento-1",
  kind: "souvenir",
  visitId: "visit-1",
  title: { zh: "烟台港", ja: "煙台港", en: "Yantai Port" },
  date: { zh: "2026-10-01", ja: "2026-10-01", en: "2026-10-01" },
  place: { zh: "烟台", ja: "煙台", en: "Yantai" },
  vendor: { zh: "", ja: "", en: "" },
  price: "",
  coords: [121.37, 37.55],
  essay: {
    zh: "在烟台到大连的轮渡上看到了最美的大海和日落！",
    ja: "フェリーからの夕日",
    en: "Beautiful sunset from the ferry.",
  },
  photos: [
    {
      src: "https://example.com/photo1.webp",
      video: "https://example.com/photo1.mp4",
      caption: { zh: "日落", ja: "夕日", en: "Sunset" },
    },
  ],
}

const sampleMemento2: Memento = {
  id: "memento-2",
  kind: "transit",
  visitId: "visit-2",
  title: { zh: "大连湾", ja: "大連湾", en: "Dalian Bay" },
  date: { zh: "2026-10-02", ja: "2026-10-02", en: "2026-10-02" },
  place: { zh: "大连", ja: "大連", en: "Dalian" },
  vendor: { zh: "", ja: "", en: "" },
  price: "",
  coords: [121.65, 38.92],
  essay: { zh: "晨光中的大连港。", ja: "朝の大連", en: "Morning Dalian." },
  photos: [],
}

const sampleJourney: Journey = {
  id: "journey-1",
  slug: "dongbei-road-trip",
  title: { zh: "国庆东北自驾", ja: "東北ドライブ", en: "Northeast Road Trip" },
  dates: { zh: "2026-10-01 — 2026-10-07", ja: "2026-10-01 — 2026-10-07", en: "2026-10-01 — 2026-10-07" },
  place: { zh: "辽宁 · 吉林", ja: "遼寧・吉林", en: "Liaoning & Jilin" },
  route: [
    [121.37, 37.55],
    [121.65, 38.92],
  ],
  legModes: ["car", "ferry"],
  weather: {
    1: { icon: "i-lucide-cloud", label: "阴 · 16–21°C" },
    2: { icon: "i-lucide-sun", label: "晴 · 12–22°C" },
  },
  distanceKm: 2890,
  drivingHours: 36,
  visits: [],
  mementos: [sampleMemento, sampleMemento2],
}

describe("Playback Engine", () => {
  it("formats time and chapters correctly", () => {
    expect(formatTime(0)).toBe("0:00")
    expect(formatTime(75)).toBe("1:15")
    expect(formatTime(3605)).toBe("60:05")

    const chapters = [
      { label: "国庆东北自驾", timeSeconds: 0 },
      { label: "第 1 天 · 2026-10-01", timeSeconds: 4 },
      { label: "烟台港", timeSeconds: 15 },
    ]
    expect(formatChapters(chapters)).toBe("0:00 国庆东北自驾\n0:04 第 1 天 · 2026-10-01\n0:15 烟台港\n")
  })

  it("generates structured playback steps (intro, days, start/end waypoints, stops, summary)", () => {
    const steps = generatePlaybackSteps(sampleJourney, "zh")
    expect(steps.length).toBe(8)

    // Step 0: Intro
    expect(steps[0].type).toBe("intro")
    if (steps[0].type === "intro") {
      expect(steps[0].title).toBe("国庆东北自驾")
      expect(steps[0].dayCount).toBe(2)
      expect(steps[0].stopCount).toBe(2)
      expect(steps[0].distanceKm).toBe(2890)
    }

    // Step 1: Day 1
    expect(steps[1].type).toBe("day")
    if (steps[1].type === "day") {
      expect(steps[1].day).toBe(1)
      expect(steps[1].date).toBe("2026-10-01")
      expect(steps[1].weather?.label).toBe("阴 · 16–21°C")
    }

    // Step 2: Start Waypoint (Departure)
    expect(steps[2].type).toBe("waypoint")
    if (steps[2].type === "waypoint") {
      expect(steps[2].role).toBe("start")
      expect(steps[2].coords).toEqual([121.37, 37.55])
      expect(steps[2].distanceKm).toBe(0)
      expect(steps[2].date).toBe("2026-10-01")
      expect(steps[2].time).toBeUndefined()
    }

    // Step 3: Stop 1
    expect(steps[3].type).toBe("stop")
    if (steps[3].type === "stop") {
      expect(steps[3].memento.id).toBe("memento-1")
      expect(steps[3].coords).toEqual([121.37, 37.55])
    }

    // Step 4: Day 2
    expect(steps[4].type).toBe("day")
    if (steps[4].type === "day") {
      expect(steps[4].day).toBe(2)
      expect(steps[4].date).toBe("2026-10-02")
      expect(steps[4].weather?.label).toBe("晴 · 12–22°C")
    }

    // Step 5: Stop 2
    expect(steps[5].type).toBe("stop")
    if (steps[5].type === "stop") {
      expect(steps[5].memento.id).toBe("memento-2")
    }

    // Step 6: End Waypoint (Arrival/Return)
    expect(steps[6].type).toBe("waypoint")
    if (steps[6].type === "waypoint") {
      expect(steps[6].role).toBe("end")
      expect(steps[6].coords).toEqual([121.65, 38.92])
      expect(steps[6].distanceKm).toBe(2890)
      expect(steps[6].date).toBe("2026-10-02")
      expect(steps[6].time).toBeUndefined()
    }

    // Step 7: Summary
    expect(steps[7].type).toBe("summary")
    if (steps[7].type === "summary") {
      expect(steps[7].stopCount).toBe(2)
      expect(steps[7].placeCount).toBe(2)
      expect(steps[7].drivingHours).toBe(36)
    }
  })

  it("calculates step dwell durations with content-aware algorithm and speed multiplier", () => {
    const steps = generatePlaybackSteps(sampleJourney, "zh")
    const introDuration = getStepDuration(steps[0], 1)
    expect(introDuration).toBe(3500)

    const waypointDuration = getStepDuration(steps[2], 1)
    expect(waypointDuration).toBe(3000)

    const stop1Duration = getStepDuration(steps[3], 1)
    // 1 photo (= 1200), 22 chars (= 880), base dwell 6080 + 800 (0km drive cushion) -> 6880
    expect(stop1Duration).toBe(6880)

    // Stop 2 travels from Yantai to Dalian (~154km): travelDriveMs is 1200 + (154.4 / 12) * 1000 = 14067, dwell is 5000
    const stop2Duration = getStepDuration(steps[5], 1)
    expect(stop2Duration).toBeGreaterThan(18000)

    // At 2x speed, duration halves
    expect(getStepDuration(steps[0], 2)).toBe(1750)
  })

  it("never invents city names, even for coordinates inside the retired Japanese demo rectangles", () => {
    const japanJourney: Journey = {
      ...sampleJourney,
      id: "journey-jp",
      slug: "jp-trip",
      title: { zh: "东日本", ja: "東日本", en: "East Japan" },
      place: { zh: "关东 · 关西", ja: "関東・関西", en: "Kanto & Kansai" },
      route: [
        [139.7, 35.68],
        [135.5, 34.69],
      ],
      mementos: [sampleMemento],
    }
    for (const lang of ["ja", "en", "zh"] as const) {
      const steps = generatePlaybackSteps(japanJourney, lang)
      const texts = steps.map((s) => (s.type === "stop" ? "" : s.type === "intro" || s.type === "summary" ? s.title : s.type === "waypoint" ? `${s.title} ${s.place}` : s.places.join(" ")))
      const joined = texts.join("\n")
      for (const city of ["東京", "东京", "Tokyo", "川口", "大阪", "Osaka", "京都", "Kyoto", "名古屋", "Nagoya"]) {
        expect(joined).not.toContain(city)
      }
    }
    // Authored journey place is used for waypoint places.
    const steps = generatePlaybackSteps(japanJourney, "en")
    const wp = steps.filter((s) => s.type === "waypoint") as Extract<PlaybackStep, { type: "waypoint" }>[]
    expect(wp[0].place).toBe("Kanto & Kansai (Origin)")
    expect(wp[1].place).toBe("Kanto & Kansai (Destination)")
  })

  it("falls back to neutral origin/destination labels when no authored place exists", () => {
    const anonymousJourney: Journey = {
      ...sampleJourney,
      id: "journey-anon",
      slug: "anon",
      title: { zh: "旅程", ja: "旅", en: "Journey" },
      place: { zh: "", ja: "", en: "" },
      mementos: [sampleMemento],
    }
    expect(generatePlaybackSteps(anonymousJourney, "en").filter((s) => s.type === "waypoint").map((s) => (s as Extract<PlaybackStep, { type: "waypoint" }>).place)).toEqual(["Origin", "Destination"])
    expect(generatePlaybackSteps(anonymousJourney, "ja").filter((s) => s.type === "waypoint").map((s) => (s as Extract<PlaybackStep, { type: "waypoint" }>).place)).toEqual(["出発地", "到着地"])
    expect(generatePlaybackSteps(anonymousJourney, "zh").filter((s) => s.type === "waypoint").map((s) => (s as Extract<PlaybackStep, { type: "waypoint" }>).place)).toEqual(["出发地", "到达地"])
  })

  it("omits waypoint times unless authored, and keeps authored times", () => {
    // sampleJourney has no departureTime/arrivalTime: no step carries a time.
    for (const step of generatePlaybackSteps(sampleJourney, "en")) {
      if (step.type === "waypoint") {
        expect(step.time).toBeUndefined()
      }
    }
    const timedJourney: Journey = {
      ...sampleJourney,
      departureTime: "06:15",
      arrivalTime: "21:40",
    }
    const waypoints = generatePlaybackSteps(timedJourney, "en").filter((s) => s.type === "waypoint") as Extract<PlaybackStep, { type: "waypoint" }>[]
    expect(waypoints.map((w) => w.time)).toEqual(["06:15", "21:40"])
  })

  it("manages playback state transitions via controller", () => {
    vi.useFakeTimers()
    const steps = generatePlaybackSteps(sampleJourney, "zh")
    const controller = new PlaybackController(steps)

    let state = controller.getState()
    expect(state.currentStepIndex).toBe(0)
    expect(state.isPlaying).toBe(false)

    controller.play()
    state = controller.getState()
    expect(state.isPlaying).toBe(true)

    // Advance to next step
    controller.next()
    expect(controller.getState().currentStepIndex).toBe(1)

    controller.previous()
    expect(controller.getState().currentStepIndex).toBe(0)

    controller.seek(4)
    expect(controller.getState().currentStepIndex).toBe(4)

    controller.pause()
    expect(controller.getState().isPlaying).toBe(false)

    controller.destroy()
    vi.useRealTimers()
  })
})
