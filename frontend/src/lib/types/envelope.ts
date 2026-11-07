/**
 * Strict TypeScript Definitions mirroring Go `services/weather_ingest/pkg/gateway/envelope.go`
 * and `services/weather_ingest/pkg/models/*.go`
 */

// -----------------------------------------------------------------------------
// 1. Client Actions & Server Events
// -----------------------------------------------------------------------------

export const ClientAction = {
  SUBSCRIBE: 'SUBSCRIBE',
  UNSUBSCRIBE: 'UNSUBSCRIBE',
  GET_CONDITIONS: 'GET_CONDITIONS',
  EVALUATE_MINIMUMS: 'EVALUATE_MINIMUMS',
  EVALUATE_KINEMATICS: 'EVALUATE_KINEMATICS',
  PING: 'PING'
} as const;

export type ClientActionType = (typeof ClientAction)[keyof typeof ClientAction];

export const ServerEvent = {
  WEATHER_SNAPSHOT: 'WEATHER_SNAPSHOT',
  WEATHER_UPDATE: 'WEATHER_UPDATE',
  MINIMUMS_EVALUATION: 'MINIMUMS_EVALUATION',
  KINEMATICS_EVALUATION: 'KINEMATICS_EVALUATION',
  HAZARD_ALERT: 'HAZARD_ALERT',
  PONG: 'PONG',
  ACK: 'ACK',
  ERROR: 'ERROR'
} as const;

export type ServerEventType = (typeof ServerEvent)[keyof typeof ServerEvent];

// -----------------------------------------------------------------------------
// 2. Domain Enums & Constants
// -----------------------------------------------------------------------------

export type FlightCategory = 'VFR' | 'MVFR' | 'IFR' | 'LIFR';

export type CloudCoverage = 'CLR' | 'SKC' | 'FEW' | 'SCT' | 'BKN' | 'OVC' | 'VV';

export type TAFChangeType = 'BASE' | 'FM' | 'TEMPO' | 'PROB30' | 'PROB40' | 'BECMG';

export type HazardType =
  | 'CONVECTIVE_SIGMET'
  | 'SEVERE_TURBULENCE'
  | 'SEVERE_ICING'
  | 'IFR_CLOUD_COVER'
  | 'MOUNTAIN_OBSCURATION'
  | 'VOLCANIC_ASH'
  | 'DUST_SANDSTORM';

export type HazardSeverity = 'LIGHT' | 'MODERATE' | 'SEVERE' | 'EXTREME';

export type DispatchVerdict =
  | 'UNSPECIFIED'
  | 'APPROVED'
  | 'MARGINAL'
  | 'BELOW_MINIMUMS'
  | 'DISQUALIFIED'
  | 'IFR_REQUIRED'
  | 'GO'
  | 'CAUTION'
  | 'NO-GO';

export type LimitingFactor =
  | 'LIMITING_FACTOR_NONE'
  | 'FAR_135_609_CEILING_BELOW_MINIMUMS'
  | 'FAR_135_609_VISIBILITY_BELOW_MINIMUMS'
  | 'FAR_135_615_OBSTACLE_CLEARANCE_VIOLATION'
  | 'FAR_135_209_VFR_RESERVE_VIOLATION'
  | 'ACTIVE_CONVECTIVE_SIGMET'
  | 'SEVERE_ICING_HAZARD'
  | 'SEVERE_TURBULENCE_HAZARD'
  | 'UNFLYABLE_WIND'
  | 'IFR_ALTERNATE_MISSING'
  | 'NO_WEATHER_REPORTING_WITHIN_15NM'
  | 'FAR_135_615_CEILING_BELOW_MINIMUMS'
  | 'FAR_135_615_VISIBILITY_BELOW_MINIMUMS';

// -----------------------------------------------------------------------------
// 3. Weather & Aeronautical Entities
// -----------------------------------------------------------------------------

export interface CloudLayer {
  coverage: CloudCoverage;
  base_ft_agl: number;
  is_ceiling: boolean;
  modifier?: string; // "CB" or "TCU"
}

export interface METAR {
  station_id: string;
  observation_time_utc: string;
  wind_dir_deg: number;
  wind_speed_kts: number;
  wind_gust_kts: number;
  is_variable_wind: boolean;
  var_min_deg?: number;
  var_max_deg?: number;
  visibility_sm: number;
  visibility_less_than: boolean;
  visibility_greater_than: boolean;
  ceiling_agl_ft: number;
  has_ceiling: boolean;
  cloud_layers: CloudLayer[];
  temperature_c: number;
  dewpoint_c: number;
  altimeter_inhg: number;
  present_weather: string[];
  flight_category: FlightCategory;
  density_altitude_ft: number;
  pressure_altitude_ft: number;
  raw_text: string;
}

