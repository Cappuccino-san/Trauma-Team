import {
  ServerEvent,
  type ServerEnvelope,
  type METAR,
  type WeatherHazard,
  type ComplianceReport
} from '../types/envelope';
import type { TraumaTeamWebSocketRouter } from '../websocket';
import { activeCorridor } from '../stores/verdict';
import { activeHazards } from '../stores/weather';

/**
 * Deterministic NOAA AWC & FAA 14 CFR Part 135 Subpart L Simulation Fixtures
 * Mirrors `services/weather_ingest/pkg/noaa/fixtures.go` so the dispatcher dashboard
 * can be inspected and audited immediately even when offline.
 */

export const FIXTURE_METARS: Record<string, METAR> = {
  KAPA: {
    station_id: 'KAPA',
    observation_time_utc: '2026-11-05T05:50:00Z',
    wind_dir_deg: 180,
    wind_speed_kts: 10,
    wind_gust_kts: 0,
    is_variable_wind: false,
    visibility_sm: 10,
    visibility_less_than: false,
    visibility_greater_than: false,
    ceiling_agl_ft: 25000,
    has_ceiling: false,
    cloud_layers: [{ coverage: 'SCT', base_ft_agl: 5000, is_ceiling: false }],
    temperature_c: 18,
    dewpoint_c: 4,
    altimeter_inhg: 29.95,
    present_weather: [],
    flight_category: 'VFR',
    density_altitude_ft: 6840,
    pressure_altitude_ft: 5855,
    raw_text: 'KAPA 050550Z 18010KT 10SM SCT050 18/04 A2995 RMK AO2 SLP134'
  },
  KDEN: {
    station_id: 'KDEN',
    observation_time_utc: '2026-11-05T05:50:00Z',
    wind_dir_deg: 280,
    wind_speed_kts: 14,
    wind_gust_kts: 0,
    is_variable_wind: false,
    visibility_sm: 10,
    visibility_less_than: false,
    visibility_greater_than: false,
    ceiling_agl_ft: 10000,
    has_ceiling: true,
    cloud_layers: [{ coverage: 'BKN', base_ft_agl: 10000, is_ceiling: true }],
    temperature_c: 15,
    dewpoint_c: -5,
    altimeter_inhg: 30.1,
    present_weather: [],
    flight_category: 'VFR',
    density_altitude_ft: 5910,
    pressure_altitude_ft: 5255,
    raw_text: 'KDEN 050550Z 28014KT 10SM BKN100 15/M05 A3010 RMK AO2'
  },
  KLXV: {
    station_id: 'KLXV',
    observation_time_utc: '2026-11-05T05:50:00Z',
    wind_dir_deg: 260,
    wind_speed_kts: 12,
    wind_gust_kts: 0,
    is_variable_wind: false,
    visibility_sm: 10,
    visibility_less_than: false,
    visibility_greater_than: false,
    ceiling_agl_ft: 25000,
    has_ceiling: false,
    cloud_layers: [{ coverage: 'SCT', base_ft_agl: 8000, is_ceiling: false }],
    temperature_c: 28,
    dewpoint_c: 2,
    altimeter_inhg: 29.7,
    present_weather: [],
    flight_category: 'VFR',
    density_altitude_ft: 13628,
    pressure_altitude_ft: 10140,
    raw_text: 'KLXV 050550Z 26012KT 10SM SCT080 28/02 A2970 RMK AO2'
  },
  KASE: {
    station_id: 'KASE',
    observation_time_utc: '2026-11-05T05:50:00Z',
    wind_dir_deg: 340,
    wind_speed_kts: 5,
    wind_gust_kts: 0,
    is_variable_wind: false,
    visibility_sm: 2,
    visibility_less_than: false,
    visibility_greater_than: false,
    ceiling_agl_ft: 900,
    has_ceiling: true,
    cloud_layers: [{ coverage: 'OVC', base_ft_agl: 900, is_ceiling: true }],
    temperature_c: 0,
    dewpoint_c: -1,
    altimeter_inhg: 29.85,
    present_weather: ['BR'],
    flight_category: 'IFR',
    density_altitude_ft: 7910,
    pressure_altitude_ft: 7890,
    raw_text: 'KASE 050550Z 34005KT 2SM OVC009 00/M01 A2985 RMK AO2'
  },
  KBJC: {
    station_id: 'KBJC',
    observation_time_utc: '2026-11-05T05:50:00Z',
    wind_dir_deg: 0,
    wind_speed_kts: 3,
    wind_gust_kts: 0,
    is_variable_wind: true,
    visibility_sm: 0.5,
    visibility_less_than: false,
    visibility_greater_than: false,
    ceiling_agl_ft: 200,
    has_ceiling: true,
    cloud_layers: [{ coverage: 'VV', base_ft_agl: 200, is_ceiling: true }],
    temperature_c: 1,
    dewpoint_c: 1,
    altimeter_inhg: 30.05,
    present_weather: ['FG'],
    flight_category: 'LIFR',
    density_altitude_ft: 4920,
    pressure_altitude_ft: 5535,
    raw_text: 'KBJC 050550Z VRB03KT 1/2SM FG VV002 01/01 A3005 RMK AO2'
  }
};

