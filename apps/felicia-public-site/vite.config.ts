import { defineConfig } from "vite"
import tailwindcss from "@tailwindcss/vite"
import { svelte } from "@sveltejs/vite-plugin-svelte"

// https://vite.dev/config/
export default defineConfig({
  plugins: [svelte(), tailwindcss()],
  base: process.env.BASE_PATH ?? "/",
  build: {
    chunkSizeWarningLimit: 2200,
  },
  server: {
    host: "0.0.0.0",
    allowedHosts: ["harus-mini"],
    proxy: {
      "/api/v1": {
        target: "http://localhost:8080",
        changeOrigin: true,
      },
    },
  },
})
