import { useEffect, useRef, useState } from 'react';
import L from 'leaflet';

// A station in the (MVP, offline) JR + Tokyo Metro catalog — name → coords.
interface Station {
  name: string;
  ja: string;
  coords: [number, number];
}

// Transit is EDGE-anchored (from → to); other kinds are point-anchored.
interface TransitLeg {
  operator: string;
  line: string;
  from: Station;
  to: Station;
}

// Define the structure of a travel Memento (matching Notion template/ER schema)
interface Memento {
  id: string;
  serial: string;
  title: string;
  kind: 'ticket' | 'stamp' | 'goods' | 'transit';
  vendor: string;
  price: string;
  date: string;
  barcode: string;
  coords: [number, number];
  essay: string;
  photo: string;
  caption: string;
  transit?: TransitLeg; // present only when kind === 'transit'
}

// MVP station catalog — JR + Tokyo Metro stops on the trip's lines (bundled, offline).
const stations: Station[] = [
  { name: 'Tokyo', ja: '東京', coords: [35.6812, 139.7671] },
  { name: 'Shinagawa', ja: '品川', coords: [35.6285, 139.7387] },
  { name: 'Shinjuku', ja: '新宿', coords: [35.6896, 139.7006] },
  { name: 'Shibuya', ja: '渋谷', coords: [35.6580, 139.7016] },
  { name: 'Ginza', ja: '銀座', coords: [35.6717, 139.7640] },
  { name: 'Asakusa', ja: '浅草', coords: [35.7148, 139.7967] },
  { name: 'Ueno', ja: '上野', coords: [35.7141, 139.7774] },
  { name: 'Mitaka', ja: '三鷹', coords: [35.7027, 139.5604] },
  { name: 'Shin-Yokohama', ja: '新横浜', coords: [35.5079, 139.6173] },
  { name: 'Nagoya', ja: '名古屋', coords: [35.1709, 136.8815] },
  { name: 'Kyoto', ja: '京都', coords: [34.9858, 135.7588] },
  { name: 'Shin-Osaka', ja: '新大阪', coords: [34.7335, 135.5003] },
  { name: 'Osaka-Umeda', ja: '大阪梅田', coords: [34.7025, 135.4959] },
];

const st = (name: string): Station =>
  stations.find(s => s.name === name) ?? stations[0];

const midpoint = (a: [number, number], b: [number, number]): [number, number] =>
  [(a[0] + b[0]) / 2, (a[1] + b[1]) / 2];

const operators = ['JR Central', 'JR East', 'JR West', 'Tokyo Metro', 'Toei Subway'];

