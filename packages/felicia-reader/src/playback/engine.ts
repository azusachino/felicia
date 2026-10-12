import type { Coordinates, Journey, Memento, PhotoAsset, WeatherInfo } from "@felicia/model"
import { calculateLegDriveDurationMs, haversineDistance } from "./timeline"

export type StepType = "intro" | "day" | "waypoint" | "stop" | "summary"

export interface IntroStep {
  type: "intro"
  title: string
  dateRange: string
  dayCount: number
  stopCount: number
  photoCount: number
  distanceKm?: number
  drivingHours?: number
  covers: PhotoAsset[]
}

export interface DayStep {
  type: "day"
  day: number
  date: string
  places: string[]
  weather?: WeatherInfo
  stopIndex: number
}

export interface WaypointStep {
  type: "waypoint"
  role: "start" | "end"
  title: string
  place: string
  day: number
  coords: Coordinates
  distanceKm?: number
  travelDistanceKm?: number
  time?: string
  date?: string
}

export interface StopStep {
  type: "stop"
  day: number
  stopIndex: number
  memento: Memento
  coords: Coordinates
  anchor: string
  travelDistanceKm?: number
}

export interface SummaryStep {
  type: "summary"
  title: string
  dayCount: number
  stopCount: number
  placeCount: number
  photoCount: number
  distanceKm?: number
  drivingHours?: number
}

export type PlaybackStep = IntroStep | DayStep | WaypointStep | StopStep | SummaryStep

export interface Chapter {
  label: string
  timeSeconds: number
}

export function formatTime(seconds: number): string {
  const mins = Math.floor(seconds / 60)
  const secs = seconds % 60
  return `${mins}:${secs.toString().padStart(2, "0")}`
}

export function formatChapters(chapters: Chapter[]): string {
  return chapters.map((c) => `${formatTime(c.timeSeconds)} ${c.label}`).join("\n") + "\n"
}

export function getStepDuration(step: PlaybackStep, speed = 1): number {
  let baseMs = 3500
  if (step.type === "intro") {
    baseMs = 3500
  } else if (step.type === "day") {
    baseMs = 2600
  } else if (step.type === "waypoint") {
    if (step.role === "start") {
      baseMs = 3000
    } else {
      const driveMs = calculateLegDriveDurationMs(step.travelDistanceKm ?? 0)
      baseMs = driveMs + 5000
    }
  } else if (step.type === "stop") {
    const photoCount = step.memento.photos.length
    const textLen = (step.memento.essay.zh || step.memento.essay.ja || step.memento.essay.en || "").length
    const driveMs = calculateLegDriveDurationMs(step.travelDistanceKm ?? 0)
    // Dwell duration after arrival for reading & viewing photos
    const dwellMs = Math.max(5000, Math.min(8500, 4000 + photoCount * 1200 + textLen * 40))
    baseMs = driveMs + dwellMs
  } else if (step.type === "summary") {
    baseMs = 5000
  }
  return Math.round(baseMs / speed)
}

function parseDateKey(dateStr: string): string {
  return dateStr.slice(0, 10)
}

export function resolveWaypointPlace(
  coords: Coordinates,
  role: "start" | "end",
  journeyPlace: string,
  lang: "ja" | "en" | "zh" = "ja"
): string {
  const [lon, lat] = coords
  // Greater Tokyo / Saitama Border (Kawaguchi / Kita-ku area)
  if (lat >= 35.78 && lat <= 35.85 && lon >= 139.68 && lon <= 139.75) {
    return lang === "ja" ? "東京・川口" : lang === "zh" ? "东京・川口" : "Tokyo (Kawaguchi)"
  }
  // Greater Tokyo / Kanto Area
  if (lat >= 35.3 && lat <= 36.2 && lon >= 139.2 && lon <= 140.2) {
    return lang === "ja" ? "東京" : lang === "zh" ? "东京" : "Tokyo"
  }
  // Greater Osaka / Kansai Area
  if (lat >= 34.4 && lat <= 35.0 && lon >= 135.2 && lon <= 135.8) {
    return lang === "ja" ? "大阪" : lang === "zh" ? "大阪" : "Osaka"
  }
  // Kyoto Area
  if (lat >= 34.9 && lat <= 35.2 && lon >= 135.6 && lon <= 135.9) {
    return lang === "ja" ? "京都" : lang === "zh" ? "京都" : "Kyoto"
  }
  // Nagoya Area
  if (lat >= 35.0 && lat <= 35.3 && lon >= 136.7 && lon <= 137.1) {
    return lang === "ja" ? "名古屋" : lang === "zh" ? "名古屋" : "Nagoya"
  }
  if (role === "start") {
    return lang === "ja" ? `${journeyPlace}（出発地）` : lang === "zh" ? `${journeyPlace}（出发地）` : `${journeyPlace} (Origin)`
  }
  return lang === "ja" ? `${journeyPlace}（到着地）` : lang === "zh" ? `${journeyPlace}（到达地）` : `${journeyPlace} (Destination)`
}

