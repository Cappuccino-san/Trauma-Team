<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import * as maplibregl from 'maplibre-gl';
  import 'maplibre-gl/dist/maplibre-gl.css';
  import type { Feature, FeatureCollection } from 'geojson';
  import { activeHazards } from '../stores/weather';
  import { activeCorridor, currentVerdictTier } from '../stores/verdict';
  import type { GeoCoordinate } from '../types/envelope';

  let mapContainer: HTMLDivElement;
  let map: maplibregl.Map | null = null;
  let markers: maplibregl.Marker[] = [];
  let isExpanded = false;
  let lastCorridorKey = '';

  interface StationMeta {
    coords: [number, number];
    elevFt: number;
    role: string;
  }

  // Certified corridor station coordinates & field elevations (ft MSL)
  const STATION_META: Record<string, StationMeta> = {
    KAPA: { coords: [-104.849, 39.57], elevFt: 5885, role: 'BASE' },
    KDEN: { coords: [-104.673, 39.861], elevFt: 5434, role: 'HUB' },
    KLXV: { coords: [-106.317, 39.223], elevFt: 9927, role: 'HIGH-DA' },
    KASE: { coords: [-106.868, 39.223], elevFt: 7820, role: 'MTN-IFR' },
    KBJC: { coords: [-105.117, 39.908], elevFt: 5673, role: 'METRO' },
    KCOS: { coords: [-104.7, 38.805], elevFt: 6187, role: 'SOUTH' },
    KPUB: { coords: [-104.496, 38.289], elevFt: 4729, role: 'PLAINS' },
    KCYS: { coords: [-104.811, 41.155], elevFt: 6159, role: 'NORTH' },
    KGJT: { coords: [-108.526, 39.122], elevFt: 4858, role: 'ALT' },
    KBOS: { coords: [-71.005, 42.364], elevFt: 20, role: 'NE-BASE' },
    KMHT: { coords: [-71.438, 42.934], elevFt: 266, role: 'CORRIDOR' },
    KPWM: { coords: [-70.309, 43.646], elevFt: 76, role: 'NE-DEST' },
    KPVD: { coords: [-71.428, 41.724], elevFt: 55, role: 'ALT' },
    KCON: { coords: [-71.502, 43.203], elevFt: 342, role: 'ENROUTE' },
    KPSM: { coords: [-70.823, 43.078], elevFt: 100, role: 'COASTAL' }
  };

  const TERRAIN_OBSTACLES: Array<{ label: string; elevFt: number; coords: [number, number] }> = [
    { label: 'MT ELBERT', elevFt: 14440, coords: [-106.445, 39.118] },
    { label: 'MOSQUITO PASS', elevFt: 13185, coords: [-106.186, 39.281] },
    { label: 'MT BLUE SKY', elevFt: 14265, coords: [-105.644, 39.588] }
  ];

  // Procedural 100% air-gapped GeoJSON graticule & tactical terrain sectors (zero external tiles)
  function buildProceduralGraticule(): FeatureCollection {
    const features: Feature[] = [];
    for (let lon = -112; lon <= -66; lon += 1) {
      features.push({
        type: 'Feature',
        properties: { major: lon % 2 === 0 },
        geometry: {
          type: 'LineString',
          coordinates: [
            [lon, 35],
            [lon, 47]
          ]
        }
      });
    }
    for (let lat = 35; lat <= 47; lat += 1) {
      features.push({
        type: 'Feature',
        properties: { major: lat % 2 === 0 },
        geometry: {
          type: 'LineString',
          coordinates: [
            [-112, lat],
            [-66, lat]
          ]
        }
      });
    }
    return { type: 'FeatureCollection', features };
  }

  function buildProceduralTerrainGeoJson(): FeatureCollection {
    return {
      type: 'FeatureCollection',
      features: [
        // Colorado High Rockies / Continental Divide 10,000+ ft Terrain Envelope
        {
          type: 'Feature',
          properties: { name: 'ROCKIES >10,000 FT MSL' },
          geometry: {
            type: 'Polygon',
            coordinates: [
              [
                [-107.3, 38.4],
                [-105.6, 38.5],
                [-105.35, 39.6],
                [-105.55, 40.6],
                [-106.7, 40.6],
                [-107.2, 39.4],
                [-107.3, 38.4]
              ]
            ]
          }
        },
        // Sawatch / Leadville Alpine Ridge (>12,000 ft peaks)
        {
          type: 'Feature',
          properties: { name: 'SAWATCH ALPINE >12,000 FT' },
          geometry: {
            type: 'Polygon',
            coordinates: [
              [
                [-106.75, 38.9],
                [-106.05, 38.95],
                [-106.05, 39.55],
                [-106.7, 39.55],
                [-106.75, 38.9]
              ]
            ]
          }
        },
        // New England Coastal & White Mountain Corridor Sector
        {
          type: 'Feature',
          properties: { name: 'NEW ENGLAND COASTAL SECTOR' },
          geometry: {
            type: 'Polygon',
            coordinates: [
              [
                [-72.0, 41.5],
                [-70.0, 41.6],
                [-69.8, 44.0],
                [-71.9, 44.0],
                [-72.0, 41.5]
              ]
            ]
          }
        }
      ]
    };
  }

  function buildGeoJsonFromHazards(hazards: typeof $activeHazards): FeatureCollection {
    const features: Feature[] = [];

    hazards.forEach((h) => {
      if (!h.active || !h.polygon || h.polygon.length < 3) return;

      const ring: [number, number][] = h.polygon.map((pt: GeoCoordinate) => [
        pt.longitude,
        pt.latitude
      ]);
      if (ring[0][0] !== ring[ring.length - 1][0] || ring[0][1] !== ring[ring.length - 1][1]) {
        ring.push([...ring[0]]);
      }

      features.push({
        type: 'Feature',
        properties: {
          id: h.hazard_id || h.advisory_id,
          type: h.hazard_type,
          severity: h.severity,
          summary: h.summary
        },
        geometry: {
          type: 'Polygon',
          coordinates: [ring]
        }
      });
    });

    return { type: 'FeatureCollection', features };
  }

  function getCorridorStations(corridor: typeof $activeCorridor): string[] {
    return [corridor.departure, ...(corridor.intermediates || []), corridor.destination]
      .map((s) => s.trim().toUpperCase())
      .filter((s) => Boolean(STATION_META[s]));
  }

  function buildRouteGeoJson(corridor: typeof $activeCorridor): FeatureCollection {
    const stList = getCorridorStations(corridor);
    const coordinates = stList.map((s) => STATION_META[s].coords);

    const features: Feature[] = [];
    if (coordinates.length >= 2) {
      features.push({
        type: 'Feature',
        properties: { name: 'Flight Corridor' },
        geometry: {
          type: 'LineString',
          coordinates
        }
      });
    }

    stList.forEach((st) => {
      features.push({
        type: 'Feature',
        properties: { icao: st, elevFt: STATION_META[st].elevFt },
        geometry: {
          type: 'Point',
          coordinates: STATION_META[st].coords
        }
      });
    });

    return { type: 'FeatureCollection', features };
  }

  function getVerdictStrokeColor(tier: string): string {
    switch (tier) {
      case 'NO-GO':
        return '#ff1744';
      case 'CAUTION':
        return '#ffab00';
      case 'IFR':
        return '#00e5ff';
      default:
        return '#00e676';
    }
  }

  function syncWaypointMarkers() {
    if (!map) return;
    markers.forEach((m) => m.remove());
    markers = [];

    const stList = getCorridorStations($activeCorridor);
    const accentColor = getVerdictStrokeColor($currentVerdictTier);

    stList.forEach((icao, idx) => {
      const meta = STATION_META[icao];
      if (!meta || !map) return;
      const el = document.createElement('div');
      el.className = 'icao-chip-marker';
      el.style.borderColor = accentColor;
      const tag = idx === 0 ? 'DEP' : idx === stList.length - 1 ? 'DST' : 'WPT';
      el.innerHTML = `<span class="chip-tag">${tag}</span><span class="chip-icao">${icao}</span><span class="chip-elev">${meta.elevFt.toLocaleString()}'</span>`;
      const m = new maplibregl.Marker({ element: el, anchor: 'bottom', offset: [0, -8] })
        .setLngLat(meta.coords)
        .addTo(map);
      markers.push(m);
    });

    const isColorado = stList.some((s) => STATION_META[s] && STATION_META[s].coords[0] < -90);
    if (isColorado) {
      TERRAIN_OBSTACLES.forEach((obs) => {
        if (!map) return;
        const el = document.createElement('div');
        el.className = 'terrain-peak-marker';
        el.textContent = `▲ ${obs.label} ${obs.elevFt.toLocaleString()}'`;
        const m = new maplibregl.Marker({ element: el, anchor: 'top', offset: [0, 6] })
          .setLngLat(obs.coords)
          .addTo(map);
        markers.push(m);
      });
    }
  }

  function updateLayers() {
    if (!map || !map.isStyleLoaded()) return;

    const hazardSource = map.getSource('hazards') as maplibregl.GeoJSONSource | undefined;
    if (hazardSource) {
      hazardSource.setData(buildGeoJsonFromHazards($activeHazards));
    }

    const routeSource = map.getSource('route') as maplibregl.GeoJSONSource | undefined;
    if (routeSource) {
      routeSource.setData(buildRouteGeoJson($activeCorridor));
    }

    const strokeColor = getVerdictStrokeColor($currentVerdictTier);
    if (map.getLayer('route-line')) {
      map.setPaintProperty('route-line', 'line-color', strokeColor);
    }
    if (map.getLayer('route-points')) {
      map.setPaintProperty('route-points', 'circle-stroke-color', strokeColor);
    }

    syncWaypointMarkers();
  }

  $: if (map && ($activeHazards || $activeCorridor || $currentVerdictTier)) {
    updateLayers();
  }

  function focusCorridor() {
    if (!map) return;
    const stList = getCorridorStations($activeCorridor);

    if (stList.length >= 2) {
      const bounds = new maplibregl.LngLatBounds(
        STATION_META[stList[0]].coords,
        STATION_META[stList[1]].coords
      );
      stList.forEach((st) => bounds.extend(STATION_META[st].coords));
      map.fitBounds(bounds, { padding: 70, maxZoom: 8.2, duration: 350 });
    } else if (stList.length === 1) {
      map.flyTo({ center: STATION_META[stList[0]].coords, zoom: 7.5 });
    }
  }

  $: if (map && $activeCorridor) {
    const key = `${$activeCorridor.departure}-${($activeCorridor.intermediates || []).join(',')}-${$activeCorridor.destination}`;
    if (key !== lastCorridorKey) {
      lastCorridorKey = key;
      focusCorridor();
    }
  }

  function toggleExpand() {
    isExpanded = !isExpanded;
    setTimeout(() => {
      if (map) {
        map.resize();
        focusCorridor();
      }
    }, 60);
  }

  onMount(() => {
    // 100% air-gapped procedural aeronautical vector style (no external tiles or glyphs)
    const offlineStyle: maplibregl.StyleSpecification = {
      version: 8,
      sources: {
        graticule: {
          type: 'geojson',
          data: buildProceduralGraticule()
        },
        'terrain-sectors': {
          type: 'geojson',
          data: buildProceduralTerrainGeoJson()
        }
      },
      layers: [
        {
          id: 'tactical-void-bg',
          type: 'background',
          paint: {
            'background-color': '#05080c'
          }
        },
        {
          id: 'graticule-minor',
          type: 'line',
          source: 'graticule',
          filter: ['==', ['get', 'major'], false],
          paint: {
            'line-color': '#0d141f',
            'line-width': 1
          }
        },
        {
          id: 'graticule-major',
          type: 'line',
          source: 'graticule',
          filter: ['==', ['get', 'major'], true],
          paint: {
            'line-color': '#162133',
            'line-width': 1.2
          }
        },
        {
          id: 'terrain-sector-fill',
          type: 'fill',
          source: 'terrain-sectors',
          paint: {
            'fill-color': '#0d1624',
            'fill-opacity': 0.72
          }
        },
        {
          id: 'terrain-sector-outline',
          type: 'line',
          source: 'terrain-sectors',
          paint: {
            'line-color': '#1f304a',
            'line-width': 1.2,
            'line-dasharray': [4, 3]
          }
        }
      ]
    };

    map = new maplibregl.Map({
      container: mapContainer,
      style: offlineStyle,
      center: [-105.6, 39.4],
      zoom: 6.4,
      attributionControl: false
    });

    map.addControl(
      new maplibregl.NavigationControl({ showCompass: true, showZoom: true }),
      'top-right'
    );

    map.on('load', () => {
      map?.addSource('hazards', {
        type: 'geojson',
        data: buildGeoJsonFromHazards($activeHazards)
      });

      map?.addLayer({
        id: 'hazards-fill',
        type: 'fill',
        source: 'hazards',
        paint: {
          'fill-color': '#ff1744',
          'fill-opacity': 0.28
        }
      });

      map?.addLayer({
        id: 'hazards-line',
        type: 'line',
        source: 'hazards',
        paint: {
          'line-color': '#ff1744',
          'line-width': 2.5,
          'line-dasharray': [2, 2]
        }
      });

      map?.addSource('route', {
        type: 'geojson',
        data: buildRouteGeoJson($activeCorridor)
      });

      map?.addLayer({
        id: 'route-line',
        type: 'line',
        source: 'route',
        filter: ['==', '$type', 'LineString'],
        paint: {
          'line-color': getVerdictStrokeColor($currentVerdictTier),
          'line-width': 3
        }
      });

      map?.addLayer({
        id: 'route-points',
        type: 'circle',
        source: 'route',
        filter: ['==', '$type', 'Point'],
        paint: {
          'circle-radius': 6,
          'circle-color': '#05080c',
          'circle-stroke-width': 2.5,
          'circle-stroke-color': getVerdictStrokeColor($currentVerdictTier)
        }
      });

      updateLayers();
      setTimeout(focusCorridor, 100);
    });
  });

  onDestroy(() => {
    markers.forEach((m) => m.remove());
    markers = [];
    if (map) {
      map.remove();
      map = null;
    }
  });
