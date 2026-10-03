// URL construction only. Matching, parameters and history belong to
// svelte-spa-router; the route table lives in routes.ts.
export function journeyDetailHash(id: string): string {
  return `#/journey/${encodeURIComponent(id)}`
}

export function mementoEditHash(journeyId: string, mementoId: string): string {
  return `#/journey/${encodeURIComponent(journeyId)}/memento/${encodeURIComponent(mementoId)}`
}

export const listHash = "#/"
export const siteHash = "#/site"
