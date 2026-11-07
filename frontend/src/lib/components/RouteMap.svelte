<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import * as maplibregl from 'maplibre-gl';
  import 'maplibre-gl/dist/maplibre-gl.css';
  import type { Feature, FeatureCollection } from 'geojson';
  import { activeHazards } from '../stores/weather';
  import { activeCorridor } from '../stores/verdict';
  import type { GeoCoordinate } from '../types/envelope';

  let mapContainer: HTMLDivElement;
  let map: maplibregl.Map | null = null;

  // Approximate station coordinates for standard flight corridor plotting
  const STATION_COORDS: Record<string, [number, number]> = {
    KAPA: [-104.849, 39.570],
    KDEN: [-104.673, 39.861],
    KLXV: [-106.317, 39.223],
    KASE: [-106.868, 39.223],
    KBJC: [-105.117, 39.908],
    KCOS: [-104.700, 38.805],
    KPUB: [-104.496, 38.289],
    KBOS: [-71.005, 42.364],
    KPWM: [-70.309, 43.646],
    KMHT: [-71.438, 42.934],
    KPVD: [-71.428, 41.724]
  };

  function buildGeoJsonFromHazards(hazards: typeof $activeHazards): FeatureCollection {
    const features: Feature[] = [];

    hazards.forEach((h) => {
      if (!h.active || !h.polygon || h.polygon.length < 3) return;

      const ring: [number, number][] = h.polygon.map((pt: GeoCoordinate) => [
        pt.longitude,
        pt.latitude
      ]);
      // Close the ring if not already closed
      if (
        ring[0][0] !== ring[ring.length - 1][0] ||
        ring[0][1] !== ring[ring.length - 1][1]
      ) {
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

    return {
      type: 'FeatureCollection',
      features
    };
  }

  function buildRouteGeoJson(corridor: typeof $activeCorridor): FeatureCollection {
    const stList = [
      corridor.departure,
      ...(corridor.intermediates || []),
      corridor.destination
    ]
      .map((s) => s.trim().toUpperCase())
      .filter((s) => Boolean(STATION_COORDS[s]));

    const coordinates = stList.map((s) => STATION_COORDS[s]);

    const routeFeature: Feature = {
      type: 'Feature',
      properties: { name: 'Flight Corridor' },
      geometry: {
        type: 'LineString',
        coordinates
      }
    };

    const pointFeatures: Feature[] = stList.map((st) => ({
      type: 'Feature',
      properties: { icao: st },
      geometry: {
        type: 'Point',
        coordinates: STATION_COORDS[st]
      }
    }));

    return {
      type: 'FeatureCollection',
      features: [routeFeature, ...pointFeatures]
    };
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
  }

  $: if (map && ($activeHazards || $activeCorridor)) {
    updateLayers();
  }

  onMount(() => {
    // Zero-token standalone dark tactical aeronautical style with high contrast
    const darkStyle: maplibregl.StyleSpecification = {
      version: 8,
      sources: {
        'osm-tactical-dark': {
          type: 'raster',
          tiles: [
            'https://tile.openstreetmap.org/{z}/{x}/{y}.png'
          ],
          tileSize: 256,
          attribution: '&copy; OpenStreetMap contributors'
        }
      },
      layers: [
        {
          id: 'osm-tactical-dark-layer',
          type: 'raster',
          source: 'osm-tactical-dark',
          minzoom: 0,
          maxzoom: 19,
          paint: {
            'raster-brightness-max': 0.32,
            'raster-contrast': 0.35,
            'raster-saturation': -0.95
          }
        }
      ]
    };

    map = new maplibregl.Map({
      container: mapContainer,
      style: darkStyle,
      center: [-105.5, 39.5], // Centered along Denver/Rockies
      zoom: 6.2,
      attributionControl: false
    });

    map.on('load', () => {
      // 1. Add Hazard Polygons Source & Layers
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
          'line-width': 2,
          'line-dasharray': [2, 2]
        }
      });

      // 2. Add Route Corridor Source & Layers
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
          'line-color': '#00e676',
          'line-width': 2.5
        }
      });

      map?.addLayer({
        id: 'route-points',
        type: 'circle',
        source: 'route',
        filter: ['==', '$type', 'Point'],
        paint: {
          'circle-radius': 5,
          'circle-color': '#07080a',
          'circle-stroke-width': 2,
          'circle-stroke-color': '#00e676'
        }
      });

      updateLayers();
    });
  });

  onDestroy(() => {
    if (map) {
      map.remove();
      map = null;
    }
  });

  let isExpanded = false;

  function toggleExpand() {
    isExpanded = !isExpanded;
    setTimeout(() => {
      if (map) {
        map.resize();
        focusCorridor();
      }
    }, 50);
  }

  function focusCorridor() {
    if (!map) return;
    const stList = [
      $activeCorridor.departure,
      ...($activeCorridor.intermediates || []),
      $activeCorridor.destination
    ]
      .map((s) => s.trim().toUpperCase())
      .filter((s) => Boolean(STATION_COORDS[s]));

    if (stList.length >= 2) {
      const bounds = new maplibregl.LngLatBounds(STATION_COORDS[stList[0]], STATION_COORDS[stList[1]]);
      stList.forEach((st) => bounds.extend(STATION_COORDS[st]));
      map.fitBounds(bounds, { padding: 60, maxZoom: 8.5 });
    } else if (stList.length === 1) {
      map.flyTo({ center: STATION_COORDS[stList[0]], zoom: 7.5 });
    }
  }

  let lastCorridorKey = '';
  $: if (map && $activeCorridor) {
    const key = `${$activeCorridor.departure}-${($activeCorridor.intermediates || []).join(',')}-${$activeCorridor.destination}`;
    if (key !== lastCorridorKey) {
      lastCorridorKey = key;
      focusCorridor();
    }
  }

  onMount(() => {
    // Zero-token standalone dark tactical aeronautical style with high contrast
    const darkStyle: maplibregl.StyleSpecification = {
      version: 8,
      sources: {
        'osm-tactical-dark': {
          type: 'raster',
          tiles: [
            'https://tile.openstreetmap.org/{z}/{x}/{y}.png'
          ],
          tileSize: 256,
          attribution: '&copy; OpenStreetMap contributors'
        }
      },
      layers: [
        {
          id: 'osm-tactical-dark-layer',
          type: 'raster',
          source: 'osm-tactical-dark',
          minzoom: 0,
          maxzoom: 19,
          paint: {
            'raster-brightness-max': 0.32,
            'raster-contrast': 0.35,
            'raster-saturation': -0.95
          }
        }
      ]
    };

    map = new maplibregl.Map({
      container: mapContainer,
      style: darkStyle,
      center: [-105.5, 39.5], // Centered along Denver/Rockies
      zoom: 6.2,
      attributionControl: false
    });

    map.addControl(new maplibregl.NavigationControl({ showCompass: true, showZoom: true }), 'top-right');

    map.on('load', () => {
      // 1. Add Hazard Polygons Source & Layers
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

      // 2. Add Route Corridor Source & Layers
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
          'line-color': '#00e676',
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
          'circle-color': '#07080a',
          'circle-stroke-width': 2.5,
          'circle-stroke-color': '#00e676'
        }
      });

      updateLayers();
      setTimeout(focusCorridor, 200);
    });
  });

  onDestroy(() => {
    if (map) {
      map.remove();
      map = null;
    }
  });
