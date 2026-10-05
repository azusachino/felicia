import { guardedGoto } from "$lib/navigation-guard"
import { resolve } from "$app/paths"
import { page } from "$app/state"

// First leave the authoring page through its dirty/pending navigation guard.
// Never switch the backend beneath a page whose navigation was cancelled.
export async function prepareWorkspaceSwitch(): Promise<boolean> {
  try {
    if (!(await guardedGoto(resolve("/")))) return false
  } catch (cause) {
    if (page.route.id !== "/") return false
    throw cause
  }
  return page.route.id === "/"
}
