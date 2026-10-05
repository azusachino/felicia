# Reader font licenses

Font binaries are build dependencies, not repository assets. The public reader
imports the existing fixed family/weight/style faces from pinned Fontsource
packages in `src/fonts.css`; Bun's lockfile records their package integrity.
Vite bundles the CSS and fingerprints local font assets under `assets/`, including
all supplied Unicode subsets. The desktop embeds that same reader build.

The original Inter, Outfit, Share Tech Mono, Spectral and Zen Old Mincho CSS
family declarations remain unchanged. Fixed imports retain those family names,
weights and styles; Japanese Zen Old Mincho subsets are included. No Google
Fonts requests are needed at runtime. Map tiles remain an online dependency.

The five `*-OFL.txt` files are copied verbatim from each pinned package's
`LICENSE` and travel with every reader build. `src/fonts.test.ts` verifies the
imports, package versions, license equality, local WOFF2 signatures and Japanese
subset presence. The desktop browser test loads all five families with external
traffic blocked.

References:

- [Fontsource installation and fixed-face imports](https://fontsource.org/docs/getting-started/install)
- [Fontsource subsets](https://fontsource.org/docs/getting-started/subsets)
- [Vite asset bundling](https://vite.dev/guide/assets.html)

Local npm TLS failures may be worked around with the owner-approved
`https://npm.okcoin.tokyo/` mirror. Do not disable TLS verification or downgrade
unrelated locked packages. No persistent registry setting is required.
