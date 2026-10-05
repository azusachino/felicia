import { readFileSync, readdirSync } from "node:fs"
import { createRequire } from "node:module"
import { dirname, join } from "node:path"
import { expect, test } from "vitest"

const require = createRequire(import.meta.url)
const faces = {
  inter: ["300", "400", "500", "600"],
  outfit: ["400", "500", "600", "700", "800"],
  "share-tech-mono": ["400"],
  spectral: ["400", "600", "400-italic", "500-italic"],
  "zen-old-mincho": ["400", "700", "900"],
}
const licenseNames = { inter: "inter", outfit: "outfit", "share-tech-mono": "sharetechmono", spectral: "spectral", "zen-old-mincho": "zenoldmincho" }

test("reader fonts use pinned Fontsource dependencies, local glyph subsets and matching licenses", () => {
  const css = readFileSync(new URL("./fonts.css", import.meta.url), "utf8")
  const imports = [...css.matchAll(/@import\s+"([^"]+)"/g)].map((match) => match[1])
  const expected = Object.entries(faces).flatMap(([family, weights]) => weights.map((weight) => `@fontsource/${family}/${weight}.css`))
  expect(new Set(imports)).toEqual(new Set(expected))
  const app = JSON.parse(readFileSync(new URL("../package.json", import.meta.url), "utf8")) as { dependencies: Record<string, string> }
  for (const [family, licenseName] of Object.entries(licenseNames)) {
    const packageDir = dirname(require.resolve(`@fontsource/${family}/400.css`))
    const pkg = JSON.parse(readFileSync(join(packageDir, "package.json"), "utf8")) as { version: string; license: string }
    expect(app.dependencies[`@fontsource/${family}`]).toMatch(/^\d+\.\d+\.\d+$/)
    expect(pkg.version).toBe(app.dependencies[`@fontsource/${family}`])
    expect(pkg.license).toBe("OFL-1.1")
    const license = readFileSync(join(packageDir, "LICENSE"), "utf8")
    expect(license).toContain("SIL OPEN FONT LICENSE Version 1.1")
    expect(readFileSync(new URL(`../public/fonts/${licenseName}-OFL.txt`, import.meta.url), "utf8")).toBe(license)
  }
  for (const entry of imports) {
    const path = require.resolve(entry)
    const face = readFileSync(path, "utf8")
    expect(face).toContain("@font-face")
    expect(face).toContain("unicode-range:")
    expect(face).not.toMatch(/url\(\s*["']?https?:/)
    const assets = [...face.matchAll(/url\(\s*["']?([^)'"\s]+\.woff2)["']?\s*\)/g)].map((match) => match[1])
    expect(assets.length).toBeGreaterThan(0)
    for (const asset of assets)
      expect(
        readFileSync(join(dirname(path), asset))
          .subarray(0, 4)
          .toString(),
      ).toBe("wOF2")
  }
  const japanese = readFileSync(require.resolve("@fontsource/zen-old-mincho/400.css"), "utf8")
  const ranges = [...japanese.matchAll(/unicode-range:\s*([^;]+)/g)]
    .flatMap((match) => match[1].split(","))
    .map((range) => {
      const bounds = /^U\+([0-9a-f]+)(?:-([0-9a-f]+))?$/i.exec(range.trim())
      if (!bounds) throw new Error(`Unsupported Unicode range: ${range}`)
      return [parseInt(bounds[1], 16), parseInt(bounds[2] ?? bounds[1], 16)]
    })
  for (const glyph of "京都の旅") {
    const code = glyph.codePointAt(0)!
    expect(ranges.some(([first, last]) => first <= code && code <= last)).toBe(true)
  }
  expect(readdirSync(new URL("../public/fonts/", import.meta.url)).filter((name) => /\.woff2?$/.test(name))).toEqual([])
  expect(readFileSync(new URL("./main.ts", import.meta.url), "utf8")).toContain('import "./fonts.css"')
  const html = readFileSync(new URL("../index.html", import.meta.url), "utf8")
  expect(html).not.toContain("fonts.googleapis.com")
  expect(html).not.toContain("fonts.gstatic.com")
  expect(html).not.toContain("reader-fonts.css")
})
