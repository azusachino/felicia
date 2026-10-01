<script lang="ts">
  import maplibregl from "maplibre-gl"
  import { onMount } from "svelte"
  import type { Journey, Lang, Theme } from "@felicia/model"

  let {
    journeys,
    selectedJourneyId,
    lang,
    theme,
    onSelect,
  }: {
    journeys: Journey[]
    selectedJourneyId: string | null
    lang: Lang
    theme: Theme
    onSelect: (id: string) => void
  } = $props()

  let container: HTMLDivElement
  let map: maplibregl.Map | undefined
  let loaded = $state(false)
  let resizeObserver: ResizeObserver | undefined
  let prefersReducedMotion = false
  // Real DOM marker buttons for each place, so every map marker has an
  // accessible name and a keyboard path (shared interaction grammar,
  // reader-ui-ux-contract.md) -- the canvas circle layer this replaced had
  // neither.
  // eslint-disable-next-line svelte/prefer-svelte-reactivity -- imperative maplibre marker cache, not reactive UI state
  const markers = new Map<string, maplibregl.Marker>()

  const style = "https://tiles.openfreemap.org/styles/liberty"

  function routeData() {
    return {
      type: "FeatureCollection" as const,
      features: journeys
        .filter((journey) => journey.route.length > 1)
        .map((journey) => ({
          type: "Feature" as const,
          properties: { id: journey.id, selected: journey.id === selectedJourneyId },
          geometry: { type: "LineString" as const, coordinates: journey.route },
        })),
    }
  }

  function placeData() {
    return {
      type: "FeatureCollection" as const,
      features: journeys.flatMap((journey) =>
        journey.visits.map((visit) => ({
          type: "Feature" as const,
          properties: { id: journey.id, label: visit.label[lang] || visit.label.en },
          geometry: { type: "Point" as const, coordinates: visit.coords },
        })),
      ),
    }
  }

  function fitWorld() {
    if (!map) return
    const coords = journeys.flatMap((journey) => [...journey.route, ...journey.visits.map((visit) => visit.coords)])
    if (!coords.length) return
    const bounds = new maplibregl.LngLatBounds(coords[0], coords[0])
    coords.forEach((coord) => bounds.extend(coord))
    map.fitBounds(bounds, { padding: 48, maxZoom: 3.2, duration: prefersReducedMotion ? 0 : 500 })
  }

  function refreshData() {
    if (!map) return
    ;(map.getSource("journeys") as maplibregl.GeoJSONSource | undefined)?.setData(routeData())
    ;(map.getSource("places") as maplibregl.GeoJSONSource | undefined)?.setData(placeData())
    rebuildMarkers()
    fitWorld()
  }

  function placeMarkerElement(journey: Journey, label: string) {
    const button = document.createElement("button")
    button.type = "button"
    button.className = "techo-index-place"
    button.setAttribute("aria-label", `${label} — ${journey.title[lang] || journey.title.en}`)
    button.addEventListener("click", (event) => {
      event.stopPropagation()
      onSelect(journey.id)
    })
    return button
  }

  function rebuildMarkers() {
    if (!map) return
    markers.forEach((marker) => marker.remove())
    markers.clear()
    journeys.forEach((journey) => {
      journey.visits.forEach((visit) => {
        const marker = new maplibregl.Marker({
          element: placeMarkerElement(journey, visit.label[lang] || visit.label.en),
          anchor: "center",
        })
          .setLngLat(visit.coords)
          .addTo(map!)
        markers.set(`${journey.id}:${visit.id}`, marker)
      })
    })
    syncActiveMarkers()
  }

  function syncActiveMarkers() {
    markers.forEach((marker, key) => {
      marker.getElement().classList.toggle("is-active", key.startsWith(`${selectedJourneyId}:`))
    })
  }

  onMount(() => {
    const motionQuery = window.matchMedia("(prefers-reduced-motion: reduce)")
    prefersReducedMotion = motionQuery.matches
    const onMotionChange = (event: MediaQueryListEvent) => (prefersReducedMotion = event.matches)
    motionQuery.addEventListener("change", onMotionChange)

    map = new maplibregl.Map({
      container,
      style,
      center: [10, 30],
      zoom: 1.6,
      attributionControl: {},
    })
    map.addControl(new maplibregl.NavigationControl({ showCompass: false }), "top-right")
    // Routes stay a mouse-only bonus click target (a line has no natural DOM
    // marker equivalent); places are real DOM marker buttons below, so they
    // are the keyboard- and screen-reader-reachable path to selecting a
    // journey.
    map.on("click", "journey-routes", (event) => {
      const id = event.features?.[0]?.properties?.id
      if (typeof id === "string") onSelect(id)
    })
    map.on("mouseenter", "journey-routes", () => map?.getCanvas().classList.add("is-clickable"))
    map.on("mouseleave", "journey-routes", () => map?.getCanvas().classList.remove("is-clickable"))

    resizeObserver = new ResizeObserver(() => map?.resize())
    resizeObserver.observe(container)

    map.on("load", () => {
      if (!map) return
      // Read the theme's own runtime-overridable tokens once at setup,
      // instead of duplicating their default values as literals -- an
      // author-configured --accent would otherwise silently stop applying
      // to the map the moment the view switches away from CSS-styled chrome.
      const styles = getComputedStyle(container)
      const terracotta = styles.getPropertyValue("--terracotta").trim() || "#d9674c"
      const ink = styles.getPropertyValue("--ink").trim() || "#3a2f1c"

      map.addSource("journeys", { type: "geojson", data: routeData() })
      map.addLayer({
        id: "journey-routes",
        type: "line",
        source: "journeys",
        layout: { "line-cap": "round", "line-join": "round" },
        paint: {
          "line-color": ["case", ["get", "selected"], terracotta, "#7aa8a6"],
          "line-width": ["case", ["get", "selected"], 4, 2],
          "line-opacity": ["case", ["get", "selected"], 0.95, 0.5],
        },
      })
      map.addSource("places", { type: "geojson", data: placeData() })
      // The circle used to be a canvas paint layer here -- unreachable by
      // keyboard and unnamed to a screen reader. rebuildMarkers() below
      // draws the actual clickable dot as a real DOM marker button instead;
      // this layer stays for the always-visible place labels only.
      map.addLayer({
        id: "journey-place-labels",
        type: "symbol",
        source: "places",
        layout: {
          "text-field": ["get", "label"],
          "text-size": 11,
          "text-offset": [0, 1.1],
          "text-anchor": "top",
        },
        paint: {
          "text-color": ink,
          "text-halo-color": "#fff8ed",
          "text-halo-width": 1.5,
        },
      })
      rebuildMarkers()
      map.resize()
      fitWorld()
      loaded = true
    })

    return () => {
      resizeObserver?.disconnect()
      resizeObserver = undefined
      motionQuery.removeEventListener("change", onMotionChange)
      markers.forEach((marker) => marker.remove())
      markers.clear()
      map?.remove()
      map = undefined
    }
  })

  $effect(() => {
    void journeys
    void selectedJourneyId
    void lang
    if (loaded) refreshData()
  })

  $effect(() => {
    void theme
  })
