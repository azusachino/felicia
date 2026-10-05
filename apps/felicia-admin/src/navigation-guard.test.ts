import { beforeEach, describe, expect, it, vi, type Mock } from "vitest"
import type { ResolvedPathname } from "$app/types"

const mocks = vi.hoisted(() => ({ beforeNavigate: vi.fn(), goto: vi.fn() }))
vi.mock("$app/navigation", () => mocks)
import { guardedGoto, guardDirtyNavigation } from "./lib/navigation-guard"

type Navigation = { to: { url: URL } | null; willUnload: boolean; cancel: () => void }
const target = "/" as ResolvedPathname
let navigate: (navigation: Navigation) => void
let dirty: boolean
let pending: boolean
let answer: (discard: boolean) => void
let confirmDiscard: Mock<() => Promise<boolean>>

beforeEach(() => {
  vi.clearAllMocks()
  dirty = true
  pending = false
  confirmDiscard = vi.fn(
    () =>
      new Promise<boolean>((resolve) => {
        answer = resolve
      }),
  )
  mocks.beforeNavigate.mockImplementation((callback) => {
    navigate = callback
  })
  mocks.goto.mockImplementation(async (path: string) => {
    navigate({ to: { url: new URL(path, "https://example.invalid") }, willUnload: false, cancel: vi.fn() })
    // SvelteKit's cancelled goto resolves too; the dialog must own completion.
  })
  guardDirtyNavigation({ dirty: () => dirty, pending: () => pending, prompt: () => "Unsaved", confirmDiscard })
})

describe("dirty navigation continuation", () => {
  it("waits for a visible decision even when cancelled goto resolves", async () => {
    let settled = false
    const result = guardedGoto(target).then((allowed) => {
      settled = true
      return allowed
    })
    await vi.waitFor(() => expect(confirmDiscard).toHaveBeenCalledOnce())
    expect(settled).toBe(false)
    answer(true)
    expect(await result).toBe(true)
    expect(mocks.goto).toHaveBeenCalledTimes(2)
  })

  it("keeps the draft and refuses workspace switching on cancellation", async () => {
    const result = guardedGoto(target)
    await vi.waitFor(() => expect(confirmDiscard).toHaveBeenCalledOnce())
    answer(false)
    expect(await result).toBe(false)
    expect(mocks.goto).toHaveBeenCalledOnce()
  })

  it("does not resume if a write becomes pending while confirming", async () => {
    const result = guardedGoto(target)
    await vi.waitFor(() => expect(confirmDiscard).toHaveBeenCalledOnce())
    pending = true
    answer(true)
    expect(await result).toBe(false)
    expect(mocks.goto).toHaveBeenCalledOnce()
  })

  it("reports pending goto refusal even when SvelteKit resolves it", async () => {
    pending = true
    expect(await guardedGoto(target)).toBe(false)
    expect(confirmDiscard).not.toHaveBeenCalled()
  })

  it("blocks pending navigation and unload without an asynchronous prompt", () => {
    const cancel = vi.fn()
    pending = true
    navigate({ to: { url: new URL("https://example.invalid/") }, willUnload: false, cancel })
    pending = false
    navigate({ to: null, willUnload: true, cancel })
    expect(cancel).toHaveBeenCalledTimes(2)
    expect(confirmDiscard).not.toHaveBeenCalled()
  })
})
