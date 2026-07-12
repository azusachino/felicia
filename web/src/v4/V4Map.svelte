<script lang="ts">
  import maplibregl, { type StyleSpecification } from 'maplibre-gl'
  import 'maplibre-gl/dist/maplibre-gl.css'
  import { onMount } from 'svelte'
  import type { Coordinates, Journey, Memento, Theme } from '../data'

  let {
    journeys,
    activeJourneyId,
    activeMementoId,
    theme,
    onSelect,
  }: {
    journeys: Journey[]
    activeJourneyId: string | null
    activeMementoId: string | null
    theme: Theme
    onSelect: (id: string) => void
  } = $props()

  let container = $state<HTMLDivElement>()
  let map: maplibregl.Map | undefined
  let loaded = $state(false)
  let resizeObserver: ResizeObserver | undefined
  // eslint-disable-next-line svelte/prefer-svelte-reactivity -- imperative MapLibre marker cache
  const markers = new Map<string, maplibregl.Marker>()

  const mapStyle: StyleSpecification = {
    version: 8,
    sources: {
      dark: {
        type: 'raster',
        tiles: ['https://a.basemaps.cartocdn.com/dark_all/{z}/{x}/{y}.png'],
        tileSize: 256,
        attribution: '&copy; OpenStreetMap contributors &copy; CARTO',
      },
      light: {
        type: 'raster',
        tiles: ['https://a.basemaps.cartocdn.com/light_all/{z}/{x}/{y}.png'],
        tileSize: 256,
        attribution: '&copy; OpenStreetMap contributors &copy; CARTO',
      },
    },
    layers: [
      { id: 'base-dark', type: 'raster', source: 'dark' },
      { id: 'base-light', type: 'raster', source: 'light', layout: { visibility: 'none' } },
    ],
  }

  function routeGeoJson() {
    return {
      type: 'FeatureCollection' as const,
      features: journeys.flatMap((journey) => {
        const segments = journey.routeSegments?.length ? journey.routeSegments : [journey.route]
        return segments
          .filter((segment) => segment.length > 0)
          .map((coordinates, index) => ({
            type: 'Feature' as const,
            geometry: { type: 'LineString' as const, coordinates },
            properties: { journeyId: journey.id, segment: index },
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
    map.fitBounds(bounds, { padding: 80, maxZoom: 9, duration: 700 })
  }

  function markerElement(memento: Memento, index: number) {
    const button = document.createElement('button')
    button.type = 'button'
    button.className = 'v4-marker'
    button.setAttribute('aria-label', `${index + 1}. ${memento.title.en}`)
    button.innerHTML = `<span>${index + 1}</span>${
      memento.photos.length ? `<i>${memento.photos.length}</i>` : ''
    }`
    button.addEventListener('click', (event) => {
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
          anchor: 'center',
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
      const ownerJourney = journeys.find((journey) =>
        journey.mementos.some((item) => item.id === id),
      )
      marker.getElement().classList.toggle('is-active', id === activeMementoId)
      marker.getElement().classList.toggle('is-dimmed', ownerJourney?.id !== activeJourneyId)
    })
  }

  function refresh() {
    if (!loaded || !map) return
    ;(map.getSource('routes') as maplibregl.GeoJSONSource | undefined)?.setData(routeGeoJson())
    map.setFilter('route-active', ['==', ['get', 'journeyId'], activeJourneyId ?? ''])
    map.setFilter('route-active-glow', ['==', ['get', 'journeyId'], activeJourneyId ?? ''])
    rebuildMarkers()
    syncMarkers()
    fitActiveJourney()
  }

  function applyTheme() {
    if (!map) return
    map.setLayoutProperty('base-light', 'visibility', theme === 'light' ? 'visible' : 'none')
    map.setLayoutProperty('base-dark', 'visibility', theme === 'light' ? 'none' : 'visible')
  }

  onMount(() => {
    if (!container) return
    map = new maplibregl.Map({
      container,
      style: mapStyle,
      center: [138, 38],
      zoom: 4.5,
      attributionControl: false,
    })
    map.addControl(new maplibregl.AttributionControl({ compact: true }), 'bottom-left')
    resizeObserver = new ResizeObserver(() => map?.resize())
    resizeObserver.observe(container)

    map.on('load', () => {
      if (!map) return
      map.addSource('routes', { type: 'geojson', data: routeGeoJson() })
      map.addLayer({
        id: 'routes-all',
        type: 'line',
        source: 'routes',
        layout: { 'line-cap': 'round', 'line-join': 'round' },
        paint: { 'line-color': '#ff7b3a', 'line-width': 2, 'line-opacity': 0.22 },
      })
      map.addLayer({
        id: 'route-active-glow',
        type: 'line',
        source: 'routes',
        filter: ['==', ['get', 'journeyId'], activeJourneyId ?? ''],
        paint: { 'line-color': '#ff7b3a', 'line-width': 8, 'line-opacity': 0.14, 'line-blur': 4 },
      })
      map.addLayer({
        id: 'route-active',
        type: 'line',
        source: 'routes',
        filter: ['==', ['get', 'journeyId'], activeJourneyId ?? ''],
        layout: { 'line-cap': 'round', 'line-join': 'round' },
        paint: { 'line-color': '#ff7b3a', 'line-width': 3, 'line-opacity': 0.8 },
      })
      loaded = true
      refresh()
      applyTheme()
      map.resize()
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

  $effect(() => {
    void journeys
    void activeJourneyId
    if (loaded) refresh()
  })

  $effect(() => {
    void activeMementoId
    if (loaded) syncMarkers()
  })

  $effect(() => {
    void theme
    if (loaded) applyTheme()
  })
</script>

<div bind:this={container} style="position: absolute; inset: 0;"></div>

<style>
  :global(.v4-marker) {
    display: grid;
    position: relative;
    width: 2rem;
    height: 2rem;
    place-items: center;
    border: 2px solid #f7ead7;
    border-radius: 999px;
    background: #bd5724;
    box-shadow:
      0 0 0 3px #171412aa,
      0 4px 12px #0008;
    color: #fff;
    font:
      700 0.75rem/1 ui-sans-serif,
      system-ui,
      sans-serif;
    cursor: default;
  }

  :global(.v4-marker.is-dimmed) {
    opacity: 0.28;
  }

  :global(.v4-marker.is-active) {
    z-index: 2;
    background: #ff7b3a;
    transform: scale(1.16);
  }

  :global(.v4-marker i) {
    position: absolute;
    top: -0.45rem;
    right: -0.45rem;
    min-width: 1rem;
    padding: 0.15rem;
    border-radius: 999px;
    background: #f7ead7;
    color: #30251d;
    font-size: 0.6rem;
    font-style: normal;
  }
</style>
