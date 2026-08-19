// The design-language registry — the single source of truth for the reader
// compositions available to every Felicia reader host.
import type { Component } from "svelte"
import type { Journey, Lang } from "../data"
import type { MessageKey } from "../i18n/catalog"
import type { Theme } from "./themes"
import Cartography from "./cartography/Cartography.svelte"
import Cabinet from "./cabinet/Cabinet.svelte"
import Techo from "./techo/Techo.svelte"
import Atlas from "./atlas/Atlas.svelte"

export type DesignId = "cartography" | "cabinet" | "techo" | "atlas"

export interface DesignLanguage {
  id: DesignId
  // URL hash that selects this design ('' = default). Deep-linkable + shareable.
  hash: string
  // Static system-locale key; never derived from user content.
  labelKey: MessageKey
  component: Component<{ lang?: Lang; theme?: Theme; loadJourneys: () => Promise<Journey[]> }>
}

// Ordered as they appear in the switcher. The first entry is the default
// (empty hash): Cartography, the map reader (felicia:decision:map-first-landing).
export const designLanguages: DesignLanguage[] = [
  {
    id: "cartography",
    hash: "",
    labelKey: "design.cartography",
    component: Cartography as unknown as DesignLanguage["component"],
  },
  {
    id: "cabinet",
    hash: "#cabinet",
    labelKey: "design.cabinet",
    component: Cabinet as unknown as DesignLanguage["component"],
  },
  {
    id: "techo",
    hash: "#techo",
    labelKey: "design.techo",
    component: Techo as unknown as DesignLanguage["component"],
  },
  {
    id: "atlas",
    hash: "#atlas",
    labelKey: "design.atlas",
    component: Atlas as unknown as DesignLanguage["component"],
  },
]

export function designLanguageFromHash(hash: string): DesignLanguage {
  const designHash = hash.split("/")[0]
  return designLanguages.find((design) => design.hash === designHash) ?? designLanguages[0]
}

export function designLanguageFromId(id: string | undefined): DesignLanguage {
  return designLanguages.find((design) => design.id === id) ?? designLanguages[0]
}
