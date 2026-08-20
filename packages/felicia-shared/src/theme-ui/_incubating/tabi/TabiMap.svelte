<script lang="ts">
  import maplibregl from "maplibre-gl"
  import { select, type Selection } from "d3-selection"
  import { curveStepAfter, line } from "d3-shape"
  import { geoDistance, geoInterpolate } from "d3-geo"
  import { timer, type Timer } from "d3-timer"
  import { easeCubicInOut } from "d3-ease"
  import { onMount, untrack } from "svelte"
  import type { Coordinates, MementoKind } from "../../data"

  // Tabi (旅, journey) detail map — MapLibre owns the basemap/route/markers
  // (a thin copy of techo/TripMap.svelte's plumbing); D3 owns a separate SVG
  // overlay for the cat sprite, kept in sync with the map's own projection.
  // The two never touch the same coordinate system directly — D3 only ever
  // sees screen pixels that `map.project()` already converted.
  interface Place {
    key: string
    coords: Coordinates
    seq: number
    count: number
    kind: MementoKind
  }

  let {
    places,
    route,
    activeKey,
    runToken,
    cartoonFilter,
    onSelect,
  }: {
    places: Place[]
    route: Coordinates[]
    activeKey: string | null
    // Bumped by the parent's "Play" control to (re)start a run. Not a boolean
    // "should it run" flag -- a run is a one-shot event, and a counter is the
    // idiomatic way to make a repeated *same* value (clicking Play twice in a
    // row while idle) still trigger the effect below each time.
    runToken: number
    // Strips the basemap to flat land/water shapes -- no roads, buildings,
    // labels, or borders -- toggleable back to Positron's own real styling.
    cartoonFilter: boolean
    onSelect: (key: string) => void
  } = $props()

  let container: HTMLDivElement
  let overlay: SVGSVGElement
  let map: maplibregl.Map | undefined
  let loaded = $state(false)
  let resizeObserver: ResizeObserver | undefined
  // eslint-disable-next-line svelte/prefer-svelte-reactivity -- imperative maplibre marker cache, not reactive UI state
  const markers = new Map<string, maplibregl.Marker>()

  // One element today, but bound as data (not just appended) so a later
  // lesson can grow this to "one per journey" or "one per active stop"
  // without changing how the join works — only what's in catData.
  interface CatDatum {
    id: string
  }
  const catData: CatDatum[] = [{ id: "cat" }]
  let catSelection: Selection<SVGGElement, CatDatum, SVGSVGElement, unknown> | undefined

  // Where the cat actually is, in lng/lat -- the source of truth. Screen
  // position is always a pure function of this plus the map's current view,
  // recomputed on every map "move" and every run-timer tick, never cached.
  let catAt: Coordinates | undefined
  let catFacingLeft = false
  let mode: "idle" | "running" | "cuddling" = "idle"
  let gaitElapsed = 0
  let runTimer: Timer | undefined

  // Everything wrong with the first version came from one thing: the cat's
  // own <g> was only ever translated. A run needs a *second*, independent
  // motion layered on top of the route position — a gait cycle that has
  // nothing to do with geography and everything to do with legs hitting the
  // ground. STRIDE_MS is that cycle's own period, unrelated to how long the
  // whole route takes.
  const STRIDE_MS = 300

  function gaitOffsets(elapsedMs: number) {
    const phase = (elapsedMs % STRIDE_MS) / STRIDE_MS
    // Math.sin(phase * PI), not phase * 2*PI: rectified to a single 0->1->0
    // arch per stride (a hop never goes negative), rather than a full
    // sine wave that would dip the cat *below* the route.
    const hop = Math.sin(phase * Math.PI)
    return {
      bobPx: -hop * 5, // SVG y is down, so "up" is negative
      tiltDeg: Math.sin(phase * Math.PI * 2) * 4,
      squash: hop * 0.12, // stretched tall at the peak of the hop, squashed at contact
      legSwing: Math.sin(phase * Math.PI * 2), // -1..1: front/back legs swing in opposition
    }
  }

  // Four patterns, keyed by the stop's own first memento's `kind` -- not
  // picked at random and not one-pattern-per-stop-index. Grouped by the
  // feeling a kind actually carries: a souvenir or shrine stamp is something
  // you settle in with, a ticket or receipt is something you inspect, goods
  // or a live show is high-energy, transit is a brief, comic pass-through.
  type CuddlePattern = "curl" | "sniff" | "roll" | "chase"
  const patternByKind: Record<MementoKind, CuddlePattern> = {
    souvenir: "curl",
    stamp: "curl",
    ticket: "sniff",
    receipt: "sniff",
    goods: "roll",
    live: "roll",
    transit: "chase",
  }

  // What the cat is actually cuddling: shown as a small glyph beside it during
  // the cuddle, so the stop reads as "this cat found your ticket stub," not a
  // generic pose with no connection to what's actually at that place.
  const artifactGlyphByKind: Record<MementoKind, string> = {
    souvenir: "🎁",
    stamp: "⛩️",
    ticket: "🎫",
    receipt: "🧾",
    goods: "🛍️",
    live: "🎤",
    transit: "🚃",
  }

  let cuddlePattern: CuddlePattern = "curl"
  let cuddleGlyph = artifactGlyphByKind.souvenir
  let cuddleElapsed = 0
  let cuddleDurationMs = 0

  // Fades the artifact glyph in, holds it, fades it out -- rather than a hard
  // cut -- over the cuddle's own 0..1 progress.
  function envelope(p: number): number {
    if (p < 0.15) return p / 0.15
    if (p > 0.85) return (1 - p) / 0.15
    return 1
  }

  // Every pattern is built from the same transform channels the run cycle
  // already uses (bob/rotate/scale) plus one extra element (the heart, curl
  // only) -- no new shapes needed per pattern, just a different curve through
  // the same knobs. `t` is 0..1 progress through the cuddle's own duration.
  function cuddlePose(pattern: CuddlePattern, t: number) {
    const p = Math.min(1, Math.max(0, t))
    switch (pattern) {
      case "curl": {
        const breathe = Math.sin(p * Math.PI * 5)
        return { bobPx: -2, rotateDeg: 0, scaleX: 1.05 + breathe * 0.03, scaleY: 0.9 + breathe * 0.02, heartOpacity: Math.abs(Math.sin(p * Math.PI * 3)) }
      }
      case "sniff": {
        const bob = Math.abs(Math.sin(p * Math.PI * 4))
        return { bobPx: -bob * 4, rotateDeg: Math.sin(p * Math.PI * 4) * 6, scaleX: 1, scaleY: 1, heartOpacity: 0 }
      }
      case "roll": {
        const stretch = Math.sin(p * Math.PI)
        return { bobPx: 0, rotateDeg: p * 360, scaleX: 1 + stretch * 0.15, scaleY: 1 - stretch * 0.1, heartOpacity: 0 }
      }
      case "chase": {
        const hop = Math.abs(Math.sin(p * Math.PI * 6))
        return { bobPx: -hop * 3, rotateDeg: p * 360 * 3, scaleX: 1, scaleY: 1, heartOpacity: 0 }
      }
    }
  }

  function updateLegs(legSwing: number) {
    if (!catSelection) return
    const backPawX = -10 + legSwing * 8
    const frontPawX = 12 - legSwing * 8
    catSelection.select(".tabi-cat-leg-back").attr("d", `M -10,19 L ${backPawX},26`)
    catSelection.select(".tabi-cat-leg-front").attr("d", `M 12,20 L ${frontPawX},27`)
  }

  function positionCat() {
    if (!map || !catSelection || !catAt) return
    const point = map.project(catAt)
    const flip = catFacingLeft ? -1 : 1

    let bobPx = 0
    let rotateDeg = 0
    let scaleX = flip
    let scaleY = 1
    let heartOpacity = 0
    let artifactOpacity = 0

    if (mode === "running") {
      const gait = gaitOffsets(gaitElapsed)
      bobPx = gait.bobPx
      rotateDeg = gait.tiltDeg * flip
      scaleX = flip * (1 + gait.squash * 0.6)
      scaleY = 1 - gait.squash
      updateLegs(gait.legSwing)
    } else if (mode === "cuddling") {
      const p = cuddleDurationMs ? cuddleElapsed / cuddleDurationMs : 0
      const pose = cuddlePose(cuddlePattern, p)
      bobPx = pose.bobPx
      rotateDeg = pose.rotateDeg
      scaleX = flip * pose.scaleX
      scaleY = pose.scaleY
      heartOpacity = pose.heartOpacity
      artifactOpacity = envelope(Math.min(1, Math.max(0, p)))
      updateLegs(0)
    } else {
      updateLegs(0)
    }

    catSelection.attr("transform", `translate(${point.x}, ${point.y + bobPx}) rotate(${rotateDeg}) scale(${scaleX}, ${scaleY})`)
    catSelection.select(".tabi-cat-heart").attr("opacity", heartOpacity)
    catSelection.select(".tabi-cat-artifact").attr("opacity", artifactOpacity).text(cuddleGlyph)
  }

  function setCatAt(at: Coordinates) {
    catAt = at
    positionCat()
  }

  // The cat runs between the journey's *stops*, not the raw GPS trace: a real
  // `route` is the literal recorded track (thousands of points, including the
  // hours-long highway drive to get there), while `places` is only the spots
  // a memento actually anchors to. Animating the raw track made the run
  // technically correct and practically invisible -- one Izu day-trip's route
  // started in Tokyo, so fitting the camera to the whole track and interpolating
  // along it meant the visible motion near the actual stops was a few screen
  // pixels out of a multi-hour, city-spanning line.
  function runPathOf(places: Place[], route: Coordinates[]): Coordinates[] {
    return places.length >= 2 ? places.map((place) => place.coords) : route
  }

  // Great-circle distance, in degrees rather than d3-geo's native radians --
  // every existing threshold/scale constant here (ARRIVAL_THRESHOLD,
  // SPEED_MS_PER_DEGREE, isDistinctOrigin's 0.0005) was calibrated against
  // Math.hypot on raw lng/lat, so converting geoDistance's radians back to
  // that same degree unit keeps them meaningful without re-tuning every one.
  // The difference from the flat Math.hypot this replaced is invisible at
  // this journey's scale (one prefecture) -- it stops being invisible for a
  // trip spanning a large stretch of the globe, where a straight line in
  // lng/lat space visibly diverges from the real great-circle path.
  const DEGREES_PER_RADIAN = 180 / Math.PI

  function greatCircleDistance(a: Coordinates, b: Coordinates): number {
    return geoDistance(a, b) * DEGREES_PER_RADIAN
  }

  // Arc-length parameterization: walk a multi-point path at constant speed
  // regardless of how unevenly its points are spaced (a recorded GPS track
  // has long straight highway segments and short winding ones in the same
  // array). `cumulative[i]` is the distance travelled by the time point `i`
  // is reached; `pointAlong` finds which segment a given fraction of the
  // *total* distance falls in and interpolates within just that segment.
  interface PathWalker {
    points: Coordinates[]
    cumulative: number[]
    total: number
  }

  function buildPathWalker(points: Coordinates[]): PathWalker {
    const cumulative = [0]
    for (let i = 1; i < points.length; i++) {
      cumulative.push(cumulative[i - 1] + greatCircleDistance(points[i - 1], points[i]))
    }
    return { points, cumulative, total: cumulative.at(-1) ?? 0 }
  }

  // `geoInterpolate`, not `d3-interpolate`'s plain `interpolate` -- a slerp
  // along the sphere between the two segment endpoints, not a lerp straight
  // through the earth's lng/lat coordinate space.
  function pointAlong(walker: PathWalker, t: number): Coordinates {
    if (walker.points.length < 2) return walker.points[0] ?? [0, 0]
    const target = Math.min(1, Math.max(0, t)) * walker.total
    let i = 1
    while (i < walker.cumulative.length - 1 && walker.cumulative[i] < target) i++
    const segStart = walker.cumulative[i - 1]
    const segLen = walker.cumulative[i] - segStart || 1
    return geoInterpolate(walker.points[i - 1], walker.points[i])((target - segStart) / segLen) as Coordinates
  }

  // The index of `route`'s own point closest to `target`, over its full
  // length -- the fallback when a directional search (below) finds nothing
  // within ARRIVAL_THRESHOLD at all.
  function nearestRouteIndex(route: Coordinates[], target: Coordinates): number {
    let best = 0
    let bestDist = Infinity
    for (let i = 0; i < route.length; i++) {
      const d = greatCircleDistance(route[i], target)
      if (d < bestDist) {
        bestDist = d
        best = i
      }
    }
    return best
  }

  // ~2km at these latitudes -- "close enough to call it arrival."
  const ARRIVAL_THRESHOLD = 0.02

  // A day-trip's recorded track is a single there-and-back LineString, not
  // two separate legs -- `route[0]` and `route.at(-1)` are usually both
  // "home," a few hundred meters apart, and a road on the way out is often
  // the *same* road on the way back. A plain "closest point anywhere in the
  // array" search doesn't know which pass of a revisited road is the right
  // one -- searching forward from the start for the *first* approach finds
  // the real arrival; searching backward from the end for the *last*
  // approach finds the real departure. (The bug this replaced used a single
  // undirected nearest-point search for both ends of the homecoming leg,
  // which put `route[0]`'s trivial self-match on the wrong side of the
  // comparison and sliced/reversed the *outbound* half of the track instead
  // of the actual return road.)
  function forwardArrivalIndex(route: Coordinates[], target: Coordinates): number {
    for (let i = 0; i < route.length; i++) {
      if (greatCircleDistance(route[i], target) < ARRIVAL_THRESHOLD) return i
    }
    return nearestRouteIndex(route, target)
  }

  function backwardDepartureIndex(route: Coordinates[], target: Coordinates): number {
    for (let i = route.length - 1; i >= 0; i--) {
      if (greatCircleDistance(route[i], target) < ARRIVAL_THRESHOLD) return i
    }
    return nearestRouteIndex(route, target)
  }

  // The real recorded track from the very start of `route` up through the
  // first arrival near `firstStop`.
  function establishingPath(route: Coordinates[], firstStop: Coordinates): Coordinates[] {
    if (route.length < 2) return [route[0] ?? firstStop, firstStop]
    const slice = route.slice(0, forwardArrivalIndex(route, firstStop) + 1)
    return slice.length >= 2 ? slice : [route[0], firstStop]
  }

  // The real recorded track from the last departure near `lastStop` through
  // to the very end of `route` -- not `route[0]`'s index, which is a
  // different point in time than "home" at the end of a round trip.
  function homecomingPath(route: Coordinates[], lastStop: Coordinates): Coordinates[] {
    if (route.length < 2) return [lastStop, route.at(-1) ?? lastStop]
    const slice = route.slice(backwardDepartureIndex(route, lastStop))
    return slice.length >= 2 ? slice : [lastStop, route.at(-1) ?? lastStop]
  }

  interface TravelStep {
    kind: "travel"
    path: Coordinates[]
    durationMs: number
    // Refits the camera right as this leg starts, if set -- "wide" for the
    // establishing leg home <-> first stop (where the real distance needs
    // the zoomed-out view to even fit on screen), "local" to zoom back in
    // once the stop-to-stop portion begins. Unset legs don't touch the camera.
    camera?: "wide" | "local"
  }
  interface CuddleStep {
    kind: "cuddle"
    at: Coordinates
    pattern: CuddlePattern
    mementoKind: MementoKind
    durationMs: number
  }
  type RunStep = TravelStep | CuddleStep

  // One speed, shared by every travel step -- a local hop between two nearby
  // stops and the country-spanning establishing/homecoming leg used to run on
  // two disconnected budgets (a distance-scaled one locally, a flat 6000ms
  // wide leg regardless of how far that actually was), which is what made the
  // whole run feel uneven: the wide legs moved at several times the implied
  // speed of the local ones. MIN/MAX are a floor so a near-zero hop still
  // reads as *an* animation and a ceiling so a very long establishing leg on
  // an unusually far-flung trip doesn't stall the whole sequence.
  const SPEED_MS_PER_DEGREE = 2000
  const MIN_LEG_MS = 500
  const MAX_LEG_MS = 6000
  const CUDDLE_MS = 1000

  function durationForDistance(distanceDeg: number): number {
    return Math.min(MAX_LEG_MS, Math.max(MIN_LEG_MS, distanceDeg * SPEED_MS_PER_DEGREE))
  }

  // A meaningfully different point from the first stop -- guards the
  // establishing/homecoming legs so a journey whose GPS track happens to
  // start right at its first memento doesn't get a pointless zero-length leg.
  function isDistinctOrigin(origin: Coordinates | undefined, firstStop: Coordinates): origin is Coordinates {
    return !!origin && greatCircleDistance(origin, firstStop) > 0.0005
  }

  // The whole trip as a sequence: an establishing run from the true origin
  // (route[0] -- where the GPS track actually starts, typically home) to the
  // first stop, then stop to stop with a cuddle at each (a pattern keyed to
  // that stop's own first memento's kind), then a homecoming leg all the way
  // back to the origin. The *stop-to-stop* portion still uses a straight
  // 2-point path between `places`, not `route`, for the same reason
  // `runPathOf` picked places over the raw GPS track in lesson 1 -- there's
  // nothing to cuddle at a highway waypoint. The establishing/homecoming legs
  // are the opposite case: the real road shape is already drawn on screen the
  // whole time, so a straight line across it would visibly contradict what's
  // right there -- `establishingPath`/`homecomingPath` walk the actual
  // recorded track, and its own arc length (via `buildPathWalker`, not a
  // flat budget) feeds the same `durationForDistance` every other leg uses.
  function buildSequence(places: Place[], route: Coordinates[]): RunStep[] {
    if (places.length < 2) {
      const path = runPathOf(places, route)
      return path.length < 2 ? [] : [{ kind: "travel", path, durationMs: durationForDistance(buildPathWalker(path).total) }]
    }
    const path = places.map((place) => place.coords)
    const origin = route[0]
    const showOrigin = isDistinctOrigin(origin, path[0])
    const steps: RunStep[] = []

    if (showOrigin) {
      const establishing = establishingPath(route, path[0])
      steps.push({ kind: "travel", path: establishing, durationMs: durationForDistance(buildPathWalker(establishing).total), camera: "wide" })
    }

    for (let i = 0; i < path.length - 1; i++) {
      steps.push({ kind: "travel", path: [path[i], path[i + 1]], durationMs: durationForDistance(greatCircleDistance(path[i], path[i + 1])), camera: i === 0 && showOrigin ? "local" : undefined })
      steps.push({ kind: "cuddle", at: path[i + 1], pattern: patternByKind[places[i + 1].kind], mementoKind: places[i + 1].kind, durationMs: CUDDLE_MS })
    }

    const last = path[path.length - 1]
    const homecoming = showOrigin ? homecomingPath(route, last) : [last, path[0]]
    steps.push({
      kind: "travel",
      path: homecoming,
      durationMs: durationForDistance(buildPathWalker(homecoming).total),
      camera: showOrigin ? "wide" : undefined,
    })
    return steps
  }

  let sequence: RunStep[] = []
  let sequenceIndex = 0

  function runRoute() {
    runTimer?.stop()
    sequence = buildSequence(places, route)
    sequenceIndex = 0
    advanceSequence()
  }

  function advanceSequence() {
    const step = sequence[sequenceIndex]
    if (!step) {
      mode = "idle"
      positionCat()
      return
    }
    if (step.kind === "travel") runTravelStep(step)
    else runCuddleStep(step)
  }

  // The "d3.timer + d3.geo + d3.ease" step: `timer()` drives one
  // requestAnimationFrame loop for this one leg. `easeCubicInOut` reshapes
  // raw elapsed-time progress into a start-slow/cruise/end-slow curve for
  // the leg -- layered underneath the per-stride gait cycle, not instead of
  // it. Position along the leg comes from `pointAlong`, which walks
  // `step.path` at constant real-world speed via `geoInterpolate` (see
  // lesson 5) rather than a naive lerp through lng/lat coordinate space.
  function runTravelStep(step: TravelStep) {
    mode = "running"
    if (step.camera === "wide") wideFit()
    else if (step.camera === "local") fitJourney()
    const walker = buildPathWalker(step.path)
    const destination = step.path.at(-1) ?? step.path[0]
    runTimer = timer((elapsed) => {
      gaitElapsed = elapsed
      const t = easeCubicInOut(Math.min(1, step.durationMs ? elapsed / step.durationMs : 1))
      const at = pointAlong(walker, t)
      if (catAt) catFacingLeft = at[0] < catAt[0]
      setCatAt(at)

      if (elapsed >= step.durationMs) {
        runTimer?.stop()
        setCatAt(destination)
        sequenceIndex += 1
        advanceSequence()
      }
    })
  }

  function runCuddleStep(step: CuddleStep) {
    mode = "cuddling"
    cuddlePattern = step.pattern
    cuddleGlyph = artifactGlyphByKind[step.mementoKind]
    cuddleDurationMs = step.durationMs
    setCatAt(step.at)
    runTimer = timer((elapsed) => {
      cuddleElapsed = elapsed
      positionCat()

      if (elapsed >= step.durationMs) {
        runTimer?.stop()
        sequenceIndex += 1
        advanceSequence()
      }
    })
  }

  // The tail is generated, not hand-tuned: three control points run through
  // d3-shape's `line`, stepped with `curveStepAfter` -- a real right-angle
  // staircase between each pair, the same way an 8-bit sprite draws a
  // diagonal one grid cell at a time, instead of `curveBumpX`'s smooth
  // S-curve (right for TicketStub.svelte's paper edge, wrong for a
  // pixel-sprite cat). Coordinates are local to the cat's own <g> -- (0, 0)
  // is roughly the base of the tail at the body.
  const tailPath =
    line<[number, number]>().curve(curveStepAfter)([
      [-9, 6],
      [-18, -4],
      [-27, -15],
    ]) ?? ""

  // Positron, not Liberty (techo/atlas/cartography's shared choice): flat and
  // mostly label-light, closer to a cartoon paper map than full street detail.
  // Still real cartography, though -- the cartoon mode below goes further and
  // strips it to bare land/water shapes, since the actual roads/place names
  // were never the point; the trip is.
  const mapStyle = "https://tiles.openfreemap.org/styles/positron"

  // Everything that isn't the land/water/park silhouette itself: every text
  // label (place names, road names, water names -- all `symbol` layers),
  // every road/rail/runway class, administrative borders, buildings, minor
  // waterways, and polar-only landcover this journey will never touch.
  function isRealismLayer(id: string, type: string): boolean {
    if (type === "symbol") return true
    if (id === "waterway" || id === "building" || id === "landuse_residential") return true
    if (id === "landcover_ice_shelf" || id === "landcover_glacier") return true
    return /^(tunnel_|aeroway|road_|highway_|railway|boundary)/.test(id)
  }

  interface RecolorTarget {
    id: string
    property: "background-color" | "fill-color"
    cartoon: string
  }
  // Flat, bold fills instead of Positron's own muted cartography colors --
  // the land/water/park shapes are the entire basemap once isRealismLayer
  // has hidden everything else, so they're what actually reads as "cartoon."
  // Saturated enough to hold their own next to the cat's own orange/pink --
  // the first pass (soft cream/pastel-blue/mint) was flat but still timid.
  const RECOLOR_TARGETS: RecolorTarget[] = [
    { id: "background", property: "background-color", cartoon: "#fed7aa" },
    { id: "water", property: "fill-color", cartoon: "#5ec8f0" },
    { id: "park", property: "fill-color", cartoon: "#86efac" },
    { id: "landcover_wood", property: "fill-color", cartoon: "#86efac" },
  ]

  interface OutlineLayer {
    id: string
    sourceLayer: "water" | "park"
    color: string
  }
  // The cat's own shapes read as "cartoon" partly because of their white
  // outline stroke (lesson 2) -- flat fills alone, with no line drawing their
  // edge, don't carry that same ink-line look. These add exactly that stroke
  // to the coastline and park boundaries, in a darker shade of each fill.
  const OUTLINE_LAYERS: OutlineLayer[] = [
    { id: "tabi-water-outline", sourceLayer: "water", color: "#0ea5e9" },
    { id: "tabi-park-outline", sourceLayer: "park", color: "#22c55e" },
  ]

  let realismLayerIds: string[] = []
  // eslint-disable-next-line svelte/prefer-svelte-reactivity -- imperative style-property cache, not reactive UI state
  const originalColors = new Map<string, unknown>()

  // Captures what to hide and what each recolor target originally looked
  // like, once, right after the style loads -- so toggling back to
  // "realistic" restores Positron's own colors exactly rather than a second
  // hand-picked guess at what they were.
  function captureRealism() {
    if (!map) return
    const layers = map.getStyle()?.layers ?? []
    realismLayerIds = layers.filter((layer) => isRealismLayer(layer.id, layer.type)).map((layer) => layer.id)
    for (const target of RECOLOR_TARGETS) {
      if (layers.some((layer) => layer.id === target.id)) {
        originalColors.set(target.id, map.getPaintProperty(target.id, target.property))
      }
    }
  }

  // Adds the outline layers once, above the fills they trace, hidden by
  // default -- setCartoonMode is what shows them.
  function addOutlineLayers() {
    if (!map) return
    for (const outline of OUTLINE_LAYERS) {
      if (map.getLayer(outline.id)) continue
      map.addLayer({
        id: outline.id,
        type: "line",
        source: "openmaptiles",
        "source-layer": outline.sourceLayer,
        layout: { visibility: "none", "line-join": "round" },
        paint: { "line-color": outline.color, "line-width": 1.8 },
      })
    }
  }

  function setCartoonMode(on: boolean) {
    if (!map) return
    for (const id of realismLayerIds) {
      map.setLayoutProperty(id, "visibility", on ? "none" : "visible")
    }
    for (const outline of OUTLINE_LAYERS) {
      map.setLayoutProperty(outline.id, "visibility", on ? "visible" : "none")
    }
    for (const target of RECOLOR_TARGETS) {
      if (!originalColors.has(target.id)) continue
      map.setPaintProperty(target.id, target.property, on ? target.cartoon : originalColors.get(target.id))
    }
  }

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

  function boundsOf(coords: Coordinates[]) {
    if (coords.length === 0) {
      return new maplibregl.LngLatBounds([138, 38], [138, 38])
    }
    const bounds = new maplibregl.LngLatBounds(coords[0], coords[0])
    for (const coord of coords) bounds.extend(coord)
    return bounds
  }

  const fitPadding = { top: 120, bottom: 120, left: 120, right: 120 }

  // Framed on the stops, not the full recorded route: techo's own map fits
  // both because its map isn't trying to make travel *between* them legible.
  // Here it has to be -- fitting the raw multi-hour GPS track (see runPathOf)
  // would zoom out far enough that the cat's motion between actual stops
  // all but disappears.
  function fitJourney() {
    if (!map) return
    const coords = places.map((place) => place.coords)
    if (!coords.length) return
    map.fitBounds(boundsOf(coords), { padding: fitPadding, maxZoom: 9, duration: 700 })
  }

  // The wide establishing view -- stops *and* the trip's real starting point
  // (route[0], typically home), so the establishing/homecoming legs have
  // somewhere visible to run across. Only used around those two legs; the
  // stop-to-stop portion stays on fitJourney()'s tighter local frame.
  function wideFit() {
    if (!map) return
    const origin = route[0]
    const coords = origin ? [origin, ...places.map((place) => place.coords)] : places.map((place) => place.coords)
    if (!coords.length) return
    map.fitBounds(boundsOf(coords), { padding: fitPadding, maxZoom: 9, duration: 700 })
  }

  function markerElement(place: Place) {
    const button = document.createElement("button")
    button.type = "button"
    button.className = "tabi-mark"
    button.setAttribute("aria-label", `Place ${place.seq}`)
    button.innerHTML = `<span>${place.seq}</span>${place.count > 1 ? `<i class="tabi-mark-count">${place.count}</i>` : ""}`
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
      const marker = new maplibregl.Marker({ element: markerElement(place), anchor: "center" }).setLngLat(place.coords).addTo(map)
      markers.set(place.key, marker)
    }
    syncActive()
  }

  let homeMarker: maplibregl.Marker | undefined

  // A plain, unnumbered dot at route[0] -- without it, the wide establishing
  // shot is just an empty stretch of map with a cat sitting in it and no cue
  // for what that point *is*. Deliberately not a `.tabi-mark` (no seq number,
  // not clickable, no place to select): it isn't a stop, it's the trip's own
  // starting point.
  function rebuildHomeMarker() {
    if (!map) return
    homeMarker?.remove()
    homeMarker = undefined
    const origin = route[0]
    if (!origin || !isDistinctOrigin(origin, places[0]?.coords ?? origin)) return
    const el = document.createElement("div")
    el.className = "tabi-home"
    el.setAttribute("aria-label", "Start")
    homeMarker = new maplibregl.Marker({ element: el, anchor: "center" }).setLngLat(origin).addTo(map)
  }

  function syncActive() {
    markers.forEach((marker, key) => {
      marker.getElement().classList.toggle("is-active", key === activeKey)
    })
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

    resizeObserver = new ResizeObserver(() => map?.resize())
    resizeObserver.observe(container)

    map.on("load", () => {
      if (!map) return
      captureRealism()
      addOutlineLayers()
      setCartoonMode(cartoonFilter)

      map.addSource("route", { type: "geojson", data: routeGeoJson() })
      map.addLayer({
        id: "route-glow",
        type: "line",
        source: "route",
        paint: { "line-color": "#f472b6", "line-width": 10, "line-opacity": 0.18, "line-blur": 5 },
      })
      map.addLayer({
        id: "route-line",
        type: "line",
        source: "route",
        layout: { "line-cap": "round", "line-join": "round" },
        paint: { "line-color": "#fb7185", "line-width": 4, "line-opacity": 0.95 },
      })

      rebuildMarkers()
      rebuildHomeMarker()

      // The join: enter appends exactly what catData doesn't have on screen
      // yet, update (the join()'s return value) is what already-bound
      // elements get on every call. Right now catData never changes length,
      // so this only ever runs the enter branch once — the shape still
      // matters, because a later lesson (per-stop sprites) changes catData's
      // length and update/exit start doing real work for free.
      //
      // The enter callback builds the cat's shapes exactly once, so a later
      // position update (translate on the outer <g>) never re-touches most
      // of them — join()'s update branch is a no-op here for everything
      // except the two legs, which `positionCat()` redraws every frame.
      // Paint order is document order: legs and tail go in first so the
      // body/head layer over their hip/base joints.
      catSelection = select(overlay)
        .selectAll<SVGGElement, CatDatum>("g.tabi-cat")
        .data(catData, (d) => d.id)
        .join((enter) => {
          const g = enter.append("g").attr("class", "tabi-cat")
          g.append("path").attr("class", "tabi-cat-leg tabi-cat-leg-back").attr("d", "M -10,19 L -10,26")
          g.append("path").attr("class", "tabi-cat-leg tabi-cat-leg-front").attr("d", "M 12,20 L 12,27")
          g.append("path").attr("class", "tabi-cat-tail").attr("d", tailPath)
          g.append("rect").attr("class", "tabi-cat-body").attr("x", -15).attr("y", 2).attr("width", 30).attr("height", 20)
          g.append("path").attr("class", "tabi-cat-ear").attr("d", "M7,-8 L8,-22 L13,-9 Z")
          g.append("path").attr("class", "tabi-cat-ear").attr("d", "M15,-9 L20,-23 L22,-9 Z")
          g.append("rect").attr("class", "tabi-cat-head").attr("x", 6).attr("y", -14).attr("width", 20).attr("height", 20)
          g.append("rect").attr("class", "tabi-cat-blush").attr("x", 12).attr("y", 0).attr("width", 4).attr("height", 4)
          g.append("rect").attr("class", "tabi-cat-eye-white").attr("x", 17).attr("y", -9).attr("width", 6).attr("height", 6)
          g.append("rect").attr("class", "tabi-cat-eye-pupil").attr("x", 19).attr("y", -7).attr("width", 3).attr("height", 3)
          g.append("rect").attr("class", "tabi-cat-nose").attr("x", 25).attr("y", -1).attr("width", 4).attr("height", 4)
          g.append("text").attr("class", "tabi-cat-heart").attr("x", 15).attr("y", -30).attr("text-anchor", "middle").attr("opacity", 0).text("♡")
          g.append("text").attr("class", "tabi-cat-artifact").attr("x", -6).attr("y", -24).attr("text-anchor", "middle").attr("opacity", 0)
          return g
        })

      map.on("move", positionCat)

      map.resize()
      wideFit()
      setCatAt(route[0] ?? runPathOf(places, route)[0] ?? [138, 38])
      loaded = true
    })

    return () => {
      runTimer?.stop()
      resizeObserver?.disconnect()
      resizeObserver = undefined
      markers.forEach((marker) => marker.remove())
      markers.clear()
      homeMarker?.remove()
      homeMarker = undefined
      map?.remove()
      map = undefined
    }
  })

  $effect(() => {
    void places
    void route
    if (!loaded || !map) return
    runTimer?.stop()
    mode = "idle"
    ;(map.getSource("route") as maplibregl.GeoJSONSource | undefined)?.setData(routeGeoJson())
    rebuildMarkers()
    rebuildHomeMarker()
    wideFit()
    setCatAt(route[0] ?? runPathOf(places, route)[0] ?? [138, 38])
  })

  // Reacts to activeKey alone (untrack()'d for loaded/map), the same reason
  // the runToken effect below does: without it, this re-fires the moment
  // `loaded` flips true on mount, and `fitJourney()`'s local frame would
  // silently overwrite the wide idle-at-home framing `wideFit()` just set.
  $effect(() => {
    void activeKey
    untrack(() => {
      if (!loaded || !map) return
      syncActive()
      if (!activeKey) fitJourney()
    })
  })

  // The sole run trigger: bumped by the parent's "Play" button. `runToken` is
  // the only tracked read outside `untrack()`, so this effect fires exactly
  // when the button is clicked -- not on mount, not when `loaded` flips true,
  // not when the journey changes (the effect above already resets to idle
  // for that case).
  $effect(() => {
    void runToken
    untrack(() => {
      if (!loaded || !map) return
      runRoute()
    })
  })

  $effect(() => {
    if (!loaded || !map) return
    setCartoonMode(cartoonFilter)
  })