export const FIXTURE_HAZARDS: WeatherHazard[] = [
  {
    hazard_id: 'SIGE-15E',
    advisory_id: 'CONVECTIVE SIGMET 15E',
    hazard_type: 'CONVECTIVE_SIGMET',
    severity: 'SEVERE',
    valid_from: '2026-11-05T05:55:00Z',
    valid_until: '2026-11-05T07:55:00Z',
    summary: 'LINE SEV TS 30 NM WIDE MOV FROM 26025KT. TOPS ABV FL450. HAIL TO 2 IN...WIND GUSTS TO 60KT POSS.',
    min_alt_ft: 0,
    max_alt_ft: 45000,
    polygon: [
      { latitude: 41.15, longitude: -104.8 },
      { latitude: 41.1, longitude: -103.0 },
      { latitude: 40.15, longitude: -103.2 }
    ],
    active: true,
    raw_text: 'WSUS31 KKCI 050555 SIGE CONVECTIVE SIGMET 15E VALID UNTIL 0755Z CO WY NE'
  },
  {
    hazard_id: 'NOV-2',
    advisory_id: 'SIGMET NOVEMBER 2',
    hazard_type: 'SEVERE_TURBULENCE',
    severity: 'SEVERE',
    valid_from: '2026-11-05T06:00:00Z',
    valid_until: '2026-11-05T10:00:00Z',
    summary: 'SEV TURB BTWN FL180 AND FL360. REPORTED BY B737.',
    min_alt_ft: 18000,
    max_alt_ft: 36000,
    polygon: [],
    active: true,
    raw_text: 'WSUS01 KKCI 050600 DEN WS 050600 SIGMET NOVEMBER 2 VALID UNTIL 051000 CO WY'
  },
  {
    hazard_id: 'SIGE-22E',
    advisory_id: 'CONVECTIVE SIGMET 22E',
    hazard_type: 'CONVECTIVE_SIGMET',
    severity: 'SEVERE',
    valid_from: '2026-11-05T05:55:00Z',
    valid_until: '2026-11-05T07:55:00Z',
    summary: 'LINE SEV TS 25 NM WIDE MOV FROM 24030KT. TOPS ABV FL450. HAIL TO 2 IN...WIND GUSTS TO 60KT POSS.',
    min_alt_ft: 0,
    max_alt_ft: 45000,
    polygon: [
      { latitude: 43.20, longitude: -71.80 },
      { latitude: 43.15, longitude: -71.10 },
      { latitude: 42.70, longitude: -71.15 },
      { latitude: 42.75, longitude: -71.85 }
    ],
    active: true,
    raw_text: 'WSUS31 KKCI 050555 SIGE CONVECTIVE SIGMET 22E VALID UNTIL 0755Z NH MA ME TOPS ABV FL450'
  }
];

