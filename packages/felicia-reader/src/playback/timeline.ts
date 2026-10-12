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

export function shortestAngleDelta(fromDeg: number, toDeg: number): number {
  return ((toDeg - fromDeg + 540) % 360) - 180
}

export function lerpAngle(fromDeg: number, toDeg: number, t: number): number {
  const diff = shortestAngleDelta(fromDeg, toDeg)
  return (fromDeg + diff * t + 360) % 360
}

/** Hermite smoothstep easing curve */
export function smoothstep(t: number): number {
  const clamped = Math.max(0, Math.min(1, t))
  return clamped * clamped * (3 - 2 * clamped)
}

/** Cubic ease-in-out curve for cinematic acceleration and braking */
export function easeInOutCubic(t: number): number {
  const clamped = Math.max(0, Math.min(1, t))
  return clamped < 0.5 ? 4 * clamped * clamped * clamped : 1 - Math.pow(-2 * clamped + 2, 3) / 2
}

/**
 * Calculates intuitive leg driving duration proportional to distance.
 * Velocity: ~12 km/s of simulated travel (1 second per 12 km), with a 1.2s start/stop cushion.
 */
export function calculateLegDriveDurationMs(distanceKm: number): number {
  if (distanceKm <= 0.05) return 800
  return Math.round(1200 + (distanceKm / 12.0) * 1000)
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

  public get startCoord(): Coordinates {
    return this.entries[0]?.coord ?? [0, 0]
  }

  public get endCoord(): Coordinates {
    return this.entries.at(-1)?.coord ?? [0, 0]
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

    // Calculate forward-looking bearing using a look-ahead window along the same segment
    // (at least 300m or 3 points ahead) to avoid micro-jitter on sharp curves
    let lookAheadCoord = curr.coord
    const lookAheadTargetDist = targetDist + 0.35 // 350 meters ahead
    for (let k = idx; k < this.entries.length; k++) {
      if (this.entries[k].seg !== curr.seg) break
      lookAheadCoord = this.entries[k].coord
      if (this.entries[k].dist >= lookAheadTargetDist) break
    }

    const bearing = calculateBearing(currentPos, lookAheadCoord)

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

  /**
   * Slice route up to a fraction [0, 1] partitioned into distinct days.
   * Enables rendering different colors per day and setting past days translucent.
   */
  public sliceByDays(
    fraction: number,
    cutoffs: DayCutoff[]
  ): {
    day: number
    coords: Coordinates[]
  }[] {
    if (this.entries.length === 0) return []
    const clampedF = Math.max(0, Math.min(1, fraction))
    const currentDist = clampedF * this.totalDistance

    if (cutoffs.length === 0) {
      return [{ day: 1, coords: this.sliceAtFraction(fraction).traveled }]
    }

    const slices: { day: number; coords: Coordinates[] }[] = []

    for (const c of cutoffs) {
      if (currentDist < c.startDist) {
        // This day has not been reached yet
        continue
      }

      const legEndDist = Math.min(currentDist, c.endDist)
      if (legEndDist <= c.startDist) {
        continue
      }

      const dayCoords: Coordinates[] = []

      // Starting coordinate of this day leg
      if (c.startDist > 0) {
        const startState = this.sliceAtFraction(c.startDist / this.totalDistance)
        dayCoords.push(startState.currentPos)
      } else {
        dayCoords.push(this.entries[0].coord)
      }

      // Intermediate entries within this day's distance window
      for (const e of this.entries) {
        if (e.dist > c.startDist && e.dist < legEndDist) {
          dayCoords.push(e.coord)
        }
      }

      // Current vehicle head position for this day
      const endState = this.sliceAtFraction(legEndDist / this.totalDistance)
      dayCoords.push(endState.currentPos)

      if (dayCoords.length >= 2) {
        slices.push({ day: c.day, coords: dayCoords })
      }
    }

    return slices
  }
}

export interface DayStyle {
  day: number
  color: string       // Main road line color
  glowColor: string   // Atmospheric route aura
  glow: string        // Aura glow color
  pulse: string       // Radar beacon border color
  label: string
}

export const DAY_PALETTE: DayStyle[] = [
  { day: 1, color: "#39C5BB", glowColor: "rgba(57, 197, 187, 0.4)", glow: "rgba(57, 197, 187, 0.8)", pulse: "#5fe0d7", label: "Day 1" }, // Hatsune Miku Signature Teal
  { day: 2, color: "#FF5E98", glowColor: "rgba(255, 94, 152, 0.4)",  glow: "rgba(255, 94, 152, 0.8)", pulse: "#ff82b2", label: "Day 2" }, // Miku Magenta / Ribbon Pink
  { day: 3, color: "#a855f7", glowColor: "rgba(168, 85, 247, 0.4)", glow: "rgba(168, 85, 247, 0.8)", pulse: "#d8b4fe", label: "Day 3" }, // Lavender Purple
  { day: 4, color: "#10b981", glowColor: "rgba(16, 185, 129, 0.4)", glow: "rgba(16, 185, 129, 0.8)", pulse: "#6ee7b7", label: "Day 4" }, // Mint Emerald
  { day: 5, color: "#38bdf8", glowColor: "rgba(56, 189, 248, 0.4)", glow: "rgba(56, 189, 248, 0.8)", pulse: "#7dd3fc", label: "Day 5" }, // Electric Sky Cyan
  { day: 6, color: "#f59e0b", glowColor: "rgba(245, 158, 11, 0.4)", glow: "rgba(245, 158, 11, 0.8)", pulse: "#fcd34d", label: "Day 6" }, // Warm Amber Gold
  { day: 7, color: "#6366f1", glowColor: "rgba(99, 102, 241, 0.4)", glow: "rgba(99, 102, 241, 0.8)", pulse: "#a5b4fc", label: "Day 7" }, // Digital Indigo
]

export function getDayColor(dayNumber: number): DayStyle {
  const idx = Math.max(0, dayNumber - 1) % DAY_PALETTE.length
  return DAY_PALETTE[idx]
}

export interface DayCutoff {
  day: number
  startDist: number
  endDist: number
  cutoffFraction?: number
}

export function calculateDayCutoffs(
  mementos: { coords: Coordinates; date: any }[],
  timeline: RouteTimeline
): DayCutoff[] {
  if (timeline.totalDistance <= 0) return []

  function getDateStr(date: any): string {
    if (typeof date === "string") return date.slice(0, 10)
    if (date && typeof date === "object") {
      const s = date.en || date.ja || date.zh || ""
      return String(s).slice(0, 10)
    }
    return ""
  }

  // Group mementos by chronological distinct dates
  const dateGroups: { date: string; mementos: { coords: Coordinates; date: any }[] }[] = []
  for (const m of mementos) {
    const d = getDateStr(m.date)
    const last = dateGroups.at(-1)
    if (last && last.date === d) {
      last.mementos.push(m)
    } else {
      dateGroups.push({ date: d, mementos: [m] })
    }
  }

  if (dateGroups.length <= 1) {
    return [{ day: 1, startDist: 0, endDist: timeline.totalDistance }]
  }

  const cutoffs: DayCutoff[] = []
  let prevEnd = 0

  for (let i = 0; i < dateGroups.length; i++) {
    const dayNumber = i + 1
    if (i === dateGroups.length - 1) {
      cutoffs.push({ day: dayNumber, startDist: prevEnd, endDist: timeline.totalDistance })
    } else {
      const lastMemento = dateGroups[i].mementos.at(-1)!
      const nextMemento = dateGroups[i + 1].mementos[0]
      const distLast = timeline.distanceAtCoordinate(lastMemento.coords)
      const distNext = timeline.distanceAtCoordinate(nextMemento.coords)

      // Look for a segment boundary transition between distLast and distNext
      let endDist = (distLast + distNext) / 2
      for (const entry of timeline.entries) {
        if (entry.dist >= distLast && entry.dist <= distNext) {
          const nextEntry = timeline.entries[timeline.entries.indexOf(entry) + 1]
          if (nextEntry && nextEntry.seg !== entry.seg) {
            endDist = entry.dist
            break
          }
        }
      }

      cutoffs.push({
        day: dayNumber,
        startDist: prevEnd,
        endDist,
        cutoffFraction: timeline.totalDistance > 0 ? endDist / timeline.totalDistance : 0,
      })
      prevEnd = endDist
    }
  }

  // Populate cutoffFraction for last cutoff as well
  if (cutoffs.length > 0) {
    const lastCutoff = cutoffs.at(-1)!
    lastCutoff.cutoffFraction = timeline.totalDistance > 0 ? lastCutoff.endDist / timeline.totalDistance : 1.0
  }

  return cutoffs
}

export interface RouteSnapResult {
  coord: Coordinates
  originalCoord: Coordinates
  distanceKm: number
  snapped: boolean
}

/**
 * Projects a point onto a line segment [a, b].
 * Uses equirectangular flat approximation scaled by cos(midLatitude).
 */
export function projectPointToSegment(
  p: Coordinates,
  a: Coordinates,
  b: Coordinates
): { point: Coordinates; t: number } {
  const phi = ((a[1] + b[1]) / 2) * (Math.PI / 180)
  const cosPhi = Math.cos(phi)
  const vx = (b[0] - a[0]) * cosPhi
  const vy = b[1] - a[1]
  const ux = (p[0] - a[0]) * cosPhi
  const uy = p[1] - a[1]
  const lenSq = vx * vx + vy * vy

  if (lenSq < 1e-14) {
    return { point: [a[0], a[1]], t: 0 }
  }

  let t = (ux * vx + uy * vy) / lenSq
  t = Math.max(0, Math.min(1, t))
  return {
    point: [a[0] + t * (b[0] - a[0]), a[1] + t * (b[1] - a[1])],
    t,
  }
}

/**
 * Snaps a coordinate to the closest point along the route polyline segments.
 * Inspired by Dawarich's photo-track matching:
 * If the minimum orthogonal/endpoint distance to any route segment is within maxDistanceKm (default 1.0 km),
 * the point is snapped to the route so that markers sit directly on the road and playback connects seamlessly.
 * Otherwise, the original coordinate is retained (snapped = false).
 */
export function snapCoordinateToRoute(
  coord: Coordinates,
  segments: Coordinates[][],
  maxDistanceKm = 1.0
): RouteSnapResult {
  if (!segments || segments.length === 0) {
    return {
      coord: [coord[0], coord[1]],
      originalCoord: [coord[0], coord[1]],
      distanceKm: Infinity,
      snapped: false,
    }
  }

  let bestDist = Infinity
  let bestPoint: Coordinates = [coord[0], coord[1]]

  for (const segment of segments) {
    if (!segment || segment.length === 0) continue
    if (segment.length === 1) {
      const d = haversineDistance(coord, segment[0])
      if (d < bestDist) {
        bestDist = d
        bestPoint = segment[0]
      }
      continue
    }

    for (let i = 0; i < segment.length - 1; i++) {
      const a = segment[i]
      const b = segment[i + 1]
      const proj = projectPointToSegment(coord, a, b)
      const d = haversineDistance(coord, proj.point)
      if (d < bestDist) {
        bestDist = d
        bestPoint = proj.point
      }
    }
  }

  if (bestDist <= maxDistanceKm) {
    return {
      coord: bestPoint,
      originalCoord: [coord[0], coord[1]],
      distanceKm: bestDist,
      snapped: true,
    }
  }

  return {
    coord: [coord[0], coord[1]],
    originalCoord: [coord[0], coord[1]],
    distanceKm: bestDist,
    snapped: false,
  }
}

