<script lang="ts">
  import maplibregl, { type StyleSpecification } from "maplibre-gl"
  import "maplibre-gl/dist/maplibre-gl.css"
  import { onMount } from "svelte"
  import { geoDistance } from "d3-geo"
  import { timer, type Timer } from "d3-timer"
  import { lineString, point } from "@turf/helpers"
  import along from "@turf/along"
  import length from "@turf/length"
  import lineSlice from "@turf/line-slice"
  import type { Coordinates, Journey, Memento, Theme } from "@felicia/model"
  import type { PlaybackStep } from "../../playback/engine"

  let {
    journeys,
    activeJourneyId,
    activeMementoId,
    currentStep = null,
    lang,
    theme,
    onSelect,
  }: {
    journeys: Journey[]
    activeJourneyId: string | null
    activeMementoId: string | null
    currentStep?: PlaybackStep | null
    lang: "ja" | "en" | "zh"
    theme: Theme
    onSelect: (id: string) => void
  } = $props()

  let container = $state<HTMLDivElement>()
  let map: maplibregl.Map | undefined
  let loaded = $state(false)
  let resizeObserver: ResizeObserver | undefined
  let prefersReducedMotion = $state(false)
  // eslint-disable-next-line svelte/prefer-svelte-reactivity -- imperative MapLibre marker cache
  const markers = new Map<string, maplibregl.Marker>()
  let vehicleMarker: maplibregl.Marker | undefined
  let activeAnimationTimer: Timer | null = null

  const mapStyle: StyleSpecification = {
    version: 8,
    sources: {
      // 1. High-detail Esri Topo Map: contains topographic elevation contours, mountain relief, forest cover, highways and local roads
      "topo-base": {
        type: "raster",
        tiles: ["https://server.arcgisonline.com/ArcGIS/rest/services/World_Topo_Map/MapServer/tile/{z}/{y}/{x}"],
        tileSize: 256,
        maxzoom: 19,
        attribution: "&copy; Esri, HERE, Garmin, Intermap, USGS, METI/NASA",
      },
      // 2. High-contrast Esri Dark Canvas Base for sleek dark spatial journal aesthetic
      "dark-base": {
        type: "raster",
        tiles: ["https://server.arcgisonline.com/ArcGIS/rest/services/Canvas/World_Dark_Gray_Base/MapServer/tile/{z}/{y}/{x}"],
        tileSize: 256,
        maxzoom: 16,
      },
      // 3. Transportation reference overlay: roads, highways, rail labels
      "transport-ref": {
        type: "raster",
        tiles: ["https://server.arcgisonline.com/ArcGIS/rest/services/Reference/World_Transportation/MapServer/tile/{z}/{y}/{x}"],
        tileSize: 256,
        maxzoom: 19,
      },
      // 4. Boundaries and places reference overlay: cities, towns, administrative boundaries, coastal place names
      "places-ref": {
        type: "raster",
        tiles: ["https://server.arcgisonline.com/ArcGIS/rest/services/Reference/World_Boundaries_and_Places/MapServer/tile/{z}/{y}/{x}"],
        tileSize: 256,
        maxzoom: 19,
      },
      // 5. Dark Canvas reference overlay
      "dark-canvas-ref": {
        type: "raster",
        tiles: ["https://server.arcgisonline.com/ArcGIS/rest/services/Canvas/World_Dark_Gray_Reference/MapServer/tile/{z}/{y}/{x}"],
        tileSize: 256,
        maxzoom: 16,
      },
    },
    layers: [
      // Light theme: Rich Topo Map with transport and place names
      { id: "layer-topo-base", type: "raster", source: "topo-base", layout: { visibility: "none" } },
      // Dark theme: Dark base + Road Transportation network + Boundaries & Places
      { id: "layer-dark-base", type: "raster", source: "dark-base", layout: { visibility: "visible" } },
      { id: "layer-dark-ref", type: "raster", source: "dark-canvas-ref", layout: { visibility: "visible" } },
      { id: "layer-transport-ref", type: "raster", source: "transport-ref", paint: { "raster-opacity": 0.82 } },
      { id: "layer-places-ref", type: "raster", source: "places-ref", paint: { "raster-opacity": 0.9 } },
    ],
  }

  function routeGeoJson() {
    return {
      type: "FeatureCollection" as const,
      features: journeys.flatMap((journey) => {
        const segments = journey.routeSegments?.length ? journey.routeSegments : [journey.route]
        return segments
          .filter((segment) => segment.length > 0)
          .map((coordinates, index) => ({
            type: "Feature" as const,
            geometry: { type: "LineString" as const, coordinates },
            properties: {
              journeyId: journey.id,
              segment: index,
              legMode: journey.legModes?.[index] ?? "car",
            },
          }))
      }),
    }
  }

  function activeJourney() {
    return journeys.find((journey) => journey.id === activeJourneyId) ?? null
  }

  function coordinatesFor(journey: Journey) {
    const route = journey.routeSegments?.length ? journey.routeSegments.flat() : journey.route
    return [...route, ...journey.mementos.map((memento) => memento.coords)]
  }

  function boundsOf(coordinates: Coordinates[]) {
    if (!coordinates.length) return undefined
    const bounds = new maplibregl.LngLatBounds(coordinates[0], coordinates[0])
    coordinates.slice(1).forEach((coordinate) => bounds.extend(coordinate))
    return bounds
  }

  function fitActiveJourney() {
    const journey = activeJourney()
    if (!map || !journey) return
    const bounds = boundsOf(coordinatesFor(journey))
    if (!bounds) return
    const distance = journeyDistance(journey)
    map.fitBounds(bounds, { padding: 80, maxZoom: 11, duration: prefersReducedMotion ? 0 : Math.min(900, 360 + distance * 180) })
  }

  function journeyDistance(journey: Journey) {
    const coordinates = journey.routeSegments?.length ? journey.routeSegments.flat() : journey.route
    return coordinates.slice(1).reduce((total, coordinate, index) => total + geoDistance(coordinates[index], coordinate), 0)
  }

  function markerElement(memento: Memento, index: number) {
    const button = document.createElement("button")
    button.type = "button"
    button.className = "atlas-marker"
    button.setAttribute("aria-label", `${index + 1}. ${memento.title[lang]}`)
    button.innerHTML = `<span>${index + 1}</span>${memento.photos.length ? `<i>${memento.photos.length}</i>` : ""}`
    button.addEventListener("click", (event) => {
      event.stopPropagation()
      onSelect(memento.id)
    })
    return button
  }

  function rebuildMarkers() {
    if (!map) return
    markers.forEach((marker) => marker.remove())
    markers.clear()
    journeys.forEach((journey) => {
      journey.mementos.forEach((memento, index) => {
        const marker = new maplibregl.Marker({
          element: markerElement(memento, index),
          anchor: "center",
        })
          .setLngLat(memento.coords)
          .addTo(map!)
        markers.set(memento.id, marker)
      })
    })
    syncMarkers()
  }

  function syncMarkers() {
    markers.forEach((marker, id) => {
      const ownerJourney = journeys.find((journey) => journey.mementos.some((item) => item.id === id))
      marker.getElement().classList.toggle("is-active", id === activeMementoId)
      marker.getElement().classList.toggle("is-dimmed", ownerJourney?.id !== activeJourneyId)
    })
  }

  function flyToMemento(id: string | null) {
    if (!map || !id) return
    const m = journeys.flatMap((j) => j.mementos).find((item) => item.id === id)
    if (!m) return
    map.flyTo({
      center: m.coords,
      zoom: 13.5,
      pitch: 45,
      bearing: 20,
      speed: 1.1,
      curve: 1.3,
      essential: true,
      padding: { top: 60, bottom: 80, left: 340, right: 60 },
    })
  }

  let lastAnimatedStopIndex = -1

  function animateAlongRoute(fromCoords: Coordinates, toCoords: Coordinates, durationMs = 2600) {
    if (!map || prefersReducedMotion) return
    const journey = activeJourney()
    if (!journey) return

    const allRouteCoords = journey.routeSegments?.length ? journey.routeSegments.flat() : journey.route
    if (allRouteCoords.length < 2) return

    try {
      const fullLine = lineString(allRouteCoords)
      const startPt = point(fromCoords)
      const stopPt = point(toCoords)
      const sliced = lineSlice(startPt, stopPt, fullLine)
      const sliceDistance = length(sliced, { units: "kilometers" })
      if (sliceDistance <= 0.05) return

      if (activeAnimationTimer) {
        activeAnimationTimer.stop()
        activeAnimationTimer = null
      }

      if (!vehicleMarker) {
        const el = document.createElement("div")
        el.className = "atlas-traveling-dot"
        vehicleMarker = new maplibregl.Marker({ element: el, anchor: "center" }).setLngLat(fromCoords).addTo(map)
      } else {
        vehicleMarker.getElement().style.display = "block"
        vehicleMarker.setLngLat(fromCoords)
      }

      const startTime = performance.now()
      activeAnimationTimer = timer(() => {
        const elapsed = performance.now() - startTime
        const progress = Math.min(1, elapsed / durationMs)
        const currentDist = progress * sliceDistance
        const pt = along(sliced, currentDist, { units: "kilometers" })
        const [lng, lat] = pt.geometry.coordinates as [number, number]

        vehicleMarker?.setLngLat([lng, lat])

        if (progress >= 1) {
          activeAnimationTimer?.stop()
          activeAnimationTimer = null
          if (vehicleMarker) {
            vehicleMarker.getElement().style.display = "none"
          }
        }
      })
    } catch {
      // Graceful fallback if slice calculation has zero length
    }
  }

  function refresh() {
    if (!loaded || !map) return
    ;(map.getSource("routes") as maplibregl.GeoJSONSource | undefined)?.setData(routeGeoJson())
    map.setFilter("route-active", ["==", ["get", "journeyId"], activeJourneyId ?? ""])
    map.setFilter("route-active-glow", ["==", ["get", "journeyId"], activeJourneyId ?? ""])
    map.setFilter("route-active-dashed", [
      "all",
      ["==", ["get", "journeyId"], activeJourneyId ?? ""],
      ["in", ["get", "legMode"], ["literal", ["ferry", "train", "plane"]]],
    ])
    rebuildMarkers()
    syncMarkers()
    fitActiveJourney()
  }

  function applyTheme() {
    if (!map) return
    const isLight = theme === "light"
    map.setLayoutProperty("layer-topo-base", "visibility", isLight ? "visible" : "none")
    map.setLayoutProperty("layer-dark-base", "visibility", isLight ? "none" : "visible")
    map.setLayoutProperty("layer-dark-ref", "visibility", isLight ? "none" : "visible")
  }

  onMount(() => {
    if (!container) return
    const motionQuery = window.matchMedia("(prefers-reduced-motion: reduce)")
    prefersReducedMotion = motionQuery.matches
    const onMotionChange = (event: MediaQueryListEvent) => (prefersReducedMotion = event.matches)
    motionQuery.addEventListener("change", onMotionChange)
    map = new maplibregl.Map({
      container,
      style: mapStyle,
      center: [138, 38],
      zoom: 4.5,
      attributionControl: false,
    })
    map.addControl(new maplibregl.AttributionControl({ compact: true }), "bottom-left")
    resizeObserver = new ResizeObserver(() => map?.resize())
    resizeObserver.observe(container)

    map.on("load", () => {
      if (!map) return
      map.addSource("routes", { type: "geojson", data: routeGeoJson() })
      map.addLayer({
        id: "routes-all",
        type: "line",
        source: "routes",
        layout: { "line-cap": "round", "line-join": "round" },
        paint: { "line-color": "#7fd8cb", "line-width": 2, "line-opacity": 0.3 },
      })
      map.addLayer({
        id: "route-active-glow",
        type: "line",
        source: "routes",
        filter: ["==", ["get", "journeyId"], activeJourneyId ?? ""],
        paint: { "line-color": "#ff9b72", "line-width": 9, "line-opacity": 0.25, "line-blur": 4 },
      })
      map.addLayer({
        id: "route-active",
        type: "line",
        source: "routes",
        filter: ["==", ["get", "journeyId"], activeJourneyId ?? ""],
        layout: { "line-cap": "round", "line-join": "round" },
        paint: { "line-color": "#ff9b72", "line-width": 3.5, "line-opacity": 0.95 },
      })
      map.addLayer({
        id: "route-active-dashed",
        type: "line",
        source: "routes",
        filter: [
          "all",
          ["==", ["get", "journeyId"], activeJourneyId ?? ""],
          ["in", ["get", "legMode"], ["literal", ["ferry", "train", "plane"]]],
        ],
        layout: { "line-cap": "round", "line-join": "round" },
        paint: {
          "line-color": "#ff9b72",
          "line-width": 3,
          "line-dasharray": [2, 2],
          "line-opacity": 0.95,
        },
      })
      loaded = true
      refresh()
      applyTheme()
      map.resize()
    })

    return () => {
      activeAnimationTimer?.stop()
      activeAnimationTimer = null
      resizeObserver?.disconnect()
      motionQuery.removeEventListener("change", onMotionChange)
      resizeObserver = undefined
      vehicleMarker?.remove()
      vehicleMarker = undefined
      markers.forEach((marker) => marker.remove())
      markers.clear()
      map?.remove()
      map = undefined
    }
  })

  $effect(() => {
    void journeys
    void activeJourneyId
    if (loaded) refresh()
  })

  $effect(() => {
    void activeMementoId
    if (loaded) {
      syncMarkers()
      if (activeMementoId) {
        flyToMemento(activeMementoId)
      }
    }
  })

  $effect(() => {
    if (!loaded || !currentStep) return
    if (currentStep.type === "intro" || currentStep.type === "summary") {
      fitActiveJourney()
      lastAnimatedStopIndex = -1
    } else if (currentStep.type === "day") {
      const journey = activeJourney()
      if (journey) {
        const dayMementos = journey.mementos.filter((m) => {
          const d = m.date.en || m.date.zh || m.date.ja || ""
          return d.startsWith(currentStep.date)
        })
        if (dayMementos.length) {
          const bounds = boundsOf(dayMementos.map((m) => m.coords))
          if (bounds) {
            map?.fitBounds(bounds, {
              padding: { top: 70, bottom: 90, left: 340, right: 60 },
              maxZoom: 12.5,
              duration: prefersReducedMotion ? 0 : 800,
            })
          }
        }
      }
    } else if (currentStep.type === "stop") {
      flyToMemento(currentStep.memento.id)
      const journey = activeJourney()
      if (journey && lastAnimatedStopIndex >= 0 && lastAnimatedStopIndex !== currentStep.stopIndex) {
        const prevMemento = journey.mementos[lastAnimatedStopIndex]
        if (prevMemento) {
          animateAlongRoute(prevMemento.coords, currentStep.coords)
        }
      }
      lastAnimatedStopIndex = currentStep.stopIndex
    }
  })

  $effect(() => {
    void theme
    if (loaded) applyTheme()
  })
