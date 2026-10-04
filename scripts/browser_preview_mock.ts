// Local browser fixture for the public reader.
// Run `make browser-mock`, then start the public site with
// `VITE_API_BASE=http://127.0.0.1:8099 make web-dev`.

import { createServer } from "node:http"

const port = Number(process.env.BROWSER_MOCK_PORT ?? 8099)
const journey = {
  id: "browser-journey",
  journal_id: "browser-journal",
  slug: "browser-journey",
  title: "Izu summer line",
  place: "Izu",
  country: "Japan",
  region: "Shizuoka",
  date_start: "2026-08-01",
  date_end: "2026-08-03",
  gps_route: {
    type: "MultiLineString",
    coordinates: [
      [
        [138.98, 34.66],
        [139.1, 34.91],
        [139.2, 34.8],
      ],
    ],
  },
  authored_fields: [],
}

const mementos = [
  {
    id: "browser-ticket",
    journey_id: journey.id,
    kind: "ticket",
    seq: 1,
    occurred_at: "2026-08-02T11:43:22+09:00",
    occurred_tz: "Asia/Tokyo",
    title: "Omuroyama Lift Ticket",
    place: "Omuroyama",
    geom: { type: "Point", coordinates: [139.1, 34.91] },
    kind_data: { ticket_variant: "mountain", name: "Omuroyama Lift Ticket" },
    essay: "The paper stub outlived the weather.",
    photos: [],
  },
  {
    id: "browser-souvenir",
    journey_id: journey.id,
    kind: "souvenir",
    seq: 2,
    occurred_at: "2026-08-03T15:08:20+09:00",
    occurred_tz: "Asia/Tokyo",
    title: "Coast walk",
    place: "Shimoda",
    geom: { type: "Point", coordinates: [139.2, 34.8] },
    kind_data: { name: "Coast walk" },
    essay: "A final blue hour before the line turned inland.",
    photos: [],
  },
]

const routes: Record<string, unknown> = {
  "/api/v1/site.json": { title: "Felicia", description: "A map for the moments", design: "atlas", default_language: "en", default_theme: "dark", accent: "#ff9b72" },
  "/api/v1/journeys.json": [
    {
      ...journey,
      representative_dots: [
        [138.98, 34.66],
        [139.1, 34.91],
        [139.2, 34.8],
      ],
    },
  ],
  [`/api/v1/journeys/${journey.id}.json`]: journey,
  [`/api/v1/journeys/${journey.id}/mementos.json`]: mementos,
}

const server = createServer((request, response) => {
  const path = new URL(request.url ?? "/", `http://127.0.0.1:${port}`).pathname
  const body = Object.hasOwn(routes, path) ? routes[path] : undefined
  response.writeHead(body === undefined ? 404 : 200, { "content-type": "application/json", "access-control-allow-origin": "*" })
  response.end(JSON.stringify(body ?? { error: "not found" }))
})
server.listen(port, "127.0.0.1", () => {
  const address = server.address()
  const boundPort = address && typeof address !== "string" ? address.port : port
  console.log(`Felicia browser mock listening on http://127.0.0.1:${boundPort}`)
})
