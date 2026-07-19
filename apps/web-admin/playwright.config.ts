// ADMIN-01.8 closed-loop E2E config. Process lifecycle (the API server and
// this app's `bun run dev`) is owned entirely by scripts/e2e_admin_gui.py,
// which is why there is no `webServer` block here — it only points at
// whatever the Python harness already started (E2E_BASE_URL) and passes
// through the rest of that harness's context via env vars the spec reads
// directly (E2E_API_BASE, E2E_JOURNEY_ID, E2E_OUT_DIR).
import { defineConfig, devices } from "@playwright/test"

export default defineConfig({
  testDir: "./e2e",
  timeout: 30_000,
  retries: 0,
  reporter: "list",
  use: {
    baseURL: process.env.E2E_BASE_URL,
    trace: "retain-on-failure",
  },
  projects: [{ name: "chromium", use: { ...devices["Desktop Chrome"] } }],
})