export function resolveWaypointTitle(
  role: "start" | "end",
  lang: "ja" | "en" | "zh" = "ja"
): string {
  if (role === "start") {
    return lang === "ja" ? "旅の出発" : lang === "zh" ? "旅程出发" : "Journey Departure"
  }
  return lang === "ja" ? "帰路・到着" : lang === "zh" ? "归途・到达" : "Return & Arrival"
}

export function generatePlaybackSteps(journey: Journey, lang: "ja" | "en" | "zh" = "ja"): PlaybackStep[] {
  const mementos = [...journey.mementos]
  const allPhotos = mementos.flatMap((m) => m.photos)
  const uniquePlaces = [...new Set(mementos.map((m) => m.place[lang] || m.place.ja || m.place.en || m.place.zh))]

  // Route bounds / coordinates
  const segments = journey.routeSegments?.length ? journey.routeSegments : (journey.route?.length ? [journey.route] : [])
  const validSegments = segments.filter((s) => s.length >= 2)
  const originCoord = validSegments[0]?.[0]
  const lastSegment = validSegments.at(-1)
  const destCoord = lastSegment?.at(-1)
  const journeyPlaceName = journey.place[lang] || journey.place.ja || journey.place.en || journey.place.zh || ""

  // Group mementos by distinct date
  const dateGroups: { date: string; mementos: Memento[] }[] = []
  for (const m of mementos) {
    const d = parseDateKey(m.date[lang] || m.date.ja || m.date.en || m.date.zh || "")
    const last = dateGroups.at(-1)
    if (last && last.date === d) {
      last.mementos.push(m)
    } else {
      dateGroups.push({ date: d, mementos: [m] })
    }
  }

  const dayCount = Math.max(dateGroups.length, 1)
  const covers = journey.covers?.length ? journey.covers : allPhotos.slice(0, 3)

  const intro: IntroStep = {
    type: "intro",
    title: journey.title[lang] || journey.title.ja || journey.title.en || journey.title.zh,
    dateRange: journey.dates[lang] || journey.dates.ja || journey.dates.en || journey.dates.zh,
    dayCount,
    stopCount: mementos.length,
    photoCount: allPhotos.length,
    distanceKm: journey.distanceKm,
    drivingHours: journey.drivingHours,
    covers,
  }

  const steps: PlaybackStep[] = [intro]
  let globalStopIndex = 0
  let prevCoords: Coordinates | undefined = originCoord

  dateGroups.forEach((group, dayIdx) => {
    const dayNumber = dayIdx + 1
    const dayPlaces = [...new Set(group.mementos.map((m) => m.place[lang] || m.place.ja || m.place.en || m.place.zh))]
    const weather = journey.weather?.[dayNumber] ?? journey.weather?.[group.date]

    const dayStep: DayStep = {
      type: "day",
      day: dayNumber,
      date: group.date,
      places: dayPlaces,
      weather,
      stopIndex: globalStopIndex,
    }
    steps.push(dayStep)

    // On Day 1, if we have a departure origin, insert Start Waypoint
    if (dayIdx === 0 && originCoord) {
      steps.push({
        type: "waypoint",
        role: "start",
        title: resolveWaypointTitle("start", lang),
        place: resolveWaypointPlace(originCoord, "start", journeyPlaceName, lang),
        day: 1,
        date: group.date,
        time: journey.departureTime || "08:30",
        coords: originCoord,
        distanceKm: 0,
        travelDistanceKm: 0,
      })
      prevCoords = originCoord
    }

    for (const memento of group.mementos) {
      const distKm = prevCoords ? haversineDistance(prevCoords, memento.coords) : 0
      steps.push({
        type: "stop",
        day: dayNumber,
        stopIndex: globalStopIndex,
        memento,
        coords: memento.coords,
        anchor: memento.id,
        travelDistanceKm: Math.round(distKm * 10) / 10,
      })
      prevCoords = memento.coords
      globalStopIndex++
    }

    // On the final day, after all mementos, if we have a return destination, insert End Waypoint
    if (dayIdx === dateGroups.length - 1 && destCoord) {
      const distKm = prevCoords ? haversineDistance(prevCoords, destCoord) : 0
      steps.push({
        type: "waypoint",
        role: "end",
        title: resolveWaypointTitle("end", lang),
        place: resolveWaypointPlace(destCoord, "end", journeyPlaceName, lang),
        day: dayNumber,
        date: group.date,
        time: journey.arrivalTime || "18:30",
        coords: destCoord,
        distanceKm: journey.distanceKm,
        travelDistanceKm: Math.round(distKm * 10) / 10,
      })
    }
  })

  // If there were no date groups (e.g. empty mementos) but route exists:
  if (dateGroups.length === 0 && originCoord && destCoord) {
    steps.push({
      type: "waypoint",
      role: "start",
      title: resolveWaypointTitle("start", lang),
      place: resolveWaypointPlace(originCoord, "start", journeyPlaceName, lang),
      day: 1,
      time: journey.departureTime || "08:30",
      coords: originCoord,
      distanceKm: 0,
    })
    steps.push({
      type: "waypoint",
      role: "end",
      title: resolveWaypointTitle("end", lang),
      place: resolveWaypointPlace(destCoord, "end", journeyPlaceName, lang),
      day: 1,
      time: journey.arrivalTime || "18:30",
      coords: destCoord,
      distanceKm: journey.distanceKm,
    })
  }

  const summary: SummaryStep = {
    type: "summary",
    title: journey.title[lang] || journey.title.ja || journey.title.en || journey.title.zh,
    dayCount,
    stopCount: mementos.length,
    placeCount: uniquePlaces.length,
    photoCount: allPhotos.length,
    distanceKm: journey.distanceKm,
    drivingHours: journey.drivingHours,
  }
  steps.push(summary)

  return steps
}

