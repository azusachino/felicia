import type { Coordinates } from "@felicia/model"

export interface TimelineEntry {
  coord: Coordinates
  dist: number // Cumulative distance in kilometers
  seg: number  // Segment index
}

const R_EARTH_KM = 6371.0088

export function haversineDistance(c1: Coordinates, c2: Coordinates): number {
  const [lon1, lat1] = c1
  const [lon2, lat2] = c2
  const dLat = ((lat2 - lat1) * Math.PI) / 180
  const dLon = ((lon2 - lon1) * Math.PI) / 180
  const a =
    Math.sin(dLat / 2) * Math.sin(dLat / 2) +
    Math.cos((lat1 * Math.PI) / 180) *
      Math.cos((lat2 * Math.PI) / 180) *
      Math.sin(dLon / 2) *
      Math.sin(dLon / 2)
  const c = 2 * Math.atan2(Math.sqrt(a), Math.sqrt(1 - a))
  return R_EARTH_KM * c
}

export function lerpCoord(a: Coordinates, b: Coordinates, t: number): Coordinates {
  return [a[0] + (b[0] - a[0]) * t, a[1] + (b[1] - a[1]) * t]
}

export function calculateBearing(from: Coordinates, to: Coordinates): number {
  const [lon1, lat1] = [from[0] * (Math.PI / 180), from[1] * (Math.PI / 180)]
  const [lon2, lat2] = [to[0] * (Math.PI / 180), to[1] * (Math.PI / 180)]
  const y = Math.sin(lon2 - lon1) * Math.cos(lat2)
  const x = Math.cos(lat1) * Math.sin(lat2) - Math.sin(lat1) * Math.cos(lat2) * Math.cos(lon2 - lon1)
  const deg = (Math.atan2(y, x) * 180) / Math.PI
  return (deg + 360) % 360
}

/**
 * Bounded Chaikin Corner-Cutting for GPS Polylines.
 * Smooths sharp GPS corners into gentle arcs while bounding cut distances to maxCornerM (default 30m)
 * so tracks never drift away from actual roads or highway curves.
 */
export function smoothSegmentCoords(
  coords: Coordinates[],
  iterations = 2,
  maxCornerM = 30,
  minEdgeM = 10
): Coordinates[] {
  if (!Array.isArray(coords) || coords.length < 3) return coords

  let current = coords
  for (let round = 0; round < iterations; round++) {
    const out: Coordinates[] = [current[0]]
    for (let i = 1; i < current.length - 1; i++) {
      const vertex = current[i]
      const prev = current[i - 1]
      const next = current[i + 1]
      const lenPrevM = haversineDistance(prev, vertex) * 1000
      const lenNextM = haversineDistance(vertex, next) * 1000

      if (lenPrevM < minEdgeM && lenNextM < minEdgeM) {
        out.push(vertex)
      } else {
        const ratioPrev = lenPrevM > 0 ? Math.min(0.25, maxCornerM / lenPrevM) : 0
        const ratioNext = lenNextM > 0 ? Math.min(0.25, maxCornerM / lenNextM) : 0
        out.push(lerpCoord(vertex, prev, ratioPrev))
        out.push(lerpCoord(vertex, next, ratioNext))
      }
    }
    out.push(current[current.length - 1])
    current = out
  }
  return current
}

export class RouteTimeline {
  public entries: TimelineEntry[] = []
  public totalDistance = 0

  constructor(segments: Coordinates[][], smooth = true) {
    let segIdx = 0
    let cumulative = 0

    for (const rawSegment of segments) {
      if (rawSegment.length < 2) continue
      const segment = smooth ? smoothSegmentCoords(rawSegment) : rawSegment

      for (let i = 0; i < segment.length; i++) {
        const coord = segment[i]
        if (i > 0) {
          cumulative += haversineDistance(segment[i - 1], coord)
        }
        this.entries.push({
          coord,
          dist: cumulative,
          seg: segIdx,
        })
      }
      segIdx++
    }
    this.totalDistance = cumulative
  }

  /**
   * Slice route up to a fraction [0, 1] of total distance.
   * Returns traveled coordinates, current vehicle head position, and forward bearing.
   */
  public sliceAtFraction(fraction: number): {
    traveled: Coordinates[]
    currentPos: Coordinates
    bearing: number
  } {
    if (this.entries.length === 0) {
      return { traveled: [], currentPos: [0, 0], bearing: 0 }
    }
    if (this.entries.length === 1) {
      return { traveled: [this.entries[0].coord], currentPos: this.entries[0].coord, bearing: 0 }
    }

    const clampedF = Math.max(0, Math.min(1, fraction))
    const targetDist = clampedF * this.totalDistance

    // Binary search for entry interval
    let low = 0
    let high = this.entries.length - 1
    while (low <= high) {
      const mid = (low + high) >> 1
      if (this.entries[mid].dist < targetDist) {
        low = mid + 1
      } else {
        high = mid - 1
      }
    }

    const idx = Math.max(1, Math.min(this.entries.length - 1, low))
    const prev = this.entries[idx - 1]
    const curr = this.entries[idx]

    const span = curr.dist - prev.dist
    const t = span > 0 ? (targetDist - prev.dist) / span : 0
    const currentPos = lerpCoord(prev.coord, curr.coord, Math.max(0, Math.min(1, t)))

    const traveled = this.entries.slice(0, idx).map((e) => e.coord)
    traveled.push(currentPos)

    // Calculate bearing slightly ahead (or from prev to currentPos)
    const bearing = calculateBearing(prev.coord, curr.coord)

    return {
      traveled,
      currentPos,
      bearing,
    }
  }

  /**
   * Find nearest timeline distance for a given geographic coordinate.
   */
  public distanceAtCoordinate(coord: Coordinates): number {
    if (this.entries.length === 0) return 0
    let bestDist = Infinity
    let bestTimelineDist = 0

    for (const entry of this.entries) {
      const d = haversineDistance(coord, entry.coord)
      if (d < bestDist) {
        bestDist = d
        bestTimelineDist = entry.dist
      }
    }
    return bestTimelineDist
  }
}
