<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import * as maplibregl from 'maplibre-gl';
  import maplibreWorkerUrl from 'maplibre-gl/dist/maplibre-gl-worker.mjs?url';
  import 'maplibre-gl/dist/maplibre-gl.css';
  import type { Feature, FeatureCollection } from 'geojson';
  import { activeHazards, stationsById } from '../stores/weather';
  import { activeCorridor, currentVerdictTier } from '../stores/verdict';
  import { latestKinematics } from '../stores/kinematics';
  import type { GeoCoordinate } from '../types/envelope';

  let mapContainer: HTMLDivElement;
  let map: maplibregl.Map | null = null;
  let markers: maplibregl.Marker[] = [];
  let isExpanded = false;
  let mapLoaded = false;
  let basemapLabel: 'CARTO DARK' | 'OFFLINE VECTOR' = 'CARTO DARK';
  let lastCorridorKey = '';
  let focusTimer: ReturnType<typeof setTimeout> | null = null;
  let resizeTimer: ReturnType<typeof setTimeout> | null = null;
  let pendingRafId: number | null = null;
  let syncDirty = false;
  let resizeObserver: ResizeObserver | null = null;

  let appliedHazardsRef: typeof $activeHazards | null = null;
  let appliedCorridorSig = '';
  let appliedStrokeColor = '';
  let appliedMarkersSig = '';

  $: effectiveVerdictTier =
    $currentVerdictTier === 'NO-GO' || !$latestKinematics.is_feasible
      ? 'NO-GO'
      : $currentVerdictTier;

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

  interface ScreenBox {
    left: number;
    top: number;
    right: number;
    bottom: number;
  }

  type MarkerAnchor =
    | 'bottom'
    | 'top'
    | 'left'
    | 'right'
    | 'bottom-left'
    | 'bottom-right'
    | 'top-left'
    | 'top-right';

  const ANCHOR_OFFSETS: Record<MarkerAnchor, [number, number]> = {
    bottom: [0, -10],
    top: [0, 10],
    left: [12, 0],
    right: [-12, 0],
    'bottom-left': [10, -8],
    'bottom-right': [-10, -8],
    'top-left': [10, 8],
    'top-right': [-10, 8]
  };

  function computeLabelBox(
    px: number,
    py: number,
    width: number,
    height: number,
    anchor: MarkerAnchor
  ): ScreenBox {
    const [ox, oy] = ANCHOR_OFFSETS[anchor];
    const ax = px + ox;
    const ay = py + oy;
    switch (anchor) {
      case 'bottom':
        return { left: ax - width / 2, right: ax + width / 2, top: ay - height, bottom: ay };
      case 'top':
        return { left: ax - width / 2, right: ax + width / 2, top: ay, bottom: ay + height };
      case 'left':
        return { left: ax, right: ax + width, top: ay - height / 2, bottom: ay + height / 2 };
      case 'right':
        return { left: ax - width, right: ax, top: ay - height / 2, bottom: ay + height / 2 };
      case 'bottom-left':
        return { left: ax, right: ax + width, top: ay - height, bottom: ay };
      case 'bottom-right':
        return { left: ax - width, right: ax, top: ay - height, bottom: ay };
      case 'top-left':
        return { left: ax, right: ax + width, top: ay, bottom: ay + height };
      case 'top-right':
        return { left: ax - width, right: ax, top: ay, bottom: ay + height };
    }
  }

  function boxesOverlap(a: ScreenBox, b: ScreenBox, pad = 6): boolean {
    return !(
      a.right + pad <= b.left ||
      a.left - pad >= b.right ||
      a.bottom + pad <= b.top ||
      a.top - pad >= b.bottom
    );
  }

  function boxWithinViewport(box: ScreenBox, vw: number, vh: number): boolean {
    if (vw <= 0 || vh <= 0) return true;
    return box.left >= 8 && box.right <= vw - 44 && box.top >= 8 && box.bottom <= vh - 56;
  }

  function pickNonOverlappingPlacement(
    coords: [number, number],
    width: number,
    height: number,
    candidates: MarkerAnchor[],
    occupied: ScreenBox[],
    allowFallback: boolean
  ): { anchor: MarkerAnchor; offset: [number, number]; box: ScreenBox } | null {
    if (!map || typeof map.project !== 'function') {
      const anchor = candidates[0];
      return { anchor, offset: ANCHOR_OFFSETS[anchor], box: { left: 0, right: width, top: 0, bottom: height } };
    }
    const pt = map.project(coords);
    const vw = mapContainer?.clientWidth || 0;
    const vh = mapContainer?.clientHeight || 0;

    for (const anchor of candidates) {
      const box = computeLabelBox(pt.x, pt.y, width, height, anchor);
      if (boxWithinViewport(box, vw, vh) && !occupied.some((o) => boxesOverlap(box, o))) {
        return { anchor, offset: ANCHOR_OFFSETS[anchor], box };
      }
    }

    for (const anchor of candidates) {
      const box = computeLabelBox(pt.x, pt.y, width, height, anchor);
      if (!occupied.some((o) => boxesOverlap(box, o))) {
        return { anchor, offset: ANCHOR_OFFSETS[anchor], box };
      }
    }

    if (!allowFallback) return null;
    const fallback = candidates[0];
    return {
      anchor: fallback,
      offset: ANCHOR_OFFSETS[fallback],
      box: computeLabelBox(pt.x, pt.y, width, height, fallback)
    };
  }

  function syncWaypointMarkers() {
    if (!map) return;
    markers.forEach((m) => m.remove());
    markers = [];

    const stList = getCorridorStations($activeCorridor);
    const accentColor = getVerdictStrokeColor(effectiveVerdictTier);
    const occupiedBoxes: ScreenBox[] = [];

    // Reserve small dot exclusion boxes at each station coordinate so labels avoid covering route circles
    if (typeof map.project === 'function') {
      stList.forEach((icao) => {
        const meta = STATION_META[icao];
        if (!meta || !map) return;
        const pt = map.project(meta.coords);
        occupiedBoxes.push({ left: pt.x - 8, right: pt.x + 8, top: pt.y - 8, bottom: pt.y + 8 });
      });
    }

    const stationAnchors: MarkerAnchor[] = [
      'bottom',
      'top',
      'left',
      'right',
      'bottom-left',
      'bottom-right',
      'top-left',
      'top-right'
    ];

    stList.forEach((icao, idx) => {
      const meta = STATION_META[icao];
      if (!meta || !map) return;
      const tag = idx === 0 ? 'DEP' : idx === stList.length - 1 ? 'DST' : 'WPT';
      const elevStr = `${meta.elevFt.toLocaleString()}'`;
      const estWidth = 36 + (tag.length + icao.length + elevStr.length) * 6.4;
      const placement = pickNonOverlappingPlacement(
        meta.coords,
        estWidth,
        22,
        stationAnchors,
        occupiedBoxes,
        true
      );
      if (!placement) return;
      occupiedBoxes.push(placement.box);

      const el = document.createElement('div');
      el.className = 'icao-chip-marker';
      el.style.borderColor = accentColor;
      el.style.zIndex = '20';
      const stMetar = $stationsById[icao]?.metar;
      if (stMetar?.raw_text) {
        el.title = `${icao} (${stMetar.flight_category}): ${stMetar.raw_text}`;
      }
      el.innerHTML = `<span class="chip-tag">${tag}</span><span class="chip-icao">${icao}</span><span class="chip-elev">${elevStr}</span>`;
      const m = new maplibregl.Marker({
        element: el,
        anchor: placement.anchor,
        offset: placement.offset
      })
        .setLngLat(meta.coords)
        .addTo(map);
      markers.push(m);
    });

    const isColorado = stList.some((s) => STATION_META[s] && STATION_META[s].coords[0] < -90);
    if (isColorado) {
      const peakAnchors: MarkerAnchor[] = [
        'top',
        'bottom',
        'right',
        'left',
        'top-right',
        'top-left',
        'bottom-right',
        'bottom-left'
      ];
      TERRAIN_OBSTACLES.forEach((obs) => {
        if (!map) return;
        const labelText = `▲ ${obs.label} ${obs.elevFt.toLocaleString()}'`;
        const estWidth = 16 + labelText.length * 6.0;
        const placement = pickNonOverlappingPlacement(
          obs.coords,
          estWidth,
          20,
          peakAnchors,
          occupiedBoxes,
          false
        );
        if (!placement) return;
        occupiedBoxes.push(placement.box);

        const el = document.createElement('div');
        el.className = 'terrain-peak-marker';
        el.style.zIndex = '10';
        el.textContent = labelText;
        const m = new maplibregl.Marker({
          element: el,
          anchor: placement.anchor,
          offset: placement.offset
        })
          .setLngLat(obs.coords)
          .addTo(map);
        markers.push(m);
      });
    }
  }

  function ensureOverlaySourcesAndLayers(): boolean {
    if (!map) return false;
    let added = false;

    if (!map.getSource('hazards')) {
      map.addSource('hazards', {
        type: 'geojson',
        data: buildGeoJsonFromHazards($activeHazards)
      });
      added = true;
    }

    if (!map.getLayer('hazards-fill')) {
      map.addLayer({
        id: 'hazards-fill',
        type: 'fill',
        source: 'hazards',
        paint: {
          'fill-color': '#ff1744',
          'fill-opacity': 0.28
        }
      });
      added = true;
    }

    if (!map.getLayer('hazards-line')) {
      map.addLayer({
        id: 'hazards-line',
        type: 'line',
        source: 'hazards',
        paint: {
          'line-color': '#ff1744',
          'line-width': 2.5,
          'line-dasharray': [2, 2]
        }
      });
      added = true;
    }

    if (!map.getSource('route')) {
      map.addSource('route', {
        type: 'geojson',
        data: buildRouteGeoJson($activeCorridor)
      });
      added = true;
    }

    const initialColor = getVerdictStrokeColor(effectiveVerdictTier);
    if (!map.getLayer('route-line')) {
      map.addLayer({
        id: 'route-line',
        type: 'line',
        source: 'route',
        filter: ['==', '$type', 'LineString'],
        paint: {
          'line-color': initialColor,
          'line-width': 3
        }
      });
      added = true;
    }

    if (!map.getLayer('route-points')) {
      map.addLayer({
        id: 'route-points',
        type: 'circle',
        source: 'route',
        filter: ['==', '$type', 'Point'],
        paint: {
          'circle-radius': 6,
          'circle-color': '#05080c',
          'circle-stroke-width': 2.5,
          'circle-stroke-color': initialColor
        }
      });
      added = true;
    }

    return added;
  }

  function applyLayerSync(forceMarkers = false) {
    if (!map || !mapLoaded) return;

    ensureOverlaySourcesAndLayers();

    const hazardsVal = $activeHazards;
    if (hazardsVal !== appliedHazardsRef) {
      appliedHazardsRef = hazardsVal;
      const hazardSource = map.getSource('hazards') as maplibregl.GeoJSONSource | undefined;
      if (hazardSource) {
        hazardSource.setData(buildGeoJsonFromHazards(hazardsVal));
      }
    }

    const corridorSig = `${$activeCorridor.departure}-${($activeCorridor.intermediates || []).join(',')}-${$activeCorridor.destination}`;
    if (corridorSig !== appliedCorridorSig) {
      appliedCorridorSig = corridorSig;
      const routeSource = map.getSource('route') as maplibregl.GeoJSONSource | undefined;
      if (routeSource) {
        routeSource.setData(buildRouteGeoJson($activeCorridor));
      }
    }

    const strokeColor = getVerdictStrokeColor(effectiveVerdictTier);
    if (strokeColor !== appliedStrokeColor) {
      appliedStrokeColor = strokeColor;
      if (map.getLayer('route-line')) {
        map.setPaintProperty('route-line', 'line-color', strokeColor);
      }
      if (map.getLayer('route-points')) {
        map.setPaintProperty('route-points', 'circle-stroke-color', strokeColor);
      }
    }

    const stationCats = getCorridorStations($activeCorridor)
      .map((s) => `${s}:${$stationsById[s]?.metar?.flight_category ?? ''}`)
      .join('|');
    const markersSig = `${corridorSig}::${strokeColor}::${stationCats}`;
    if (forceMarkers || markersSig !== appliedMarkersSig) {
      appliedMarkersSig = markersSig;
      syncWaypointMarkers();
    }
  }

  function cancelPendingSync() {
    if (pendingRafId !== null) {
      if (typeof cancelAnimationFrame === 'function') {
        cancelAnimationFrame(pendingRafId);
      } else {
        clearTimeout(pendingRafId);
      }
      pendingRafId = null;
    }
    syncDirty = false;
  }

  function scheduleLayerSync() {
    if (!map || !mapLoaded) return;
    syncDirty = true;
    if (pendingRafId !== null) return;

    const raf =
      typeof requestAnimationFrame === 'function'
        ? requestAnimationFrame
        : (cb: FrameRequestCallback) => setTimeout(() => cb(Date.now()), 0) as unknown as number;

    pendingRafId = raf(() => {
      pendingRafId = null;
      if (!syncDirty || !map || !mapLoaded) return;
      syncDirty = false;
      applyLayerSync();
    });
  }

  $: if (map && mapLoaded && ($activeHazards || $activeCorridor || $stationsById || effectiveVerdictTier)) {
    scheduleLayerSync();
  }

  function focusCorridor(duration = 350) {
    if (!map) return;
    if (focusTimer) {
      clearTimeout(focusTimer);
      focusTimer = null;
    }
    const stList = getCorridorStations($activeCorridor);

    if (stList.length >= 2) {
      const bounds = new maplibregl.LngLatBounds(
        STATION_META[stList[0]].coords,
        STATION_META[stList[1]].coords
      );
      stList.forEach((st) => bounds.extend(STATION_META[st].coords));
      map.fitBounds(bounds, { padding: 72, maxZoom: 8.2, duration });
    } else if (stList.length === 1) {
      map.flyTo({ center: STATION_META[stList[0]].coords, zoom: 7.5, duration });
    }
  }

  $: if (map && mapLoaded && $activeCorridor) {
    const key = `${$activeCorridor.departure}-${($activeCorridor.intermediates || []).join(',')}-${$activeCorridor.destination}`;
    if (key !== lastCorridorKey) {
      lastCorridorKey = key;
      focusCorridor();
    }
  }

  function toggleExpand() {
    isExpanded = !isExpanded;
    if (resizeTimer) clearTimeout(resizeTimer);
    resizeTimer = setTimeout(() => {
      if (map) {
        map.resize();
        focusCorridor(200);
      }
    }, 220);
  }

  onMount(() => {
    if (typeof window !== 'undefined' && typeof maplibregl.setWorkerUrl === 'function') {
      maplibregl.setWorkerUrl(new URL(maplibreWorkerUrl, window.location.href).href);
    }

    // CARTO Dark Matter raster basemap with procedural aeronautical graticule & terrain overlay/fallback
    const mapStyle: maplibregl.StyleSpecification = {
      version: 8,
      sources: {
        'carto-dark': {
          type: 'raster',
          tiles: [
            'https://a.basemaps.cartocdn.com/dark_all/{z}/{x}/{y}@2x.png',
            'https://b.basemaps.cartocdn.com/dark_all/{z}/{x}/{y}@2x.png',
            'https://c.basemaps.cartocdn.com/dark_all/{z}/{x}/{y}@2x.png',
            'https://d.basemaps.cartocdn.com/dark_all/{z}/{x}/{y}@2x.png'
          ],
          tileSize: 256,
          attribution: '© OpenStreetMap © CARTO'
        },
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
          id: 'carto-dark-layer',
          type: 'raster',
          source: 'carto-dark',
          paint: {
            'raster-opacity': 0.95
          }
        },
        {
          id: 'graticule-minor',
          type: 'line',
          source: 'graticule',
          filter: ['==', ['get', 'major'], false],
          paint: {
            'line-color': '#0d141f',
            'line-width': 1,
            'line-opacity': 0.35
          }
        },
        {
          id: 'graticule-major',
          type: 'line',
          source: 'graticule',
          filter: ['==', ['get', 'major'], true],
          paint: {
            'line-color': '#162133',
            'line-width': 1.2,
            'line-opacity': 0.45
          }
        },
        {
          id: 'terrain-sector-fill',
          type: 'fill',
          source: 'terrain-sectors',
          paint: {
            'fill-color': '#0d1624',
            'fill-opacity': 0.16
          }
        },
        {
          id: 'terrain-sector-outline',
          type: 'line',
          source: 'terrain-sectors',
          paint: {
            'line-color': '#1f304a',
            'line-width': 1.2,
            'line-dasharray': [4, 3],
            'line-opacity': 0.65
          }
        }
      ]
    };

    map = new maplibregl.Map({
      container: mapContainer,
      style: mapStyle,
      center: [-105.6, 39.4],
      zoom: 6.4,
      attributionControl: false
    });

    map.addControl(
      new maplibregl.NavigationControl({ showCompass: true, showZoom: true }),
      'top-right'
    );

    map.on('error', (err: maplibregl.ErrorEvent & { sourceId?: string }) => {
      if (err?.sourceId === 'carto-dark' && basemapLabel !== 'OFFLINE VECTOR') {
        basemapLabel = 'OFFLINE VECTOR';
        if (map?.getLayer('carto-dark-layer')) {
          map.setLayoutProperty('carto-dark-layer', 'visibility', 'none');
        }
        if (map?.getLayer('terrain-sector-fill')) {
          map.setPaintProperty('terrain-sector-fill', 'fill-opacity', 0.72);
        }
        if (map?.getLayer('graticule-minor')) {
          map.setPaintProperty('graticule-minor', 'line-opacity', 1);
        }
        if (map?.getLayer('graticule-major')) {
          map.setPaintProperty('graticule-major', 'line-opacity', 1);
        }
        if (map?.getLayer('terrain-sector-outline')) {
          map.setPaintProperty('terrain-sector-outline', 'line-opacity', 1);
        }
      }
    });

    map.on('load', () => {
      mapLoaded = true;
      ensureOverlaySourcesAndLayers();
      lastCorridorKey = `${$activeCorridor.departure}-${($activeCorridor.intermediates || []).join(',')}-${$activeCorridor.destination}`;
      map?.resize();
      focusCorridor(0);
      applyLayerSync(true);
    });

    map.on('styledata', () => {
      if (!mapLoaded) return;
      if (ensureOverlaySourcesAndLayers() || syncDirty) {
        scheduleLayerSync();
      }
    });

    map.on('zoomend', syncWaypointMarkers);
    map.on('moveend', syncWaypointMarkers);

    if (typeof ResizeObserver !== 'undefined' && mapContainer) {
      resizeObserver = new ResizeObserver(() => {
        if (map && mapLoaded) {
          map.resize();
          syncWaypointMarkers();
        }
      });
      resizeObserver.observe(mapContainer);
    }
  });

  onDestroy(() => {
    mapLoaded = false;
    cancelPendingSync();
    if (focusTimer) {
      clearTimeout(focusTimer);
      focusTimer = null;
    }
    if (resizeTimer) {
      clearTimeout(resizeTimer);
      resizeTimer = null;
    }
    if (resizeObserver) {
      resizeObserver.disconnect();
      resizeObserver = null;
    }
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
      <span
        class="corridor-summary"
        class:summary-nogo={effectiveVerdictTier === 'NO-GO'}
        class:summary-caution={effectiveVerdictTier === 'CAUTION'}
        class:summary-ifr={effectiveVerdictTier === 'IFR'}
      >
        {$activeCorridor.departure || 'DEP'} → {($activeCorridor.intermediates || []).join(' → ') || 'DIRECT'} → {$activeCorridor.destination || 'DEST'}
      </span>
    </div>
    <div class="map-actions">
      <button type="button" class="btn-action" on:click={() => focusCorridor()}>CENTER ROUTE</button>
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
        <span
          class="hud-val"
          class:text-emerald-400={basemapLabel === 'CARTO DARK'}
          class:text-amber-400={basemapLabel === 'OFFLINE VECTOR'}
          data-testid="basemap-status"
        >
          {basemapLabel}
        </span>
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

  .corridor-summary.summary-nogo {
    color: var(--color-nogo);
    background: rgba(255, 23, 68, 0.12);
    border-color: rgba(255, 23, 68, 0.4);
  }

  .corridor-summary.summary-caution {
    color: var(--color-caution);
    background: rgba(255, 171, 0, 0.12);
    border-color: rgba(255, 171, 0, 0.4);
  }

  .corridor-summary.summary-ifr {
    color: var(--color-ifr);
    background: rgba(0, 229, 255, 0.12);
    border-color: rgba(0, 229, 255, 0.4);
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
    z-index: 20;
    display: inline-flex;
    width: max-content;
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
    z-index: 10;
    display: inline-block;
    width: max-content;
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
