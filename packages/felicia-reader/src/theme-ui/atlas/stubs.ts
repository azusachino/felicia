import type { MementoKind } from "@felicia/model"

export interface StubTemplate {
  id: MementoKind
  label: string
}

// The registry is deliberately small: kind selects the visual form while the
// memento remains the shared data contract used by every Atlas surface.
export const stubTemplates: Record<MementoKind, StubTemplate> = {
  transit: { id: "transit", label: "transit ticket" },
  stamp: { id: "stamp", label: "stamp" },
  goods: { id: "goods", label: "goods tag" },
  receipt: { id: "receipt", label: "receipt" },
  souvenir: { id: "souvenir", label: "souvenir card" },
  live: { id: "live", label: "concert stub" },
  ticket: { id: "ticket", label: "admission ticket" },
}

export function templateFor(kind: string) {
  return stubTemplates[kind as MementoKind]
}
