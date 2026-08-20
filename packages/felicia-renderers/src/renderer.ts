import type { JourneyEvent, JourneyScene, SemanticAction } from "@felicia/runtime"

export interface JourneyRenderer {
  mount(host: HTMLElement): void
  render(scene: JourneyScene): void
  present(event: JourneyEvent): void
  play(action: SemanticAction): Promise<void>
  dispose(): void
}
