import type { Memento } from "../../data"
import type { JourneyScene, JourneyStop } from "../runtime/scene"

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
