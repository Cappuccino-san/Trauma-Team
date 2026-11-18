import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render } from '@testing-library/svelte';

const { setWorkerUrlMock, mapConstructorMock } = vi.hoisted(() => {
  const onMock = vi.fn();
  const addControlMock = vi.fn();
  const removeMock = vi.fn();
  const isStyleLoadedMock = vi.fn(() => false);
  const fitBoundsMock = vi.fn();

  return {
    setWorkerUrlMock: vi.fn(),
    mapConstructorMock: vi.fn(() => ({
      on: onMock,
      addControl: addControlMock,
      remove: removeMock,
      isStyleLoaded: isStyleLoadedMock,
      fitBounds: fitBoundsMock
    }))
  };
});

vi.mock('maplibre-gl', () => ({
  setWorkerUrl: setWorkerUrlMock,
  Map: mapConstructorMock,
  NavigationControl: vi.fn(),
  Marker: vi.fn(() => ({
    setLngLat: vi.fn().mockReturnThis(),
    addTo: vi.fn().mockReturnThis(),
    remove: vi.fn()
  })),
  LngLatBounds: vi.fn(() => ({
    extend: vi.fn()
  }))
}));

import RouteMap from '../src/lib/components/RouteMap.svelte';

describe('RouteMap static worker configuration', () => {
  beforeEach(() => {
    setWorkerUrlMock.mockClear();
    mapConstructorMock.mockClear();
  });

  it('configures MapLibre workerUrl before constructing the Map instance', () => {
    render(RouteMap);

    expect(setWorkerUrlMock).toHaveBeenCalledTimes(1);
    const workerUrl = setWorkerUrlMock.mock.calls[0][0] as string;
    expect(workerUrl).toContain('maplibre-gl-worker');
    expect(workerUrl).toMatch(/\.mjs($|\?)/);
    expect(mapConstructorMock).toHaveBeenCalledTimes(1);
    expect(setWorkerUrlMock.mock.invocationCallOrder[0]).toBeLessThan(
      mapConstructorMock.mock.invocationCallOrder[0]
    );
  });
});
