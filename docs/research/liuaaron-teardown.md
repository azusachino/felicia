# liuaaron.com teardown — how the reference actually works

> Live-site analysis of [liuaaron.com](https://liuaaron.com/) ("Aaron's Waypoints"),
> 2026-06-12, via headless browser + bundle inspection. Complements the screenshots
> (`liuaaron-desktop.png`, `liuaaron-mobile.png`); this is what's under the hood.

## Method

Headless Chromium (playwright-cli): loaded the page, captured all network requests,
clicked a ticket open, screenshotted the detail view. Downloaded and grepped the Vite
JS chunks for the embedded data.

## Findings

### No backend at all

The site is a **fully static Vite SPA**. The only network traffic is Mapbox tiles/fonts,
Cloudflare RUM, and umami analytics — **no data API, no CMS, no images CDN of its own**.
Every journey is a **lazy-loaded JS chunk** (`assets/index-*.js`, one per trip plus
shared UI), so trip data ships inside the bundle.

### Routes

- Embedded GeoJSON `FeatureCollection` per trip, made of **many short `LineString`
  segments named per day** (`"Jeju walk (2026-01-09)"`, `"Jeju walk (2026-01-10)"`) —
  not one continuous line per journey.
- Coordinates are rounded to **4 decimal places** (~11 m) — keeps the chunk small and
  fuzzes precision.
- Map config: style `mapbox/dark-v11`, track color `#ff7b3a`, width `3`, opacity `0.8`.

### Tickets are components, not photos

Each ticket is a **hand-built React component recreating the physical ticket in
HTML/CSS**: the Jeju admission ticket has a tear-off stub area, perforation curve,
UNESCO logos, Korean payment labels (`카드발매`); receipts are rendered with
`react-barcode`. Ticket "data" is **hardcoded props**:

```js
venueNameKo: "성산일출봉", price: "5,000", date: "2026-01-10",
time: "12:36:54", ticketId: "A3-2025052500185", paymentMethod: "카드발매"
```

The venue photo sits *inside* the ticket art (`/images/seongsan-ilchulbong.png`).

### Detail view & shell

Clicking a ticket opens a **paper-toned panel**: the rendered ticket on top, the title,
a short *italic essay paragraph*, then a `PHOTOS` section with **polaroid-style**
framed images. The shell is a sidebar listing journeys (`Jeju`, `Hokkaido`) with
country + date range, each ticket row showing a **photo count badge**, and a
`Newest ⇄ Oldest` sort toggle.

## Implications for felicia

1. **Content model validated** — journey → tickets → essay + photos matches what we
   designed; the photo-count badge and sort toggle are cheap wins to keep.
2. **Stub rendering fork** — the reference's crispness comes from *rendered* tickets,
   not photographed ones. Decision (2026-06-12): **type-templates filled from OCR'd
   fields, photo fallback** — crisp + animatable without per-ticket coding
   (see design §6/§8).
3. **Route geometry** — the reference uses per-day segments (`MultiLineString`-ish),
   not one `LineString`. Multi-day trips genuinely have gaps (flights, sleep). Open
   question logged in design §8.
4. **Coordinate rounding** (~4 dp) is a nice privacy/size trick for the public API —
   pairs well with our EXIF-strip invariant.
5. Where the reference hardcodes chunks, felicia generates from Postgres — our A+E
   pipeline is the "CMS" the reference doesn't have.
