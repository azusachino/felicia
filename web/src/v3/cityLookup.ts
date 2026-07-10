// v3 (techo) — a hand-placed, roughly-proportional lon/lat sketch
// projection helper for the stylized landing map.
import type { Coordinates } from '../data'

// Stylized bounding box for the sketch map (lon min/max, lat min/max), padded
// beyond the actual dot range so labels don't clip at the panel edges.
const LON_RANGE: [number, number] = [128.5, 144.5]
const LAT_RANGE: [number, number] = [31.5, 45.0]

export function project([lon, lat]: Coordinates): { x: number; y: number } {
  const x = ((lon - LON_RANGE[0]) / (LON_RANGE[1] - LON_RANGE[0])) * 100
  const y = (1 - (lat - LAT_RANGE[0]) / (LAT_RANGE[1] - LAT_RANGE[0])) * 100
  return { x, y }
}
