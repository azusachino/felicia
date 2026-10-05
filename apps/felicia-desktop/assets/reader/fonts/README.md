# Reader fonts

The reader self-hosts its existing Inter, Outfit, Share Tech Mono, Zen Old Mincho,
and Spectral typefaces. The owner selected bundling rather than changing the
paper-memento typography or retaining external font requests.

`reader-fonts.css` retains the families, styles, weights, display behavior and
Unicode subsets served by the two original Google Fonts stylesheet URLs recorded
in `sources.json`. Only font URLs are replaced with local filenames. The stylesheets
were fetched with a WOFF2-capable Chrome user agent on 2026-10-04. All 425 font-face
rules and 396 unique WOFF2 files are retained, including Japanese subsets: do not
silently prune glyph coverage. Font binaries total approximately 8.7 MB.

`sources.json` records each original Google-hosted font URL, local filename and
content SHA-256. The filenames are derived from the URL SHA-256; content hashes
verify the actual downloaded binaries. These provenance URLs are data, not
runtime requests. The HTML uses Vite's base-aware local stylesheet path and font
URLs are relative to that stylesheet, supporting both root and subpath output.

All five families use SIL Open Font License 1.1. Their unmodified copyright and
license files are included alongside the assets:

- `inter-OFL.txt`: <https://github.com/google/fonts/tree/main/ofl/inter>
- `outfit-OFL.txt`: <https://github.com/google/fonts/tree/main/ofl/outfit>
- `sharetechmono-OFL.txt`: <https://github.com/google/fonts/tree/main/ofl/sharetechmono>
- `zenoldmincho-OFL.txt`: <https://github.com/google/fonts/tree/main/ofl/zenoldmincho>
- `spectral-OFL.txt`: <https://github.com/google/fonts/tree/main/ofl/spectral>

Refresh deliberately from the recorded stylesheet URLs, retaining licenses,
checking WOFF2 signatures and content hashes, and reviewing family/weight/subset
changes. Verify publication and the embedded desktop reader without external
font requests before accepting a refresh. No runtime download or new package
dependency is needed.
