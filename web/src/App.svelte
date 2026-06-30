<script lang="ts">
  import maplibregl, { type StyleSpecification } from 'maplibre-gl';
  import { onMount, tick } from 'svelte';

  type Coordinates = [number, number];
  type Mode = 'artifact' | 'review' | 'creator';
  type MementoKind = 'goods' | 'transit' | 'stamp';

  interface Station {
    name: string;
    ja: string;
    coords: Coordinates;
  }

  interface Memento {
    id: string;
    kind: MementoKind;
    title: string;
    date: string;
    place: string;
    vendor: string;
    price: string;
    coords: Coordinates;
    essay: string;
    photos: { src: string; caption: string }[];
    source: 'Immich' | 'Manual' | 'Transit creator';
    confidence: number;
    reviewState: 'confirm' | 'merge' | 'private';
    transit?: {
      operator: string;
      line: string;
      from: Station;
      to: Station;
      fare: string;
    };
  }

  const stations: Station[] = [
    { name: 'Tokyo', ja: '東京', coords: [139.7671, 35.6812] },
    { name: 'Shibuya', ja: '渋谷', coords: [139.7016, 35.6580] },
    { name: 'Asakusa', ja: '浅草', coords: [139.7967, 35.7148] },
    { name: 'Kyoto', ja: '京都', coords: [135.7588, 34.9858] },
    { name: 'Osaka-Umeda', ja: '大阪梅田', coords: [135.4959, 34.7025] }
  ];

  const station = (name: string) => stations.find(item => item.name === name) ?? stations[0];

  const route: Coordinates[] = [
    [139.7671, 35.6812],
    [139.7016, 35.6580],
    [139.7967, 35.7148],
    [138.389, 35.322],
    [136.8815, 35.1709],
    [135.7588, 34.9858],
    [135.5013, 34.6687]
  ];

  const mementos: Memento[] = [
    {
      id: 'fuwamiku',
      kind: 'goods',
      title: 'Fuwamiku Mascot Plush',
      date: '2026.05.16',
      place: 'Dotonbori, Osaka',
      vendor: 'Mascot Cafe & Shop',
      price: 'JPY 2,400',
      coords: [135.5013, 34.6687],
      essay:
        'After the neon crush of Dotonbori, the cafe felt quiet enough to hear the ice settle in our glasses. The tiny plush was sitting beside the register, absurdly soft, and somehow became the thing that carried the whole afternoon home.',
      photos: [{ src: '/osaka_plushie.jpg', caption: 'The back-fillable object: still photographable months later.' }],
      source: 'Immich',
      confidence: 86,
      reviewState: 'confirm'
    },
    {
      id: 'ginza-line',
      kind: 'transit',
      title: 'Shibuya to Asakusa',
      date: '2026.05.12',
      place: 'Tokyo Metro Ginza Line',
      vendor: 'Tokyo Metro',
      price: 'JPY 210',
      coords: [139.7492, 35.6864],
      essay:
        'The orange train stitched two Tokyos together: Shibuya noise in the west, temple smoke in the east. It is less a point on the map than a line that made the day legible.',
      photos: [],
      source: 'Transit creator',
      confidence: 100,
      reviewState: 'confirm',
      transit: {
        operator: 'Tokyo Metro',
        line: 'Ginza Line',
        from: station('Shibuya'),
        to: station('Asakusa'),
        fare: 'JPY 210'
      }
    },
    {
      id: 'kinkakuji',
      kind: 'stamp',
      title: 'Kinkaku-ji Goshuin',
      date: '2026.05.14',
      place: 'Kinkaku-ji, Kyoto',
      vendor: 'Temple office',
      price: 'JPY 500',
      coords: [135.7292, 35.0394],
      essay:
        'The gold pavilion was loud in the light, but the stamp was quiet: brush, red seal, paper. It felt less like buying a souvenir and more like receiving a small piece of the place.',
      photos: [{ src: '/kyoto_temple.jpg', caption: 'Kinkaku-ji framed by early color.' }],
      source: 'Immich',
      confidence: 72,
      reviewState: 'merge'
    }
  ];

  const priorities = [
    {
      label: '1',
      title: 'Public artifact moat',
      detail: 'Route + designed memento + essay/gallery. This is the first critical feature.'
    },
    {
      label: '2',
      title: 'Curate queue',
      detail: 'Immich and Dawarich propose anchors; author confirms, merges, or hides them.'
    },
    {
      label: '3',
      title: 'Transit creator',
      detail: 'Manual edge-anchored rail legs can become route geometry without passive GPS.'
    }
  ];

  let mode: Mode = 'artifact';
  let selected = mementos[0];
  let mapContainer: HTMLDivElement;
  let map: maplibregl.Map | undefined;
  const markers = new Map<string, maplibregl.Marker>();

  const routeGeoJson = {
    type: 'Feature' as const,
    geometry: {
      type: 'LineString' as const,
      coordinates: route
    },
    properties: {}
  };

  const transitGeoJson = {
    type: 'FeatureCollection' as const,
    features: mementos
      .filter(memento => memento.transit)
      .map(memento => ({
        type: 'Feature' as const,
        geometry: {
          type: 'LineString' as const,
          coordinates: [memento.transit!.from.coords, memento.transit!.to.coords]
        },
        properties: { id: memento.id }
      }))
  };

  const mapStyle: StyleSpecification = {
    version: 8,
    sources: {
      dark: {
        type: 'raster',
        tiles: ['https://a.basemaps.cartocdn.com/dark_all/{z}/{x}/{y}.png'],
        tileSize: 256,
        attribution: '&copy; OpenStreetMap contributors &copy; CARTO'
      }
    },
    layers: [
      {
        id: 'dark',
        type: 'raster',
        source: 'dark'
      }
    ]
  };

  function selectMemento(memento: Memento) {
    selected = memento;
    mode = 'artifact';
    focusMap(memento);
  }

  function focusMap(memento: Memento) {
    if (!map) return;

    if (memento.transit) {
      const bounds = new maplibregl.LngLatBounds(memento.transit.from.coords, memento.transit.to.coords);
      map.fitBounds(bounds, {
        padding: { top: 90, bottom: 90, left: 80, right: 520 },
        maxZoom: 10.5,
        duration: 700
      });
      return;
    }

    map.flyTo({
      center: memento.coords,
      zoom: 9.6,
      duration: 700,
      essential: true
    });
  }

  function markerElement(memento: Memento) {
    const button = document.createElement('button');
    button.className = `map-marker map-marker--${memento.kind}`;
    button.type = 'button';
    button.setAttribute('aria-label', memento.title);
    button.innerHTML = `<span>${memento.kind === 'transit' ? '→' : memento.kind === 'stamp' ? '印' : '◆'}</span>`;
    button.addEventListener('click', () => selectMemento(memento));
    return button;
  }

  function syncMarkers() {
    markers.forEach((marker, id) => {
      const element = marker.getElement();
      element.classList.toggle('is-active', id === selected.id);
    });
  }

  function setupMap() {
    map = new maplibregl.Map({
      container: mapContainer,
      style: mapStyle,
      center: [137.2, 35.0],
      zoom: 5.8,
      attributionControl: false
    });

    map.addControl(new maplibregl.NavigationControl({ showCompass: false }), 'top-left');

    map.on('load', () => {
      if (!map) return;

      map.addSource('route', { type: 'geojson', data: routeGeoJson });
      map.addLayer({
        id: 'route-glow',
        type: 'line',
        source: 'route',
        paint: {
          'line-color': '#f97316',
          'line-width': 8,
          'line-opacity': 0.16,
          'line-blur': 4
        }
      });
      map.addLayer({
        id: 'route',
        type: 'line',
        source: 'route',
        paint: {
          'line-color': '#fb923c',
          'line-width': 3,
          'line-opacity': 0.9
        }
      });

      map.addSource('transit', { type: 'geojson', data: transitGeoJson });
      map.addLayer({
        id: 'transit',
        type: 'line',
        source: 'transit',
        paint: {
          'line-color': '#fde68a',
          'line-width': 4,
          'line-opacity': 0.9
        }
      });

      for (const memento of mementos) {
        const marker = new maplibregl.Marker({ element: markerElement(memento), anchor: 'center' })
          .setLngLat(memento.coords)
          .addTo(map);
        markers.set(memento.id, marker);
      }

      syncMarkers();
      focusMap(selected);
    });
  }

  $: if (markers.size) {
    syncMarkers();
  }

  onMount(() => {
    setupMap();

    return () => {
      markers.forEach(marker => marker.remove());
      markers.clear();
      map?.remove();
    };
  });

  async function openMode(nextMode: Mode) {
    mode = nextMode;
    await tick();
    syncMarkers();
  }
