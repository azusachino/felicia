import { useEffect, useRef, useState } from 'react';
import L from 'leaflet';

// Define the structure of a travel Memento (matching Notion template/ER schema)
interface Memento {
  id: string;
  serial: string;
  title: string;
  kind: 'ticket' | 'stamp' | 'goods';
  vendor: string;
  price: string;
  date: string;
  barcode: string;
  coords: [number, number];
  essay: string;
  photo: string;
  caption: string;
}

const mementosData: Memento[] = [
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
    id: "memento-kyoto",
    serial: "FL-26-002",
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
    serial: "FL-26-003",
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

export default function App() {
  const mapRef = useRef<HTMLDivElement>(null);
  const mapInstance = useRef<L.Map | null>(null);
  const markersRef = useRef<{ [key: string]: L.Marker }>({});

  const [selectedMemento, setSelectedMemento] = useState<Memento | null>(null);
  const [isCollapsed, setIsCollapsed] = useState(true);

  // Initialize Map
  useEffect(() => {
    if (!mapRef.current || mapInstance.current) return;

    // Create Map Instance
    const map = L.map(mapRef.current, {
      zoomControl: false,
      attributionControl: false
    }).setView([35.3, 137.6], 7);

    mapInstance.current = map;

    // Add custom zoom control
    L.control.zoom({ position: 'topleft' }).addTo(map);

    // Dark Map Tiles (CartoDB Dark Matter)
    L.tileLayer('https://{s}.basemaps.cartocdn.com/dark_all/{z}/{x}/{y}{r}.png', {
      maxZoom: 19,
      subdomains: 'abcd'
    }).addTo(map);

    // Draw Glowing Amber Route
    L.polyline(routeCoordinates, {
      color: '#ff6a00',
      weight: 3,
      opacity: 0.85,
      lineJoin: 'round'
    }).addTo(map);

    // Clean up on unmount
    return () => {
      if (mapInstance.current) {
        mapInstance.current.remove();
        mapInstance.current = null;
      }
    };
  }, []);

  // Update markers when selection changes
  useEffect(() => {
    if (!mapInstance.current) return;

    const map = mapInstance.current;

    // Remove old markers if any
    Object.values(markersRef.current).forEach(m => m.remove());
    markersRef.current = {};

    // Place new custom markers
    mementosData.forEach(memento => {
      const isActive = selectedMemento?.id === memento.id;

      // Custom DivIcon markup
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

      const marker = L.marker(memento.coords, { icon: customIcon }).addTo(map);
      markersRef.current[memento.id] = marker;

      // Click to Select
      marker.on('click', () => {
        handleSelect(memento);
      });
    });
  }, [selectedMemento]);

  const handleSelect = (memento: Memento) => {
    setSelectedMemento(memento);
    setIsCollapsed(false);

    if (mapInstance.current) {
      mapInstance.current.setView(memento.coords, 9, {
        animate: true,
        duration: 0.8
      });
    }
  };

  const handleClose = () => {
    setIsCollapsed(true);
    setSelectedMemento(null);
  };

  function getIconPath(kind: string) {
    if (kind === 'ticket') {
      return `<rect width="18" height="12" x="3" y="6" rx="2"/><path d="M9 10v.01M15 10v.01M9 14v.01M15 14v.01"/>`;
    } else if (kind === 'stamp') {
      return `<path d="M12 22a7 7 0 0 0 7-7V9a3 3 0 0 0-6 0v6a3 3 0 0 1-6 0v-4a2 2 0 0 1 4 0"/><path d="M14 6V3a1 1 0 0 0-1-1h-2a1 1 0 0 0-1 1v3"/>`;
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
              <p className="trip-meta">May 10 – May 18, 2026 • 3 stops</p>
              <p className="trip-summary">
                A journey through Japan's historic temples, glowing neon cities, and cozy back-alley cafes. Each memory is anchored by a collectible memento along the amber path.
              </p>
              <div className="instruction-box">
                <p>Select an amber stub on the map or choose a memory below to open the scrapbook:</p>
                <div className="quick-list">
                  {mementosData.map(m => (
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
            </div>

            <article className="scrapbook-body">
              <div className="essay-container">
                <h4 className="essay-heading">The Story</h4>
                <p className="essay-text">{selectedMemento.essay}</p>
              </div>

              <div className="gallery-container">
                <h4 className="essay-heading">Photo Gallery</h4>
                <div className="gallery-wrapper">
                  <img className="gallery-image" src={selectedMemento.photo} alt={selectedMemento.title} />
                  <div className="gallery-caption">{selectedMemento.caption}</div>
                </div>
              </div>
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
          <span className="stat-value">3 Collected</span>
        </div>
      </div>
    </div>
  );
}
