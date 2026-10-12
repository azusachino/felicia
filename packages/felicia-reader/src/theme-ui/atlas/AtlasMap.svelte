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
  import { RouteTimeline, easeInOutCubic, haversineDistance, getDayColor, calculateDayCutoffs, type DayCutoff } from "../../playback/timeline"

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
    button.innerHTML = `<span>${index + 1}</span>`
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
    button.setAttribute("aria-label", role === "start" ? startText : endText)
    if (role === "start") {
      button.innerHTML = `
        <svg class="atlas-marker-icon" viewBox="0 0 24 24" fill="currentColor">
          <polygon points="6 3 20 12 6 21 6 3"></polygon>
        </svg>
      `
    } else {
      button.innerHTML = `
        <svg class="atlas-marker-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
          <path d="M4 15s1-1 4-1 5 2 8 2 4-1 4-1V3s-1 1-4 1-5-2-8-2-4 1-4 1z" fill="currentColor"></path>
          <line x1="4" y1="22" x2="4" y2="15"></line>
        </svg>
      `
    }
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
        const isActive = id === activeMementoId
        marker.getElement().classList.toggle("is-active", isActive)
        marker.getElement().classList.toggle("is-dimmed", ownerJourney?.id !== activeJourneyId)
        if (isActive && currentStep && "day" in currentStep) {
          const dayPalette = getDayColor(currentStep.day)
          marker.getElement().style.borderColor = "#ffffff"
          marker.getElement().style.boxShadow = `0 0 0 3px ${dayPalette.color}, 0 0 16px ${dayPalette.glow}, 0 4px 14px rgba(0, 0, 0, 0.7)`
        } else if (!isActive) {
          marker.getElement().style.borderColor = ""
          marker.getElement().style.boxShadow = ""
        }
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
  let cachedJourneyId: string | null = null
  let cachedTimeline: RouteTimeline | null = null
  let cachedDayCutoffs: DayCutoff[] = []

  function getTimelineForJourney(journey: Journey): { timeline: RouteTimeline; cutoffs: DayCutoff[] } | null {
    const segments = journey.routeSegments?.length ? journey.routeSegments : (journey.route?.length ? [journey.route] : [])
    const validSegments = segments.filter((s) => s.length >= 2)
    if (!validSegments.length) return null

    if (cachedTimeline && cachedJourneyId === journey.id) {
      return { timeline: cachedTimeline, cutoffs: cachedDayCutoffs }
    }

    const timeline = new RouteTimeline(validSegments, true)
    const cutoffs = calculateDayCutoffs(journey.mementos, timeline)
    cachedJourneyId = journey.id
    cachedTimeline = timeline
    cachedDayCutoffs = cutoffs
    return { timeline, cutoffs }
  }

  function emptyFeatureCollection() {
    return {
      type: "FeatureCollection" as const,
      features: [],
    }
  }

  function updateTraveledRoute(
    daySlices: Array<{ day: number; coords: Coordinates[] }>,
    activeDay: number = 1,
    isSummary: boolean = false
  ) {
    const source = map?.getSource("routes-traveled") as maplibregl.GeoJSONSource | undefined
    if (!source) return
    if (!daySlices.length) {
      source.setData(emptyFeatureCollection())
      return
    }

    const features = daySlices
      .filter((s) => s.coords.length >= 2)
      .map((slice) => {
        const palette = getDayColor(slice.day)
        const isCurrentDay = !isSummary && slice.day === activeDay
        // When overlapping/collapsed: past days render with subtle transparency and thinner line
        const opacity = isSummary ? 0.85 : isCurrentDay ? 0.95 : 0.32
        const glowOpacity = isSummary ? 0.25 : isCurrentDay ? 0.45 : 0.12
        const width = isSummary ? 3.8 : isCurrentDay ? 4.5 : 3.0
        const glowWidth = isCurrentDay ? 12 : 7

        return {
          type: "Feature" as const,
          geometry: {
            type: "LineString" as const,
            coordinates: slice.coords,
          },
          properties: {
            day: slice.day,
            color: palette.color,
            glowColor: palette.glow,
            opacity,
            glowOpacity,
            width,
            glowWidth,
            isCurrentDay: isCurrentDay ? 1 : 0,
          },
        }
      })

    source.setData({
      type: "FeatureCollection",
      features,
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

  function ensureVehicleMarker(coord: Coordinates, bearingDeg: number = 0, dayNumber: number = 1) {
    const dayPalette = getDayColor(dayNumber)
    if (!vehicleMarker && map) {
      const wrap = document.createElement("div")
      wrap.className = "atlas-vehicle-indicator"
      wrap.innerHTML = `
        <div class="atlas-vehicle-pulse" style="border-color: ${dayPalette.pulse};"></div>
        <div class="atlas-vehicle-body" style="background: ${dayPalette.color}; box-shadow: 0 0 14px ${dayPalette.glow}, 0 4px 12px rgba(0, 0, 0, 0.6);">
          <svg class="atlas-vehicle-heading" viewBox="0 0 24 24" fill="currentColor">
            <polygon points="12 2 19 21 12 17 5 21 12 2"></polygon>
          </svg>
        </div>
      `
      vehicleMarker = new maplibregl.Marker({
        element: wrap,
        anchor: "center",
      })
        .setLngLat(coord)
        .addTo(map)
      updateVehicleBearing(bearingDeg)
    } else if (vehicleMarker) {
      vehicleMarker.getElement().style.display = "grid"
      vehicleMarker.setLngLat(coord)
      updateVehicleBearing(bearingDeg)

      const body = vehicleMarker.getElement().querySelector(".atlas-vehicle-body") as HTMLElement | null
      const pulse = vehicleMarker.getElement().querySelector(".atlas-vehicle-pulse") as HTMLElement | null
      if (body) {
        body.style.background = dayPalette.color
        body.style.boxShadow = `0 0 14px ${dayPalette.glow}, 0 4px 12px rgba(0, 0, 0, 0.6)`
      }
      if (pulse) {
        pulse.style.borderColor = dayPalette.pulse
      }
    }
  }

  function updateVehicleBearing(bearingDeg: number) {
    if (!vehicleMarker) return
    const arrow = vehicleMarker.getElement().querySelector(".atlas-vehicle-heading") as HTMLElement | null
    if (arrow) {
      arrow.style.transform = `rotate(${Math.round(bearingDeg)}deg)`
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
    targetDay: number = 1,
    customDurationMs?: number
  ) {
    if (!map) return
    const journey = activeJourney()
    if (!journey) return

    const info = getTimelineForJourney(journey)
    if (!info || info.timeline.totalDistance <= 0.05) {
      if (destinationMementoId) flyToMemento(destinationMementoId)
      else map.flyTo({ center: toCoords, zoom: 12.2, pitch: 0, bearing: 0, padding: { top: 60, bottom: 80, left: 340, right: 60 }, duration: 600 })
      return
    }

    const { timeline, cutoffs } = info
    const targetDist = timeline.distanceAtCoordinate(toCoords)

    if (prefersReducedMotion) {
      if (activeAnimationTimer) {
        activeAnimationTimer.stop()
        activeAnimationTimer = null
      }
      const fraction = targetDist / timeline.totalDistance
      const slices = timeline.sliceByDays(fraction, cutoffs)
      updateTraveledRoute(slices, targetDay, false)
      ensureVehicleMarker(toCoords, 0, targetDay)
      if (destinationMementoId) flyToMemento(destinationMementoId)
      else map.flyTo({ center: toCoords, zoom: 12.2, pitch: 0, bearing: 0, padding: { top: 60, bottom: 80, left: 340, right: 60 }, duration: 600 })
      return
    }

    try {
      const startDist = timeline.distanceAtCoordinate(fromCoords)
      const travelDist = targetDist - startDist

      // If already at destination or reverse seek, reveal directly
      if (travelDist <= 0.05) {
        const fraction = targetDist / timeline.totalDistance
        const slices = timeline.sliceByDays(fraction, cutoffs)
        updateTraveledRoute(slices, targetDay, false)
        ensureVehicleMarker(toCoords, 0, targetDay)
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

      ensureVehicleMarker(fromCoords, 0, targetDay)

      // Dynamic duration based on travel distance:
      // When auto-playing: paced duration matching step dwell (3.2s - 8.5s).
      // When stepping manually: brisk, smooth driving animation (1.2s - 2.2s).
      const calculatedDuration = Math.round(
        Math.min(8500, Math.max(3200, 2600 + Math.sqrt(Math.max(0, travelDist)) * 390))
      )
      const durationMs =
        customDurationMs ??
        (isPlaying ? calculatedDuration : Math.min(2200, Math.max(1200, Math.round(calculatedDuration * 0.35))))

      // Synchronized cinematic transition:
      // MapLibre's built-in geodesic flight curve (van Wijk & Nuij) smoothly adjusts
      // altitude and pans across the landscape to arrive at toCoords in perfect unison with the car.
      map.flyTo({
        center: toCoords,
        zoom: 12.2,
        pitch: 0,
        bearing: 0,
        padding: { top: 60, bottom: 80, left: 340, right: 60 },
        duration: durationMs,
        curve: travelDist > 30 ? 1.25 : 0.8,
        speed: 1.0,
        easing: (t) => (t < 0.5 ? 4 * t * t * t : 1 - Math.pow(-2 * t + 2, 3) / 2),
        essential: true,
      })

      const startTime = performance.now()

      activeAnimationTimer = timer(() => {
        const elapsed = performance.now() - startTime
        const rawProgress = Math.min(1, elapsed / durationMs)

        // Constant speed along polyline (linear velocity km/s)
        const currentTotalDist = startDist + rawProgress * travelDist
        const fraction = currentTotalDist / timeline.totalDistance

        const { currentPos, bearing } = timeline.sliceAtFraction(fraction)
        const slices = timeline.sliceByDays(fraction, cutoffs)

        // Location indicator smoothly tracks vehicle position along the route with forward heading & day tint
        ensureVehicleMarker(currentPos, bearing, targetDay)
        updateTraveledRoute(slices, targetDay, false)

        if (rawProgress >= 1) {
          activeAnimationTimer?.stop()
          activeAnimationTimer = null
          const finalFraction = targetDist / timeline.totalDistance
          const finalSlices = timeline.sliceByDays(finalFraction, cutoffs)
          const finalState = timeline.sliceAtFraction(finalFraction)
          updateTraveledRoute(finalSlices, targetDay, false)
          ensureVehicleMarker(toCoords, finalState.bearing, targetDay)
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

      // Traveled route (bold, glowing, progressive path drawing with per-day color and history transparency)
      map.addLayer({
        id: "route-traveled-glow",
        type: "line",
        source: "routes-traveled",
        layout: { "line-cap": "round", "line-join": "round" },
        paint: {
          "line-color": ["coalesce", ["get", "glowColor"], "#ff9b72"],
          "line-width": ["coalesce", ["get", "glowWidth"], 10],
          "line-opacity": ["coalesce", ["get", "glowOpacity"], 0.35],
          "line-blur": 5,
        },
      })
      map.addLayer({
        id: "route-traveled",
        type: "line",
        source: "routes-traveled",
        layout: { "line-cap": "round", "line-join": "round" },
        paint: {
          "line-color": ["coalesce", ["get", "color"], "#ff7e47"],
          "line-width": ["coalesce", ["get", "width"], 4.5],
          "line-opacity": ["coalesce", ["get", "opacity"], 0.95],
        },
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
        const info = getTimelineForJourney(journey)
        if (info) {
          const slices = info.timeline.sliceByDays(1.0, info.cutoffs)
          updateTraveledRoute(slices, 1, true)
        }
      }
      lastAnimatedCoords = null
    } else if (currentStep.type === "day") {
      if (activeAnimationTimer) {
        activeAnimationTimer.stop()
        activeAnimationTimer = null
      }
      if (journey) {
        const info = getTimelineForJourney(journey)
        const originCoord = info?.timeline.startCoord ?? journey.mementos[0]?.coords

        if (currentStep.day === 1 && originCoord) {
          // Day 1 starts at Tokyo departure origin, NOT at Kujuppama destination!
          updateTraveledRoute([], 1)
          ensureVehicleMarker(originCoord, 0, currentStep.day)
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
          // For Day > 1, show route up to the start of this day
          if (info && currentStep.day > 1) {
            const prevCutoff = info.cutoffs[currentStep.day - 2]?.cutoffFraction ?? 0
            const slices = info.timeline.sliceByDays(prevCutoff, info.cutoffs)
            updateTraveledRoute(slices, currentStep.day, false)
          }
          if (lastAnimatedCoords) {
            ensureVehicleMarker(lastAnimatedCoords, 0, currentStep.day)
          }
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
        updateTraveledRoute([], currentStep.day)
        ensureVehicleMarker(currentStep.coords, 0, currentStep.day)
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
          animateAlongRoute(from, currentStep.coords, undefined, currentStep.day)
        }
        lastAnimatedCoords = currentStep.coords
      }
    } else if (currentStep.type === "stop") {
      if (journey) {
        const from = resolveLegOrigin(journey, currentStep.coords)
        animateAlongRoute(from, currentStep.coords, currentStep.memento.id, currentStep.day)
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
    border: 2px solid #e2e8f0;
    border-radius: 999px;
    background: #0f172a;
    box-shadow:
      0 0 0 2px rgba(15, 23, 42, 0.6),
      0 4px 12px rgba(0, 0, 0, 0.5);
    color: #f8fafc;
    font:
      700 0.8rem/1 ui-sans-serif,
      system-ui,
      sans-serif;
    transition: background 0.2s ease, border-color 0.2s ease, box-shadow 0.2s ease;
  }

  /* Keep visible marker at 2rem but grow hit area to touch-target minimum */
  :global(.atlas-marker)::after {
    content: "";
    position: absolute;
    inset: calc((2rem - 2.75rem) / 2);
  }

  :global(.atlas-marker.is-dimmed) {
    opacity: 0.32;
  }

  :global(.atlas-marker.is-active) {
    z-index: 20;
    background: #ff7e47;
    border-color: #ffffff;
    color: #ffffff;
    box-shadow:
      0 0 0 3px #ff7e47,
      0 0 16px rgba(255, 126, 71, 0.8),
      0 4px 14px rgba(0, 0, 0, 0.7);
  }

  :global(.atlas-marker-start) {
    background: #065f46;
    border-color: #34d399;
    color: #ffffff;
  }

  :global(.atlas-marker-start.is-active) {
    background: #059669;
    border-color: #6ee7b7;
    box-shadow:
      0 0 0 3px #059669,
      0 0 16px rgba(52, 211, 153, 0.8),
      0 4px 14px rgba(0, 0, 0, 0.7);
  }

  :global(.atlas-marker-end) {
    background: #78350f;
    border-color: #fbbf24;
    color: #ffffff;
  }

  :global(.atlas-marker-end.is-active) {
    background: #d97706;
    border-color: #fde68a;
    box-shadow:
      0 0 0 3px #d97706,
      0 0 16px rgba(251, 191, 36, 0.8),
      0 4px 14px rgba(0, 0, 0, 0.7);
  }

  :global(.atlas-marker-icon) {
    display: block;
    width: 0.85rem;
    height: 0.85rem;
  }

  :global(.atlas-vehicle-indicator) {
    display: grid;
    place-items: center;
    width: 2.5rem;
    height: 2.5rem;
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
      0 0 14px rgba(255, 126, 71, 0.9),
      0 4px 12px rgba(0, 0, 0, 0.6);
    color: #ffffff;
    position: relative;
    z-index: 2;
  }

  :global(.atlas-vehicle-heading) {
    display: block;
    width: 0.9rem;
    height: 0.9rem;
    transform-origin: center center;
    transition: transform 0.1s linear;
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
