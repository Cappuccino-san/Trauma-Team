import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render } from '@testing-library/svelte';
import { resetRouterStores, wsRouter } from '../src/lib/websocket';
import { resetWeatherStore, stationsById } from '../src/lib/stores/weather';
import { resetVerdictStore, activeCorridor } from '../src/lib/stores/verdict';
import { latestKinematics, defaultKinematics } from '../src/lib/stores/kinematics';
import { loadSimulationScenario, FIXTURE_METARS } from '../src/lib/simulation/fixtures';
import { ServerEvent } from '../src/lib/types/envelope';

const {
  setWorkerUrlMock,
  mapConstructorMock,
  setDataRouteMock,
  setDataHazardsMock,
  setPaintPropertyMock,
  setLayoutPropertyMock,
  mapRemoveMock,
  markerCalls,
  markerRemoveMock
} = vi.hoisted(() => {
  const setDataRouteMock = vi.fn();
  const setDataHazardsMock = vi.fn();
  const setPaintPropertyMock = vi.fn();
  const setLayoutPropertyMock = vi.fn();
  const mapRemoveMock = vi.fn();
  const markerRemoveMock = vi.fn();
  const markerCalls: Array<{ anchor?: string; text?: string; zIndex?: string; lngLat?: unknown }> = [];

  const handlers: Record<string, Array<(payload?: unknown) => void>> = {};
  const mapInstance = {
    on: vi.fn((event: string, cb: (payload?: unknown) => void) => {
      handlers[event] = handlers[event] || [];
      handlers[event].push(cb);
    }),
    addControl: vi.fn(),
    addSource: vi.fn(),
    addLayer: vi.fn(),
    getSource: vi.fn((id: string) => {
      if (id === 'route') return { setData: setDataRouteMock };
      if (id === 'hazards') return { setData: setDataHazardsMock };
      return undefined;
    }),
    getLayer: vi.fn((id: string) =>
      [
        'carto-dark-layer',
        'graticule-minor',
        'graticule-major',
        'terrain-sector-fill',
        'terrain-sector-outline',
        'hazards-fill',
        'hazards-line',
        'route-line',
        'route-points'
      ].includes(id)
        ? {}
        : undefined
    ),
    setPaintProperty: setPaintPropertyMock,
    setLayoutProperty: setLayoutPropertyMock,
    // Simulate MapLibre worker still busy (isStyleLoaded === false) during concurrent updates
    isStyleLoaded: vi.fn(() => false),
    fitBounds: vi.fn(),
    flyTo: vi.fn(),
    resize: vi.fn(),
    remove: mapRemoveMock,
    project: vi.fn((coords: [number, number]) => ({
      x: (coords[0] + 108) * 180,
      y: (41 - coords[1]) * 180
    })),
    __trigger: (event: string, payload?: unknown) => {
      (handlers[event] || []).forEach((cb) => cb(payload));
    },
    __resetHandlers: () => {
      for (const k of Object.keys(handlers)) delete handlers[k];
    }
  };

  return {
    setWorkerUrlMock: vi.fn(),
    mapConstructorMock: vi.fn((_opts?: unknown) => mapInstance),
    setDataRouteMock,
    setDataHazardsMock,
    setPaintPropertyMock,
    setLayoutPropertyMock,
    mapRemoveMock,
    markerCalls,
    markerRemoveMock
  };
});

vi.mock('maplibre-gl', () => ({
  setWorkerUrl: setWorkerUrlMock,
  Map: mapConstructorMock,
  NavigationControl: vi.fn(),
  Marker: vi.fn((opts?: { element?: HTMLElement; anchor?: string }) => {
    const entry = {
      anchor: opts?.anchor,
      text: opts?.element?.textContent || '',
      zIndex: opts?.element?.style?.zIndex || '',
      lngLat: undefined as unknown
    };
    markerCalls.push(entry);
    return {
      setLngLat: vi.fn(function (this: unknown, coords: unknown) {
        entry.lngLat = coords;
        return this;
      }),
      addTo: vi.fn().mockReturnThis(),
      remove: markerRemoveMock
    };
  }),
  LngLatBounds: vi.fn(() => ({
    extend: vi.fn()
  }))
}));

import RouteMap from '../src/lib/components/RouteMap.svelte';

async function flushRaf(): Promise<void> {
  await new Promise((resolve) => setTimeout(resolve, 25));
}

