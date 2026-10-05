import { readFileSync } from "node:fs"
import { createHash } from "node:crypto"
import { expect, test } from "vitest"

const fonts = new URL("../public/fonts/", import.meta.url)

test("reader typography is locally bundled with intact glyph subsets and licenses", () => {
  const css = readFileSync(new URL("reader-fonts.css", fonts), "utf8")
  const manifest = JSON.parse(readFileSync(new URL("sources.json", fonts), "utf8")) as { assets: { file: string; sha256: string }[] }
  expect(css).not.toMatch(/url\(\s*["']?https?:/)
  expect(css.match(/@font-face\s*\{/g)).toHaveLength(425)
  expect(css.match(/unicode-range:/g)).toHaveLength(425)
  expect(new Set([...css.matchAll(/font-family:\s*["']([^"']+)/g)].map((match) => match[1]))).toEqual(new Set(["Inter", "Outfit", "Share Tech Mono", "Spectral", "Zen Old Mincho"]))
  const referenced = new Set([...css.matchAll(/url\(["']?([^)'"\s]+)["']?\)/g)].map((match) => match[1]))
  expect(referenced).toEqual(new Set(manifest.assets.map((asset) => asset.file)))
  for (const asset of manifest.assets) {
    const bytes = readFileSync(new URL(asset.file, fonts))
    expect(bytes.subarray(0, 4).toString()).toBe("wOF2")
    expect(createHash("sha256").update(bytes).digest("hex")).toBe(asset.sha256)
  }
  for (const family of ["inter", "outfit", "sharetechmono", "spectral", "zenoldmincho"]) {
    expect(readFileSync(new URL(`${family}-OFL.txt`, fonts), "utf8")).toContain("SIL OPEN FONT LICENSE Version 1.1")
  }
  const html = readFileSync(new URL("../index.html", import.meta.url), "utf8")
  expect(html).toContain("%BASE_URL%fonts/reader-fonts.css")
  expect(html).not.toContain("fonts.googleapis.com")
  expect(html).not.toContain("fonts.gstatic.com")
})
