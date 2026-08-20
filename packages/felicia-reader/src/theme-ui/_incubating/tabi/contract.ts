import type { CharacterAction, SceneAction, ThemeManifest } from "@felicia/runtime"

export const tabiManifest = {
  id: "tabi",
  label: "Tabi",
  capabilities: ["journey-scene", "stop-events", "artifact-reveal", "memory-inspection", "home-archive"],
} as const satisfies ThemeManifest

export type TabiModelId = "traveller" | "board" | "stop" | "artifact" | "archive"

export interface TabiModelDefinition {
  id: TabiModelId
  actions: readonly (CharacterAction | SceneAction)[]
}

export const tabiModelDefinitions = [
  { id: "traveller", actions: ["idle", "travel", "arrive", "celebrate"] },
  { id: "board", actions: ["depart", "resume", "return-home", "archive"] },
  { id: "stop", actions: ["arrive", "reveal", "inspect"] },
  { id: "artifact", actions: ["reveal", "inspect", "carry", "place"] },
  { id: "archive", actions: ["archive"] },
] as const satisfies readonly TabiModelDefinition[]
