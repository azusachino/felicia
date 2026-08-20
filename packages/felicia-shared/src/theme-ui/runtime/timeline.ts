export type JourneyPhase =
  | "home-ready"
  | "departing"
  | "travelling"
  | "arriving-at-stop"
  | "revealing-memento"
  | "inspecting-memory"
  | "resuming"
  | "destination-reached"
  | "returning-home"
  | "home-archive"

export interface JourneyTimeline {
  phase: JourneyPhase
  currentStopId?: string
  currentMementoId?: string
}
