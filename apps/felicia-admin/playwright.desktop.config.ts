import { defineConfig, devices } from "@playwright/test"

export default defineConfig({
  testDir: "./e2e-desktop",
  timeout: 30_000,
  workers: 1,
  forbidOnly: Boolean(process.env.CI),
  retries: process.env.CI ? 2 : 0,
  failOnFlakyTests: Boolean(process.env.CI),
  reporter: "list",
  outputDir: "../../../../.tmp/felicia-desktop-redesign/test-results",
  use: {
    viewport: { width: 1100, height: 760 },
    locale: "en-US",
    trace: "retain-on-failure",
    screenshot: "only-on-failure",
  },
  projects: [
    { name: "chromium", use: { ...devices["Desktop Chrome"], viewport: { width: 1100, height: 760 } } },
    { name: "webkit", use: { ...devices["Desktop Safari"], viewport: { width: 1100, height: 760 } } },
  ],
})
