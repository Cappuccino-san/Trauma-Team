import { writable } from 'svelte/store';

/**
 * Global Display Unit Store (`KG/MM` <-> `LBS/IN`).
 * Mirrors dimensional conversion constants from `crates/traumateam_rfm/src/units.rs`.
 * Converts displayed weights, fuel quantities, fuel flow, and CG arms across the UI
 * without altering underlying SI calculations.
 */
export type UnitSystem = 'KG_MM' | 'LBS_IN';

export const LBS_TO_KG = 0.45359237;
export const KG_TO_LBS = 1.0 / LBS_TO_KG;
export const INCHES_TO_MM = 25.4;
export const MM_TO_INCHES = 1.0 / INCHES_TO_MM;
export const JET_A_STD_DENSITY_KG_PER_L = 0.804;
export const US_GAL_TO_LITERS = 3.785411784;

export const unitSystem = writable<UnitSystem>('KG_MM');

export function toggleUnitSystem(): UnitSystem {
  let next: UnitSystem = 'KG_MM';
  unitSystem.update((cur) => {
    next = cur === 'KG_MM' ? 'LBS_IN' : 'KG_MM';
    return next;
  });
  return next;
}

export function resetUnitStore(): void {
  unitSystem.set('KG_MM');
}

export function convertWeight(kg: number, system: UnitSystem): number {
  return system === 'LBS_IN' ? kg * KG_TO_LBS : kg;
}

export function convertArm(mm: number, system: UnitSystem): number {
  return system === 'LBS_IN' ? mm * MM_TO_INCHES : mm;
}

export function weightUnitLabel(system: UnitSystem): 'KG' | 'LBS' {
  return system === 'LBS_IN' ? 'LBS' : 'KG';
}

export function fuelFlowUnitLabel(system: UnitSystem): 'KG/H' | 'LBS/H' {
  return system === 'LBS_IN' ? 'LBS/H' : 'KG/H';
}

export function armUnitLabel(system: UnitSystem): 'MM' | 'IN' {
  return system === 'LBS_IN' ? 'IN' : 'MM';
}

export function formatWeight(
  kg: number,
  system: UnitSystem,
  options: { decimals?: number; signed?: boolean; includeUnit?: boolean } = {}
): string {
  const { decimals = 0, signed = false, includeUnit = true } = options;
  if (!Number.isFinite(kg)) return '--';
  const converted = convertWeight(kg, system);
  const rounded = decimals === 0 ? Math.round(converted) : Number(converted.toFixed(decimals));
  const sign = signed && rounded >= 0 ? '+' : '';
  const numStr = decimals === 0 ? `${rounded}` : rounded.toFixed(decimals);
  return includeUnit ? `${sign}${numStr} ${weightUnitLabel(system)}` : `${sign}${numStr}`;
}

export function formatFuelVolume(kg: number, system: UnitSystem): string {
  if (!Number.isFinite(kg)) return '--';
  const liters = Math.max(0, kg) / JET_A_STD_DENSITY_KG_PER_L;
  if (system === 'LBS_IN') {
    const gallons = liters / US_GAL_TO_LITERS;
    return `${Math.round(gallons)} GAL`;
  }
  return `${Math.round(liters)} L`;
}

export function formatFuelFlow(kgHr: number, system: UnitSystem): string {
  if (!Number.isFinite(kgHr)) return '--';
  const val = Math.round(convertWeight(kgHr, system));
  return `${val} ${fuelFlowUnitLabel(system)}`;
}

export function formatArm(
  mm: number,
  system: UnitSystem,
  options: { decimals?: number; signed?: boolean; includeUnit?: boolean } = {}
): string {
  const { decimals, signed = false, includeUnit = true } = options;
  if (!Number.isFinite(mm)) return '--';
  const converted = convertArm(mm, system);
  const effDecimals = decimals ?? (system === 'LBS_IN' ? 1 : 1);
  const sign = signed && converted >= 0 ? '+' : '';
  const numStr = effDecimals === 0 ? `${Math.round(converted)}` : converted.toFixed(effDecimals);
  return includeUnit ? `${sign}${numStr} ${armUnitLabel(system)}` : `${sign}${numStr}`;
}