export interface PlaybackControllerState {
  currentStepIndex: number
  isPlaying: boolean
  speed: number
  totalSteps: number
}

export type StateListener = (state: PlaybackControllerState, step: PlaybackStep) => void

export class PlaybackController {
  private steps: PlaybackStep[]
  private currentIndex = 0
  private isPlaying = false
  private speed = 1
  private timer: ReturnType<typeof setTimeout> | null = null
  private listeners = new Set<StateListener>()
  private chapters: Chapter[] = []
  private startTime = 0

  constructor(steps: PlaybackStep[]) {
    this.steps = steps
  }

  public subscribe(listener: StateListener): () => void {
    this.listeners.add(listener)
    listener(this.getState(), this.currentStep)
    return () => this.listeners.delete(listener)
  }

  public get currentStep(): PlaybackStep {
    return this.steps[this.currentIndex] ?? this.steps[0]
  }

  public getState(): PlaybackControllerState {
    return {
      currentStepIndex: this.currentIndex,
      isPlaying: this.isPlaying,
      speed: this.speed,
      totalSteps: this.steps.length,
    }
  }

  public getChapters(): Chapter[] {
    return [...this.chapters]
  }

  public play(): void {
    if (this.isPlaying) return
    this.isPlaying = true
    if (this.chapters.length === 0) {
      this.startTime = performance.now()
      this.recordChapter(this.currentStep)
    }
    this.notify()
    this.scheduleNext()
  }

  public pause(): void {
    if (!this.isPlaying) return
    this.isPlaying = false
    this.clearTimer()
    this.notify()
  }

  public togglePlay(): void {
    if (this.isPlaying) {
      this.pause()
    } else {
      this.play()
    }
  }

  public setSpeed(multiplier: number): void {
    this.speed = multiplier
    this.notify()
    if (this.isPlaying) {
      this.clearTimer()
      this.scheduleNext()
    }
  }

  public next(): void {
    if (this.currentIndex < this.steps.length - 1) {
      this.seek(this.currentIndex + 1)
    } else {
      this.pause()
    }
  }

  public previous(): void {
    if (this.currentIndex > 0) {
      this.seek(this.currentIndex - 1)
    }
  }

  public seek(index: number): void {
    const target = Math.max(0, Math.min(index, this.steps.length - 1))
    this.currentIndex = target
    this.clearTimer()
    this.recordChapter(this.currentStep)
    this.notify()
    if (this.isPlaying) {
      if (this.currentIndex === this.steps.length - 1) {
        // Stop automatically after summary dwell
        const duration = getStepDuration(this.currentStep, this.speed)
        this.timer = setTimeout(() => this.pause(), duration)
      } else {
        this.scheduleNext()
      }
    }
  }

  private scheduleNext(): void {
    this.clearTimer()
    if (!this.isPlaying) return
    const duration = getStepDuration(this.currentStep, this.speed)
    this.timer = setTimeout(() => {
      if (this.currentIndex < this.steps.length - 1) {
        this.next()
      } else {
        this.pause()
      }
    }, duration)
  }

  private clearTimer(): void {
    if (this.timer !== null) {
      clearTimeout(this.timer)
      this.timer = null
    }
  }

  private recordChapter(step: PlaybackStep): void {
    if (step.type === "intro" || step.type === "day" || step.type === "waypoint" || step.type === "summary") {
      const elapsedSec = Math.floor((performance.now() - (this.startTime || performance.now())) / 1000)
      const label =
        step.type === "intro"
          ? step.title
          : step.type === "day"
            ? `Day ${step.day} · ${step.date}`
            : step.type === "waypoint"
              ? `${step.title} · ${step.place}`
              : "Summary"
      const last = this.chapters.at(-1)
      if (last && elapsedSec - last.timeSeconds < 5) {
        last.label = label
      } else {
        this.chapters.push({ label, timeSeconds: elapsedSec })
      }
    }
  }

  private notify(): void {
    const state = this.getState()
    const step = this.currentStep
    this.listeners.forEach((l) => l(state, step))
  }

  public destroy(): void {
    this.clearTimer()
    this.listeners.clear()
  }
}
