// Standalone built-reader font/static-subpath regression: builds the real
// production public-site bundle into a bounded workstation scratch directory
// (never apps/felicia-public-site/dist or embedded assets), previews it through
// loopback-only Vite, and runs the focused font spec. Not part of `make
// validate`; this is not native Wails visual acceptance and does not cover
// published journal content.
import { defineConfig, devices } from "@playwright/test"
import { join } from "node:path"

// Overridable only to demonstrate the expected-red negative receipt (a build
// made with the wrong base path served under the documented subpath must
// fail); normal runs build and serve the documented static subpath. Overrides
// are validated synchronously (before any build) against a strict allowlist:
// absolute slash-terminated subpaths of [A-Za-z0-9._-] segments, no traversal.
// They are then safe to place inside shell double quotes without expansion
// risk, because the allowlist forbids $, backticks, quotes and whitespace.
function validatedBase(name: string): string {
  const value = process.env[name]
  if (value === undefined) return "/reader-fonts/"
  if (!/^\/(?:[A-Za-z0-9._-]+\/)+$/.test(value) || value.includes(".."))
    throw new Error(`Invalid ${name}: must be an absolute slash-terminated subpath of [A-Za-z0-9._-] segments without "..", got: ${value}`)
  return value
}
const buildBase = validatedBase("READER_FONTS_BUILD_BASE")
const serveBase = process.env.READER_FONTS_SERVE_BASE === undefined ? buildBase : validatedBase("READER_FONTS_SERVE_BASE")
const scratch = join(import.meta.dirname, "../../../../.tmp/felicia-desktop-polish/reader-fonts")
const outDir = `${scratch}/site`
const quotedOutDir = `'${outDir.replace(/'/g, "'\\''")}'`
const port = 4179

export default defineConfig({
  testDir: "./e2e-reader-fonts",
  timeout: 60_000,
  workers: 1,
  forbidOnly: Boolean(process.env.CI),
  retries: 0,
  reporter: "list",
  outputDir: `${scratch}/test-results`,
  use: {
    baseURL: `http://127.0.0.1:${port}`,
    viewport: { width: 1100, height: 760 },
    trace: "retain-on-failure",
  },
  webServer: {
    // One serialized command: production build with the non-root base path,
    // then Vite preview bound to loopback. Playwright starts and cleans it up.
    // buildBase/serveBase are allowlist-validated above; outDir is wrapped in
    // escaped shell single quotes so a checkout path containing shell characters
    // cannot word-split or expand. Vite comes from the workspace's installed
    // dependency, never an implicit bun x download.
    command: `cd ../felicia-public-site && ./node_modules/.bin/vite build --base "${buildBase}" --outDir ${quotedOutDir} --emptyOutDir && ./node_modules/.bin/vite preview --base "${serveBase}" --outDir ${quotedOutDir} --host 127.0.0.1 --port ${port} --strictPort`,
    url: `http://127.0.0.1:${port}${serveBase}`,
    // "false" by default so every normal run rebuilds from source. The
    // READER_FONTS_REUSE_SERVER escape hatch exists solely for the negative
    // receipt: it lets the spec run against an already-built, hand-mutated
    // scratch bundle (e.g. built CSS missing one face) without the webServer
    // build overwriting the mutation first. Never set it for normal runs.
    reuseExistingServer: process.env.READER_FONTS_REUSE_SERVER === "1",
    stdout: "pipe",
    stderr: "pipe",
  },
  projects: [
    { name: "chromium", use: { ...devices["Desktop Chrome"], viewport: { width: 1100, height: 760 } } },
    { name: "webkit", use: { ...devices["Desktop Safari"], viewport: { width: 1100, height: 760 } } },
  ],
})
