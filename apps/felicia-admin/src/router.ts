// URL construction only. Matching, parameters and history belong to
// SvelteKit; pages live under src/routes/.
export function journeyDetailPath(id: string): `/journey/${string}` {
  return `/journey/${encodeURIComponent(id)}`
}

export function journeyEditPath(id: string): `/journey/${string}/edit` {
  return `/journey/${encodeURIComponent(id)}/edit`
}

export function mementoCreatePath(id: string): `/journey/${string}/memento/new` {
  return `/journey/${encodeURIComponent(id)}/memento/new`
}

export function mementoEditPath(journeyId: string, mementoId: string): `/journey/${string}/memento/${string}` {
  return `/journey/${encodeURIComponent(journeyId)}/memento/${encodeURIComponent(mementoId)}`
}

// Unwired creation/import views still use hash URLs until their UI slice.
export function journeyDetailHash(id: string): string {
  return `#${journeyDetailPath(id)}`
}

export function mementoEditHash(journeyId: string, mementoId: string): string {
  return `#${mementoEditPath(journeyId, mementoId)}`
}

export const listHash = "#/"
export const siteHash = "#/site"
