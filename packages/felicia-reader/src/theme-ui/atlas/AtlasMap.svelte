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
  import { RouteTimeline, easeInOutCubic, haversineDistance } from "../../playback/timeline"

  let {
    journeys,
    activeJourneyId,
    activeMementoId,
    currentStep = null,
    isPlaying = false,
    lang,
    theme,
    onSelect,
  }: {
    journeys: Journey[]
    activeJourneyId: string | null
    activeMementoId: string | null
    currentStep?: PlaybackStep | null
    isPlaying?: boolean
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
      // 3. Dark Canvas reference overlay: clean town/city labels and subtle boundaries
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
      // Dark theme: Dark base + Subtle Reference Labels
      { id: "layer-dark-base", type: "raster", source: "dark-base", layout: { visibility: "visible" } },
      { id: "layer-dark-ref", type: "raster", source: "dark-canvas-ref", layout: { visibility: "visible" }, paint: { "raster-opacity": 0.75 } },
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

  function waypointMarkerElement(journey: Journey, role: "start" | "end", coord: Coordinates) {
    const button = document.createElement("button")
    button.type = "button"
    button.className = `atlas-marker atlas-marker-special atlas-marker-${role}`
    const startText = lang === "ja" ? "出発" : lang === "zh" ? "出发" : "Start"
    const endText = lang === "ja" ? "到着" : lang === "zh" ? "到达" : "Goal"
    const icon = role === "start" ? "🚗" : "🏁"
    const label = `${icon} ${role === "start" ? startText : endText}`
    button.setAttribute("aria-label", label)
    button.innerHTML = `<span>${icon}</span>`
    button.addEventListener("click", (event) => {
      event.stopPropagation()
      if (map?.isMoving()) map.stop()
      map?.flyTo({
        center: coord,
        zoom: 12.2,
        pitch: 0,
        bearing: 0,
        padding: { top: 60, bottom: 80, left: 340, right: 60 },
        duration: prefersReducedMotion ? 0 : 1000,
      })
    })
    return button
  }

  function rebuildMarkers() {
    if (!map) return
    markers.forEach((marker) => marker.remove())
    markers.clear()
    journeys.forEach((journey) => {
      // 1. Memento markers (1, 2, 3...)
      journey.mementos.forEach((memento, index) => {
        const marker = new maplibregl.Marker({
          element: markerElement(memento, index),
          anchor: "center",
        })
          .setLngLat(memento.coords)
          .addTo(map!)
        markers.set(memento.id, marker)
      })

      // 2. Journey Start & End waypoints (special circular milestone markers)
      const segments = journey.routeSegments?.length ? journey.routeSegments : (journey.route?.length ? [journey.route] : [])
      const validSegments = segments.filter((s) => s.length >= 2)
      const originCoord = validSegments[0]?.[0]
      const lastSeg = validSegments.at(-1)
      const destCoord = lastSeg?.at(-1)

      if (originCoord) {
        // If destCoord is very close to originCoord (< 0.005 deg / ~500m), offset slightly so both circles sit side-by-side
        const isLoopTrip = destCoord && geoDistance(originCoord, destCoord) < 0.005
        const startMarker = new maplibregl.Marker({
          element: waypointMarkerElement(journey, "start", originCoord),
          anchor: "center",
          offset: isLoopTrip ? [-18, 0] : [0, 0],
        })
          .setLngLat(originCoord)
          .addTo(map!)
        markers.set(`${journey.id}:start`, startMarker)

        if (destCoord) {
          const endMarker = new maplibregl.Marker({
            element: waypointMarkerElement(journey, "end", destCoord),
            anchor: "center",
            offset: isLoopTrip ? [18, 0] : [0, 0],
          })
            .setLngLat(destCoord)
            .addTo(map!)
          markers.set(`${journey.id}:end`, endMarker)
        }
      }
    })
    syncMarkers()
  }

  function syncMarkers() {
    markers.forEach((marker, id) => {
      if (id.endsWith(":start") || id.endsWith(":end")) {
        const journeyId = id.split(":")[0]
        const isCurrentStepWaypoint =
          currentStep?.type === "waypoint" &&
          ((currentStep.role === "start" && id.endsWith(":start")) ||
           (currentStep.role === "end" && id.endsWith(":end")))
        marker.getElement().classList.toggle("is-active", isCurrentStepWaypoint)
        marker.getElement().classList.toggle("is-dimmed", journeyId !== activeJourneyId)
      } else {
        const ownerJourney = journeys.find((journey) => journey.mementos.some((item) => item.id === id))
        marker.getElement().classList.toggle("is-active", id === activeMementoId)
        marker.getElement().classList.toggle("is-dimmed", ownerJourney?.id !== activeJourneyId)
      }
    })
    updateSpokes()
  }

  function flyToMemento(id: string | null) {
    if (!map || !id) return
    const m = journeys.flatMap((j) => j.mementos).find((item) => item.id === id)
    if (!m) return
    if (map.isMoving()) map.stop()
    map.flyTo({
      center: m.coords,
      zoom: 12.2,
      pitch: 0,
      bearing: 0,
      speed: 1.0,
      curve: 1.2,
      essential: true,
      padding: { top: 60, bottom: 80, left: 340, right: 60 },
    })
  }

  let lastAnimatedCoords: Coordinates | null = null
  let currentTraveledCoords: Coordinates[] = []

  function emptyFeatureCollection() {
    return {
      type: "FeatureCollection" as const,
      features: [],
    }
  }

  function updateTraveledRoute(coords: Coordinates[]) {
    currentTraveledCoords = coords
    const source = map?.getSource("routes-traveled") as maplibregl.GeoJSONSource | undefined
    if (!source) return
    if (coords.length < 2) {
      source.setData(emptyFeatureCollection())
      return
    }
    source.setData({
      type: "FeatureCollection",
      features: [
        {
          type: "Feature",
          geometry: {
            type: "LineString",
            coordinates: coords,
          },
          properties: {},
        },
      ],
    })
  }

  function spokesGeoJson() {
    const journey = activeJourney()
    if (!journey) {
      return {
        lines: emptyFeatureCollection(),
        points: emptyFeatureCollection(),
      }
    }

    const lineFeatures: any[] = []
    const pointFeatures: any[] = []

    journey.mementos.forEach((m) => {
      if (m.originalCoords) {
        const distKm = haversineDistance(m.coords, m.originalCoords)
        // If distance > 30 meters, show spoke connector to photo EXIF location
        if (distKm > 0.03) {
          const isActive = m.id === activeMementoId
          lineFeatures.push({
            type: "Feature",
            geometry: {
              type: "LineString",
              coordinates: [m.coords, m.originalCoords],
            },
            properties: {
              mementoId: m.id,
              active: isActive ? 1 : 0,
            },
          })
          pointFeatures.push({
            type: "Feature",
            geometry: {
              type: "Point",
              coordinates: m.originalCoords,
            },
            properties: {
              mementoId: m.id,
              active: isActive ? 1 : 0,
            },
          })
        }
      }
    })

    return {
      lines: {
        type: "FeatureCollection" as const,
        features: lineFeatures,
      },
      points: {
        type: "FeatureCollection" as const,
        features: pointFeatures,
      },
    }
  }

  function updateSpokes() {
    if (!loaded || !map) return
    const data = spokesGeoJson()
    ;(map.getSource("routes-spokes") as maplibregl.GeoJSONSource | undefined)?.setData(data.lines)
    ;(map.getSource("routes-photo-spots") as maplibregl.GeoJSONSource | undefined)?.setData(data.points)
  }

  function ensureVehicleMarker(coord: Coordinates) {
    if (!vehicleMarker && map) {
      const wrap = document.createElement("div")
      wrap.className = "atlas-vehicle-indicator"
      wrap.innerHTML = `
        <div class="atlas-vehicle-pulse"></div>
        <div class="atlas-vehicle-body">🚗</div>
      `
      vehicleMarker = new maplibregl.Marker({
        element: wrap,
        anchor: "center",
      })
        .setLngLat(coord)
        .addTo(map)
    } else if (vehicleMarker) {
      vehicleMarker.getElement().style.display = "grid"
      vehicleMarker.setLngLat(coord)
    }
  }

  function hideVehicleMarker() {
    if (vehicleMarker) {
      vehicleMarker.getElement().style.display = "none"
    }
  }

  function resolveLegOrigin(journey: Journey, currentCoords: Coordinates): Coordinates {
    if (lastAnimatedCoords) return lastAnimatedCoords

    const mementos = journey.mementos
    const mementoIdx = mementos.findIndex(
      (m) => Math.abs(m.coords[0] - currentCoords[0]) < 0.0001 && Math.abs(m.coords[1] - currentCoords[1]) < 0.0001
    )
    if (mementoIdx > 0) {
      return mementos[mementoIdx - 1].coords
    }

    const segments = journey.routeSegments?.length ? journey.routeSegments : (journey.route?.length ? [journey.route] : [])
    const validSegments = segments.filter((s) => s.length >= 2)
    return validSegments[0]?.[0] ?? currentCoords
  }

  function animateAlongRoute(
    fromCoords: Coordinates,
    toCoords: Coordinates,
    destinationMementoId?: string,
    customDurationMs?: number
  ) {
    if (!map) return
    if (prefersReducedMotion || !isPlaying) {
      if (activeAnimationTimer) {
        activeAnimationTimer.stop()
        activeAnimationTimer = null
      }
      const journey = activeJourney()
      if (journey) {
        const segments = journey.routeSegments?.length ? journey.routeSegments : [journey.route]
        const validSegments = segments.filter((s) => s.length >= 2)
        if (validSegments.length) {
          const timeline = new RouteTimeline(validSegments, true)
          const targetDist = timeline.distanceAtCoordinate(toCoords)
          const finalState = timeline.sliceAtFraction(targetDist / timeline.totalDistance)
          updateTraveledRoute(finalState.traveled)
        }
      }
      ensureVehicleMarker(toCoords)
      if (destinationMementoId) flyToMemento(destinationMementoId)
      else map.flyTo({ center: toCoords, zoom: 12.2, pitch: 0, bearing: 0, padding: { top: 60, bottom: 80, left: 340, right: 60 }, duration: 600 })
      return
    }
    const journey = activeJourney()
    if (!journey) return

    const segments = journey.routeSegments?.length ? journey.routeSegments : [journey.route]
    const validSegments = segments.filter((s) => s.length >= 2)
    if (!validSegments.length) return

    try {
      const timeline = new RouteTimeline(validSegments, true)
      if (timeline.totalDistance <= 0.05) return

      const startDist = timeline.distanceAtCoordinate(fromCoords)
      const targetDist = timeline.distanceAtCoordinate(toCoords)
      const travelDist = targetDist - startDist

      // If already at destination or reverse seek, reveal directly
      if (travelDist <= 0.05) {
        const finalState = timeline.sliceAtFraction(targetDist / timeline.totalDistance)
        updateTraveledRoute(finalState.traveled)
        ensureVehicleMarker(toCoords)
        if (destinationMementoId) flyToMemento(destinationMementoId)
        else {
          map.flyTo({
            center: toCoords,
            zoom: 12.2,
            pitch: 0,
            bearing: 0,
            padding: { top: 60, bottom: 80, left: 340, right: 60 },
            duration: 600,
          })
        }
        return
      }

      if (activeAnimationTimer) {
        activeAnimationTimer.stop()
        activeAnimationTimer = null
      }
      if (map.isMoving()) {
        map.stop()
      }

      ensureVehicleMarker(fromCoords)

      // Dynamic duration based on travel distance:
      // Short hops (~2-10km): ~3.2s - 4.2s.
      // Long highway legs (~197km): ~7.5s - 8.2s.
      const calculatedDuration = Math.round(
        Math.min(8500, Math.max(3200, 2600 + Math.sqrt(Math.max(0, travelDist)) * 390))
      )
      const durationMs = customDurationMs ?? calculatedDuration

      // Pre-frame leg bounds so all map tiles load cleanly with zero request cancellations
      const legBounds = boundsOf([fromCoords, toCoords])
      if (legBounds) {
        map.fitBounds(legBounds, {
          padding: { top: 90, bottom: 90, left: 370, right: 90 },
          maxZoom: travelDist > 30 ? 10.5 : 12.2,
          pitch: 0,
          bearing: 0,
          duration: 700,
        })
      }

      const startTime = performance.now()
      let cameraFocusedOnArrival = false

      activeAnimationTimer = timer(() => {
        const elapsed = performance.now() - startTime
        const rawProgress = Math.min(1, elapsed / durationMs)

        // STATIC SPEED (constant linear velocity km/s, no cubic easing whiplash)
        const currentTotalDist = startDist + rawProgress * travelDist
        const fraction = currentTotalDist / timeline.totalDistance

        const { traveled, currentPos } = timeline.sliceAtFraction(fraction)

        // Location indicator smoothly tracks vehicle position along the route
        ensureVehicleMarker(currentPos)
        updateTraveledRoute(traveled)

        // Smoothly focus on destination when vehicle approaches (last 15% of travel)
        if (rawProgress >= 0.85 && !cameraFocusedOnArrival) {
          cameraFocusedOnArrival = true
          if (destinationMementoId) {
            flyToMemento(destinationMementoId)
          } else {
            map?.flyTo({
              center: toCoords,
              zoom: 12.2,
              pitch: 0,
              bearing: 0,
              padding: { top: 60, bottom: 80, left: 340, right: 60 },
              duration: 800,
            })
          }
        }

        if (rawProgress >= 1) {
          activeAnimationTimer?.stop()
          activeAnimationTimer = null
          const finalState = timeline.sliceAtFraction(targetDist / timeline.totalDistance)
          updateTraveledRoute(finalState.traveled)
          ensureVehicleMarker(toCoords)

          if (destinationMementoId) {
            flyToMemento(destinationMementoId)
          } else {
            map?.flyTo({
              center: toCoords,
              zoom: 12.2,
              pitch: 0,
              bearing: 0,
              padding: { top: 60, bottom: 80, left: 340, right: 60 },
              duration: 600,
            })
          }
        }
      })
    } catch {
      // Graceful fallback if timeline fails
      if (destinationMementoId) flyToMemento(destinationMementoId)
      else map?.flyTo({ center: toCoords, zoom: 12.2, pitch: 0, bearing: 0 })
    }
  }

  function refresh() {
    if (!loaded || !map) return
    ;(map.getSource("routes") as maplibregl.GeoJSONSource | undefined)?.setData(routeGeoJson())
    map.setFilter("route-active", ["==", ["get", "journeyId"], activeJourneyId ?? ""])
    if (map.getLayer("route-active-glow")) {
      map.setFilter("route-active-glow", ["==", ["get", "journeyId"], activeJourneyId ?? ""])
    }
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
      map.addSource("routes-traveled", { type: "geojson", data: emptyFeatureCollection() })

      // Planned / Background routes (subtle translucent)
      map.addLayer({
        id: "routes-all",
        type: "line",
        source: "routes",
        layout: { "line-cap": "round", "line-join": "round" },
        paint: { "line-color": "#94a3b8", "line-width": 1.5, "line-opacity": 0.12 },
      })
      map.addLayer({
        id: "route-active",
        type: "line",
        source: "routes",
        filter: ["==", ["get", "journeyId"], activeJourneyId ?? ""],
        layout: { "line-cap": "round", "line-join": "round" },
        paint: { "line-color": "#ff9b72", "line-width": 2.5, "line-opacity": 0.35, "line-dasharray": [2, 2] },
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
          "line-width": 2.5,
          "line-dasharray": [2, 3],
          "line-opacity": 0.35,
        },
      })

      // Traveled route (bold, glowing, progressive path drawing)
      map.addLayer({
        id: "route-traveled-glow",
        type: "line",
        source: "routes-traveled",
        layout: { "line-cap": "round", "line-join": "round" },
        paint: { "line-color": "#ff9b72", "line-width": 10, "line-opacity": 0.35, "line-blur": 5 },
      })
      map.addLayer({
        id: "route-traveled",
        type: "line",
        source: "routes-traveled",
        layout: { "line-cap": "round", "line-join": "round" },
        paint: { "line-color": "#ff7e47", "line-width": 4.5, "line-opacity": 0.95 },
      })

      // Photo scenic viewpoint connectors (Dawarich-style route snapping indicator)
      map.addSource("routes-spokes", { type: "geojson", data: emptyFeatureCollection() })
      map.addSource("routes-photo-spots", { type: "geojson", data: emptyFeatureCollection() })

      map.addLayer({
        id: "routes-spokes-line",
        type: "line",
        source: "routes-spokes",
        layout: { "line-cap": "round", "line-join": "round" },
        paint: {
          "line-color": "#ff9b72",
          "line-width": ["case", ["==", ["get", "active"], 1], 2, 1.2],
          "line-dasharray": [2, 2],
          "line-opacity": ["case", ["==", ["get", "active"], 1], 0.85, 0.4],
        },
      })

      map.addLayer({
        id: "routes-photo-spots-circle",
        type: "circle",
        source: "routes-photo-spots",
        paint: {
          "circle-radius": ["case", ["==", ["get", "active"], 1], 4.5, 3],
          "circle-color": "#ff7e47",
          "circle-stroke-width": 1.5,
          "circle-stroke-color": "#ffffff",
          "circle-opacity": ["case", ["==", ["get", "active"], 1], 0.95, 0.5],
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
      // Only trigger flyTo when user directly selects a memento outside of active playback steps
      if (activeMementoId && !currentStep) {
        flyToMemento(activeMementoId)
      }
    }
  })

  $effect(() => {
    if (!loaded) return
    syncMarkers()
    if (!currentStep) return
    const journey = activeJourney()

    if (currentStep.type === "intro") {
      if (activeAnimationTimer) {
        activeAnimationTimer.stop()
        activeAnimationTimer = null
      }
      hideVehicleMarker()
      fitActiveJourney()
      updateTraveledRoute([])
      lastAnimatedCoords = null
    } else if (currentStep.type === "summary") {
      if (activeAnimationTimer) {
        activeAnimationTimer.stop()
        activeAnimationTimer = null
      }
      hideVehicleMarker()
      fitActiveJourney()
      if (journey) {
        const allRouteCoords = journey.routeSegments?.length ? journey.routeSegments.flat() : journey.route
        updateTraveledRoute(allRouteCoords)
      }
      lastAnimatedCoords = null
    } else if (currentStep.type === "day") {
      if (activeAnimationTimer) {
        activeAnimationTimer.stop()
        activeAnimationTimer = null
      }
      if (journey) {
        const segments = journey.routeSegments?.length ? journey.routeSegments : (journey.route?.length ? [journey.route] : [])
        const validSegments = segments.filter((s) => s.length >= 2)
        const originCoord = validSegments[0]?.[0] ?? journey.mementos[0]?.coords

        if (currentStep.day === 1 && originCoord) {
          // Day 1 starts at Tokyo departure origin, NOT at Kujuppama destination!
          updateTraveledRoute([])
          hideVehicleMarker()
          map?.flyTo({
            center: originCoord,
            zoom: 12.0,
            pitch: 0,
            bearing: 0,
            padding: { top: 60, bottom: 80, left: 340, right: 60 },
            duration: prefersReducedMotion ? 0 : 1200,
          })
          lastAnimatedCoords = originCoord
        } else {
          hideVehicleMarker()
          const dayMementos = journey.mementos.filter((m) => {
            const d = m.date.en || m.date.ja || m.date.zh || ""
            return d.startsWith(currentStep.date)
          })
          if (dayMementos.length) {
            const bounds = boundsOf(dayMementos.map((m) => m.coords))
            if (bounds) {
              map?.fitBounds(bounds, {
                padding: { top: 80, bottom: 90, left: 360, right: 80 },
                maxZoom: 12.0,
                pitch: 0,
                bearing: 0,
                duration: prefersReducedMotion ? 0 : 1200,
              })
            }
          }
        }
      }
    } else if (currentStep.type === "waypoint") {
      if (currentStep.role === "start") {
        if (activeAnimationTimer) {
          activeAnimationTimer.stop()
          activeAnimationTimer = null
        }
        updateTraveledRoute([currentStep.coords])
        hideVehicleMarker()
        map?.flyTo({
          center: currentStep.coords,
          zoom: 12.0,
          pitch: 0,
          bearing: 0,
          padding: { top: 60, bottom: 80, left: 340, right: 60 },
          duration: prefersReducedMotion ? 0 : 1000,
        })
        lastAnimatedCoords = currentStep.coords
      } else if (currentStep.role === "end") {
        if (journey) {
          const from = resolveLegOrigin(journey, currentStep.coords)
          animateAlongRoute(from, currentStep.coords, undefined)
        }
        lastAnimatedCoords = currentStep.coords
      }
    } else if (currentStep.type === "stop") {
      if (journey) {
        const from = resolveLegOrigin(journey, currentStep.coords)
        animateAlongRoute(from, currentStep.coords, currentStep.memento.id)
      } else {
        flyToMemento(currentStep.memento.id)
      }
      lastAnimatedCoords = currentStep.coords
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
    position: absolute;
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
    z-index: 20;
    background: #ff9b72;
    color: #07131f;
    box-shadow:
      0 0 0 3px #ff9b72,
      0 0 16px rgba(255, 155, 114, 0.8),
      0 4px 14px rgba(0, 0, 0, 0.7);
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

  :global(.atlas-marker-special) {
    border-width: 2.5px;
  }

  :global(.atlas-marker-start) {
    background: #065f46;
    border-color: #34d399;
    color: #ffffff;
  }

  :global(.atlas-marker-end) {
    background: #78350f;
    border-color: #fbbf24;
    color: #ffffff;
  }

  :global(.atlas-marker-special span) {
    font-size: 0.85rem;
    line-height: 1;
  }

  :global(.atlas-vehicle-indicator) {
    display: grid;
    place-items: center;
    width: 2.4rem;
    height: 2.4rem;
    pointer-events: none;
    z-index: 50;
    position: absolute;
  }

  :global(.atlas-vehicle-body) {
    width: 2rem;
    height: 2rem;
    display: grid;
    place-items: center;
    border-radius: 999px;
    background: #ff7e47;
    border: 2px solid #ffffff;
    box-shadow:
      0 0 14px #ff7e47,
      0 4px 12px rgba(0, 0, 0, 0.6);
    font-size: 1rem;
    line-height: 1;
    position: relative;
    z-index: 2;
  }

  :global(.atlas-vehicle-pulse) {
    position: absolute;
    inset: -6px;
    border-radius: 999px;
    border: 2px solid #ff9b72;
    opacity: 0.8;
    animation: vehicle-radar-pulse 1.4s infinite ease-out;
    pointer-events: none;
    z-index: 1;
  }

  @keyframes vehicle-radar-pulse {
    0% {
      transform: scale(0.85);
      opacity: 0.9;
    }
    100% {
      transform: scale(1.6);
      opacity: 0;
    }
  }
</style>
