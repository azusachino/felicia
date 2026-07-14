import js from "@eslint/js"
import globals from "globals"
import ts from "typescript-eslint"
import svelte from "eslint-plugin-svelte"

// Flat config. .svelte components are linted (previously ignored) using the
// svelte parser with typescript-eslint for <script lang="ts"> blocks.
export default ts.config(
  { ignores: ["dist/"] },
  js.configs.recommended,
  ...ts.configs.recommended,
  ...svelte.configs.recommended,
  {
    files: ["**/*.svelte", "**/*.svelte.ts"],
    languageOptions: {
      parserOptions: {
        parser: ts.parser,
        extraFileExtensions: [".svelte"],
      },
    },
  },
  {
    languageOptions: {
      globals: { ...globals.browser },
    },
    rules: {
      // The reader uses reactive closures ($: t = (v) => v[lang]) so template
      // calls re-render when `lang` changes; converting them to plain functions
      // (as this rule wants) would drop the reactive dependency and break
      // language switching. The rule's autofixer is also incompatible with
      // ESLint 10 (uses the removed SourceCode.isSpaceBetweenTokens).
      "svelte/no-reactive-functions": "off",
    },
  },
)