describe('RouteMap static worker, coalesced concurrent updates & unmount cleanup', () => {
  beforeEach(() => {
    resetRouterStores();
    resetWeatherStore();
    resetVerdictStore();
    latestKinematics.set(structuredClone(defaultKinematics));
    setWorkerUrlMock.mockClear();
    mapConstructorMock.mockClear();
    setDataRouteMock.mockClear();
    setDataHazardsMock.mockClear();
    setPaintPropertyMock.mockClear();
    setLayoutPropertyMock.mockClear();
    mapRemoveMock.mockClear();
    markerRemoveMock.mockClear();
    markerCalls.length = 0;
  });

  it('configures CARTO Dark basemap at bottom of layer stack, coalesces concurrent updates, falls back on tile error, and cleans up on unmount', async () => {
    const { unmount, getByTestId } = render(RouteMap);

    expect(setWorkerUrlMock).toHaveBeenCalledTimes(1);
    const workerUrl = setWorkerUrlMock.mock.calls[0][0] as string;
    expect(workerUrl).toContain('maplibre-gl-worker');
    expect(workerUrl).toMatch(/\.mjs($|\?)/);
    expect(mapConstructorMock).toHaveBeenCalledTimes(1);
    expect(setWorkerUrlMock.mock.invocationCallOrder[0]).toBeLessThan(
      mapConstructorMock.mock.invocationCallOrder[0]
    );

    const mapOptions = mapConstructorMock.mock.calls[0][0] as unknown as {
      style: {
        sources: Record<string, { type: string; tileSize?: number; tiles?: string[]; attribution?: string }>;
        layers: Array<{ id: string; type: string; source?: string }>;
      };
    };
    expect(mapOptions.style.sources['carto-dark']?.type).toBe('raster');
    expect(mapOptions.style.sources['carto-dark']?.tileSize).toBe(256);
    expect(mapOptions.style.sources['carto-dark']?.tiles).toHaveLength(4);
    expect(mapOptions.style.sources['carto-dark']?.attribution).toBe('© OpenStreetMap © CARTO');
    expect(mapOptions.style.layers[0]).toMatchObject({
      id: 'carto-dark-layer',
      type: 'raster',
      source: 'carto-dark'
    });
    expect(getByTestId('basemap-status').textContent?.trim()).toBe('CARTO DARK');

    const mapInstance = mapConstructorMock.mock.results[0].value as {
      __trigger: (event: string, payload?: unknown) => void;
    };
    mapInstance.__trigger('load');

    expect(setDataRouteMock).toHaveBeenCalledTimes(1);
    expect(setDataHazardsMock).toHaveBeenCalledTimes(1);

    // Station chips must have higher z-index than terrain callouts
    const stationMarkers = markerCalls.filter((m) => m.text?.includes('KAPA') || m.text?.includes('KLXV'));
    const terrainMarkers = markerCalls.filter((m) => m.text?.includes('▲'));
    expect(stationMarkers.length).toBeGreaterThanOrEqual(2);
    stationMarkers.forEach((m) => expect(Number(m.zIndex)).toBeGreaterThan(10));
    terrainMarkers.forEach((m) => expect(Number(m.zIndex)).toBeLessThan(20));

    setDataRouteMock.mockClear();
    setDataHazardsMock.mockClear();
    setPaintPropertyMock.mockClear();

    // Burst concurrent updates in a single synchronous turn (stationsById + scenario presets + styledata)
    wsRouter.routeEnvelope({
      event: ServerEvent.WEATHER_SNAPSHOT,
      timestamp_utc: '2026-11-18T12:00:00Z',
      seq: 201,
      station_id: 'KBOS',
      payload: FIXTURE_METARS.KAPA
    });
    loadSimulationScenario(wsRouter, 'HIGH_DA_KLXV');
    loadSimulationScenario(wsRouter, 'NEW_ENGLAND_SIGMET');
    stationsById.update((s) => ({ ...s }));
    mapInstance.__trigger('styledata');

    // Coalesced into a single RAF flush
    await flushRaf();

    expect(setDataRouteMock).toHaveBeenCalledTimes(1);
    expect(setDataHazardsMock).toHaveBeenCalledTimes(1);
    expect(setPaintPropertyMock).toHaveBeenCalledWith('route-line', 'line-color', '#ff1744');

    // Simulate raster tile error on carto-dark -> automatic fallback to OFFLINE VECTOR
    mapInstance.__trigger('error', { sourceId: 'carto-dark', error: new Error('Tile network error') });
    await Promise.resolve();
    expect(getByTestId('basemap-status').textContent?.trim()).toBe('OFFLINE VECTOR');
    expect(setLayoutPropertyMock).toHaveBeenCalledWith('carto-dark-layer', 'visibility', 'none');
    expect(setPaintPropertyMock).toHaveBeenCalledWith('terrain-sector-fill', 'fill-opacity', 0.72);

    // Queue another update and immediately unmount before RAF fires to verify clean cancellation
    const cancelSpy = vi.spyOn(globalThis, 'cancelAnimationFrame');
    activeCorridor.set({ departure: 'KAPA', intermediates: ['KDEN'], destination: 'KLXV' });
    await Promise.resolve();
    const activeMarkersCount = markerCalls.length;
    expect(activeMarkersCount).toBeGreaterThan(0);

    unmount();

    expect(cancelSpy).toHaveBeenCalled();
    expect(markerRemoveMock).toHaveBeenCalled();
    expect(mapRemoveMock).toHaveBeenCalledTimes(1);
    cancelSpy.mockRestore();
  });
});
