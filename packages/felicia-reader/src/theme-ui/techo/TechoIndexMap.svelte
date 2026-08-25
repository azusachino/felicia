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
    map.fitBounds(bounds, { padding: 48, maxZoom: 3.2, duration: 500 })
  }

  function refreshData() {
    if (!map) return
    ;(map.getSource("journeys") as maplibregl.GeoJSONSource | undefined)?.setData(routeData())
    ;(map.getSource("places") as maplibregl.GeoJSONSource | undefined)?.setData(placeData())
    fitWorld()
  }

  onMount(() => {
    map = new maplibregl.Map({
      container,
      style,
      center: [10, 30],
      zoom: 1.6,
      attributionControl: {},
    })
    map.addControl(new maplibregl.NavigationControl({ showCompass: false }), "top-right")
    map.on("click", "journey-routes", (event) => {
      const id = event.features?.[0]?.properties?.id
      if (typeof id === "string") onSelect(id)
    })
    map.on("click", "journey-places", (event) => {
      const id = event.features?.[0]?.properties?.id
      if (typeof id === "string") onSelect(id)
    })
    map.on("mouseenter", "journey-routes", () => map?.getCanvas().classList.add("is-clickable"))
    map.on("mouseleave", "journey-routes", () => map?.getCanvas().classList.remove("is-clickable"))
    map.on("mouseenter", "journey-places", () => map?.getCanvas().classList.add("is-clickable"))
    map.on("mouseleave", "journey-places", () => map?.getCanvas().classList.remove("is-clickable"))

    resizeObserver = new ResizeObserver(() => map?.resize())
    resizeObserver.observe(container)

    map.on("load", () => {
      if (!map) return
      map.addSource("journeys", { type: "geojson", data: routeData() })
      map.addLayer({
        id: "journey-routes",
        type: "line",
        source: "journeys",
        layout: { "line-cap": "round", "line-join": "round" },
        paint: {
          "line-color": ["case", ["get", "selected"], "#ff9b72", "#7aa8a6"],
          "line-width": ["case", ["get", "selected"], 4, 2],
          "line-opacity": ["case", ["get", "selected"], 0.95, 0.5],
        },
      })
      map.addSource("places", { type: "geojson", data: placeData() })
      map.addLayer({
        id: "journey-places",
        type: "circle",
        source: "places",
        paint: {
          "circle-color": "#ff9b72",
          "circle-radius": 5,
          "circle-stroke-color": "#fff8ed",
          "circle-stroke-width": 2,
        },
      })
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
          "text-color": "#4c3a27",
          "text-halo-color": "#fff8ed",
          "text-halo-width": 1.5,
        },
      })
      map.resize()
      fitWorld()
      loaded = true
    })

    return () => {
      resizeObserver?.disconnect()
      resizeObserver = undefined
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
</style>
