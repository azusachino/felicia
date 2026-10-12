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
  import { RouteTimeline, calculateBearing, lerpAngle, easeInOutCubic } from "../../playback/timeline"

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
    button.className = `atlas-marker-waypoint atlas-marker-${role}`
    const startText = lang === "ja" ? "出発" : lang === "zh" ? "出发" : "START"
    const endText = lang === "ja" ? "到着" : lang === "zh" ? "到达" : "GOAL"
    const icon = role === "start" ? "🚗" : "🏁"
    const label = role === "start" ? `${icon} ${startText}` : `${icon} ${endText}`
    button.setAttribute("aria-label", label)
    button.innerHTML = `<span>${icon}</span><span>${role === "start" ? startText : endText}</span>`
    button.addEventListener("click", (event) => {
      event.stopPropagation()
      if (map?.isMoving()) map.stop()
      map?.flyTo({
        center: coord,
        zoom: 13.8,
        pitch: 38,
        bearing: 0,
        padding: { top: 60, bottom: 80, left: 340, right: 60 },
        duration: prefersReducedMotion ? 0 : 1200,
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

      // 2. Journey Start & End waypoints
      const segments = journey.routeSegments?.length ? journey.routeSegments : (journey.route?.length ? [journey.route] : [])
      const validSegments = segments.filter((s) => s.length >= 2)
      const originCoord = validSegments[0]?.[0]
      const lastSeg = validSegments.at(-1)
      const destCoord = lastSeg?.at(-1)

      if (originCoord) {
        const startMarker = new maplibregl.Marker({
          element: waypointMarkerElement(journey, "start", originCoord),
          offset: [-34, 0],
        })
          .setLngLat(originCoord)
          .addTo(map!)
        markers.set(`${journey.id}:start`, startMarker)
      }

      if (destCoord) {
        const endMarker = new maplibregl.Marker({
          element: waypointMarkerElement(journey, "end", destCoord),
          offset: [34, 0],
        })
          .setLngLat(destCoord)
          .addTo(map!)
        markers.set(`${journey.id}:end`, endMarker)
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
  }

  function flyToMemento(id: string | null, customBearing?: number) {
    if (!map || !id) return
    const m = journeys.flatMap((j) => j.mementos).find((item) => item.id === id)
    if (!m) return
    if (map.isMoving()) map.stop()
    map.flyTo({
      center: m.coords,
      zoom: 14.2,
      pitch: 40,
      bearing: customBearing ?? 15,
      speed: 1.1,
      curve: 1.3,
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

  function animateAlongRoute(
    fromCoords: Coordinates,
    toCoords: Coordinates,
    destinationMementoId?: string,
    durationMs = 3800
  ) {
    if (!map) return
    if (prefersReducedMotion) {
      if (destinationMementoId) flyToMemento(destinationMementoId)
      else map.flyTo({ center: toCoords, zoom: 13.5, pitch: 35, padding: { top: 60, bottom: 80, left: 340, right: 60 } })
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
        if (destinationMementoId) flyToMemento(destinationMementoId)
        else {
          map.flyTo({
            center: toCoords,
            zoom: 13.5,
            pitch: 35,
            padding: { top: 60, bottom: 80, left: 340, right: 60 },
            duration: 800,
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

      if (!vehicleMarker) {
        const el = document.createElement("div")
        el.className = "atlas-traveling-dot"
        vehicleMarker = new maplibregl.Marker({ element: el, anchor: "center" }).setLngLat(fromCoords).addTo(map)
      } else {
        vehicleMarker.getElement().style.display = "block"
        vehicleMarker.setLngLat(fromCoords)
      }

      const startTime = performance.now()
      let currentBearing = map.getBearing()
      let lastCamFrame = 0

      activeAnimationTimer = timer(() => {
        const elapsed = performance.now() - startTime
        const rawProgress = Math.min(1, elapsed / durationMs)
        const easedProgress = easeInOutCubic(rawProgress)
        const currentTotalDist = startDist + easedProgress * travelDist
        const fraction = currentTotalDist / timeline.totalDistance

        const { traveled, currentPos, bearing: targetBearing } = timeline.sliceAtFraction(fraction)

        vehicleMarker?.setLngLat(currentPos)
        updateTraveledRoute(traveled)

        // Interpolate camera bearing smoothly to prevent sharp angular jumps
        currentBearing = lerpAngle(currentBearing, targetBearing, 0.08)

        // Throttled camera follow: update every ~32ms (30fps) using smooth jumpTo
        const now = performance.now()
        if (rawProgress < 0.96 && map && now - lastCamFrame >= 30) {
          lastCamFrame = now
          map.jumpTo({
            center: currentPos,
            bearing: currentBearing,
            pitch: 42,
            zoom: 13.8,
            padding: { top: 60, bottom: 80, left: 340, right: 60 },
          })
        }

        if (rawProgress >= 1) {
          activeAnimationTimer?.stop()
          activeAnimationTimer = null
          if (vehicleMarker) {
            vehicleMarker.getElement().style.display = "none"
          }
          const finalState = timeline.sliceAtFraction(targetDist / timeline.totalDistance)
          updateTraveledRoute(finalState.traveled)

          if (destinationMementoId) {
            flyToMemento(destinationMementoId, currentBearing)
          } else {
            map?.flyTo({
              center: toCoords,
              zoom: 13.5,
              pitch: 35,
              bearing: currentBearing,
              padding: { top: 60, bottom: 80, left: 340, right: 60 },
              duration: 800,
            })
          }
        }
      })
    } catch {
      // Graceful fallback if timeline fails
      if (destinationMementoId) flyToMemento(destinationMementoId)
      else map?.flyTo({ center: toCoords, zoom: 13.5, pitch: 35 })
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
      if (vehicleMarker) {
        vehicleMarker.getElement().style.display = "none"
      }
      fitActiveJourney()
      updateTraveledRoute([])
      lastAnimatedCoords = null
    } else if (currentStep.type === "summary") {
      if (activeAnimationTimer) {
        activeAnimationTimer.stop()
        activeAnimationTimer = null
      }
      if (vehicleMarker) {
        vehicleMarker.getElement().style.display = "none"
      }
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
      if (vehicleMarker) {
        vehicleMarker.getElement().style.display = "none"
      }
      if (journey) {
        const dayMementos = journey.mementos.filter((m) => {
          const d = m.date.en || m.date.ja || m.date.zh || ""
          return d.startsWith(currentStep.date)
        })
        if (dayMementos.length) {
          const bounds = boundsOf(dayMementos.map((m) => m.coords))
          if (bounds) {
            map?.fitBounds(bounds, {
              padding: { top: 80, bottom: 90, left: 360, right: 80 },
              maxZoom: 12.5,
              pitch: 30,
              bearing: 0,
              duration: prefersReducedMotion ? 0 : 1200,
            })
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
        if (!vehicleMarker && map) {
          const el = document.createElement("div")
          el.className = "atlas-traveling-dot"
          vehicleMarker = new maplibregl.Marker({ element: el, anchor: "center" }).setLngLat(currentStep.coords).addTo(map)
        } else if (vehicleMarker) {
          vehicleMarker.getElement().style.display = "block"
          vehicleMarker.setLngLat(currentStep.coords)
        }
        map?.flyTo({
          center: currentStep.coords,
          zoom: 13.5,
          pitch: 38,
          bearing: 0,
          padding: { top: 60, bottom: 80, left: 340, right: 60 },
          duration: prefersReducedMotion ? 0 : 1500,
        })
        lastAnimatedCoords = currentStep.coords
      } else if (currentStep.role === "end") {
        if (lastAnimatedCoords) {
          animateAlongRoute(lastAnimatedCoords, currentStep.coords, undefined, 4200)
        } else if (journey) {
          const allRouteCoords = journey.routeSegments?.length ? journey.routeSegments.flat() : journey.route
          updateTraveledRoute(allRouteCoords)
          map?.flyTo({
            center: currentStep.coords,
            zoom: 13.5,
            pitch: 38,
            bearing: 0,
            padding: { top: 60, bottom: 80, left: 340, right: 60 },
            duration: prefersReducedMotion ? 0 : 1200,
          })
        }
        lastAnimatedCoords = currentStep.coords
      }
    } else if (currentStep.type === "stop") {
      if (journey) {
        if (lastAnimatedCoords) {
          animateAlongRoute(lastAnimatedCoords, currentStep.coords, currentStep.memento.id, 3800)
        } else {
          const segments = journey.routeSegments?.length ? journey.routeSegments : [journey.route]
          const validSegments = segments.filter((s) => s.length >= 2)
          const timeline = validSegments.length ? new RouteTimeline(validSegments, true) : null
          if (timeline && timeline.totalDistance > 0) {
            const targetDist = timeline.distanceAtCoordinate(currentStep.coords)
            const fraction = targetDist / timeline.totalDistance
            const { traveled, bearing } = timeline.sliceAtFraction(fraction)
            updateTraveledRoute(traveled)
            flyToMemento(currentStep.memento.id, bearing)
          } else {
            flyToMemento(currentStep.memento.id)
          }
        }
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

  :global(.atlas-marker-waypoint) {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: 0.25rem;
    height: 1.85rem;
    padding: 0 0.55rem;
    border-radius: 999px;
    font-size: 0.7rem;
    font-weight: 700;
    letter-spacing: 0.02em;
    cursor: pointer;
    border: 2px solid #ffffff;
    box-shadow:
      0 0 0 2px #000000aa,
      0 4px 12px #00000088;
    transition: transform 0.2s cubic-bezier(0.16, 1, 0.3, 1), box-shadow 0.2s;
    user-select: none;
    z-index: 1;
  }

  :global(.atlas-marker-waypoint:hover) {
    transform: scale(1.1);
  }

  :global(.atlas-marker-start) {
    background: #059669;
    color: #ffffff;
  }

  :global(.atlas-marker-end) {
    background: #d97706;
    color: #ffffff;
  }

  :global(.atlas-marker-waypoint.is-dimmed) {
    opacity: 0.28;
  }

  :global(.atlas-marker-waypoint.is-active) {
    transform: scale(1.18);
    box-shadow:
      0 0 0 3px #ffffff,
      0 0 16px #ff9b72;
    z-index: 4;
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