</script>

<div class="map-card" class:expanded={isExpanded} aria-label="Aeronautical Route & SIGMET Overlay Map">
  <div class="panel-header">
    <div class="flex items-center gap-2 min-w-0">
      <span class="truncate">03 // AERONAUTICAL CORRIDOR &amp; SIGMET GEODESIC</span>
      <span class="corridor-summary">
        {$activeCorridor.departure || 'DEP'} → {($activeCorridor.intermediates || []).join(' → ') || 'DIRECT'} → {$activeCorridor.destination || 'DEST'}
      </span>
    </div>
    <div class="map-actions">
      <button type="button" class="btn-action" on:click={focusCorridor}>CENTER ROUTE</button>
      <button type="button" class="btn-action btn-expand" on:click={toggleExpand}>
        {isExpanded ? 'STANDARD (480PX)' : 'EXPAND (680PX)'}
      </button>
    </div>
  </div>

  <div class="map-container-relative">
    <div class="map-viewport" class:viewport-expanded={isExpanded} bind:this={mapContainer}></div>

    <div class="map-hud-overlay">
      <div class="hud-item">
        <span class="hud-label">CORRIDOR</span>
        <span class="hud-val text-cyan-400">{$activeCorridor.departure} → {$activeCorridor.destination}</span>
      </div>
      <div class="hud-item">
        <span class="hud-label">ACTIVE HAZARDS</span>
        <span class="hud-val" class:text-rose-400={$activeHazards.length > 0} class:text-emerald-400={$activeHazards.length === 0}>
          {$activeHazards.length} {$activeHazards.length === 1 ? 'SIGMET' : 'SIGMETS'}
        </span>
      </div>
      <div class="hud-item">
        <span class="hud-label">BASEMAP</span>
        <span class="hud-val text-emerald-400">OFFLINE VECTOR</span>
      </div>
    </div>
  </div>