</script>

<div bind:this={container} class="index-map"></div>

<style>
  .index-map {
    position: absolute;
    inset: 0;
    overflow: hidden;
    border-radius: 0.15rem;
  }

  :global(.is-clickable) {
    cursor: pointer;
  }

  /* The place dot itself stays visually small (matches the former canvas
     circle-radius: 5). The hit area grows via ::after, but this is a
     world-zoom index of every journey's visits, where two points can sit a
     handful of CSS pixels apart -- the project's usual 44px touch-target
     extension (AtlasMap's recipe) would overlap at that density, so this
     uses the WCAG 2.5.8 AA 24px floor instead. Neighboring visits within one
     journey share a handler (selecting either opens the same journey), so an
     overlap there is harmless; real marker clustering would be needed to
     fully remove the residual risk between two *different* journeys' visits
     landing this close together, which is out of scope for this pass. */
  :global(.techo-index-place) {
    position: relative;
    width: 0.625rem;
    height: 0.625rem;
    border: 2px solid #fff8ed;
    border-radius: 50%;
    background: #7aa8a6;
    padding: 0;
  }

  :global(.techo-index-place)::after {
    content: "";
    position: absolute;
    inset: calc((0.625rem - 1.5rem) / 2);
  }

  :global(.techo-index-place.is-active) {
    z-index: 1;
    background: var(--terracotta, #d9674c);
  }
</style>
