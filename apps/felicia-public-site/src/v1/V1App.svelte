<script lang="ts">
  import maplibregl, { type StyleSpecification } from "maplibre-gl"
  import { onMount, tick } from "svelte"
  import { cubicOut } from "svelte/easing"
  import { crossfade, fade } from "svelte/transition"
  import { loadJourneys } from "../api/source"
  import { kindLabel, uiText, type Coordinates, type Journey, type L, type Lang, type Memento, type MementoKind, type Station, type Theme } from "../data"
  import { message, type MessageKey } from "../i18n/catalog"

  // v1 — the liuaaron-aligned map reader: journey index rail -> map hero ->
  // paper detail. The map is the index. Reached from v2 as the "more" view.
  export let lang: Lang = "ja"
  export let theme: Theme = "dark"
  export let toMemories: (() => void) | undefined = undefined

  let journeys: Journey[] = []
  let selectedJourneyId = ""
  let selected: Memento | undefined
  let isLoading = true
  let error: string | null = null
  let mapContainer: HTMLDivElement
  let map: maplibregl.Map | undefined
  // eslint-disable-next-line svelte/prefer-svelte-reactivity -- imperative maplibre marker cache, not reactive UI state
  const markers = new Map<string, maplibregl.Marker>()

  // Shared-element open: a transient "ghost" stands in for the clicked
  // (imperative, non-Svelte) map marker so a crossfade pair has something to
  // grow from — see selectMemento() / markerScreenPos() (ADR-0031).
  let ghostVisible = false
  let ghostKey = ""
  let ghostOrigin: { x: number; y: number } | undefined
  let ghostKind: MementoKind | undefined
  let detailHeadingEl: HTMLHeadingElement | undefined
  let prefersReducedMotion = false

  const [sendStub, receiveStub] = crossfade({
    duration: (d: number) => (prefersReducedMotion ? 0 : Math.min(500, 220 + Math.sqrt(d) * 8)),
    easing: cubicOut,
    fallback(node) {
      const opacity = +getComputedStyle(node).opacity || 1
      return {
        duration: prefersReducedMotion ? 0 : 260,
        easing: cubicOut,
        css: (t: number) => `opacity: ${t * opacity}`,
      }
    },
  })

  $: t = (value: L | MessageKey) => (typeof value === "string" ? message(lang, value) : value[lang])
  $: stationName = (s: Station) => (lang === "en" ? s.name : s.ja)

  // kind_data is the template registry's untyped bag (ADR-0006): read it
  // defensively so a memento authored with only its required field still
  // renders a deliberate stub rather than an empty box.
  $: kindText = (key: string): string => {
    const value = selected?.kindData?.[key]
    return typeof value === "string" ? value.trim() : ""
  }
  // `station` / `venue` fields carry `{ name, coords }` rather than a bare string.
  $: kindName = (key: string): string => {
    const value = selected?.kindData?.[key]
    if (typeof value === "string") return value.trim()
    if (typeof value === "object" && value !== null && !Array.isArray(value)) {
      const name = (value as Record<string, unknown>).name
      if (typeof name === "string") return name.trim()
    }
    return ""
  }
  // Receipt `items` is authored as free text; accept a list too.
  $: kindLines = (key: string): string[] => {
    const value = selected?.kindData?.[key]
    const raw: unknown[] = Array.isArray(value) ? value : kindText(key).split("\n")
    return raw.map((line) => (typeof line === "string" ? line.trim() : "")).filter(Boolean)
  }

  function linkHost(url: string): string {
    try {
      return new URL(url).host
    } catch {
      return ""
    }
  }
  $: selectedJourney = journeys.find((j) => j.id === selectedJourneyId) ?? journeys[0]
  $: countLabel = (n: number) => (lang === "en" ? `${n} mementos` : `${n}件`)
  $: liveMessage = selected ? `${t(kindLabel[selected.kind])}: ${t(selected.title)}` : ""
  $: ghostSeq = selectedJourney ? selectedJourney.mementos.findIndex((m) => m.id === ghostKey) + 1 : 0

  function routesGeoJson() {
    return {
      type: "FeatureCollection" as const,
      features: journeys.map((journey) => ({
        type: "Feature" as const,
        geometry: { type: "LineString" as const, coordinates: journey.route },
        properties: { journeyId: journey.id },
      })),
    }
  }

  function transitFeatures(journey: Journey) {
    return {
      type: "FeatureCollection" as const,
      features: journey.mementos
        .filter((memento) => memento.transit)
        .map((memento) => ({
          type: "Feature" as const,
          geometry: {
            type: "LineString" as const,
            coordinates: [memento.transit!.from.coords, memento.transit!.to.coords],
          },
          properties: { id: memento.id },
        })),
    }
  }

  const mapStyle: StyleSpecification = {
    version: 8,
    sources: {
      dark: {
        type: "raster",
        tiles: ["https://a.basemaps.cartocdn.com/dark_all/{z}/{x}/{y}.png"],
        tileSize: 256,
        attribution: "&copy; OpenStreetMap contributors &copy; CARTO",
      },
      light: {
        type: "raster",
        tiles: ["https://a.basemaps.cartocdn.com/light_all/{z}/{x}/{y}.png"],
        tileSize: 256,
        attribution: "&copy; OpenStreetMap contributors &copy; CARTO",
      },
    },
    layers: [
      { id: "base-dark", type: "raster", source: "dark" },
      { id: "base-light", type: "raster", source: "light", layout: { visibility: "none" } },
    ],
  }

  function boundsOf(coords: Coordinates[]) {
    if (!coords.length) return undefined
    const bounds = new maplibregl.LngLatBounds(coords[0], coords[0])
    for (const coord of coords) bounds.extend(coord)
    return bounds
  }

  const fitPadding = { top: 80, bottom: 80, left: 80, right: 460 }

  function fitAll() {
    if (!map) return
    const bounds = boundsOf(journeys.flatMap((journey) => journey.route))
    if (!bounds) return
    map.fitBounds(bounds, {
      padding: fitPadding,
      maxZoom: 6.5,
      duration: 800,
    })
  }

  function fitJourney(journey: Journey) {
    if (!map) return
    const bounds = boundsOf(journey.route)
    if (!bounds) return
    map.fitBounds(bounds, { padding: fitPadding, maxZoom: 9, duration: 800 })
  }

  function markerElement(memento: Memento, seq: number) {
    const button = document.createElement("button")
    button.className = `map-marker map-marker--${memento.kind}`
    button.type = "button"
    button.setAttribute("aria-label", `${seq}. ${memento.title[lang]}`)
    button.innerHTML = `<span>${seq}</span>`
    button.addEventListener("click", () => selectMemento(memento))
    return button
  }

  function rebuildMarkers(journey: Journey) {
    if (!map) return
    markers.forEach((marker) => marker.remove())
    markers.clear()
    journey.mementos.forEach((memento, index) => {
      const marker = new maplibregl.Marker({
        element: markerElement(memento, index + 1),
        anchor: "center",
      })
        .setLngLat(memento.coords)
        .addTo(map!)
      markers.set(memento.id, marker)
    })
    syncMarkers()
  }

  function syncMarkers() {
    markers.forEach((marker, id) => {
      marker.getElement().classList.toggle("is-active", id === selected?.id)
    })
  }

  function updateJourneyLayers(journey: Journey) {
    if (!map || !map.getLayer("route-active")) return
    map.setFilter("route-active", ["==", ["get", "journeyId"], journey.id])
    map.setFilter("route-active-glow", ["==", ["get", "journeyId"], journey.id])
    ;(map.getSource("transit") as maplibregl.GeoJSONSource).setData(transitFeatures(journey))
    rebuildMarkers(journey)
  }

  function selectJourney(journey: Journey) {
    selectedJourneyId = journey.id
    selected = journey.mementos[0]
    updateJourneyLayers(journey)
    fitJourney(journey)
    void focusDetailHeading()
  }

  // Screen-space position (viewport-relative, for position:fixed) of a
  // memento's imperative map marker, if the map + marker for it exist yet.
  function markerScreenPos(memento: Memento) {
    if (!map || !mapContainer || !markers.has(memento.id)) return undefined
    const point = map.project(memento.coords)
    const rect = mapContainer.getBoundingClientRect()
    return { x: rect.left + point.x, y: rect.top + point.y }
  }

  async function selectMemento(memento: Memento, { focusHeading = true } = {}) {
    const origin = prefersReducedMotion ? undefined : markerScreenPos(memento)
    if (origin) {
      // Render the ghost at the marker for one tick, then swap it out for
      // the real stub-card in the same flush that selects the memento —
      // crossfade pairs the ghost's out-transition with the stub-card's
      // in-transition (same key) and morphs between their positions.
      ghostOrigin = origin
      ghostKind = memento.kind
      ghostKey = memento.id
      ghostVisible = true
      await tick()
      selected = memento
      ghostVisible = false
    } else {
      selected = memento
    }
    focusMap(memento)
    if (focusHeading) await focusDetailHeading()
  }

  async function focusDetailHeading() {
    await tick()
    detailHeadingEl?.focus()
  }

  // "Close" in v1's always-visible 3-column layout means returning to a
  // neutral state — the journey's first memento — and moving focus back to
  // the index rail rather than hiding the (non-modal) detail panel.
  async function closeDetail() {
    if (!selectedJourney) return
    const first = selectedJourney.mementos[0]
    if (first && first.id !== selected?.id) {
      await selectMemento(first, { focusHeading: false })
    } else {
      fitJourney(selectedJourney)
    }
    await tick()
    document.querySelector<HTMLButtonElement>(".timeline-item.active")?.focus()
  }

  function focusMap(memento: Memento) {
    if (!map) return
    if (memento.transit) {
      const bounds = boundsOf([memento.transit.from.coords, memento.transit.to.coords])
      if (!bounds) return
      map.fitBounds(bounds, {
        padding: fitPadding,
        maxZoom: 10.5,
        duration: 700,
      })
      return
    }
    map.flyTo({ center: memento.coords, zoom: 9.6, duration: 700, essential: true })
  }

  function applyMapTheme(next: Theme) {
    if (!map || !map.getLayer("base-light")) return
    map.setLayoutProperty("base-light", "visibility", next === "light" ? "visible" : "none")
    map.setLayoutProperty("base-dark", "visibility", next === "light" ? "none" : "visible")
  }

  function toggleTheme() {
    theme = theme === "dark" ? "light" : "dark"
  }

  function setupMap() {
    if (!mapContainer) return
    map = new maplibregl.Map({
      container: mapContainer,
      style: mapStyle,
      center: [138.0, 38.0],
      zoom: 4.4,
      attributionControl: false,
    })

    map.addControl(new maplibregl.NavigationControl({ showCompass: false }), "top-right")

    map.on("load", () => {
      if (!map) return

      map.addSource("routes", { type: "geojson", data: routesGeoJson() })
      // All journeys, dim — the world index.
      map.addLayer({
        id: "routes-all",
        type: "line",
        source: "routes",
        paint: { "line-color": "#fb923c", "line-width": 2, "line-opacity": 0.28 },
      })
      // Selected journey, bright + glow.
      map.addLayer({
        id: "route-active-glow",
        type: "line",
        source: "routes",
        filter: ["==", ["get", "journeyId"], selectedJourneyId],
        paint: { "line-color": "#f97316", "line-width": 8, "line-opacity": 0.16, "line-blur": 4 },
      })
      map.addLayer({
        id: "route-active",
        type: "line",
        source: "routes",
        filter: ["==", ["get", "journeyId"], selectedJourneyId],
        paint: { "line-color": "#fb923c", "line-width": 3, "line-opacity": 0.95 },
      })

      map.addSource("transit", { type: "geojson", data: transitFeatures(selectedJourney) })
      map.addLayer({
        id: "transit",
        type: "line",
        source: "transit",
        paint: { "line-color": "#fde68a", "line-width": 4, "line-opacity": 0.9 },
      })

      rebuildMarkers(selectedJourney)
      applyMapTheme(theme)
      fitAll()
    })
  }

  // Re-highlight the active marker whenever the selected memento changes.
  $: if (selected && markers.size) {
    syncMarkers()
  }

  $: applyMapTheme(theme)

  onMount(() => {
    loadJourneys()
      .then(async (data) => {
        journeys = data
        selectedJourneyId = data[0]?.id ?? ""
        selected = data[0]?.mementos[0]
        isLoading = false
        await tick()
        setupMap()
      })
      .catch((reason) => {
        error = reason instanceof Error ? reason.message : String(reason)
        isLoading = false
      })

    const motionQuery = window.matchMedia("(prefers-reduced-motion: reduce)")
    prefersReducedMotion = motionQuery.matches
    const onMotionChange = (event: MediaQueryListEvent) => {
      prefersReducedMotion = event.matches
    }
    motionQuery.addEventListener("change", onMotionChange)

    return () => {
      markers.forEach((marker) => marker.remove())
      markers.clear()
      map?.remove()
      motionQuery.removeEventListener("change", onMotionChange)
    }
  })