</script>

<main class="app-shell">
  <section class="map-stage" aria-label="Journey map">
    <div bind:this={mapContainer} class="map-canvas"></div>

    <header class="topbar">
      <div>
        <p class="eyebrow">felicia decision demo</p>
        <h1>Japan Golden Route</h1>
      </div>
      <div class="trip-stat">
        <span>May 10-18, 2026</span>
        <strong>{mementos.length} mementos</strong>
      </div>
    </header>

    <nav class="mode-switch" aria-label="Demo modes">
      <button class:active={mode === 'artifact'} on:click={() => openMode('artifact')}>Artifact</button>
      <button class:active={mode === 'review'} on:click={() => openMode('review')}>Review Queue</button>
      <button class:active={mode === 'creator'} on:click={() => openMode('creator')}>Transit Creator</button>
    </nav>

    <aside class="priority-strip" aria-label="Roadmap priorities">
      {#each priorities as priority}
        <article>
          <span>{priority.label}</span>
          <div>
            <strong>{priority.title}</strong>
            <p>{priority.detail}</p>
          </div>
        </article>
      {/each}
    </aside>
  </section>

  <aside class="decision-panel" aria-label="Decision panel">
    {#if mode === 'artifact'}
      <section class="panel-section artifact-view">
        <div class="section-head">
          <p class="eyebrow">public artifact moat</p>
          <h2>{selected.title}</h2>
        </div>

        <button class="stub-card {selected.kind}" on:click={() => focusMap(selected)}>
          {#if selected.kind === 'transit' && selected.transit}
            <div class="ticket-face">
              <div class="ticket-line">
                <span>{selected.transit.operator}</span>
                <strong>{selected.transit.line}</strong>
              </div>
              <div class="station-pair">
                <span>{selected.transit.from.ja}</span>
                <b>→</b>
                <span>{selected.transit.to.ja}</span>
              </div>
              <div class="ticket-meta">
                <span>{selected.date}</span>
                <span>{selected.transit.fare}</span>
              </div>
            </div>
          {:else if selected.kind === 'stamp'}
            <div class="stamp-face">
              <span>御朱印</span>
              <strong>{selected.place}</strong>
              <small>{selected.date}</small>
            </div>
          {:else}
            <div class="goods-face">
              <span>GOODS</span>
              <strong>{selected.title}</strong>
              <small>{selected.vendor} · {selected.price}</small>
            </div>
          {/if}
        </button>

        <article class="essay">
          <span>The Story</span>
          <p>{selected.essay}</p>
        </article>

        {#if selected.photos.length}
          <div class="gallery">
            {#each selected.photos as photo}
              <figure>
                <img src={photo.src} alt={selected.title} />
                <figcaption>{photo.caption}</figcaption>
              </figure>
            {/each}
          </div>
        {/if}

        <div class="memento-list">
          {#each mementos as memento}
            <button class:active={memento.id === selected.id} on:click={() => selectMemento(memento)}>
              <span>{memento.kind}</span>
              <strong>{memento.title}</strong>
            </button>
          {/each}
        </div>
      </section>
    {:else if mode === 'review'}
      <section class="panel-section">
        <div class="section-head">
          <p class="eyebrow">source/ingest review queue</p>
          <h2>Confirm what becomes a memento</h2>
        </div>

        <div class="queue-list">
          {#each mementos as memento}
            <article class="queue-item">
              <div>
                <span>{memento.source}</span>
                <strong>{memento.title}</strong>
                <p>{memento.place} · {memento.date}</p>
              </div>
              <meter min="0" max="100" value={memento.confidence}>{memento.confidence}</meter>
              <div class="queue-actions">
                <button class:selected={memento.reviewState === 'confirm'}>Confirm</button>
                <button class:selected={memento.reviewState === 'merge'}>Merge</button>
                <button class:selected={memento.reviewState === 'private'}>Private</button>
              </div>
            </article>
          {/each}
        </div>
      </section>
    {:else}
      <section class="panel-section">
        <div class="section-head">
          <p class="eyebrow">manual edge-anchored creator</p>
          <h2>Transit can draw the route</h2>
        </div>

        <div class="creator-card">
          <label>
            Operator
            <select>
              <option>Tokyo Metro</option>
              <option>JR Central</option>
            </select>
          </label>
          <label>
            Line
            <input value="Ginza Line" />
          </label>
          <div class="creator-grid">
            <label>
              From
              <select>
                <option>Shibuya</option>
                <option>Tokyo</option>
              </select>
            </label>
            <label>
              To
              <select>
                <option>Asakusa</option>
                <option>Kyoto</option>
              </select>
            </label>
          </div>
          <button>Create transit memento</button>
        </div>

        <article class="roadmap-note">
          <strong>Roadmap read:</strong>
          <p>
            This should come after the public artifact view, but before passive automation if
            Japan rail trips are central. It proves route segments can be authored, not only ingested.
          </p>
        </article>
      </section>
    {/if}
  </aside>
</main>
