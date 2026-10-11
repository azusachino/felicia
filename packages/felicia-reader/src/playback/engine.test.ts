import { describe, expect, it, vi } from "vitest"
import type { Journey, Memento } from "@felicia/model"
import {
  formatChapters,
  formatTime,
  generatePlaybackSteps,
  getStepDuration,
  PlaybackController,
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

  it("generates structured playback steps (intro, days, stops, summary)", () => {
    const steps = generatePlaybackSteps(sampleJourney, "zh")
    expect(steps.length).toBe(6)

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

    // Step 2: Stop 1
    expect(steps[2].type).toBe("stop")
    if (steps[2].type === "stop") {
      expect(steps[2].memento.id).toBe("memento-1")
      expect(steps[2].coords).toEqual([121.37, 37.55])
    }

    // Step 3: Day 2
    expect(steps[3].type).toBe("day")
    if (steps[3].type === "day") {
      expect(steps[3].day).toBe(2)
      expect(steps[3].date).toBe("2026-10-02")
      expect(steps[3].weather?.label).toBe("晴 · 12–22°C")
    }

    // Step 4: Stop 2
    expect(steps[4].type).toBe("stop")
    if (steps[4].type === "stop") {
      expect(steps[4].memento.id).toBe("memento-2")
    }

    // Step 5: Summary
    expect(steps[5].type).toBe("summary")
    if (steps[5].type === "summary") {
      expect(steps[5].stopCount).toBe(2)
      expect(steps[5].placeCount).toBe(2)
      expect(steps[5].drivingHours).toBe(36)
    }
  })

  it("calculates step dwell durations with content-aware algorithm and speed multiplier", () => {
    const steps = generatePlaybackSteps(sampleJourney, "zh")
    const introDuration = getStepDuration(steps[0], 1)
    expect(introDuration).toBe(3500)

    const stop1Duration = getStepDuration(steps[2], 1)
    // 1 photo (= 2000), 23 chars (= 2760), min 5000 -> 5000
    expect(stop1Duration).toBe(5000)

    // At 2x speed, duration halves
    expect(getStepDuration(steps[0], 2)).toBe(1750)
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
