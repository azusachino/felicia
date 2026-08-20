import type { Coordinates } from "../../data"

export type JourneyEvent =
  | { type: "journey-start"; origin: Coordinates }
  | { type: "travel"; from: Coordinates; to: Coordinates; stopId?: string }
  | { type: "arrive"; stopId: string }
  | { type: "reveal"; stopId: string; mementoId: string }
  | { type: "inspect"; stopId: string; mementoId: string }
  | { type: "resume"; stopId: string }
  | { type: "destination-reached"; destination: Coordinates }
  | { type: "return-home"; target: "origin" | "authored-home" }
  | { type: "archive" }