const initialMementos: Memento[] = [
  {
    id: "memento-tokyo",
    serial: "FL-26-001",
    title: "Ghibli Museum Admission",
    kind: "ticket",
    vendor: "Lawson Mitaka / Studio Ghibli",
    price: "¥1,000",
    date: "2026.05.11",
    barcode: "MEM-TOKYO-0511",
    coords: [35.6963, 139.5704], // Mitaka, Tokyo
    essay: "Stepping into the Ghibli Museum felt like crossing the threshold into a beautifully animated dream. The winding spiral staircases, stained-glass windows depicting Totoro, and the giant robot sentinel standing guard on the rooftop garden under the soft afternoon sky. We sat in the tiny Saturn Theater, watching an exclusive short film, childhood wonder fully restored. It's not just a museum; it's a house built of memories and dreams.",
    photo: "/tokyo_night.jpg",
    caption: "Dusk over Tokyo skyline from Shibuya Sky, taken later that evening."
  },
  {
    id: "memento-metro-ginza",
    serial: "FL-26-002",
    title: "Shibuya → Asakusa",
    kind: "transit",
    vendor: "Tokyo Metro · Ginza Line",
    price: "¥210",
    date: "2026.05.12",
    barcode: "TR-SHIBUYA-ASAKUSA",
    coords: midpoint(st('Shibuya').coords, st('Asakusa').coords),
    essay: "The oldest subway line in Asia, the Ginza Line, rattled us east beneath the city from the youthful chaos of Shibuya to the temple-town calm of Asakusa. Standing room only, the orange-liveried carriage swaying, salarymen and tourists pressed shoulder to shoulder. We surfaced into the incense haze of Senso-ji as the lanterns flickered on.",
    photo: "",
    caption: "",
    transit: { operator: 'Tokyo Metro', line: 'Ginza Line', from: st('Shibuya'), to: st('Asakusa') }
  },
  {
    id: "memento-shinkansen",
    serial: "FL-26-003",
    title: "Tokyo → Kyoto",
    kind: "transit",
    vendor: "JR Central · Tōkaidō Shinkansen",
    price: "¥13,320",
    date: "2026.05.13",
    barcode: "TR-TOKYO-KYOTO",
    coords: midpoint(st('Tokyo').coords, st('Kyoto').coords),
    essay: "Nozomi 221, departing Tokyo at 09:00 sharp — to the second. We bought ekiben and cold green tea on the platform and settled in as the city gave way to suburbs, then to the wide Shizuoka plain. For a few breathless minutes Mt. Fuji filled the right-hand windows, impossibly large and snow-capped. 320 km/h, and the mountain barely seemed to move.",
    photo: "",
    caption: "",
    transit: { operator: 'JR Central', line: 'Tōkaidō Shinkansen', from: st('Tokyo'), to: st('Kyoto') }
  },
  {
    id: "memento-kyoto",
    serial: "FL-26-004",
    title: "Golden Pavilion Goshuin",
    kind: "stamp",
    vendor: "Kinkaku-ji Temple Officials",
    price: "¥500",
    date: "2026.05.14",
    barcode: "MEM-KYOTO-0514",
    coords: [35.0394, 135.7292], // Kinkaku-ji, Kyoto
    essay: "The Golden Pavilion rose out of the mirror-like pond, its brilliant gold leaf reflecting the vibrant red and orange autumn maples. We stood in line in absolute quiet as the calligrapher monk, with absolute grace and fluid motion, dipped his thick bamboo brush in black ink and hand-wrote the temple's blessing into our notebook, pressing the crimson vermilion seals into the paper. A moment of deep silence frozen in ink.",
    photo: "/kyoto_temple.jpg",
    caption: "The majestic Kinkaku-ji (Golden Pavilion) framed by early autumn colors."
  },
  {
    id: "memento-osaka",
    serial: "FL-26-005",
    title: "Fuwamiku Mascot Plush",
    kind: "goods",
    vendor: "Mascot Cafe & Shop Osaka",
    price: "¥2,400",
    date: "2026.05.16",
    barcode: "MEM-OSAKA-0516",
    coords: [34.6687, 135.5013], // Dotonbori, Osaka
    essay: "After a long afternoon wandering through the crowded, glowing neon alleys of Dotonbori, we found refuge in a quiet back-alley cafe. Sitting on the counter next to our matcha latte was this tiny, pink fluffy mascot plush. We couldn't leave without it. Now it sits on our desk—a constant, soft reminder of Osaka's warm cafes, cozy afternoons, and the hum of city lights outside the window.",
    photo: "/osaka_plushie.jpg",
    caption: "Our new plush companion sitting cozy next to a custom matcha latte."
  }
];

const routeCoordinates: [number, number][] = [
  [35.6963, 139.5704], // Tokyo stop
  [35.3500, 139.1000], // Hakone pass-through
  [35.1814, 136.9066], // Nagoya pass-through
  [35.0394, 135.7292], // Kyoto stop
  [34.6687, 135.5013]  // Osaka stop
];

interface CreatorForm {
  operator: string;
  line: string;
  from: string;
  to: string;
  date: string;
  fare: string;
}

const emptyForm: CreatorForm = {
  operator: 'JR Central',
  line: '',
  from: 'Tokyo',
  to: 'Kyoto',
  date: '2026-05-13',
  fare: '',
};

