export type ThemeCapability = "journey-scene" | "stop-events" | "artifact-reveal" | "memory-inspection" | "home-archive"

export interface ThemeManifest {
  id: string
  label: string
  capabilities: readonly ThemeCapability[]
}
