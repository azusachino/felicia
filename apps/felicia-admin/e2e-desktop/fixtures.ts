import { test as base, expect } from "@playwright/test"
import { spawn } from "node:child_process"
import { randomBytes } from "node:crypto"
import { appendFileSync, mkdirSync, mkdtempSync } from "node:fs"
import { fileURLToPath } from "node:url"
import { join } from "node:path"

const repo = fileURLToPath(new URL("../../../", import.meta.url))
const scratch = fileURLToPath(new URL("../../../../../.tmp/felicia-desktop-redesign/runs/", import.meta.url))

type Studio = { url: string; root: string; token: string }

export const test = base.extend<{ studio: Studio }>({
  studio: async ({ browserName }, use) => {
    mkdirSync(scratch, { recursive: true })
    const root = mkdtempSync(join(scratch, `studio-${browserName}-`))
    const token = randomBytes(32).toString("hex")
    const child = spawn(
      join(repo, "bin/felicia-desktop-e2e"),
      ["--e2e-addr", "127.0.0.1:0", "--db", join(root, "felicia.sqlite"), "--media-root", join(root, "media"), "--public-dir", join(root, "site"), "--preview-addr", "127.0.0.1:0"],
      { env: { ...process.env, FELICIA_E2E_TOKEN: token }, stdio: ["ignore", "pipe", "pipe"] },
    )
    child.stderr.on("data", (data: Buffer) => appendFileSync(join(root, "stderr.log"), data))
    const exited = new Promise<void>((resolve) => {
      child.once("exit", () => resolve())
      child.once("error", () => resolve())
    })
    try {
      const url = await new Promise<string>((resolve, reject) => {
        let output = ""
        const timer = setTimeout(() => reject(new Error("Desktop test transport readiness timed out")), 15_000)
        child.once("error", reject)
        child.once("exit", (code) => reject(new Error(`Desktop exited before readiness: ${code}`)))
        child.stdout.on("data", (data: Buffer) => {
          appendFileSync(join(root, "stdout.log"), data)
          output += data.toString()
          const match = output.match(/FELICIA_E2E_READY=(http:\/\/127\.0\.0\.1:\d+)/)
          if (match) {
            clearTimeout(timer)
            resolve(match[1])
          }
        })
        exited.then(() => clearTimeout(timer))
      })
      const ready = await fetch(`${url}/api/admin/journeys`, { headers: { Cookie: `felicia-e2e=${token}` } })
      expect(ready.status).toBe(200)
      await use({ url, root, token })
    } finally {
      if (child.exitCode === null && child.signalCode === null) child.kill("SIGTERM")
      const timer = setTimeout(() => child.kill("SIGKILL"), 2_000)
      await exited
      clearTimeout(timer)
    }
  },
  baseURL: async ({ studio }, use) => {
    await use(studio.url)
  },
  storageState: async ({ studio }, use) => {
    await use({
      cookies: [
        {
          name: "felicia-e2e",
          value: studio.token,
          domain: "127.0.0.1",
          path: "/",
          expires: -1,
          httpOnly: true,
          secure: false,
          sameSite: "Strict",
        },
      ],
      origins: [],
    })
  },
})

export { expect }
