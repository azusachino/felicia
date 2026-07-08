<script lang="ts">
  import maplibregl, { type StyleSpecification } from 'maplibre-gl';
  import { onMount } from 'svelte';
  import type { Coordinates, Theme } from '../data';

  // v3 detail map — reuses v1's MapLibre setup (CARTO raster basemap, route +
  // transit lines, DOM markers) but scoped to a single journey and clustered by
  // PLACE: one marker per place, a count badge when a place holds several
  // mementos. Selecting a place is bubbled up; the parent opens its memories.
  interface Place {
    key: string;
    coords: Coordinates;
    seq: number;
    count: number;
  }

  let {
    places,
    route,
    transit,
    activeKey,
    theme,
    onSelect
  }: {
    places: Place[];
    route: Coordinates[];
    transit: [Coordinates, Coordinates][];
    activeKey: string | null;
    theme: Theme;
    onSelect: (key: string) => void;
  } = $props();

  let container: HTMLDivElement;
  let map: maplibregl.Map | undefined;
  let loaded = $state(false);
  let resizeObserver: ResizeObserver | undefined;
  const markers = new Map<string, maplibregl.Marker>();

  const mapStyle: StyleSpecification = {
    version: 8,
    sources: {
      dark: {
        type: 'raster',
        tiles: ['https://a.basemaps.cartocdn.com/dark_all/{z}/{x}/{y}.png'],
        tileSize: 256,
        attribution: '&copy; OpenStreetMap contributors &copy; CARTO'
      },
      light: {
        type: 'raster',
        tiles: ['https://a.basemaps.cartocdn.com/light_all/{z}/{x}/{y}.png'],
        tileSize: 256,
        attribution: '&copy; OpenStreetMap contributors &copy; CARTO'
      }
    },
    layers: [
      { id: 'base-dark', type: 'raster', source: 'dark' },
      { id: 'base-light', type: 'raster', source: 'light', layout: { visibility: 'none' } }
    ]
  };

  function routeGeoJson() {
    return {
      type: 'FeatureCollection' as const,
      features: [
        { type: 'Feature' as const, geometry: { type: 'LineString' as const, coordinates: route }, properties: {} }
      ]
    };
  }

  function transitGeoJson() {
    return {
      type: 'FeatureCollection' as const,
      features: transit.map(pair => ({
        type: 'Feature' as const,
        geometry: { type: 'LineString' as const, coordinates: pair },
        properties: {}
      }))
    };
  }

  function boundsOf(coords: Coordinates[]) {
    const bounds = new maplibregl.LngLatBounds(coords[0], coords[0]);
    for (const coord of coords) bounds.extend(coord);
    return bounds;
  }

  const fitPadding = { top: 120, bottom: 120, left: 120, right: 460 };

  function fitJourney() {
    if (!map) return;
    const coords = route.length ? route : places.map(p => p.coords);
    if (!coords.length) return;
    map.fitBounds(boundsOf(coords), { padding: fitPadding, maxZoom: 9, duration: 700 });
  }

  function markerElement(place: Place) {
    const button = document.createElement('button');
    button.type = 'button';
    button.className = 'v3-mark';
    button.setAttribute('aria-label', `Place ${place.seq}`);
    button.innerHTML = `<span>${place.seq}</span>${
      place.count > 1 ? `<i class="v3-mark-count">${place.count}</i>` : ''
    }`;
    button.addEventListener('click', () => onSelect(place.key));
    return button;
  }

  function rebuildMarkers() {
    if (!map) return;
    markers.forEach(marker => marker.remove());
    markers.clear();
    for (const place of places) {
      const marker = new maplibregl.Marker({ element: markerElement(place), anchor: 'center' })
        .setLngLat(place.coords)
        .addTo(map);
      markers.set(place.key, marker);
    }
    syncActive();
  }

  function syncActive() {
    markers.forEach((marker, key) => {
      marker.getElement().classList.toggle('is-active', key === activeKey);
    });
  }

  function applyTheme(next: Theme) {
    if (!map || !map.getLayer('base-light')) return;
    map.setLayoutProperty('base-light', 'visibility', next === 'light' ? 'visible' : 'none');
    map.setLayoutProperty('base-dark', 'visibility', next === 'light' ? 'none' : 'visible');
  }

  onMount(() => {
    map = new maplibregl.Map({
      container,
      style: mapStyle,
      center: route[0] ?? places[0]?.coords ?? [138, 38],
      zoom: 6,
      attributionControl: false
    });
    map.addControl(new maplibregl.NavigationControl({ showCompass: false }), 'top-right');

    // The container reaches full size after mount/layout; keep the canvas in
    // sync so the map fills its box (otherwise only a corner renders).
    resizeObserver = new ResizeObserver(() => map?.resize());
    resizeObserver.observe(container);

    map.on('load', () => {
      if (!map) return;
      map.addSource('route', { type: 'geojson', data: routeGeoJson() });
      map.addLayer({
        id: 'route-glow',
        type: 'line',
        source: 'route',
        paint: { 'line-color': '#f97316', 'line-width': 8, 'line-opacity': 0.16, 'line-blur': 4 }
      });
      map.addLayer({
        id: 'route-line',
        type: 'line',
        source: 'route',
        paint: { 'line-color': '#fb923c', 'line-width': 3, 'line-opacity': 0.95 }
      });
      map.addSource('transit', { type: 'geojson', data: transitGeoJson() });
      map.addLayer({
        id: 'transit',
        type: 'line',
        source: 'transit',
        paint: { 'line-color': '#fde68a', 'line-width': 4, 'line-opacity': 0.9 }
      });

      rebuildMarkers();
      applyTheme(theme);
      map.resize();
      fitJourney();
      loaded = true;
    });

    return () => {
      resizeObserver?.disconnect();
      resizeObserver = undefined;
      markers.forEach(marker => marker.remove());
      markers.clear();
      map?.remove();
      map = undefined;
    };
  });

  // Rebuild + refit when the journey (places/route) changes.
  $effect(() => {
    void places;
    void route;
    if (!loaded || !map) return;
    (map.getSource('route') as maplibregl.GeoJSONSource | undefined)?.setData(routeGeoJson());
    (map.getSource('transit') as maplibregl.GeoJSONSource | undefined)?.setData(transitGeoJson());
    rebuildMarkers();
    fitJourney();
  });

  // Highlight + fly to the active place.
  $effect(() => {
    if (!loaded || !map) return;
    syncActive();
    const place = places.find(p => p.key === activeKey);
    if (place) map.flyTo({ center: place.coords, zoom: 8.5, duration: 600, essential: true });
  });

  $effect(() => {
    if (loaded) applyTheme(theme);
  });
</script>

<!-- Inline styles, not Tailwind utilities: maplibre-gl.css sets an UNLAYERED
     `.maplibregl-map { position: relative }` that outranks layered utilities, so
     `absolute inset-0` would be ignored and the map would collapse. -->
<div bind:this={container} style="position:absolute; inset:0;"></div>