export interface TAFPeriod {
  change_type: TAFChangeType;
  valid_from_utc: string;
  valid_to_utc: string;
  wind_dir_deg: number;
  wind_speed_kts: number;
  wind_gust_kts: number;
  visibility_sm: number;
  ceiling_agl_ft: number;
  has_ceiling: boolean;
  cloud_layers: CloudLayer[];
  present_weather: string[];
  flight_category: FlightCategory;
}

export interface TAF {
  station_id: string;
  issue_time_utc: string;
  valid_from_utc: string;
  valid_to_utc: string;
  periods: TAFPeriod[];
  raw_text: string;
}

export interface WindsAloftLayer {
  altitude_msl_ft: number;
  direction_deg: number;
  speed_kts: number;
  temperature_c: number;
}

export interface WindsAloftForecast {
  station_id: string;
  observation_time_utc: string;
  layers: WindsAloftLayer[];
  raw_text: string;
}

export interface GeoCoordinate {
  latitude: number;
  longitude: number;
}

export interface WeatherHazard {
  hazard_id: string;
  advisory_id: string;
  hazard_type: HazardType;
  severity: HazardSeverity | string;
  valid_from: string;
  valid_until: string;
  summary: string;
  min_alt_ft: number;
  max_alt_ft: number;
  polygon: GeoCoordinate[];
  active: boolean;
  raw_text: string;
}

export interface StationCompositeConditions {
  station_id: string;
  metar: METAR | null;
  taf?: TAF | null;
  winds_aloft?: WindsAloftForecast | null;
  hazards?: WeatherHazard[];
  last_updated_utc: string;
  seq: number;
}

// -----------------------------------------------------------------------------
// 4. FAA 14 CFR Part 135 Subpart L Compliance & Verdict Models
// -----------------------------------------------------------------------------

export interface EquipmentFlags {
  nvg_equipped: boolean;
  htaws_equipped: boolean;
  radar_altimeter_equipped: boolean;
  autopilot_equipped: boolean;
}

export interface CorridorParams {
  departure: string;
  destination: string;
  intermediates: string[];
}

export interface RegulatoryContextParams {
  is_day: boolean;
  is_mountainous: boolean;
  is_local: boolean;
  allow_opspec_a021_relief: boolean;
  is_ifr_certified: boolean;
  has_ifr_alternate: boolean;
  has_severe_weather_radar: boolean;
  planned_cruise_alt_msl: number;
  highest_obstacle_msl: number;
  equipment: EquipmentFlags;
}

export interface StationComplianceReport {
  station_id: string;
  is_compliant: boolean;
  actual_ceiling_ft: number;
  required_ceiling_ft: number;
  ceiling_margin_ft: number;
  actual_vis_sm: number;
  required_vis_sm: number;
  vis_margin_sm: number;
  category: FlightCategory;
  explanation: string;
}

export interface ComplianceReport {
  is_compliant: boolean;
  verdict: DispatchVerdict;
  limiting_factor: LimitingFactor;
  worst_ceiling_ft: number;
  worst_visibility_sm: number;
  min_required_ceiling_ft: number;
  min_required_vis_sm: number;
  ceiling_margin_ft: number;
  vis_margin_sm: number;
  legal_rule_applied: string;
  explanation: string;
  station_reports: StationComplianceReport[];
  active_hazards: WeatherHazard[];
  evaluation_time_utc: string;
  audit_trace: string[];
}

// -----------------------------------------------------------------------------
// 5. WebSocket Protocol Envelopes
// -----------------------------------------------------------------------------

export interface KinematicsParams {
  airframe_id: number;
  station_elevation_ft?: number;
  true_course_deg?: number;
  leg_distance_nm?: number;
  fuel_weight_kg?: number;
  payloads?: Array<{
    station_id: number;
    weight_kg: number;
  }>;
}

export interface ClientParams {
  station?: string;
  stations?: string[];
  topics?: string[];
  corridor?: CorridorParams;
  regulatory_context?: RegulatoryContextParams;
  kinematics?: KinematicsParams;
}

export interface ClientEnvelope {
  action: ClientActionType;
  request_id?: string;
  timestamp?: number; // epoch ms for PING
  params?: ClientParams;
}

export interface PongPayload {
  client_timestamp: number;
  server_epoch_ms: number;
}

export interface AckPayload {
  status: 'SUBSCRIBED' | 'UNSUBSCRIBED' | string;
  active_topics?: string[];
  unsubscribed_topics?: string[];
}

export interface ServerEnvelope<T = unknown> {
  event: ServerEventType;
  request_id?: string;
  timestamp_utc: string;
  seq: number;
  station_id?: string;
  payload?: T;
  metadata?: Record<string, string>;
  error?: string;
}

export type ConnectionState = 'DISCONNECTED' | 'CONNECTING' | 'CONNECTED' | 'RECONNECTING' | 'ERROR';
