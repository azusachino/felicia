import { goto } from "$app/navigation"
import { resolve } from "$app/paths"
import { page } from "$app/state"

// First leave the authoring page through its dirty/pending navigation guard.
// Never switch the backend beneath a page whose navigation was cancelled.
export async function prepareWorkspaceSwitch(): Promise<boolean> {
  try {
    await goto(resolve("/"))
  } catch (cause) {
    if (page.route.id !== "/") return false
    throw cause
  }
  return page.route.id === "/"
}
