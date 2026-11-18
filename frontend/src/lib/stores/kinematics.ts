import { writable, derived } from 'svelte/store';
import type { ServerEnvelope } from '../types/envelope';

/**
 * Mass and Balance snapshot evaluated against certified RFM envelope limits.
 */
export interface MassBalanceEvaluation {
  is_gw_compliant: boolean;
  is_long_cg_compliant: boolean;
  is_lat_cg_compliant: boolean;
  is_overall_compliant: boolean;
  limiting_factor: string;
  gross_weight_kg: number;
  cg_long_mm: number;
  cg_lat_mm: number;
  fwd_limit_mm: number;
  aft_limit_mm: number;
  left_limit_mm: number;
  right_limit_mm: number;
  fwd_margin_mm: number;
  aft_margin_mm: number;
  left_margin_mm: number;
  right_margin_mm: number;
}

/**
 * Full Mission Kinematics evaluation output returned by the Rust physics engine.
 */
export interface MissionKinematicsEvaluation {
  airframe: string;
  verdict: 'GO' | 'CAUTION' | 'NO-GO' | string;
  limiting_factor: string;
  is_feasible: boolean;
  is_wind_flyable: boolean;
  pressure_altitude_ft: number;
  density_altitude_ft: number;
  max_ige_hover_weight_kg: number;
  max_oge_hover_weight_kg: number;
  max_oei_cat_a_weight_kg: number;
  ige_margin_kg: number;
  oge_margin_kg: number;
  oei_cat_a_margin_kg: number;
  cruise_tas_kts: number;
  cruise_fuel_flow_kg_hr: number;
  hover_oge_fuel_flow_kg_hr: number;
  loiter_fuel_flow_kg_hr: number;
  groundspeed_kts: number;
  true_heading_deg: number;
  wind_correction_angle_deg: number;
  headwind_component_kts: number;
  crosswind_component_kts: number;
  leg_ete_minutes: number;
  leg_fuel_burn_kg: number;
  remaining_fuel_kg: number;
  reserve_fuel_20min_kg: number;
  reserve_margin_kg: number;
  mass_balance: MassBalanceEvaluation;
  landing_gross_weight_kg?: number;
  landing_cg_long_mm?: number;
  landing_cg_lat_mm?: number;
  zero_fuel_weight_kg?: number;
  zero_fuel_cg_long_mm?: number;
  zero_fuel_cg_lat_mm?: number;
  ige_hover_ceiling_ft?: number;
  oge_hover_ceiling_ft?: number;
  oei_cat_a_ceiling_ft?: number;
  required_fuel_offload_kg?: number;
  required_payload_reduction_kg?: number;
  additional_fuel_required_kg?: number;
  actionable_advisory?: string;
}

export interface KinematicsPreset {
  airframe: string;
  fieldElevFt: number;
  oatC: number;
  qnhInHg: number;
  fuelKg: number;
  payloadKg: number;
  crewKg: number;
  headwindKts: number;
  legDistanceNm?: number;
}

export const defaultKinematics: MissionKinematicsEvaluation = {
  airframe: 'Airbus H145 (BK117 D-2)',
  verdict: 'GO',
  limiting_factor: 'NONE',
  is_feasible: true,
  is_wind_flyable: true,
  pressure_altitude_ft: 5858,
  density_altitude_ft: 6827,
  max_ige_hover_weight_kg: 3700,
  max_oge_hover_weight_kg: 3437,
  max_oei_cat_a_weight_kg: 3015,
  ige_margin_kg: 808,
  oge_margin_kg: 545,
  oei_cat_a_margin_kg: 123,
  cruise_tas_kts: 138.1,
  cruise_fuel_flow_kg_hr: 217.5,
  hover_oge_fuel_flow_kg_hr: 261.0,
  loiter_fuel_flow_kg_hr: 184.9,
  groundspeed_kts: 130.1,
  true_heading_deg: 268.0,
  wind_correction_angle_deg: -2.0,
  headwind_component_kts: 8.0,
  crosswind_component_kts: -5.0,
  leg_ete_minutes: 20.8,
  leg_fuel_burn_kg: 75.2,
  remaining_fuel_kg: 344.8,
  reserve_fuel_20min_kg: 72.5,
  reserve_margin_kg: 272.3,
  landing_gross_weight_kg: 2816.8,
  landing_cg_long_mm: 3860.4,
  landing_cg_lat_mm: -13.9,
  zero_fuel_weight_kg: 2472.0,
  zero_fuel_cg_long_mm: 3844.5,
  zero_fuel_cg_lat_mm: -15.8,
  ige_hover_ceiling_ft: 14000,
  oge_hover_ceiling_ft: 10912,
  oei_cat_a_ceiling_ft: 6858,
  required_fuel_offload_kg: 0,
  required_payload_reduction_kg: 0,
  additional_fuel_required_kg: 0,
  actionable_advisory: '',
  mass_balance: {
    is_gw_compliant: true,
    is_long_cg_compliant: true,
    is_lat_cg_compliant: true,
    is_overall_compliant: true,
    limiting_factor: 'NONE',
    gross_weight_kg: 2892,
    cg_long_mm: 3863.8,
    cg_lat_mm: -13.5,
    fwd_limit_mm: 3798.1,
    aft_limit_mm: 4186.9,
    left_limit_mm: -79.7,
    right_limit_mm: 79.7,
    fwd_margin_mm: 65.7,
    aft_margin_mm: 323.1,
    left_margin_mm: 66.2,
    right_margin_mm: 93.2
  }
};

export const latestKinematics = writable<MissionKinematicsEvaluation>(defaultKinematics);
export const kinematicsPresetStore = writable<KinematicsPreset | null>(null);

export const currentGrossWeight = derived(latestKinematics, ($k) => $k.mass_balance.gross_weight_kg);
export const currentCgLong = derived(latestKinematics, ($k) => $k.mass_balance.cg_long_mm);
export const currentCgLat = derived(latestKinematics, ($k) => $k.mass_balance.cg_lat_mm);
export const isCgCompliant = derived(latestKinematics, ($k) => $k.mass_balance.is_overall_compliant);

export function ingestKinematicsEnvelope(envelope: ServerEnvelope<unknown>): void {
  const payload = envelope.payload as MissionKinematicsEvaluation | undefined;
  if (!payload || typeof payload !== 'object' || !payload.mass_balance) return;
  latestKinematics.set(payload);
}
