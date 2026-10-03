// SvelteKit decodes the initial hash before calling reroute. Normalize only
// malformed encoding here; matching and navigation remain framework-owned.
export function init() {
  try {
    decodeURIComponent(location.hash)
  } catch (error) {
    if (!(error instanceof URIError)) throw error
    const url = new URL(location.href)
    url.hash = "#/"
    history.replaceState(history.state, "", url)
  }
}
