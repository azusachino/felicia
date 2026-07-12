import type { MementoKind } from '../data'

export interface StubTemplate {
  id: MementoKind
  label: string
}

// The registry is deliberately small: kind selects the visual form while the
// memento remains the shared data contract used by every V4 surface.
export const stubTemplates: Record<MementoKind, StubTemplate> = {
  transit: { id: 'transit', label: 'transit ticket' },
  stamp: { id: 'stamp', label: 'stamp' },
  goods: { id: 'goods', label: 'goods tag' },
  receipt: { id: 'receipt', label: 'receipt' },
  souvenir: { id: 'souvenir', label: 'souvenir card' },
}

export function templateFor(kind: string) {
  return stubTemplates[kind as MementoKind]
}
