import type { CorridorParams } from './types/envelope';

export interface StationMeta {
  coords: [number, number];
  elevFt: number;
  role: string;
}

/**
 * Canonical certified corridor station coordinates & field elevations (ft MSL).
 * Shared by RouteMap rendering and geodesic distance / mission leg integration.
 */
export const STATION_META: Record<string, StationMeta> = {
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

const EARTH_RADIUS_NM = 3440.065;

export function haversineDistanceNm(a: [number, number], b: [number, number]): number {
  const toRad = (deg: number) => (deg * Math.PI) / 180;
  const dLat = toRad(b[1] - a[1]);
  const dLon = toRad(b[0] - a[0]);
  const lat1 = toRad(a[1]);
  const lat2 = toRad(b[1]);
  const h =
    Math.sin(dLat / 2) * Math.sin(dLat / 2) +
    Math.cos(lat1) * Math.cos(lat2) * Math.sin(dLon / 2) * Math.sin(dLon / 2);
  return 2 * EARTH_RADIUS_NM * Math.asin(Math.min(1, Math.sqrt(h)));
}

export function computeCorridorDistanceNm(corridor: CorridorParams): number {
  const stops = [corridor.departure, ...(corridor.intermediates || []), corridor.destination]
    .map((s) => s.trim().toUpperCase())
    .filter(Boolean);

  let total = 0;
  let segments = 0;
  let prevCoords: [number, number] | null = null;
  for (const id of stops) {
    const meta = STATION_META[id];
    if (!meta) continue;
    if (prevCoords) {
      total += haversineDistanceNm(prevCoords, meta.coords);
      segments += 1;
    }
    prevCoords = meta.coords;
  }

  return segments > 0 ? Number(total.toFixed(1)) : 45.0;
}