</script>

<main class="app-shell" class:theme-light={theme === "light"}>
  <div class="sr-only" role="status" aria-live="polite">{liveMessage}</div>
  {#if isLoading}
    <div class="v1-status">Loading…</div>
  {:else if error}
    <div class="v1-status">{error}</div>
  {:else if selectedJourney && selected}
    <!-- Index rail: journeys (world index) -> selected journey's chronological timeline. -->
    <aside class="index-rail" aria-label="Journey index">
      <div class="rail-toolbar">
        <div class="lang-switch" role="group" aria-label="Language">
          <button class:active={lang === "ja"} on:click={() => (lang = "ja")}>日本語</button>
          <button class:active={lang === "en"} on:click={() => (lang = "en")}>EN</button>
          <button class:active={lang === "zh"} on:click={() => (lang = "zh")}>中文</button>
        </div>
        <button class="theme-toggle" on:click={toggleTheme} aria-label={theme === "dark" ? "Switch to light theme" : "Switch to dark theme"}>
          {theme === "dark" ? "☀" : "☾"}
        </button>
      </div>

      <div class="rail-head">
        <p class="eyebrow">felicia</p>
        <div class="rail-head-row">
          <h1>{t(uiText.journeys)}</h1>
          <button class="all-btn" on:click={fitAll}>{t(uiText.all)}</button>
        </div>
        {#if toMemories}
          <button class="all-btn" on:click={toMemories}>
            {lang === "en" ? "Collection →" : lang === "zh" ? "藏品 →" : "コレクション →"}
          </button>
        {/if}
      </div>

      <ol class="journey-list" aria-label="Journeys">
        {#each journeys as journey (journey.id)}
          <li>
            <button class="journey-item" class:active={journey.id === selectedJourneyId} on:click={() => selectJourney(journey)}>
              <span class="journey-item-title">{t(journey.title)}</span>
              <span class="journey-item-meta">{t(journey.dates)} · {t(journey.place)}</span>
              <span class="journey-item-count">{countLabel(journey.mementos.length)}</span>
            </button>

            {#if journey.id === selectedJourneyId}
              <ol class="timeline" aria-label="Mementos in order">
                {#each journey.mementos as memento, index (memento.id)}
                  <li>
                    <button class="timeline-item" class:active={memento.id === selected.id} on:click={() => selectMemento(memento)}>
                      <span class="timeline-glyph timeline-glyph--{memento.kind}">{index + 1}</span>
                      <span class="timeline-body">
                        <span class="timeline-date">{t(memento.date)}</span>
                        <strong>{t(memento.title)}</strong>
                        <span class="timeline-place">{t(memento.place)}</span>
                      </span>
                    </button>
                  </li>
                {/each}
              </ol>
            {/if}
          </li>
        {/each}
      </ol>
    </aside>

    <!-- Map: the hero. -->
    <section class="map-stage" aria-label="Journey map">
      <div bind:this={mapContainer} class="map-canvas"></div>
      {#if ghostVisible && ghostOrigin}
        <div
          class="marker-ghost map-marker map-marker--{ghostKind}"
          style="left: {ghostOrigin.x}px; top: {ghostOrigin.y}px"
          aria-hidden="true"
          in:receiveStub={{ key: ghostKey }}
          out:sendStub={{ key: ghostKey }}
        >
          <span>{ghostSeq}</span>
        </div>
      {/if}
    </section>

    <!-- Detail: the memento opened — paper stub + essay + gallery. -->
    <aside class="detail-panel" aria-label="Memento detail">
      {#key selected.id}
        <div class="detail-inner" in:fade={{ duration: prefersReducedMotion ? 0 : 220 }}>
          <button type="button" class="detail-close" aria-label={t(uiText.close)} on:click={closeDetail}>×</button>

          <div class="section-head">
            <p class="eyebrow">{t(kindLabel[selected.kind])}</p>
            <h2 tabindex="-1" bind:this={detailHeadingEl}>{t(selected.title)}</h2>
          </div>

          <div class="stub-card {selected.kind}" in:receiveStub={{ key: selected.id }} out:sendStub={{ key: selected.id }}>
            {#if selected.kind === "transit" && selected.transit}
              <div class="ticket-face">
                <div class="ticket-line">
                  <span>{t(selected.transit.operator)}</span>
                  <strong>{t(selected.transit.line)}</strong>
                </div>
                <div class="station-pair">
                  <span>{stationName(selected.transit.from)}</span>
                  <b>→</b>
                  <span>{stationName(selected.transit.to)}</span>
                </div>
                <div class="ticket-meta">
                  <span>{t(selected.date)}</span>
                  <span>{selected.transit.fare}</span>
                </div>
              </div>
            {:else if selected.kind === "stamp"}
              <!-- 御朱印 — ink on washi; the vermilion seal block is the signature. -->
              <div class="washi-face">
                <span class="washi-seal" aria-hidden="true">御朱印</span>
                <p class="washi-sub">{kindText("shrine_or_temple") || t(selected.place)}</p>
                <strong class="washi-name">{kindText("name") || t(selected.title)}</strong>
                {#if kindText("deity")}
                  <p class="washi-deity">{kindText("deity")}</p>
                {/if}
                <small class="washi-date">{t(selected.date)}</small>
              </div>
            {:else if selected.kind === "receipt"}
              <!-- Thermal roll — narrow measure, monospace, genuinely torn hem. -->
              <div class="thermal-face">
                <div class="thermal-head">
                  <strong>{kindText("shop") || t(selected.vendor) || t(selected.title)}</strong>
                  <span>{t(selected.place)}</span>
                </div>
                <div class="thermal-rule"></div>
                <ul class="thermal-items">
                  <li class="thermal-date"><span>{t(selected.date)}</span></li>
                  {#each kindLines("items") as item, index (`${item}:${index}`)}
                    <li><span>{item}</span></li>
                  {/each}
                </ul>
                <div class="thermal-total">
                  <b>{selected.price || kindText("total")}</b>
                </div>
                <div class="thermal-code" aria-hidden="true"></div>
                <p class="thermal-foot">{t(selected.title)}</p>
              </div>
            {:else if selected.kind === "live"}
              <!-- Admission stock — the tear-off stub is cut out of the card. -->
              <div class="admission-face">
                <div class="admission-main">
                  <p class="admission-venue">{kindName("venue") || t(selected.place)}</p>
                  <strong class="admission-artist">{kindText("artist") || t(selected.title)}</strong>
                  <div class="admission-meta">
                    <span>{t(selected.date)}</span>
                    {#if kindText("seat")}
                      <span>{kindText("seat")}</span>
                    {/if}
                  </div>
                  {#if linkHost(kindText("setlist_url"))}
                    <a class="admission-link" href={kindText("setlist_url")} target="_blank" rel="noopener noreferrer">
                      {linkHost(kindText("setlist_url"))} ↗
                    </a>
                  {/if}
                </div>
                <div class="admission-stub">
                  <span class="admission-stub-kind">{t(kindLabel.live)}</span>
                  <span class="admission-stub-name">{kindText("artist") || t(selected.title)}</span>
                  <span class="admission-stub-seat">{kindText("seat") || t(selected.date)}</span>
                </div>
              </div>
            {:else if selected.kind === "souvenir"}
              <!-- Postcard back — divider rule, stamp box, postmark. -->
              <div class="postcard-face">
                <div class="postcard-note">
                  <p class="postcard-dateline">{[t(selected.place), t(selected.date)].filter(Boolean).join(" — ")}</p>
                  {#if kindText("note")}
                    <p class="postcard-message">{kindText("note")}</p>
                  {:else}
                    <span class="postcard-rules" aria-hidden="true"></span>
                  {/if}
                </div>
                <div class="postcard-address">
                  <span class="postcard-stamp" aria-hidden="true">〒</span>
                  <span class="postcard-postmark" aria-hidden="true"></span>
                  <strong>{kindText("name") || t(selected.title)}</strong>
                  <span>{kindText("origin") || t(selected.place)}</span>
                </div>
              </div>
            {:else}
              <!-- Goods — kraft swing tag; the hole and the clipped shoulders are real. -->
              <div class="tag-face">
                <span class="tag-eyelet" aria-hidden="true"></span>
                <p class="tag-kind">{t(kindLabel.goods)}</p>
                <strong class="tag-name">{kindText("name") || t(selected.title)}</strong>
                <p class="tag-source">
                  {[kindText("shop") || t(selected.vendor), kindText("manufacturer")].filter(Boolean).join(" · ") || t(selected.place)}
                </p>
                <div class="tag-foot">
                  <span>{t(selected.date)}</span>
                  {#if selected.price}
                    <b>{selected.price}</b>
                  {/if}
                </div>
              </div>
            {/if}
          </div>

          <article class="essay">
            <span>{t(uiText.story)}</span>
            <p>{t(selected.essay)}</p>
          </article>

          {#if selected.photos.length}
            <div class="gallery">
              {#each selected.photos as photo, index (`${photo.src}:${index}`)}
                <figure>
                  <img src={photo.src} alt={t(selected.title)} />
                  <figcaption>{t(photo.caption)}</figcaption>
                </figure>
              {/each}
            </div>
          {/if}
        </div>
      {/key}
    </aside>
  {:else}
    <div class="v1-status">No journeys</div>
  {/if}
</main>

<style>
  .v1-status {
    display: grid;
    grid-column: 1 / -1;
    min-height: 100%;
    place-items: center;
    color: var(--muted);
  }
</style>
