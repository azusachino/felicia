import { defineConfig } from "vite"
import tailwindcss from "@tailwindcss/vite"
import { svelte } from "@sveltejs/vite-plugin-svelte"

export default defineConfig({
  plugins: [svelte(), tailwindcss()],
  server: { host: "0.0.0.0", port: 5174 },
})