</script>

<div class="map-card" class:expanded={isExpanded} aria-label="Aeronautical Route & SIGMET Overlay Map">
  <div class="panel-header">
    <div class="flex items-center gap-2">
      <span>03 // AERONAUTICAL CORRIDOR &amp; HAZARD GEODESIC (MAPLIBRE)</span>
      <span class="corridor-summary">
        {$activeCorridor.departure || 'DEP'} → {($activeCorridor.intermediates || []).join(' → ') || 'INT'} → {$activeCorridor.destination || 'DEST'}
      </span>
    </div>
    <div class="map-actions">
      <button type="button" class="btn-action" on:click={focusCorridor}>CENTER ROUTE</button>
      <button type="button" class="btn-action btn-expand" on:click={toggleExpand}>
        {isExpanded ? 'STANDARD VIEW (480PX)' : 'EXPAND VIEW (680PX)'}
      </button>
    </div>
  </div>

  <div class="map-container-relative">
    <div class="map-viewport" class:viewport-expanded={isExpanded} bind:this={mapContainer}></div>

    <!-- Live Telemetry HUD Strip Overlaid on Map -->
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
        <span class="hud-label">AIRSPACE STATUS</span>
        <span class="hud-val text-emerald-400">CLASS G/E COMPLIANT</span>
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
    transition: all 0.2s ease-in-out;
  }

  .corridor-summary {
    font-size: 10px;
    font-weight: 700;
    color: var(--color-go);
    letter-spacing: 0.05em;
    background: rgba(0, 230, 118, 0.1);
    padding: 1px 6px;
    border: 1px solid rgba(0, 230, 118, 0.3);
  }

  .map-actions {
    display: flex;
    gap: 4px;
    align-items: center;
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
    background: #000;
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
    background: rgba(7, 8, 10, 0.88);
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
