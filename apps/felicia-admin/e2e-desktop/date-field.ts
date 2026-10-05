import type { Locator } from "@playwright/test"

// Exercise the library's keyboard editing, not an invisible native-input proxy.
export async function setDate(field: Locator, value: string) {
  const parts = value.split("-")
  for (const [index, part] of ["year", "month", "day"].entries()) {
    const segment = field.locator(`[data-segment="${part}"]`)
    // Backspace removes one digit in Bits UI; clear the whole segment before
    // typing a replacement so a retained year prefix cannot alter the date.
    for (let digit = 0; digit < 4 && /^\d+$/.test((await segment.textContent()) ?? ""); digit++) {
      await segment.focus()
      await segment.press("Backspace")
    }
    await segment.focus()
    await segment.pressSequentially(parts[index])
  }
}