// Render a transit memento as a realistic JR / Metro magnetic ticket (きっぷ),
// template-first from structured data — the kind picks the designed form.
function renderTransitTicket(m: Memento, leg: TransitLeg) {
  const isMetro = /Metro|Toei/.test(leg.operator);
  const isShinkansen = /Shinkansen|新幹線/.test(leg.line);
  const typeLabel = isShinkansen ? '新幹線 乗車券' : '乗車券';

  return (
    <div className={`jr-ticket ${isMetro ? 'jr-ticket--metro' : ''}`}>
      <div className="jr-grain" aria-hidden="true"></div>

      <div className="jr-head">
        <span className="jr-type">{typeLabel}</span>
        <span className="jr-date">{m.date}</span>
      </div>

      <div className="jr-route">
        <div className="jr-stn">
          <span className="jr-stn-jp">{leg.from.ja}</span>
          <span className="jr-stn-en">{leg.from.name}</span>
        </div>
        <span className="jr-wave">〜</span>
        <div className="jr-stn jr-stn--to">
          <span className="jr-stn-jp">{leg.to.ja}</span>
          <span className="jr-stn-en">{leg.to.name}</span>
        </div>
      </div>

      <div className="jr-via">経由：{leg.line || leg.operator}</div>

      <div className="jr-foot">
        <div className="jr-fare-block">
          <span className="jr-yen">￥</span>
          <span className="jr-fare">{m.price.replace(/^¥/, '')}</span>
        </div>
        <span className="jr-fineprint">下車前途無効</span>
      </div>

      <div className="jr-magstripe">
        <span className="jr-mag-issue">{leg.operator}発行 · {m.serial}</span>
        <span className="jr-mag-code">{m.barcode}</span>
      </div>
    </div>
  );
}

