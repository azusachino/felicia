import js from "@eslint/js"
import globals from "globals"
import ts from "typescript-eslint"
import svelte from "eslint-plugin-svelte"

export default ts.config(
  { ignores: [] },
  js.configs.recommended,
  ...ts.configs.recommended,
  ...svelte.configs.recommended,
  {
    files: ["**/*.svelte", "**/*.svelte.ts"],
    languageOptions: {
      parserOptions: { parser: ts.parser, extraFileExtensions: [".svelte"] },
    },
  },
  {
    languageOptions: { globals: { ...globals.browser } },
    rules: { "svelte/no-reactive-functions": "off" },
  },
)
