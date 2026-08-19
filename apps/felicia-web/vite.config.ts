import { defineConfig } from "vite"
import tailwindcss from "@tailwindcss/vite"
import { svelte } from "@sveltejs/vite-plugin-svelte"

export default defineConfig({
  plugins: [svelte(), tailwindcss()],
  server: {
    host: "0.0.0.0",
    port: 5173,
    allowedHosts: ["harus-macmini", "harus-mini"],
    proxy: {
      "/api": {
        target: process.env.VITE_API_PROXY || "http://127.0.0.1:8080",
        changeOrigin: true,
      },
    },
  },
})
