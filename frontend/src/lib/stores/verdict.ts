import { writable, derived } from 'svelte/store';
import type {
  ComplianceReport,
  CorridorParams,
  RegulatoryContextParams,
  DispatchVerdict,
  ServerEnvelope
} from '../types/envelope';

/**
 * Core reactive store for FAA 14 CFR Part 135 Subpart L Compliance Verdicts.
 * Demultiplexes `MINIMUMS_EVALUATION` events from the Go `weather_ingest` gateway.
 */

export const defaultCorridor: CorridorParams = {
  departure: 'KAPA',
  destination: 'KLXV',
  intermediates: ['KDEN']
};

export const defaultRegulatoryContext: RegulatoryContextParams = {
  is_day: true,
  is_mountainous: true,
  is_local: false,
  allow_opspec_a021_relief: false,
  is_ifr_certified: true,
  has_ifr_alternate: true,
  has_severe_weather_radar: true,
  planned_cruise_alt_msl: 11500,
  highest_obstacle_msl: 10200,
  equipment: {
    nvg_equipped: true,
    htaws_equipped: true,
    radar_altimeter_equipped: true,
    autopilot_equipped: true
  }
};

export const activeCorridor = writable<CorridorParams>({ ...defaultCorridor });
export const regulatoryContext = writable<RegulatoryContextParams>(
  structuredClone(defaultRegulatoryContext)
);
export const latestComplianceReport = writable<ComplianceReport | null>(null);
export const evaluationHistory = writable<ComplianceReport[]>([]);
export const lastEvaluationSeq = writable<number>(0);
export const isEvaluating = writable<boolean>(false);

/**
 * Derived normalized high-level verdict status:
 * Maps Go verdicts (`APPROVED`, `MARGINAL`, `BELOW_MINIMUMS`, `DISQUALIFIED`, `IFR_REQUIRED`, `GO`, `CAUTION`, `NO-GO`)
 * into canonical operational visual severity tiers.
 */
export type VerdictSeverityTier = 'GO' | 'CAUTION' | 'NO-GO' | 'IFR' | 'STANDBY';

export function mapVerdictToTier(verdict?: DispatchVerdict | null): VerdictSeverityTier {
  if (!verdict || verdict === 'UNSPECIFIED') return 'STANDBY';
  switch (verdict) {
    case 'APPROVED':
    case 'GO':
      return 'GO';
    case 'MARGINAL':
    case 'CAUTION':
      return 'CAUTION';
    case 'IFR_REQUIRED':
      return 'IFR';
    case 'BELOW_MINIMUMS':
    case 'DISQUALIFIED':
    case 'NO-GO':
      return 'NO-GO';
    default:
      return 'STANDBY';
  }
}

export const currentVerdictTier = derived(latestComplianceReport, ($report) =>
  mapVerdictToTier($report?.verdict)
);

/**
 * Derived corridor stations list in route order (Departure -> Intermediates -> Destination).
 */
export const corridorStations = derived(activeCorridor, ($corridor) => {
  const list = [$corridor.departure, ...($corridor.intermediates || []), $corridor.destination];
  return list.map((s) => s.trim().toUpperCase()).filter(Boolean);
});

/**
 * Ingests a `MINIMUMS_EVALUATION` ServerEnvelope from the Go WebSocket gateway.
 */
export function ingestMinimumsEnvelope(envelope: ServerEnvelope<unknown>): void {
  const payload = envelope.payload as ComplianceReport | undefined;
  isEvaluating.set(false);
  if (!payload || typeof payload !== 'object' || !('verdict' in payload)) {
    return;
  }

  const normalized: ComplianceReport = {
    ...payload,
    station_reports: Array.isArray(payload.station_reports) ? payload.station_reports : [],
    active_hazards: Array.isArray(payload.active_hazards) ? payload.active_hazards : [],
    audit_trace: Array.isArray(payload.audit_trace) ? payload.audit_trace : [],
    evaluation_time_utc: payload.evaluation_time_utc || envelope.timestamp_utc
  };

  lastEvaluationSeq.set(envelope.seq);
  latestComplianceReport.set(normalized);

  evaluationHistory.update((history) => [normalized, ...history].slice(0, 25));
}

/**
 * Resets verdict stores to default state (for testing or console reset).
 */
export function resetVerdictStore(): void {
  activeCorridor.set({ ...defaultCorridor, intermediates: [...defaultCorridor.intermediates] });
  regulatoryContext.set(structuredClone(defaultRegulatoryContext));
  latestComplianceReport.set(null);
  evaluationHistory.set([]);
  lastEvaluationSeq.set(0);
  isEvaluating.set(false);
}
