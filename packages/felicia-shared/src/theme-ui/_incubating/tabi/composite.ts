import type { HomeArchiveProps, JourneyBoardProps, MementoArtifactProps, StopMarkerProps } from "../../components/contracts"

export interface TabiCompositeModel {
  board: JourneyBoardProps
  stopMarkers: readonly StopMarkerProps[]
  artifact?: MementoArtifactProps
  archive: HomeArchiveProps
}
