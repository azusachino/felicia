import type { HomeArchiveProps, JourneyBoardProps, MementoArtifactProps, StopMarkerProps } from "@felicia/components"

export interface TabiCompositeModel {
  board: JourneyBoardProps
  stopMarkers: readonly StopMarkerProps[]
  artifact?: MementoArtifactProps
  archive: HomeArchiveProps
}
