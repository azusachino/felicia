import { beforeNavigate, goto } from "$app/navigation"
import type { ResolvedPathname } from "$app/types"

let decision: Promise<boolean> | null = null
let cancelled = false

// SvelteKit cancellation is synchronous; keep the visible dialog asynchronous
// and let programmatic callers (including workspace switches) await the result.
export async function guardedGoto(target: ResolvedPathname): Promise<boolean> {
  cancelled = false
  try {
    await goto(target)
    // SvelteKit also resolves goto() when beforeNavigate cancels it. The
    // authoring decision, not that initial promise, owns the continuation.
    if (decision) return decision
    return !cancelled
  } catch (cause) {
    if (decision) return decision
    // Pending or unloading navigation is deliberately refused by the guard.
    if (cause instanceof Error && /cancel/i.test(cause.message)) return false
    throw cause
  }
}

export function guardDirtyNavigation(options: {
  dirty: () => boolean
  pending: () => boolean
  allowed?: () => boolean
  prompt: () => string
  confirmDiscard: (prompt: string) => Promise<boolean>
}): void {
  let permittedTarget: string | null = null
  beforeNavigate((navigation) => {
    if (options.allowed?.()) return
    if (permittedTarget === navigation.to?.url.href) {
      permittedTarget = null
      return
    }
    if (!options.dirty() && !options.pending()) return
    navigation.cancel()
    cancelled = true
    if (options.pending() || navigation.willUnload || !navigation.to) return
    if (decision) return
    const target = navigation.to.url
    decision = options
      .confirmDiscard(options.prompt())
      .then(async (discard) => {
        if (!discard || options.pending()) return false
        permittedTarget = target.href
        // SvelteKit supplied this internal URL: its base prefix is already
        // resolved. Preserve query/hash without resolving that prefix twice.
        const destination = `${target.pathname}${target.search}${target.hash}` as ResolvedPathname
        await goto(destination)
        return true
      })
      .finally(() => {
        decision = null
      })
  })
}
