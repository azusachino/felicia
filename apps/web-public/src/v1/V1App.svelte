<script lang="ts">
  import maplibregl, { type StyleSpecification } from "maplibre-gl"
  import { onMount, tick } from "svelte"
  import { fade, fly } from "svelte/transition"
  import { loadJourneys } from "../api/source"
  import {
    kindLabel,
    uiText,
    type Coordinates,
    type Journey,
    type L,
    type Lang,
    type Memento,
    type Station,
    type Theme,
  } from "../data"
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

  $: t = (value: L | MessageKey) => (typeof value === "string" ? message(lang, value) : value[lang])
  $: stationName = (s: Station) => (lang === "en" ? s.name : s.ja)
  $: selectedJourney = journeys.find((j) => j.id === selectedJourneyId) ?? journeys[0]
  $: countLabel = (n: number) => (lang === "en" ? `${n} mementos` : `${n}件`)

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
  }

  function selectMemento(memento: Memento) {
    selected = memento
    focusMap(memento)
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

    return () => {
      markers.forEach((marker) => marker.remove())
      markers.clear()
      map?.remove()
    }
  })
</script>

<main class="app-shell" class:theme-light={theme === "light"}>
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
        <button
          class="theme-toggle"
          on:click={toggleTheme}
          aria-label={theme === "dark" ? "Switch to light theme" : "Switch to dark theme"}
        >
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
            <button
              class="journey-item"
              class:active={journey.id === selectedJourneyId}
              on:click={() => selectJourney(journey)}
            >
              <span class="journey-item-title">{t(journey.title)}</span>
              <span class="journey-item-meta">{t(journey.dates)} · {t(journey.place)}</span>
              <span class="journey-item-count">{countLabel(journey.mementos.length)}</span>
            </button>

            {#if journey.id === selectedJourneyId}
              <ol class="timeline" aria-label="Mementos in order">
                {#each journey.mementos as memento, index (memento.id)}
                  <li>
                    <button
                      class="timeline-item"
                      class:active={memento.id === selected.id}
                      on:click={() => selectMemento(memento)}
                    >
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
    </section>

    <!-- Detail: the memento opened — paper stub + essay + gallery. -->
    <aside class="detail-panel" aria-label="Memento detail">
      {#key selected.id}
        <div class="detail-inner" in:fade={{ duration: 220 }}>
          <div class="section-head">
            <p class="eyebrow">{t(kindLabel[selected.kind])}</p>
            <h2>{t(selected.title)}</h2>
          </div>

          <div class="stub-card {selected.kind}" in:fly={{ y: 12, duration: 320, delay: 60 }}>
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
              <div class="stamp-face">
                <span>御朱印</span>
                <strong>{t(selected.place)}</strong>
                <small>{t(selected.date)}</small>
              </div>
            {:else}
              <div class="goods-face">
                <span>{t(kindLabel.goods)}</span>
                <strong>{t(selected.title)}</strong>
                <small>{t(selected.vendor)} · {selected.price}</small>
              </div>
            {/if}
          </div>

          <article class="essay">
            <span>{t(uiText.story)}</span>
            <p>{t(selected.essay)}</p>
          </article>

          {#if selected.photos.length}
            <div class="gallery">
              {#each selected.photos as photo (photo.src)}
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