export const FIXTURE_SCENARIOS: Record<string, ComplianceReport> = {
  NOMINAL_GO: {
    is_compliant: true,
    verdict: 'APPROVED',
    limiting_factor: 'LIMITING_FACTOR_NONE',
    worst_ceiling_ft: 10000,
    worst_visibility_sm: 10.0,
    min_required_ceiling_ft: 1000,
    min_required_vis_sm: 3.0,
    ceiling_margin_ft: 9000,
    vis_margin_sm: 7.0,
    legal_rule_applied: '14 CFR § 135.609 Table 1 (Mountainous Day Cross-Country VFR)',
    explanation: 'All corridor stations (KAPA, KDEN, KLXV) exceed § 135.609 Table 1 VFR minimums (1000 ft / 3.0 SM) with positive obstacle clearance under § 135.615.',
    station_reports: [
      {
        station_id: 'KAPA',
        is_compliant: true,
        actual_ceiling_ft: 25000,
        required_ceiling_ft: 1000,
        ceiling_margin_ft: 24000,
        actual_vis_sm: 10,
        required_vis_sm: 3,
        vis_margin_sm: 7,
        category: 'VFR',
        explanation: 'COMPLIANT: Ceiling 25000 ft (+24000 ft), Vis 10.0 SM (+7.0 SM)'
      },
      {
        station_id: 'KDEN',
        is_compliant: true,
        actual_ceiling_ft: 10000,
        required_ceiling_ft: 1000,
        ceiling_margin_ft: 9000,
        actual_vis_sm: 10,
        required_vis_sm: 3,
        vis_margin_sm: 7,
        category: 'VFR',
        explanation: 'COMPLIANT: Ceiling 10000 ft (+9000 ft), Vis 10.0 SM (+7.0 SM)'
      },
      {
        station_id: 'KLXV',
        is_compliant: true,
        actual_ceiling_ft: 25000,
        required_ceiling_ft: 1000,
        ceiling_margin_ft: 24000,
        actual_vis_sm: 10,
        required_vis_sm: 3,
        vis_margin_sm: 7,
        category: 'VFR',
        explanation: 'COMPLIANT: Ceiling 25000 ft (+24000 ft), Vis 10.0 SM (+7.0 SM)'
      }
    ],
    active_hazards: [],
    evaluation_time_utc: '2026-11-05T20:15:00Z',
    audit_trace: [
      '[RULE] 14 CFR § 135.609 Table 1: Mountainous=true, Day=true, Local=false -> Req 1000ft / 3.0SM',
      '[STN:KAPA] Ceiling=25000ft (>=1000ft), Vis=10.0SM (>=3.0SM) -> PASS',
      '[STN:KDEN] Ceiling=10000ft (>=1000ft), Vis=10.0SM (>=3.0SM) -> PASS',
      '[STN:KLXV] Ceiling=25000ft (>=1000ft), Vis=10.0SM (>=3.0SM) -> PASS',
      '[RULE] 14 CFR § 135.615 Obstacle Clearance: Cruise 11500ft MSL vs Highest Obstacle 10200ft MSL (+1300ft >= 300ft Day Req) -> PASS',
      '[VERDICT] APPROVED (All regulatory gates satisfied)'
    ]
  },

  MARGINAL_CAUTION: {
    is_compliant: true,
    verdict: 'MARGINAL',
    limiting_factor: 'LIMITING_FACTOR_NONE',
    worst_ceiling_ft: 1100,
    worst_visibility_sm: 3.2,
    min_required_ceiling_ft: 1000,
    min_required_vis_sm: 3.0,
    ceiling_margin_ft: 100,
    vis_margin_sm: 0.2,
    legal_rule_applied: '14 CFR § 135.609 Table 1 (Mountainous Night Cross-Country w/ NVG/HTAWS)',
    explanation: 'Corridor is legally compliant under § 135.609 NVG relief (1000 ft / 3.0 SM), but KDEN ceiling margin (+100 ft) is within the 200 ft operational caution buffer.',
    station_reports: [
      {
        station_id: 'KAPA',
        is_compliant: true,
        actual_ceiling_ft: 4500,
        required_ceiling_ft: 1000,
        ceiling_margin_ft: 3500,
        actual_vis_sm: 8,
        required_vis_sm: 3,
        vis_margin_sm: 5,
        category: 'VFR',
        explanation: 'COMPLIANT: Ceiling 4500 ft (+3500 ft), Vis 8.0 SM (+5.0 SM)'
      },
      {
        station_id: 'KDEN',
        is_compliant: true,
        actual_ceiling_ft: 1100,
        required_ceiling_ft: 1000,
        ceiling_margin_ft: 100,
        actual_vis_sm: 3.2,
        required_vis_sm: 3,
        vis_margin_sm: 0.2,
        category: 'MVFR',
        explanation: 'MARGINAL: Ceiling 1100 ft (+100 ft buffer), Vis 3.2 SM (+0.2 SM)'
      }
    ],
    active_hazards: [],
    evaluation_time_utc: '2026-11-05T20:15:30Z',
    audit_trace: [
      '[RULE] 14 CFR § 135.609 Table 1: Mountainous=true, Day=false, Local=false, NVG=true -> Req 1000ft / 3.0SM',
      '[STN:KAPA] Ceiling=4500ft (>=1000ft), Vis=8.0SM (>=3.0SM) -> PASS',
      '[STN:KDEN] Ceiling=1100ft (>=1000ft), Vis=3.2SM (>=3.0SM) -> PASS (CAUTION MARGIN +100ft)',
      '[VERDICT] MARGINAL (Legal minimums met; narrow operational buffer)'
    ]
  },

  ASPEN_BELOW_MINIMUMS: {
    is_compliant: false,
    verdict: 'BELOW_MINIMUMS',
    limiting_factor: 'FAR_135_609_CEILING_BELOW_MINIMUMS',
    worst_ceiling_ft: 900,
    worst_visibility_sm: 2.0,
    min_required_ceiling_ft: 1000,
    min_required_vis_sm: 3.0,
    ceiling_margin_ft: -100,
    vis_margin_sm: -1.0,
    legal_rule_applied: '14 CFR § 135.609 Table 1 (Mountainous Night Cross-Country w/ NVG)',
    explanation: 'DISPATCH PROHIBITED VFR: Destination KASE reports 900 ft OVC (-100 ft deficit vs 1000 ft req) and 2.0 SM visibility (-1.0 SM deficit vs 3.0 SM req).',
    station_reports: [
      {
        station_id: 'KAPA',
        is_compliant: true,
        actual_ceiling_ft: 25000,
        required_ceiling_ft: 1000,
        ceiling_margin_ft: 24000,
        actual_vis_sm: 10,
        required_vis_sm: 3,
        vis_margin_sm: 7,
        category: 'VFR',
        explanation: 'COMPLIANT: Ceiling 25000 ft (+24000 ft), Vis 10.0 SM (+7.0 SM)'
      },
      {
        station_id: 'KASE',
        is_compliant: false,
        actual_ceiling_ft: 900,
        required_ceiling_ft: 1000,
        ceiling_margin_ft: -100,
        actual_vis_sm: 2,
        required_vis_sm: 3,
        vis_margin_sm: -1,
        category: 'IFR',
        explanation: 'VIOLATION: Ceiling 900 ft < 1000 ft req (-100 ft); Vis 2.0 SM < 3.0 SM req (-1.0 SM)'
      }
    ],
    active_hazards: FIXTURE_HAZARDS,
    evaluation_time_utc: '2026-11-05T20:16:00Z',
    audit_trace: [
      '[RULE] 14 CFR § 135.609 Table 1: Mountainous=true, Day=false, Local=false, NVG=true -> Req 1000ft / 3.0SM',
      '[STN:KAPA] Ceiling=25000ft (>=1000ft), Vis=10.0SM (>=3.0SM) -> PASS',
      '[STN:KASE] Ceiling=900ft (<1000ft) -> FAIL (FAR_135_609_CEILING_BELOW_MINIMUMS)',
      '[STN:KASE] Vis=2.0SM (<3.0SM) -> FAIL (FAR_135_609_VISIBILITY_BELOW_MINIMUMS)',
      '[VERDICT] BELOW_MINIMUMS (VFR dispatch legally prohibited)'
    ]
  },

  IFR_TRANSITION_REQUIRED: {
    is_compliant: true,
    verdict: 'IFR_REQUIRED',
    limiting_factor: 'FAR_135_609_CEILING_BELOW_MINIMUMS',
    worst_ceiling_ft: 900,
    worst_visibility_sm: 2.0,
    min_required_ceiling_ft: 1000,
    min_required_vis_sm: 3.0,
    ceiling_margin_ft: -100,
    vis_margin_sm: -1.0,
    legal_rule_applied: '14 CFR § 135.611 (HAA IFR Operations w/ Valid Alternate)',
    explanation: 'VFR corridor below § 135.609 minimums at KASE (900 ft / 2.0 SM), but aircraft/crew are IFR-certified with verified § 135.611 alternate. IFR clearance required.',
    station_reports: [
      {
        station_id: 'KAPA',
        is_compliant: true,
        actual_ceiling_ft: 25000,
        required_ceiling_ft: 1000,
        ceiling_margin_ft: 24000,
        actual_vis_sm: 10,
        required_vis_sm: 3,
        vis_margin_sm: 7,
        category: 'VFR',
        explanation: 'COMPLIANT: VFR departure legal'
      },
      {
        station_id: 'KASE',
        is_compliant: false,
        actual_ceiling_ft: 900,
        required_ceiling_ft: 1000,
        ceiling_margin_ft: -100,
        actual_vis_sm: 2,
        required_vis_sm: 3,
        vis_margin_sm: -1,
        category: 'IFR',
        explanation: 'VFR BELOW MINS: Qualifying for § 135.611 IFR Approach & Alternate'
      }
    ],
    active_hazards: [],
    evaluation_time_utc: '2026-11-05T20:16:30Z',
    audit_trace: [
      '[RULE] 14 CFR § 135.609 VFR Check -> FAIL at KASE (900ft / 2.0SM)',
      '[RULE] 14 CFR § 135.611 IFR Fallback: IFR_Certified=true, IFR_Alternate=true -> PASS',
      '[VERDICT] IFR_REQUIRED (Authorized under Part 135 Subpart L IFR)'
    ]
  },

  NEW_ENGLAND_SIGMET: {
    is_compliant: false,
    verdict: 'DISQUALIFIED',
    limiting_factor: 'ACTIVE_CONVECTIVE_SIGMET',
    worst_ceiling_ft: 4500,
    worst_visibility_sm: 4.0,
    min_required_ceiling_ft: 800,
    min_required_vis_sm: 2.0,
    ceiling_margin_ft: 3700,
    vis_margin_sm: 2.0,
    legal_rule_applied: 'FAA Safety Directive: Active Convective Storm Hazard',
    explanation: 'DISQUALIFIED: Flight corridor intersects active Convective SIGMET 22E (NH/MA/ME): LINE SEV TS 25 NM WIDE MOV FROM 24030KT. Severe storms blocking KMHT corridor.',
    station_reports: [
      {
        station_id: 'KBOS',
        is_compliant: true,
        actual_ceiling_ft: 8000,
        required_ceiling_ft: 800,
        ceiling_margin_ft: 7200,
        actual_vis_sm: 10,
        required_vis_sm: 2,
        vis_margin_sm: 8,
        category: 'VFR',
        explanation: 'COMPLIANT: Ceiling 8000 ft, Vis 10.0 SM'
      },
      {
        station_id: 'KMHT',
        is_compliant: false,
        actual_ceiling_ft: 4500,
        required_ceiling_ft: 800,
        ceiling_margin_ft: 3700,
        actual_vis_sm: 7,
        required_vis_sm: 2,
        vis_margin_sm: 5,
        category: 'VFR',
        explanation: 'INTERSECTS CONVECTIVE SIGMET 22E: Severe thunderstorms blocking corridor'
      },
      {
        station_id: 'KPWM',
        is_compliant: false,
        actual_ceiling_ft: 900,
        required_ceiling_ft: 800,
        ceiling_margin_ft: 100,
        actual_vis_sm: 4,
        required_vis_sm: 2,
        vis_margin_sm: 2,
        category: 'IFR',
        explanation: 'MARGINAL/IFR: Ceiling 900 ft, Vis 4.0 SM'
      }
    ],
    active_hazards: FIXTURE_HAZARDS,
    evaluation_time_utc: '2026-11-05T20:17:00Z',
    audit_trace: [
      '[ROUTE] KBOS -> KMHT -> KPWM (New England Flight Corridor)',
      '[CORRIDOR INTERSECTION] Segment KBOS-KMHT intersects Convective SIGMET 22E polygon',
      '[HAZARD] Active Convective SIGMET 22E: Cloud tops FL450, Hail 2 in, Gusts 60 kt',
      '[RULE] FAA Part 135 HAA Safety Directive: Flight into active convective SIGMET prohibited',
      '[VERDICT] DISQUALIFIED (Corridor blocked by severe convective weather)'
    ]
  }
};

