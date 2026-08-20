import type { JourneyEvent } from "../runtime/events"
import type { JourneyScene } from "../runtime/scene"
import type { SemanticAction } from "../runtime/actions"

export interface JourneyRenderer {
  mount(host: HTMLElement): void
  render(scene: JourneyScene): void
  present(event: JourneyEvent): void
  play(action: SemanticAction): Promise<void>
  dispose(): void
}