</script>

<div bind:this={container} style="position: absolute; inset: 0;"></div>

<style>
  :global(.atlas-marker) {
    display: grid;
    position: relative;
    width: 2rem;
    height: 2rem;
    place-items: center;
    border: 2px solid #e8f4ef;
    border-radius: 999px;
    background: #123344;
    box-shadow:
      0 0 0 3px #171412aa,
      0 4px 12px #0008;
    color: #fff;
    font:
      700 0.75rem/1 ui-sans-serif,
      system-ui,
      sans-serif;
  }

  /* Keep the visible marker at 2rem (map density) but grow the actual hit
     area to the project's 44px touch-target minimum. */
  :global(.atlas-marker)::after {
    content: "";
    position: absolute;
    inset: calc((2rem - 2.75rem) / 2);
  }

  :global(.atlas-marker.is-dimmed) {
    opacity: 0.28;
  }

  :global(.atlas-marker.is-active) {
    z-index: 2;
    background: #ff9b72;
    color: #07131f;
    transform: scale(1.16);
  }

  :global(.atlas-marker i) {
    position: absolute;
    top: -0.45rem;
    right: -0.45rem;
    min-width: 1rem;
    padding: 0.15rem;
    border-radius: 999px;
    background: #e8f4ef;
    color: #123344;
    font-size: 0.6rem;
    font-style: normal;
  }

  :global(.atlas-traveling-dot) {
    width: 1.25rem;
    height: 1.25rem;
    border-radius: 999px;
    background: #ff7e47;
    border: 2.5px solid #ffffff;
    box-shadow: 0 0 12px #ff7e47, 0 0 24px #ff7e47aa;
    pointer-events: none;
    z-index: 10;
    animation: traveling-pulse 1.2s infinite ease-in-out;
  }

  @keyframes traveling-pulse {
    0%, 100% {
      transform: scale(1);
    }
    50% {
      transform: scale(1.3);
    }
  }
</style>
