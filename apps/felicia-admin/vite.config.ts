import { defineConfig } from "vite"
import tailwindcss from "@tailwindcss/vite"
import adapter from "@sveltejs/adapter-static"
import { sveltekit } from "@sveltejs/kit/vite"

export default defineConfig({
  plugins: [
    sveltekit({
      adapter: adapter({ pages: "dist", assets: "dist", fallback: "index.html" }),
      router: { type: "hash" },
    }),
    tailwindcss(),
  ],
  server: {
    host: "0.0.0.0",
    port: 5174,
    allowedHosts: ["harus-macmini", "harus-mini"],
    // Same-origin dev against a local API server without CORS wiring
    // (apps/felicia-server/cmd/api doesn't set an allowed origin), mirroring how the
    // compiled artifact serves /api/* from the site origin:
    //   VITE_API_PROXY=http://127.0.0.1:8080 bun run dev
    // The admin-GUI E2E harness (scripts/e2e_admin_gui.py) relies on this.
    proxy: process.env.VITE_API_PROXY ? { "/api": { target: process.env.VITE_API_PROXY, changeOrigin: true } } : undefined,
  },
})
