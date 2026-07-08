// The design registry — the single source of truth for which front-of-house
// designs the demo can switch between. All designs are pure presentations over
// the same immutable fixtures (data.ts) and the same {journey, memento}
// contract (felicia:decision:presentation-agnostic-contract), so adding a new
// PM design = drop a `<Name>App.svelte` under web/src/<id>/ and add one entry
// here — no router surgery.
import type { Component } from 'svelte';
import type { L, Lang, Theme } from './data';
import V1App from './v1/V1App.svelte';
import V2App from './v2/V2App.svelte';
import V3App from './v3/V3App.svelte';

export interface Design {
  id: string;
  // URL hash that selects this design ('' = default). Deep-linkable + shareable.
  hash: string;
  // Switcher label, localized.
  label: L;
  component: Component<{ lang?: Lang; theme?: Theme }>;
}

// Ordered as they appear in the switcher. The first entry is the default
// (empty hash) — currently v1, the map reader (felicia:decision:map-first-landing).
export const designs: Design[] = [
  {
    id: 'v1',
    hash: '',
    label: { ja: '地図', en: 'Map', zh: '地图' },
    component: V1App as unknown as Design['component'],
  },
  {
    id: 'v2',
    hash: '#collection',
    label: { ja: 'コレクション', en: 'Collection', zh: '藏品' },
    component: V2App as unknown as Design['component'],
  },
  {
    id: 'v3',
    hash: '#techo',
    label: { ja: '手帳', en: 'Journal', zh: '手帐' },
    component: V3App as unknown as Design['component'],
  },
];

// Resolve the active design from a location hash, tolerating a couple of
// aliases (#v2/#v3) and falling back to the default.
export function designFromHash(hash: string): Design {
  const aliases: Record<string, string> = { '#v2': '#collection', '#v3': '#techo' };
  const normalized = aliases[hash] ?? hash;
  return designs.find(d => d.hash === normalized) ?? designs[0];
}
