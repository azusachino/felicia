<script lang="ts">
  import maplibregl from "maplibre-gl"
  import { onMount } from "svelte"
  import type { Coordinates, Theme } from "../data"

  // v3 detail map — MapLibre vector basemap from OpenFreeMap (OSM-derived,
  // with attribution), route + transit lines, DOM markers, scoped to a journey
  // and clustered by
  // PLACE: one marker per place, a count badge when a place holds several
  // mementos. Selecting a place is bubbled up; the parent opens its memories.
  interface Place {
    key: string
    coords: Coordinates
    seq: number
    count: number
  }

  let {
    places,
    route,
    transit,
    activeKey,
    theme,
    onSelect,
  }: {
    places: Place[]
    route: Coordinates[]
    transit: [Coordinates, Coordinates][]
    activeKey: string | null
    theme: Theme
    onSelect: (key: string) => void
  } = $props()

  let container: HTMLDivElement
  let map: maplibregl.Map | undefined
  let loaded = $state(false)
  let resizeObserver: ResizeObserver | undefined
  // eslint-disable-next-line svelte/prefer-svelte-reactivity -- imperative maplibre marker cache, not reactive UI state
  const markers = new Map<string, maplibregl.Marker>()

  const mapStyle = "https://tiles.openfreemap.org/styles/liberty"

  function routeGeoJson() {
    return {
      type: "FeatureCollection" as const,
      features: [
        {
          type: "Feature" as const,
          geometry: { type: "LineString" as const, coordinates: route },
          properties: {},
        },
      ],
    }
  }

  function transitGeoJson() {
    return {
      type: "FeatureCollection" as const,
      features: transit.map((pair) => ({
        type: "Feature" as const,
        geometry: { type: "LineString" as const, coordinates: pair },
        properties: {},
      })),
    }
  }

  function boundsOf(coords: Coordinates[]) {
    if (coords.length === 0) {
      return new maplibregl.LngLatBounds([138, 38], [138, 38])
    }
    const bounds = new maplibregl.LngLatBounds(coords[0], coords[0])
    for (const coord of coords) bounds.extend(coord)
    return bounds
  }

  const fitPadding = { top: 120, bottom: 120, left: 120, right: 460 }

  function fitJourney() {
    if (!map) return
    const coords = [...route, ...places.map((place) => place.coords)]
    if (!coords.length) return
    map.fitBounds(boundsOf(coords), { padding: fitPadding, maxZoom: 9, duration: 700 })
  }

  function markerElement(place: Place) {
    const button = document.createElement("button")
    button.type = "button"
    button.className = "v3-mark"
    button.setAttribute("aria-label", `Place ${place.seq}`)
    button.innerHTML = `<span>${place.seq}</span>${
      place.count > 1 ? `<i class="v3-mark-count">${place.count}</i>` : ""
    }`
    button.addEventListener("click", (e) => {
      e.stopPropagation()
      onSelect(place.key)
    })
    return button
  }

  function rebuildMarkers() {
    if (!map) return
    markers.forEach((marker) => marker.remove())
    markers.clear()
    for (const place of places) {
      const marker = new maplibregl.Marker({ element: markerElement(place), anchor: "center" })
        .setLngLat(place.coords)
        .addTo(map)
      markers.set(place.key, marker)
    }
    syncActive()
  }

  function syncActive() {
    markers.forEach((marker, key) => {
      marker.getElement().classList.toggle("is-active", key === activeKey)
    })
  }

  function applyTheme(next: Theme) {
    // Liberty is intentionally used for both themes: it is a readable OSM
    // vector style, and hiding attribution or swapping to OSMF's public tile
    // server would be the wrong trade-off for a local prototype.
    void next
  }

  onMount(() => {
    map = new maplibregl.Map({
      container,
      style: mapStyle,
      center: route[0] ?? places[0]?.coords ?? [138, 38],
      zoom: 6,
    })
    map.addControl(new maplibregl.NavigationControl({ showCompass: false }), "top-right")

    map.on("click", (e) => {
      if ((e.originalEvent?.target as HTMLElement)?.tagName === "CANVAS") {
        onSelect("")
      }
    })

    // The container reaches full size after mount/layout; keep the canvas in
    // sync so the map fills its box (otherwise only a corner renders).
    resizeObserver = new ResizeObserver(() => map?.resize())
    resizeObserver.observe(container)

    map.on("load", () => {
      if (!map) return
      map.addSource("route", { type: "geojson", data: routeGeoJson() })
      map.addLayer({
        id: "route-glow",
        type: "line",
        source: "route",
        paint: { "line-color": "#f97316", "line-width": 8, "line-opacity": 0.16, "line-blur": 4 },
      })
      map.addLayer({
        id: "route-line",
        type: "line",
        source: "route",
        layout: { "line-cap": "round", "line-join": "round" },
        paint: {
          "line-color": "#fb923c",
          "line-width": 4,
          "line-opacity": 0.95,
        },
      })
      map.addSource("transit", { type: "geojson", data: transitGeoJson() })
      map.addLayer({
        id: "transit",
        type: "line",
        source: "transit",
        paint: {
          "line-color": "#f6c98b",
          "line-width": 2,
          "line-opacity": 0.35,
          "line-dasharray": [1, 2],
        },
      })

      rebuildMarkers()
      applyTheme(theme)
      map.resize()
      fitJourney()
      loaded = true
    })

    return () => {
      resizeObserver?.disconnect()
      resizeObserver = undefined
      markers.forEach((marker) => marker.remove())
      markers.clear()
      map?.remove()
      map = undefined
    }
  })

  // Rebuild + refit when the journey (places/route) changes.
  $effect(() => {
    void places
    void route
    if (!loaded || !map) return
    ;(map.getSource("route") as maplibregl.GeoJSONSource | undefined)?.setData(routeGeoJson())
    ;(map.getSource("transit") as maplibregl.GeoJSONSource | undefined)?.setData(transitGeoJson())
    rebuildMarkers()
    fitJourney()
  })

  // Highlight + fly to the active place.
  $effect(() => {
    if (!loaded || !map) return
    syncActive()
    // Keep the full journey visible while the reader changes memories. The
    // active marker is enough feedback; zooming to every memory hides the route.
    if (!activeKey) fitJourney()
  })

  $effect(() => {
    if (loaded) applyTheme(theme)
  })
</script>

<!-- Inline styles, not Tailwind utilities: maplibre-gl.css sets an UNLAYERED
     `.maplibregl-map { position: relative }` that outranks layered utilities, so
     `absolute inset-0` would be ignored and the map would collapse. -->
<div bind:this={container} style="position:absolute; inset:0;"></div>
