import js from "@eslint/js"
import globals from "globals"
import ts from "typescript-eslint"
import svelte from "eslint-plugin-svelte"

export default ts.config(
  // e2e/ + playwright.config.ts run under the Playwright test runner (Node),
  // not this app's Vite/browser build — they're driven by
  // scripts/e2e_admin_gui.py (`make test-admin-e2e`), a local-only pass kept
  // out of `validate` per ADMIN-01.8, so they're kept out of this lint scope
  // too rather than stretching the browser-only `globals.browser` config
  // below to also cover Node globals like `process`.
  { ignores: ["dist/", "e2e/**", "playwright.config.ts", "playwright-report/**", "test-results/**"] },
  js.configs.recommended,
  ...ts.configs.recommended,
  ...svelte.configs.recommended,
  {
    files: ["**/*.svelte", "**/*.svelte.ts"],
    languageOptions: { parserOptions: { parser: ts.parser, extraFileExtensions: [".svelte"] } },
  },
  { languageOptions: { globals: { ...globals.browser } } },
)
