// Standalone built-reader font/static-subpath regression (see
// playwright.reader-fonts.config.ts). Proves that the production public-site
// build served under a non-root base path still downloads and actually loads
// every pinned reader typeface. Font coverage only: publication content, the
// API contract and native Wails rendering are out of scope.
import { expect, test } from "@playwright/test"

// Same declared face/weight/style inventory as
// apps/felicia-public-site/src/fonts.test.ts — expected families are declared
// here, never derived from observed requests under test.
const combos = [
  ...["300", "400", "500", "600"].map((weight) => ({ family: "Inter", weight, style: "normal", text: "Kyoto" })),
  ...["400", "500", "600", "700", "800"].map((weight) => ({ family: "Outfit", weight, style: "normal", text: "Kyoto" })),
  { family: "Share Tech Mono", weight: "400", style: "normal", text: "KYOTO" },
  { family: "Spectral", weight: "400", style: "normal", text: "Kyoto" },
  { family: "Spectral", weight: "600", style: "normal", text: "Kyoto" },
  { family: "Spectral", weight: "400", style: "italic", text: "Kyoto" },
  { family: "Spectral", weight: "500", style: "italic", text: "Kyoto" },
  ...["400", "700", "900"].map((weight) => ({ family: "Zen Old Mincho", weight, style: "normal", text: "京都の旅" })),
]

// Hard guard against silent inventory drift: the list above must stay the
// exact 17-face inventory of apps/felicia-public-site/src/fonts.test.ts.
expect(combos).toHaveLength(17)

const origin = "http://127.0.0.1:4179"

test("built reader under the static subpath downloads and loads all 17 pinned font faces without external requests", async ({ page }) => {
  const basePath = process.env.READER_FONTS_SERVE_BASE ?? process.env.READER_FONTS_BUILD_BASE ?? "/reader-fonts/"
  const prefix = new URL(basePath, origin).pathname
  const external: string[] = []
  const fonts: { url: string; status: number; contentType: string }[] = []

  // Offline boundary: every non-loopback request (including third-party map
  // tiles) is aborted and must stay empty. No real journal or API is contacted.
  await page.route("**/*", async (route) => {
    const url = new URL(route.request().url())
    if (url.origin !== origin) {
      external.push(url.toString())
      await route.abort()
      return
    }
    await route.continue()
  })
  page.on("response", (response) => {
    if (new URL(response.url()).pathname.endsWith(".woff2")) fonts.push({ url: response.url(), status: response.status(), contentType: response.headers()["content-type"] ?? "" })
  })

  await page.goto(basePath)

  // Ask the browser for every declared combination and require the EXACT
  // declared face to come back loaded: font matching would happily fall back
  // to the nearest weight/style (e.g. Spectral 400 for a missing 600), so a
  // mere non-empty result proves nothing. The Japanese sample text forces the
  // Zen Old Mincho JP glyph subset; a fallback substitution fails here.
  const loaded = await page.evaluate(
    async (combos) =>
      Promise.all(
        combos.map(async ({ family, weight, style, text }) => {
          const faces = await document.fonts.load(`${style} ${weight} 16px "${family}"`, text)
          return faces.map((face) => ({ family: face.family.replace(/["']/g, ""), weight: String(face.weight), style: face.style, status: face.status }))
        }),
      ),
    combos,
  )
  for (const [index, faces] of loaded.entries()) {
    const combo = combos[index]
    expect(
      faces.some((face) => face.family === combo.family && face.weight === combo.weight && face.style === combo.style && face.status === "loaded"),
      `combo ${index} (${combo.style} ${combo.weight} "${combo.family}") must load the exact declared face; loaded faces: ${JSON.stringify(faces)}`,
    ).toBe(true)
  }

  // Downloaded font assets must be successful, served with the correct MIME
  // type, and live below exactly the configured base path — a missing or
  // wrongly-prefixed asset fails here, not just a request-count drift.
  expect(fonts.length).toBeGreaterThan(0)
  for (const font of fonts) {
    expect(font.status, font.url).toBe(200)
    expect(font.contentType, font.url).toContain("font/woff2")
    expect(new URL(font.url).pathname.startsWith(prefix), font.url).toBe(true)
  }

  expect(external, "no external font, map or network requests").toEqual([])
})