</script>

<!-- Inline styles, not Tailwind utilities: maplibre-gl.css sets an UNLAYERED
     `.maplibregl-map { position: relative }` that outranks layered utilities, so
     `absolute inset-0` would be ignored and the map would collapse. -->
<div bind:this={container} style="position:absolute; inset:0;">
  <svg bind:this={overlay} class="tabi-overlay" role="presentation"></svg>
</div>

<style>

  :global(.tabi-mark) {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 2rem;
    height: 2rem;
    border: 2px solid #fff;
    border-radius: 999px;
    background: #fb7185;
    color: #fff;
    font-family: ui-rounded, "Nunito", system-ui, sans-serif;
    font-weight: 800;
    font-size: 0.8rem;
    box-shadow: 0 0.2rem 0.5rem rgba(190, 24, 93, 0.35);
    cursor: pointer;
  }

  :global(.tabi-mark.is-active) {
    background: #f472b6;
    transform: scale(1.15);
  }

  :global(.tabi-home) {
    width: 0.85rem;
    height: 0.85rem;
    border: 2px solid #fb7185;
    border-radius: 999px;
    background: #fff;
    box-shadow: 0 0.15rem 0.4rem rgba(190, 24, 93, 0.3);
  }

  .tabi-overlay {
    position: absolute;
    inset: 0;
    z-index: 5;
    width: 100%;
    height: 100%;
    /* MapLibre still owns click/drag/scroll; the overlay only ever draws. */
    pointer-events: none;
  }

  :global(.tabi-cat) {
    filter: drop-shadow(0 0.15rem 0.25rem rgba(190, 24, 93, 0.4));
  }

  /* Sharp-cornered rects/triangles + a bold dark outline (not the soft white
     one from the flat-vector version) is what reads as a pixel-sprite
     silhouette rather than a smoothed cartoon one -- stroke-linejoin stays
     the SVG default (miter), never round, so every corner stays a corner. */
  :global(.tabi-cat-body),
  :global(.tabi-cat-head),
  :global(.tabi-cat-ear) {
    fill: #f0975a;
    stroke: #3d2510;
    stroke-width: 2.5;
  }

  :global(.tabi-cat-tail),
  :global(.tabi-cat-leg) {
    fill: none;
    stroke: #f0975a;
    stroke-width: 4.5;
    stroke-linecap: square;
  }

  :global(.tabi-cat-eye-white) {
    fill: #fff;
    stroke: #3d2510;
    stroke-width: 1.5;
  }

  :global(.tabi-cat-eye-pupil) {
    fill: #2a1506;
  }

  :global(.tabi-cat-blush) {
    fill: #fda4af;
  }

  :global(.tabi-cat-nose) {
    fill: #be185d;
    stroke: #3d2510;
    stroke-width: 1;
  }

  :global(.tabi-cat-heart) {
    fill: #f43f5e;
    font-size: 9px;
  }

  :global(.tabi-cat-artifact) {
    font-size: 11px;
  }

  :global(.tabi-mark-count) {
    position: absolute;
    top: -0.35rem;
    right: -0.35rem;
    display: flex;
    align-items: center;
    justify-content: center;
    min-width: 1.1rem;
    height: 1.1rem;
    padding: 0 0.2rem;
    border-radius: 999px;
    background: #fff;
    color: #f472b6;
    font-size: 0.62rem;
    font-style: normal;
    font-weight: 800;
  }
</style>
