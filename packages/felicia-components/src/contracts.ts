import type { Memento } from "@felicia/model"
import type { JourneyScene, JourneyStop } from "@felicia/runtime"

export interface JourneyBoardProps {
  scene: JourneyScene
  selectedStopId?: string
}

export interface StopMarkerProps {
  stop: JourneyStop
  active: boolean
}

export interface MementoArtifactProps {
  memento: Memento
  revealed: boolean
}

export interface HomeArchiveProps {
  journeyId: JourneyScene["journeyId"]
  completedStopIds: readonly string[]
}