</div>

<style>
  .map-card {
    background: var(--bg-panel);
    border: 1px solid var(--border-hairline);
    display: flex;
    flex-direction: column;
  }

  .corridor-summary {
    font-size: 10px;
    font-weight: 700;
    color: var(--color-go);
    letter-spacing: 0.05em;
    background: rgba(0, 230, 118, 0.1);
    padding: 1px 6px;
    border: 1px solid rgba(0, 230, 118, 0.3);
    white-space: nowrap;
  }

  .map-actions {
    display: flex;
    gap: 4px;
    align-items: center;
    flex-shrink: 0;
  }

  .btn-action {
    font-size: 9px;
    padding: 2px 7px;
    font-weight: 700;
  }

  .btn-expand {
    border-color: var(--color-info);
    color: var(--color-info);
  }

  .btn-expand:hover {
    background: var(--color-info);
    color: var(--bg-void);
  }

  .map-container-relative {
    position: relative;
    width: 100%;
  }

  .map-viewport {
    width: 100%;
    height: 480px;
    min-height: 480px;
    background: #05080c;
    transition: height 0.2s ease-in-out;
  }

  .viewport-expanded {
    height: 680px;
    min-height: 680px;
  }

  .map-hud-overlay {
    position: absolute;
    bottom: 12px;
    left: 12px;
    display: flex;
    gap: 8px;
    background: rgba(7, 8, 10, 0.9);
    border: 1px solid var(--border-strong);
    padding: 4px 8px;
    backdrop-filter: blur(4px);
    z-index: 10;
    pointer-events: none;
  }

  .hud-item {
    display: flex;
    flex-direction: column;
    padding: 0 4px;
  }

  .hud-label {
    font-size: 8px;
    color: var(--text-muted);
    letter-spacing: 0.06em;
  }

  .hud-val {
    font-size: 10px;
    font-weight: 800;
    font-family: var(--font-mono);
  }

  :global(.icao-chip-marker) {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    padding: 2px 5px;
    background: rgba(5, 8, 12, 0.92);
    border: 1px solid #00e676;
    font-family: var(--font-mono, ui-monospace, monospace);
    font-size: 10px;
    line-height: 1.1;
    color: #e6edf3;
    white-space: nowrap;
    pointer-events: none;
  }

  :global(.icao-chip-marker .chip-tag) {
    font-size: 8px;
    font-weight: 700;
    color: #8c96a8;
  }

  :global(.icao-chip-marker .chip-icao) {
    font-weight: 800;
    color: #ffffff;
  }

  :global(.icao-chip-marker .chip-elev) {
    font-size: 9px;
    color: #00e5ff;
  }

  :global(.terrain-peak-marker) {
    padding: 1px 4px;
    background: rgba(18, 21, 28, 0.85);
    border: 1px solid #2d3545;
    color: #ffab00;
    font-family: var(--font-mono, ui-monospace, monospace);
    font-size: 8px;
    font-weight: 700;
    white-space: nowrap;
    pointer-events: none;
  }

  :global(.maplibregl-canvas) {
    outline: none;
  }

  :global(.maplibregl-ctrl-group) {
    background: #0c0e12 !important;
    border: 1px solid #2d3545 !important;
  }

  :global(.maplibregl-ctrl-group button) {
    border-bottom: 1px solid #1e232e !important;
  }

  :global(.maplibregl-ctrl-icon) {
    filter: invert(1);
  }
</style>
