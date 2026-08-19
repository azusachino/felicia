// v3 (techo) — a hand-placed, roughly-proportional lon/lat sketch
// projection helper for the stylized landing map.
import type { Coordinates } from "../data"

// Stylized bounding box for the full Japan sketch map. Include Okinawa rather
// than using a mainland-only range, which projects southern points outside.
const LON_RANGE: [number, number] = [122, 146]
const LAT_RANGE: [number, number] = [24, 46]
const MAP_PADDING = 4

export function project([lon, lat]: Coordinates): { x: number; y: number } {
  const rawX = ((lon - LON_RANGE[0]) / (LON_RANGE[1] - LON_RANGE[0])) * 100
  const rawY = (1 - (lat - LAT_RANGE[0]) / (LAT_RANGE[1] - LAT_RANGE[0])) * 100
  const x = MAP_PADDING + (rawX / 100) * (100 - MAP_PADDING * 2)
  const y = MAP_PADDING + (rawY / 100) * (100 - MAP_PADDING * 2)
  return { x, y }
}
