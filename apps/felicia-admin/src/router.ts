// Hash-based routing, the same idiom as the reader theme-ui registry: no router
// dependency, a pure function resolves state from `location.hash`, and the
// shell re-derives on the `hashchange` event so deep links and back/forward
// both work for free.

export type Route = { name: "list" } | { name: "detail"; id: string } | { name: "memento"; journeyId: string; id: string } | { name: "site" }

const LIST: Route = { name: "list" }

// Accepts "", "#", "#/", "#/journey/{id}", and "#/journey/{id}/memento/{id}"
// (with or without a trailing slash). Anything else falls back to the list
// route rather than erroring, since a stale or hand-edited hash shouldn't
// strand the user on a blank page.
export function parseRoute(hash: string): Route {
  const trimmed = hash.replace(/^#/, "").replace(/\/+$/, "")
  if (trimmed === "" || trimmed === "/") return LIST
  if (trimmed === "/site") return { name: "site" }

  const mementoMatch = /^\/journey\/([^/]+)\/memento\/([^/]+)$/.exec(trimmed)
  if (mementoMatch && mementoMatch[1] && mementoMatch[2]) {
    return { name: "memento", journeyId: decodeURIComponent(mementoMatch[1]), id: decodeURIComponent(mementoMatch[2]) }
  }

  const match = /^\/journey\/([^/]+)$/.exec(trimmed)
  if (match && match[1]) return { name: "detail", id: decodeURIComponent(match[1]) }

  return LIST
}

export function journeyDetailHash(id: string): string {
  return `#/journey/${encodeURIComponent(id)}`
}

export function mementoEditHash(journeyId: string, mementoId: string): string {
  return `#/journey/${encodeURIComponent(journeyId)}/memento/${encodeURIComponent(mementoId)}`
}

export const listHash = "#/"

export const siteHash = "#/site"
