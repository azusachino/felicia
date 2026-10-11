import type { Coordinates, Journey, Memento, PhotoAsset, WeatherInfo } from "@felicia/model"

export type StepType = "intro" | "day" | "stop" | "summary"

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

export interface StopStep {
  type: "stop"
  day: number
  stopIndex: number
  memento: Memento
  coords: Coordinates
  anchor: string
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

export type PlaybackStep = IntroStep | DayStep | StopStep | SummaryStep

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
    baseMs = 2400
  } else if (step.type === "stop") {
    const photoCount = step.memento.photos.length
    const textLen = (step.memento.essay.zh || step.memento.essay.ja || step.memento.essay.en || "").length
    baseMs = Math.max(5000, photoCount * 2000, textLen * 120)
  } else if (step.type === "summary") {
    baseMs = 4500
  }
  return Math.round(baseMs / speed)
}

function parseDateKey(dateStr: string): string {
  return dateStr.slice(0, 10)
}

export function generatePlaybackSteps(journey: Journey, lang: "ja" | "en" | "zh" = "zh"): PlaybackStep[] {
  const mementos = [...journey.mementos]
  const allPhotos = mementos.flatMap((m) => m.photos)
  const uniquePlaces = [...new Set(mementos.map((m) => m.place[lang] || m.place.zh || m.place.en))]

  // Group mementos by distinct date
  const dateGroups: { date: string; mementos: Memento[] }[] = []
  for (const m of mementos) {
    const d = parseDateKey(m.date.en || m.date.zh || m.date.ja || "")
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
    title: journey.title[lang] || journey.title.zh || journey.title.ja,
    dateRange: journey.dates[lang] || journey.dates.zh || journey.dates.ja,
    dayCount,
    stopCount: mementos.length,
    photoCount: allPhotos.length,
    distanceKm: journey.distanceKm,
    drivingHours: journey.drivingHours,
    covers,
  }

  const steps: PlaybackStep[] = [intro]
  let globalStopIndex = 0

  dateGroups.forEach((group, dayIdx) => {
    const dayNumber = dayIdx + 1
    const dayPlaces = [...new Set(group.mementos.map((m) => m.place[lang] || m.place.zh || m.place.en))]
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

    for (const memento of group.mementos) {
      steps.push({
        type: "stop",
        day: dayNumber,
        stopIndex: globalStopIndex,
        memento,
        coords: memento.coords,
        anchor: memento.id,
      })
      globalStopIndex++
    }
  })

  const summary: SummaryStep = {
    type: "summary",
    title: journey.title[lang] || journey.title.zh || journey.title.ja,
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
    if (step.type === "intro" || step.type === "day" || step.type === "summary") {
      const elapsedSec = Math.floor((performance.now() - (this.startTime || performance.now())) / 1000)
      const label =
        step.type === "intro"
          ? step.title
          : step.type === "day"
            ? `第 ${step.day} 天 · ${step.date}`
            : "旅程回顾"
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