let simSeq = 100;

/**
 * Seeds the WebSocket router with initial NOAA station snapshots and an initial compliance verdict.
 */
export function loadSimulationScenario(
  router: TraumaTeamWebSocketRouter,
  scenarioKey: keyof typeof FIXTURE_SCENARIOS = 'NOMINAL_GO'
): void {
  const now = new Date().toISOString();

  // Clear stale hazards from previous scenario before loading new scenario
  activeHazards.set([]);

  if (scenarioKey === 'NEW_ENGLAND_SIGMET') {
    activeCorridor.set({
      departure: 'KBOS',
      intermediates: ['KMHT'],
      destination: 'KPWM'
    });
  } else if (scenarioKey === 'ASPEN_BELOW_MINIMUMS' || scenarioKey === 'IFR_TRANSITION_REQUIRED') {
    activeCorridor.set({
      departure: 'KAPA',
      intermediates: ['KLXV'],
      destination: 'KASE'
    });
  } else {
    activeCorridor.set({
      departure: 'KAPA',
      intermediates: ['KDEN'],
      destination: 'KLXV'
    });
  }

  Object.values(FIXTURE_METARS).forEach((metar) => {
    simSeq += 1;
    const env: ServerEnvelope<METAR> = {
      event: ServerEvent.WEATHER_SNAPSHOT,
      timestamp_utc: now,
      seq: simSeq,
      station_id: metar.station_id,
      payload: metar,
      metadata: {
        source: 'NOAA_AWC_FIXTURE',
        feed_type: 'METAR'
      }
    };
    router.routeEnvelope(env);
  });

  if (scenarioKey === 'ASPEN_BELOW_MINIMUMS' || scenarioKey === 'NEW_ENGLAND_SIGMET') {
    FIXTURE_HAZARDS.forEach((hazard) => {
      simSeq += 1;
      const hEnv: ServerEnvelope<WeatherHazard> = {
        event: ServerEvent.HAZARD_ALERT,
        timestamp_utc: now,
        seq: simSeq,
        payload: hazard,
        metadata: { source: 'NOAA_AWC_AIRSIGMET' }
      };
      router.routeEnvelope(hEnv);
    });
  }

  const report = FIXTURE_SCENARIOS[scenarioKey];
  if (report) {
    simSeq += 1;
    const vEnv: ServerEnvelope<ComplianceReport> = {
      event: ServerEvent.MINIMUMS_EVALUATION,
      request_id: `sim-${scenarioKey.toLowerCase()}`,
      timestamp_utc: now,
      seq: simSeq,
      payload: {
        ...report,
        evaluation_time_utc: now
      },
      metadata: {
        evaluator: 'FAA_PART_135_SUBPART_L'
      }
    };
    router.routeEnvelope(vEnv);
  }
}
