import { getContext } from "svelte"
import type { Locale } from "../i18n"

export interface StudioState {
  readonly locale: Locale
  readonly desktop: boolean
}

export const STUDIO_CONTEXT = Symbol("studio")
export function getStudio(): StudioState {
  return getContext<StudioState>(STUDIO_CONTEXT)
}
