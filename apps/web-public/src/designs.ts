// The design registry — the single source of truth for which front-of-house
// designs the demo can switch between. All designs are pure presentations over
// the same immutable fixtures (data.ts) and the same {journey, memento}
// contract (felicia:decision:presentation-agnostic-contract), so adding a new
// PM design = drop a `<Name>App.svelte` under web/src/<id>/ and add one entry
// here — no router surgery.
import type { Component } from "svelte"
import type { Lang, Theme } from "./data"
import type { MessageKey } from "./i18n/catalog"
import V1App from "./v1/V1App.svelte"
import V2App from "./v2/V2App.svelte"
import V3App from "./v3/V3App.svelte"
import V4App from "./v4/V4App.svelte"

export interface Design {
  id: string
  // URL hash that selects this design ('' = default). Deep-linkable + shareable.
  hash: string
  // Static system-locale key; never derived from user content.
  labelKey: MessageKey
  component: Component<{ lang?: Lang; theme?: Theme }>
}

// Ordered as they appear in the switcher. The first entry is the default
// (empty hash) — currently v1, the map reader (felicia:decision:map-first-landing).
export const designs: Design[] = [
  {
    id: "v1",
    hash: "",
    labelKey: "design.map",
    component: V1App as unknown as Design["component"],
  },
  {
    id: "v2",
    hash: "#collection",
    labelKey: "design.collection",
    component: V2App as unknown as Design["component"],
  },
  {
    id: "v3",
    hash: "#techo",
    labelKey: "design.journal",
    component: V3App as unknown as Design["component"],
  },
  {
    id: "v4",
    hash: "#atlas",
    labelKey: "design.atlas",
    component: V4App as unknown as Design["component"],
  },
]

// Resolve the active design from a location hash, tolerating a couple of
// aliases (#v2/#v3) and falling back to the default.
export function designFromHash(hash: string): Design {
  const aliases: Record<string, string> = { "#v2": "#collection", "#v3": "#techo" }
  const normalized = aliases[hash] ?? hash
  const designHash = normalized.split("/")[0]
  return designs.find((d) => d.hash === designHash) ?? designs[0]
}
