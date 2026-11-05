import { writable, derived, get } from 'svelte/store';
import type {
  METAR,
  TAF,
  WindsAloftForecast,
  WeatherHazard,
  StationCompositeConditions,
  ServerEnvelope
} from '../types/envelope';

/**
 * Core reactive store for station meteorological conditions & active SIGMET hazards.
 * Demultiplexes `WEATHER_SNAPSHOT`, `WEATHER_UPDATE`, and `HAZARD_ALERT` events
 * from the Go `weather_ingest` WebSocket gateway.
 */

const initialStations: Record<string, StationCompositeConditions> = {};
const initialHazards: WeatherHazard[] = [];

export const stationsById = writable<Record<string, StationCompositeConditions>>(initialStations);
export const activeHazards = writable<WeatherHazard[]>(initialHazards);
export const selectedStationId = writable<string>('KAPA');
export const lastWeatherSeq = writable<number>(0);
export const lastWeatherTimestampUtc = writable<string>('');

/**
 * Derived list of stations sorted alphabetically by ICAO identifier.
 */
export const stationList = derived(stationsById, ($stations) =>
  Object.values($stations).sort((a, b) => a.station_id.localeCompare(b.station_id))
);

/**
 * Derived currently selected station conditions.
 */
export const selectedStation = derived(
  [stationsById, selectedStationId],
  ([$stations, $selectedId]) => ($selectedId ? $stations[$selectedId] ?? null : null)
);

/**
 * Derived count of active hazards (SIGMETs / Convective SIGMETs).
 */
export const activeHazardCount = derived(
  activeHazards,
  ($hazards) => $hazards.filter((h) => h.active).length
);

/**
 * Normalizes raw payload into a METAR or composite station update.
 */
function isMETARPayload(payload: unknown): payload is METAR {
  return (
    typeof payload === 'object' &&
    payload !== null &&
    'station_id' in payload &&
    ('visibility_sm' in payload || 'flight_category' in payload || 'raw_text' in payload) &&
    !('metar' in payload)
  );
}

interface CompositeRestPayload {
  station_id: string;
  metar?: METAR;
  taf?: TAF;
  winds_aloft?: WindsAloftForecast;
  hazards?: WeatherHazard[];
}

function isCompositePayload(payload: unknown): payload is CompositeRestPayload {
  return typeof payload === 'object' && payload !== null && 'metar' in payload;
}

/**
 * Ingests a `WEATHER_SNAPSHOT` or `WEATHER_UPDATE` ServerEnvelope into the reactive store.
 */
export function ingestWeatherEnvelope(envelope: ServerEnvelope<unknown>): void {
  const payload = envelope.payload;
  if (!payload) return;

  lastWeatherSeq.set(envelope.seq);
  lastWeatherTimestampUtc.set(envelope.timestamp_utc);

  if (isCompositePayload(payload)) {
    const stId = (payload.station_id || envelope.station_id || '').toUpperCase().trim();
    if (!stId) return;

    stationsById.update((current) => {
      const existing = current[stId];
      return {
        ...current,
        [stId]: {
          station_id: stId,
          metar: payload.metar ?? existing?.metar ?? null,
          taf: payload.taf ?? existing?.taf ?? null,
          winds_aloft: payload.winds_aloft ?? existing?.winds_aloft ?? null,
          hazards: payload.hazards ?? existing?.hazards ?? [],
          last_updated_utc: envelope.timestamp_utc || new Date().toISOString(),
          seq: envelope.seq
        }
      };
    });

    if (Array.isArray(payload.hazards) && payload.hazards.length > 0) {
      activeHazards.set(payload.hazards);
    }
    return;
  }

  if (isMETARPayload(payload)) {
    const stId = (payload.station_id || envelope.station_id || '').toUpperCase().trim();
    if (!stId) return;

    stationsById.update((current) => {
      const existing = current[stId];
      return {
        ...current,
        [stId]: {
          station_id: stId,
          metar: payload,
          taf: existing?.taf ?? null,
          winds_aloft: existing?.winds_aloft ?? null,
          hazards: existing?.hazards ?? [],
          last_updated_utc: envelope.timestamp_utc || payload.observation_time_utc,
          seq: envelope.seq
        }
      };
    });
  }
}

/**
 * Ingests a `HAZARD_ALERT` ServerEnvelope into the active hazards store.
 */
export function ingestHazardEnvelope(envelope: ServerEnvelope<unknown>): void {
  const payload = envelope.payload as WeatherHazard | undefined;
  if (!payload || typeof payload !== 'object') return;

  lastWeatherSeq.set(envelope.seq);
  lastWeatherTimestampUtc.set(envelope.timestamp_utc);

  const key = payload.hazard_id || payload.advisory_id;
  activeHazards.update((current) => {
    const filtered = current.filter((h) => (h.hazard_id || h.advisory_id) !== key);
    return [payload, ...filtered];
  });
}

/**
 * Directly seeds or updates a station's full composite record (e.g., from REST fallback or offline fixtures).
 */
export function upsertStationComposite(record: StationCompositeConditions): void {
  const stId = record.station_id.toUpperCase().trim();
  stationsById.update((current) => ({
    ...current,
    [stId]: {
      ...record,
      station_id: stId
    }
  }));
}

/**
 * Resets all weather stores to initial state (useful for deterministic unit tests).
 */
export function resetWeatherStore(): void {
  stationsById.set({});
  activeHazards.set([]);
  selectedStationId.set('KAPA');
  lastWeatherSeq.set(0);
  lastWeatherTimestampUtc.set('');
}

/**
 * Helper to retrieve a snapshot of a station synchronously.
 */
export function getStationSnapshot(stationId: string): StationCompositeConditions | undefined {
  return get(stationsById)[stationId.toUpperCase().trim()];
}
