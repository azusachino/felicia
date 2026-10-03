import type { Reroute } from "@sveltejs/kit"

// Let SvelteKit own routing; reject malformed encoding before its decoder runs.
export const reroute: Reroute = ({ url }) => {
  try {
    decodeURIComponent(url.hash)
  } catch (error) {
    if (error instanceof URIError) return "/"
    throw error
  }
}
