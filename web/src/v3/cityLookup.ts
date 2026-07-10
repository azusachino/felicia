// v3 (techo) — a small romaji city lookup for the stylized landing map. The
// map is a hand-placed, roughly-proportional lon/lat sketch (not a real
// projection) so a fixed dot list keeps it simple and legible, echoing the
// coordinates already present in ../data (stations + memento coords).
import type { Coordinates } from '../data'

export interface CityDot {
  id: string
  journeyId: string
  label: string
  coords: Coordinates
}

export const cityDots: CityDot[] = [
  { id: 'tokyo', journeyId: 'golden-route', label: 'TOKYO', coords: [139.7671, 35.6812] },
  { id: 'kyoto', journeyId: 'golden-route', label: 'KYOTO', coords: [135.7588, 34.9858] },
  { id: 'osaka', journeyId: 'golden-route', label: 'OSAKA', coords: [135.5013, 34.6687] },
  { id: 'sapporo', journeyId: 'hokkaido', label: 'SAPPORO', coords: [141.3545, 43.0618] },
  { id: 'otaru', journeyId: 'hokkaido', label: 'OTARU', coords: [141.0007, 43.1907] },
  { id: 'fukuoka', journeyId: 'kyushu', label: 'FUKUOKA', coords: [130.4207, 33.5904] },
  { id: 'kumamoto', journeyId: 'kyushu', label: 'KUMAMOTO', coords: [130.6889, 32.7845] },
]

// Stylized bounding box for the sketch map (lon min/max, lat min/max), padded
// beyond the actual dot range so labels don't clip at the panel edges.
const LON_RANGE: [number, number] = [128.5, 144.5]
const LAT_RANGE: [number, number] = [31.5, 45.0]

export function project([lon, lat]: Coordinates): { x: number; y: number } {
  const x = ((lon - LON_RANGE[0]) / (LON_RANGE[1] - LON_RANGE[0])) * 100
  const y = (1 - (lat - LAT_RANGE[0]) / (LAT_RANGE[1] - LAT_RANGE[0])) * 100
  return { x, y }
}