export default function App() {
  const mapRef = useRef<HTMLDivElement>(null);
  const mapInstance = useRef<L.Map | null>(null);
  const overlayRef = useRef<L.LayerGroup | null>(null);

  const [mementos, setMementos] = useState<Memento[]>(initialMementos);
  const [selectedMemento, setSelectedMemento] = useState<Memento | null>(null);
  const [isCollapsed, setIsCollapsed] = useState(true);
  const [showCreator, setShowCreator] = useState(false);
  const [form, setForm] = useState<CreatorForm>(emptyForm);

  // Initialize Map
  useEffect(() => {
    if (!mapRef.current || mapInstance.current) return;

    const map = L.map(mapRef.current, {
      zoomControl: false,
      attributionControl: false
    }).setView([35.3, 137.6], 7);

    mapInstance.current = map;
    overlayRef.current = L.layerGroup().addTo(map);

    L.control.zoom({ position: 'topleft' }).addTo(map);

    // Dark Map Tiles (CartoDB Dark Matter)
    L.tileLayer('https://{s}.basemaps.cartocdn.com/dark_all/{z}/{x}/{y}{r}.png', {
      maxZoom: 19,
      subdomains: 'abcd'
    }).addTo(map);

    // Scenic base journey route (the passive-track look — dashed, marching)
    L.polyline(routeCoordinates, {
      color: '#ff6a00',
      weight: 3,
      opacity: 0.6,
      lineJoin: 'round'
    }).addTo(map);

    return () => {
      if (mapInstance.current) {
        mapInstance.current.remove();
        mapInstance.current = null;
      }
    };
  }, []);

  // Redraw overlay (markers + transit legs) when mementos or selection change
  useEffect(() => {
    const map = mapInstance.current;
    const overlay = overlayRef.current;
    if (!map || !overlay) return;

    overlay.clearLayers();

    mementos.forEach(memento => {
      const isActive = selectedMemento?.id === memento.id;

      // Transit mementos draw a solid leg + station endpoints
      if (memento.transit) {
        const { from, to } = memento.transit;
        L.polyline([from.coords, to.coords], {
          className: 'transit-leg',
          color: '#ff8a3d',
          weight: isActive ? 5 : 3,
          opacity: isActive ? 1 : 0.85,
          lineJoin: 'round'
        }).addTo(overlay);

        [from, to].forEach(s => {
          L.circleMarker(s.coords, {
            className: 'station-dot',
            radius: 4,
            color: '#ff8a3d',
            weight: 2,
            fillColor: '#09090b',
            fillOpacity: 1
          }).addTo(overlay);
        });
      }

      const customIcon = L.divIcon({
        className: 'stub-marker-container',
        html: `
          <div class="stub-marker ${isActive ? 'active' : ''}" id="marker-${memento.id}">
            <svg class="stub-marker-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              ${getIconPath(memento.kind)}
            </svg>
            <div class="stub-marker-dot"></div>
          </div>
        `,
        iconSize: [32, 44],
        iconAnchor: [16, 22]
      });

      const marker = L.marker(memento.coords, { icon: customIcon }).addTo(overlay);
      marker.on('click', () => handleSelect(memento));
    });
  }, [mementos, selectedMemento]);

  const handleSelect = (memento: Memento) => {
    setSelectedMemento(memento);
    setIsCollapsed(false);

    const map = mapInstance.current;
    if (!map) return;

    if (memento.transit) {
      // Frame the whole leg, leaving room for the right-hand panel
      map.fitBounds([memento.transit.from.coords, memento.transit.to.coords], {
        paddingTopLeft: [60, 90],
        paddingBottomRight: [480, 90],
        maxZoom: 11,
        animate: true
      });
    } else {
      map.setView(memento.coords, 9, { animate: true, duration: 0.8 });
    }
  };

  const handleClose = () => {
    setIsCollapsed(true);
    setSelectedMemento(null);
  };

  const handleCreate = (e: React.FormEvent) => {
    e.preventDefault();
    const from = st(form.from);
    const to = st(form.to);
    if (from.name === to.name) return;

    const n = mementos.length + 1;
    const newMemento: Memento = {
      id: `transit-${Date.now()}`,
      serial: `FL-26-${String(n).padStart(3, '0')}`,
      title: `${from.name} → ${to.name}`,
      kind: 'transit',
      vendor: form.line ? `${form.operator} · ${form.line}` : form.operator,
      price: form.fare || '—',
      date: form.date.replace(/-/g, '.'),
      barcode: `TR-${from.name}-${to.name}`.toUpperCase().replace(/[^A-Z0-9-]/g, ''),
      coords: midpoint(from.coords, to.coords),
      essay: `A ${form.line || form.operator} leg from ${from.name} to ${to.name}. Write the story of this ride here — the view from the window, who you sat beside, where you were headed.`,
      photo: '',
      caption: '',
      transit: { operator: form.operator, line: form.line, from, to }
    };

    setMementos(prev => [...prev, newMemento]);
    setShowCreator(false);
    setForm(emptyForm);
    handleSelect(newMemento);
  };

  function getIconPath(kind: string) {
    if (kind === 'ticket') {
      return `<rect width="18" height="12" x="3" y="6" rx="2"/><path d="M9 10v.01M15 10v.01M9 14v.01M15 14v.01"/>`;
    } else if (kind === 'stamp') {
      return `<path d="M12 22a7 7 0 0 0 7-7V9a3 3 0 0 0-6 0v6a3 3 0 0 1-6 0v-4a2 2 0 0 1 4 0"/><path d="M14 6V3a1 1 0 0 0-1-1h-2a1 1 0 0 0-1 1v3"/>`;
    } else if (kind === 'transit') {
      return `<rect width="16" height="16" x="4" y="3" rx="2"/><path d="M4 11h16"/><path d="M12 3v8"/><path d="m8 19-2 3"/><path d="m18 22-2-3"/><path d="M8 15h.01"/><path d="M16 15h.01"/>`;
    } else {
      return `<polygon points="12 2 15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2"/>`;
    }
  }

  return (
    <div className="app-container">
      {/* Background map view */}
      <main ref={mapRef} className="map-view"></main>

      {/* Floating application header */}
      <header className="app-header">
        <div className="logo-area">
          <span className="logo-dot"></span>
          <h1 className="logo-title">felicia</h1>
          <span className="logo-badge">TSX Prototype</span>
        </div>
        <p className="header-subtitle">Japan Golden Route '26</p>
      </header>

      {/* Add Ticket creator — floating action button */}
      <button className="fab-add-ticket" onClick={() => setShowCreator(true)}>
        <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="12" y1="5" x2="12" y2="19"></line><line x1="5" y1="12" x2="19" y2="12"></line></svg>
        Add Ticket
      </button>

      {/* Slide-out Scrapbook Side Drawer */}
      <aside className={`side-panel ${isCollapsed ? 'collapsed' : ''}`}>
        <button className="panel-close-btn" onClick={handleClose} aria-label="Close panel">
          <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="18" y1="6" x2="6" y2="18"></line><line x1="6" y1="6" x2="18" y2="18"></line></svg>
        </button>

        {/* Welcome screen: shown when no memento is active */}
        {!selectedMemento ? (
          <div className="welcome-view">
            <div className="welcome-content">
              <div className="welcome-icon">
                <svg className="route-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
                  <path d="M20 10c0 6-8 12-8 12s-8-6-8-12a8 8 0 0 1 16 0Z"/>
                  <circle cx="12" cy="10" r="3"/>
                </svg>
              </div>
              <h2>Japan Golden Route</h2>
              <p className="trip-meta">May 10 – May 18, 2026 • {mementos.length} mementos</p>
              <p className="trip-summary">
                A journey through Japan's historic temples, glowing neon cities, and cozy back-alley cafes. Each memory is anchored by a collectible memento along the amber path — including the transit legs that stitch the route together.
              </p>
              <div className="instruction-box">
                <p>Select an amber stub on the map or choose a memory below to open the scrapbook:</p>
                <div className="quick-list">
                  {mementos.map(m => (
                    <button key={m.id} className="quick-item-btn" onClick={() => handleSelect(m)}>
                      <div className="quick-item-info">
                        <span className="quick-item-title">{m.title}</span>
                        <span className="quick-item-meta">{m.date} • {m.kind.toUpperCase()}</span>
                      </div>
                      <svg className="quick-item-arrow" xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m9 18 6-6-6-6"/></svg>
                    </button>
                  ))}
                </div>
              </div>
            </div>
          </div>
        ) : (
          /* Memento detail view: stubs, essays and galleries */
          <div className="details-view">
            <div className="stub-container-wrapper">
              {selectedMemento.transit ? (
                renderTransitTicket(selectedMemento, selectedMemento.transit)
              ) : (
                <div className="memento-stub">
                  <div className="stub-left-notch"></div>
                  <div className="stub-right-notch"></div>

                  <div className="stub-header">
                    <span className="stub-serial">{selectedMemento.serial}</span>
                    <span className={`stub-badge kind-${selectedMemento.kind}`}>
                      {selectedMemento.kind.toUpperCase()}
                    </span>
                  </div>

                  <div className="stub-body">
                    <h3 className="stub-title">{selectedMemento.title}</h3>
                    <p className="stub-vendor">{selectedMemento.vendor}</p>

                    <div className="stub-meta-grid">
                      <div className="stub-meta-item">
                        <span className="meta-label">DATE</span>
                        <span className="meta-value">{selectedMemento.date}</span>
                      </div>
                      <div className="stub-meta-item">
                        <span className="meta-label">PRICE</span>
                        <span className="meta-value">{selectedMemento.price}</span>
                      </div>
                    </div>
                  </div>

                  <div className="stub-divider">
                    <span className="divider-line"></span>
                  </div>

                  <div className="stub-footer">
                    <div className="barcode-container">
                      <div className="barcode-bars">
                        <div className="bar thin"></div>
                        <div className="bar wide"></div>
                        <div className="bar thin"></div>
                        <div className="bar medium"></div>
                        <div className="bar wide"></div>
                        <div className="bar thin"></div>
                        <div className="bar medium"></div>
                        <div className="bar thin"></div>
                        <div className="bar wide"></div>
                      </div>
                      <span className="barcode-text">{selectedMemento.barcode}</span>
                    </div>
                  </div>
                </div>
              )}
            </div>

            <article className="scrapbook-body">
              <div className="essay-container">
                <h4 className="essay-heading">The Story</h4>
                <p className="essay-text">{selectedMemento.essay}</p>
              </div>

              {selectedMemento.photo && (
                <div className="gallery-container">
                  <h4 className="essay-heading">Photo Gallery</h4>
                  <div className="gallery-wrapper">
                    <img className="gallery-image" src={selectedMemento.photo} alt={selectedMemento.title} />
                    <div className="gallery-caption">{selectedMemento.caption}</div>
                  </div>
                </div>
              )}
            </article>
          </div>
        )}
      </aside>

      {/* Floating trip metrics bar at the bottom */}
      <div className="map-stats-panel">
        <div className="stat-item">
          <span className="stat-label">JOURNEY</span>
          <span className="stat-value">Japan Golden Route</span>
        </div>
        <div className="stat-item separator"></div>
        <div className="stat-item">
          <span className="stat-label">DISTANCE</span>
          <span className="stat-value">512 km</span>
        </div>
        <div className="stat-item separator"></div>
        <div className="stat-item">
          <span className="stat-label">MEMENTOS</span>
          <span className="stat-value">{mementos.length} Collected</span>
        </div>
      </div>

      {/* Transit ticket creator modal (manual authoring — the E path) */}
      {showCreator && (
        <div className="creator-overlay" onClick={() => setShowCreator(false)}>
          <form className="creator-modal" onClick={e => e.stopPropagation()} onSubmit={handleCreate}>
            <div className="creator-header">
              <h3>New Transit Ticket</h3>
              <button type="button" className="creator-close" onClick={() => setShowCreator(false)} aria-label="Close">
                <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="18" y1="6" x2="6" y2="18"></line><line x1="6" y1="6" x2="18" y2="18"></line></svg>
              </button>
            </div>

            <div className="creator-field">
              <label>Operator</label>
              <select value={form.operator} onChange={e => setForm({ ...form, operator: e.target.value })}>
                {operators.map(o => <option key={o} value={o}>{o}</option>)}
              </select>
            </div>

            <div className="creator-field">
              <label>Line <span className="optional">(optional)</span></label>
              <input type="text" placeholder="e.g. Tōkaidō Shinkansen" value={form.line} onChange={e => setForm({ ...form, line: e.target.value })} />
            </div>

            <div className="creator-row">
              <div className="creator-field">
                <label>From</label>
                <select value={form.from} onChange={e => setForm({ ...form, from: e.target.value })}>
                  {stations.map(s => <option key={s.name} value={s.name}>{s.name}</option>)}
                </select>
              </div>
              <div className="creator-field">
                <label>To</label>
                <select value={form.to} onChange={e => setForm({ ...form, to: e.target.value })}>
                  {stations.map(s => <option key={s.name} value={s.name}>{s.name}</option>)}
                </select>
              </div>
            </div>

            <div className="creator-row">
              <div className="creator-field">
                <label>Date</label>
                <input type="date" value={form.date} onChange={e => setForm({ ...form, date: e.target.value })} />
              </div>
              <div className="creator-field">
                <label>Fare</label>
                <input type="text" placeholder="¥13,320" value={form.fare} onChange={e => setForm({ ...form, fare: e.target.value })} />
              </div>
            </div>

            {form.from === form.to && (
              <p className="creator-warning">From and To must be different stations.</p>
            )}

            <div className="creator-actions">
              <button type="button" className="btn-secondary" onClick={() => setShowCreator(false)}>Cancel</button>
              <button type="submit" className="btn-primary" disabled={form.from === form.to}>Create Ticket</button>
            </div>
          </form>
        </div>
      )}
    </div>
  );
}
