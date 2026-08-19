import type { Memento } from "../../data"

export type TicketVariant = "rail" | "metro" | "live" | "mountain" | "admission"

export const ticketVariants: TicketVariant[] = ["rail", "metro", "live", "mountain", "admission"]

function text(memento: Memento, key: string): string {
  const value = memento.kindData?.[key]
  return typeof value === "string" ? value : ""
}

export function ticketVariantFor(memento: Memento): TicketVariant {
  const explicit = text(memento, "ticket_variant")
  if (ticketVariants.includes(explicit as TicketVariant)) return explicit as TicketVariant

  if (memento.kind === "transit") {
    return text(memento, "mode") === "metro" ? "metro" : "rail"
  }

  return "admission"
}
