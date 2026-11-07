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
}

export const defaultKinematics: MissionKinematicsEvaluation = {
  airframe: 'Airbus H145 (BK117 D-2)',
  verdict: 'GO',
  limiting_factor: 'NONE',
  is_feasible: true,
  is_wind_flyable: true,
  pressure_altitude_ft: 5885,
  density_altitude_ft: 6840,
  max_ige_hover_weight_kg: 3700,
  max_oge_hover_weight_kg: 3450,
  max_oei_cat_a_weight_kg: 3050,
  ige_margin_kg: 810,
  oge_margin_kg: 560,
  oei_cat_a_margin_kg: 160,
  cruise_tas_kts: 135.5,
  cruise_fuel_flow_kg_hr: 215.0,
  hover_oge_fuel_flow_kg_hr: 258.0,
  loiter_fuel_flow_kg_hr: 182.5,
  groundspeed_kts: 128.0,
  true_heading_deg: 268.0,
  wind_correction_angle_deg: -2.0,
  headwind_component_kts: 10.0,
  crosswind_component_kts: -5.0,
  leg_ete_minutes: 21.0,
  leg_fuel_burn_kg: 75.0,
  remaining_fuel_kg: 375.0,
  reserve_fuel_20min_kg: 71.7,
  reserve_margin_kg: 303.3,
  mass_balance: {
    is_gw_compliant: true,
    is_long_cg_compliant: true,
    is_lat_cg_compliant: true,
    is_overall_compliant: true,
    limiting_factor: 'NONE',
    gross_weight_kg: 2890,
    cg_long_mm: 3864,
    cg_lat_mm: -13.5,
    fwd_limit_mm: 3796.7,
    aft_limit_mm: 4188.3,
    left_limit_mm: -80.0,
    right_limit_mm: 80.0,
    fwd_margin_mm: 67.3,
    aft_margin_mm: 324.3,
    left_margin_mm: 66.5,
    right_margin_mm: 93.5
  }
};

export const latestKinematics = writable<MissionKinematicsEvaluation>(defaultKinematics);

export const currentGrossWeight = derived(latestKinematics, ($k) => $k.mass_balance.gross_weight_kg);
export const currentCgLong = derived(latestKinematics, ($k) => $k.mass_balance.cg_long_mm);
export const currentCgLat = derived(latestKinematics, ($k) => $k.mass_balance.cg_lat_mm);
export const isCgCompliant = derived(latestKinematics, ($k) => $k.mass_balance.is_overall_compliant);

export function ingestKinematicsEnvelope(envelope: ServerEnvelope<unknown>): void {
  const payload = envelope.payload as MissionKinematicsEvaluation | undefined;
  if (!payload || typeof payload !== 'object' || !payload.mass_balance) return;
  latestKinematics.set(payload);
}
