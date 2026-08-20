import type { Coordinates, Journey, Memento, Visit } from "../../data"

export interface JourneyStop {
  visit: Visit
  mementos: readonly Memento[]
}

export interface JourneyScene {
  journeyId: Journey["id"]
  origin: Coordinates
  destination: Coordinates
  route: readonly Coordinates[]
  stops: readonly JourneyStop[]
  returnHome: boolean
}
