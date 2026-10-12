import { describe, expect, test } from "vitest"
import {
  RouteTimeline,
  calculateBearing,
  haversineDistance,
  smoothSegmentCoords,
} from "./timeline"

describe("Route Timeline & GPS Smoothing", () => {
  test("haversineDistance computes accurate geographic distance", () => {
    // Tokyo Station to Shinagawa Station (~6.8 km)
    const tokyo: [number, number] = [139.7671, 35.6812]
    const shinagawa: [number, number] = [139.7387, 35.6284]
    const dist = haversineDistance(tokyo, shinagawa)
    expect(dist).toBeGreaterThan(6.0)
    expect(dist).toBeLessThan(7.5)
  })

  test("calculateBearing returns heading in degrees [0, 360)", () => {
    const p1: [number, number] = [139.0, 35.0]
    const pNorth: [number, number] = [139.0, 36.0]
    const pEast: [number, number] = [140.0, 35.0]
    expect(Math.round(calculateBearing(p1, pNorth))).toBe(0)
    expect(Math.round(calculateBearing(p1, pEast))).toBe(90)
  })

  test("smoothSegmentCoords smooths sharp polyline without drifting far from road", () => {
    // Right angle corner (139.0, 35.0) -> (139.0, 35.01) -> (139.01, 35.01)
    const raw: [number, number][] = [
      [139.0, 35.0],
      [139.0, 35.01],
      [139.01, 35.01],
    ]
    const smoothed = smoothSegmentCoords(raw, 2, 30)
    expect(smoothed.length).toBeGreaterThan(raw.length)
    expect(smoothed[0]).toEqual(raw[0])
    expect(smoothed.at(-1)).toEqual(raw.at(-1))
  })

  test("RouteTimeline handles distance-paced slicing and nearest coordinate lookup", () => {
    const segment1: [number, number][] = [
      [139.7043, 35.8159],
      [139.7186, 35.7975],
      [139.7274, 35.7637],
    ]
    const timeline = new RouteTimeline([segment1], false)
    expect(timeline.totalDistance).toBeGreaterThan(0)
    expect(timeline.entries.length).toBe(3)

    // Slice at 0%
    const start = timeline.sliceAtFraction(0)
    expect(start.currentPos[0]).toBeCloseTo(139.7043, 3)
    expect(start.currentPos[1]).toBeCloseTo(35.8159, 3)

    // Slice at 50%
    const mid = timeline.sliceAtFraction(0.5)
    expect(mid.traveled.length).toBeGreaterThanOrEqual(2)
    expect(mid.bearing).toBeGreaterThanOrEqual(0)
    expect(mid.bearing).toBeLessThan(360)

    // Slice at 100%
    const end = timeline.sliceAtFraction(1.0)
    expect(end.currentPos[0]).toBeCloseTo(139.7274, 3)
    expect(end.currentPos[1]).toBeCloseTo(35.7637, 3)

    // Coordinate lookup
    const foundDist = timeline.distanceAtCoordinate([139.7186, 35.7975])
    expect(foundDist).toBeCloseTo(timeline.entries[1].dist, 4)
  })
})
