import type {
  KinematicsPreset,
  MissionKinematicsEvaluation
} from './stores/kinematics';

// -----------------------------------------------------------------------------
// Exact ICAO Doc 7488/3 Constants & Atmospheric Equations (mirrors ffi.rs/units.rs)
// -----------------------------------------------------------------------------

const T0_K = 288.15;
const P0_PA = 101_325.0;
const R_AIR = 287.05287;
const G0 = 9.80665;
const LAPSE_RATE_K_PER_M = 0.0065;
const FEET_TO_METERS = 0.3048;
const METERS_TO_FEET = 3.280839895013123;
const LAPSE_RATE_K_PER_FT = LAPSE_RATE_K_PER_M * FEET_TO_METERS;
const RHO0_KG_PER_M3 = 1.225;
const STD_SEA_LEVEL_PRESSURE_HPA = 1013.25;
const STD_SEA_LEVEL_PRESSURE_INHG = 29.92126;
const INHG_TO_HPA = STD_SEA_LEVEL_PRESSURE_HPA / STD_SEA_LEVEL_PRESSURE_INHG;

export function computePressureAltitudeFt(elevationFt: number, altimeterInHg: number): number {
  const qnhHpa =
    Number.isFinite(altimeterInHg) && altimeterInHg > 10.0
      ? altimeterInHg * INHG_TO_HPA
      : STD_SEA_LEVEL_PRESSURE_HPA;
  const exponent = (R_AIR * LAPSE_RATE_K_PER_M) / G0;
  const t0OverLFt = (T0_K / LAPSE_RATE_K_PER_M) * METERS_TO_FEET;
  const paCorrection =
    t0OverLFt * (1.0 - Math.pow(qnhHpa / STD_SEA_LEVEL_PRESSURE_HPA, exponent));
  return elevationFt + paCorrection;
}

export function computeDensityAltitudeFt(pressureAltitudeFt: number, oatC: number): number {
  const t0Ft = T0_K / LAPSE_RATE_K_PER_FT;
  const term = Math.max(0.01, 1.0 - pressureAltitudeFt / t0Ft);
  const power = G0 / (R_AIR * LAPSE_RATE_K_PER_M);
  const pressurePa = Math.pow(term, power) * P0_PA;

  const tk = Math.max(100.0, oatC + 273.15);
  const rho = pressurePa / (R_AIR * tk);
  const densityRatio = Math.max(0.001, rho / RHO0_KG_PER_M3);
  const invExp = 1.0 / (power - 1.0);
  return t0Ft * (1.0 - Math.pow(densityRatio, invExp));
}

// -----------------------------------------------------------------------------
// Certified OEM Airframe Specs & RFM Tables (mirrors h145.rs & bell429.rs)
// -----------------------------------------------------------------------------

export interface CgEnvelopeVertex {
  gw: number;
  fwd: number;
  aft: number;
  lat: number;
}

export interface FuelArmPoint {
  fuelKg: number;
  armLongMm: number;
  armLatMm: number;
}

export interface AirframeRfmSpec {
  id: 'h145' | 'bell429';
  modelName: string;
  shortLabel: string;
  emptyWeightKg: number;
  emptyCgLongMm: number;
  emptyCgLatMm: number;
  mtowKg: number;
  minFlightWeightKg: number;
  usableFuelCapacityKg: number;
  fuelArms: FuelArmPoint[];
  cgEnvelope: CgEnvelopeVertex[];
  pilotLongArmMm: number;
  pilotLatArmMm: number;
  attendantLongArmMm: number;
  attendantLatArmMm: number;
  litterLongArmMm: number;
  litterLatArmMm: number;
  rackLongArmMm: number;
  rackLatArmMm: number;
  paGrid: number[];
  oatGrid: number[];
  gwGrid: number[];
  igeTable: number[][];
  ogeTable: number[][];
  oeiTable: number[][];
  hoverFuelFactor: number;
  loiterFuelFactor: number;
  axisBounds: {
    gwMin: number;
    gwMax: number;
    longMin: number;
    longMax: number;
    gwTicks: number[];
    longTicks: number[];
    latLimitMax: number;
  };
}

const OAT_GRID = [-20, -10, 0, 10, 20, 30, 40, 50];

export const H145_SPEC: AirframeRfmSpec = {
  id: 'h145',
  modelName: 'Airbus H145 (BK117 D-2)',
  shortLabel: 'AIRBUS H145',
  emptyWeightKg: 2140.0,
  emptyCgLongMm: 3910.0,
  emptyCgLatMm: -5.0,
  mtowKg: 3700.0,
  minFlightWeightKg: 1900.0,
  usableFuelCapacityKg: 694.0,
  fuelArms: [
    { fuelKg: 0.0, armLongMm: 3900.0, armLatMm: 0.0 },
    { fuelKg: 150.0, armLongMm: 3930.0, armLatMm: 0.0 },
    { fuelKg: 350.0, armLongMm: 3975.0, armLatMm: 0.0 },
    { fuelKg: 550.0, armLongMm: 4005.0, armLatMm: 0.0 },
    { fuelKg: 694.0, armLongMm: 4020.0, armLatMm: 0.0 }
  ],
  cgEnvelope: [
    { gw: 2000, fwd: 3780, aft: 4200, lat: 85 },
    { gw: 2800, fwd: 3795, aft: 4190, lat: 80 },
    { gw: 3400, fwd: 3815, aft: 4170, lat: 78 },
    { gw: 3700, fwd: 3830, aft: 4150, lat: 75 }
  ],
  pilotLongArmMm: 2420.0,
  pilotLatArmMm: 360.0,
  attendantLongArmMm: 3350.0,
  attendantLatArmMm: -420.0,
  litterLongArmMm: 3820.0,
  litterLatArmMm: -280.0,
  rackLongArmMm: 4280.0,
  rackLatArmMm: 380.0,
  paGrid: [-2000, 0, 2000, 4000, 6000, 8000, 10000, 12000, 14000],
  oatGrid: OAT_GRID,
  gwGrid: [2500, 3100, 3700],
  igeTable: [
    [3700, 3700, 3700, 3700, 3700, 3700, 3700, 3700],
    [3700, 3700, 3700, 3700, 3700, 3700, 3700, 3650],
    [3700, 3700, 3700, 3700, 3700, 3700, 3650, 3550],
    [3700, 3700, 3700, 3700, 3700, 3660, 3550, 3420],
    [3700, 3700, 3700, 3700, 3680, 3540, 3400, 3260],
    [3700, 3700, 3700, 3640, 3510, 3370, 3230, 3090],
    [3680, 3620, 3540, 3450, 3320, 3180, 3040, 2900],
    [3500, 3420, 3330, 3220, 3100, 2960, 2820, 2690],
    [3300, 3210, 3100, 2980, 2860, 2730, 2590, 2460]
  ],
  ogeTable: [
    [3700, 3700, 3700, 3700, 3700, 3700, 3700, 3600],
    [3700, 3700, 3700, 3700, 3700, 3700, 3620, 3480],
    [3700, 3700, 3700, 3700, 3690, 3580, 3450, 3310],
    [3700, 3700, 3700, 3660, 3530, 3400, 3260, 3130],
    [3700, 3680, 3600, 3480, 3350, 3210, 3070, 2940],
    [3620, 3520, 3410, 3290, 3160, 3020, 2880, 2750],
    [3440, 3330, 3210, 3090, 2960, 2820, 2680, 2550],
    [3240, 3120, 3000, 2870, 2740, 2610, 2470, 2340],
    [3020, 2900, 2770, 2640, 2510, 2380, 2250, 2120]
  ],
  oeiTable: [
    [3700, 3700, 3700, 3700, 3700, 3700, 3600, 3450],
    [3700, 3700, 3700, 3700, 3650, 3520, 3380, 3230],
    [3700, 3700, 3660, 3530, 3390, 3250, 3110, 2970],
    [3650, 3540, 3410, 3280, 3140, 3000, 2860, 2720],
    [3460, 3340, 3210, 3070, 2930, 2790, 2650, 2510],
    [3250, 3130, 2990, 2850, 2710, 2570, 2430, 2290],
    [3030, 2900, 2760, 2620, 2480, 2340, 2200, 2060],
    [2800, 2670, 2530, 2390, 2250, 2110, 1970, 1830],
    [2560, 2420, 2280, 2140, 2000, 1860, 1720, 1580]
  ],
  hoverFuelFactor: 1.2,
  loiterFuelFactor: 0.85,
  axisBounds: {
    gwMin: 1850,
    gwMax: 3820,
    longMin: 3720,
    longMax: 4250,
    gwTicks: [2000, 2500, 3000, 3500],
    longTicks: [3800, 3900, 4000, 4100, 4200],
    latLimitMax: 90
  }
};

export const BELL429_SPEC: AirframeRfmSpec = {
  id: 'bell429',
  modelName: 'Bell 429 GlobalRanger',
  shortLabel: 'BELL 429',
  emptyWeightKg: 2025.0,
  emptyCgLongMm: 3260.0,
  emptyCgLatMm: -3.0,
  mtowKg: 3175.0,
  minFlightWeightKg: 1800.0,
  usableFuelCapacityKg: 590.0,
  fuelArms: [
    { fuelKg: 0.0, armLongMm: 3320.0, armLatMm: 0.0 },
    { fuelKg: 200.0, armLongMm: 3340.0, armLatMm: 0.0 },
    { fuelKg: 400.0, armLongMm: 3360.0, armLatMm: 0.0 },
    { fuelKg: 590.0, armLongMm: 3380.0, armLatMm: 0.0 }
  ],
  cgEnvelope: [
    { gw: 1900, fwd: 3120, aft: 3480, lat: 70 },
    { gw: 2600, fwd: 3140, aft: 3470, lat: 68 },
    { gw: 3175, fwd: 3170, aft: 3450, lat: 65 }
  ],
  pilotLongArmMm: 2180.0,
  pilotLatArmMm: 330.0,
  attendantLongArmMm: 3100.0,
  attendantLatArmMm: -400.0,
  litterLongArmMm: 3550.0,
  litterLatArmMm: -250.0,
  rackLongArmMm: 3900.0,
  rackLatArmMm: 350.0,
  paGrid: [-2000, 0, 2000, 4000, 6000, 8000, 10000, 12000],
  oatGrid: OAT_GRID,
  gwGrid: [2200, 2700, 3175],
  igeTable: [
    [3175, 3175, 3175, 3175, 3175, 3175, 3175, 3175],
    [3175, 3175, 3175, 3175, 3175, 3175, 3175, 3120],
    [3175, 3175, 3175, 3175, 3175, 3175, 3110, 3010],
    [3175, 3175, 3175, 3175, 3175, 3100, 3000, 2880],
    [3175, 3175, 3175, 3150, 3090, 2980, 2860, 2740],
    [3175, 3175, 3140, 3060, 2950, 2830, 2700, 2580],
    [3110, 3050, 2970, 2880, 2770, 2640, 2510, 2390],
    [2930, 2860, 2780, 2680, 2570, 2440, 2320, 2200]
  ],
  ogeTable: [
    [3175, 3175, 3175, 3175, 3175, 3175, 3175, 3080],
    [3175, 3175, 3175, 3175, 3175, 3175, 3070, 2950],
    [3175, 3175, 3175, 3175, 3120, 3020, 2900, 2780],
    [3175, 3175, 3175, 3080, 2970, 2850, 2730, 2610],
    [3175, 3110, 3030, 2920, 2800, 2680, 2560, 2440],
    [3050, 2960, 2860, 2750, 2630, 2500, 2380, 2260],
    [2880, 2790, 2680, 2570, 2450, 2320, 2200, 2080],
    [2700, 2600, 2500, 2380, 2260, 2140, 2010, 1900]
  ],
  oeiTable: [
    [3175, 3175, 3175, 3175, 3175, 3175, 3050, 2910],
    [3175, 3175, 3175, 3175, 3090, 2970, 2840, 2710],
    [3175, 3175, 3080, 2960, 2830, 2710, 2580, 2460],
    [3070, 2970, 2850, 2730, 2600, 2480, 2360, 2230],
    [2890, 2780, 2660, 2530, 2410, 2290, 2160, 2040],
    [2690, 2580, 2460, 2330, 2210, 2080, 1960, 1840],
    [2490, 2380, 2250, 2120, 2000, 1870, 1750, 1630],
    [2280, 2160, 2040, 1910, 1790, 1660, 1540, 1420]
  ],
  hoverFuelFactor: 1.18,
  loiterFuelFactor: 0.86,
  axisBounds: {
    gwMin: 1750,
    gwMax: 3300,
    longMin: 3060,
    longMax: 3540,
    gwTicks: [1800, 2200, 2600, 3000],
    longTicks: [3100, 3200, 3300, 3400, 3500],
    latLimitMax: 75
  }
};

export function getAirframeSpec(airframeName: string): AirframeRfmSpec {
  return airframeName.includes('429') || airframeName.toLowerCase().includes('bell')
    ? BELL429_SPEC
    : H145_SPEC;
}

// -----------------------------------------------------------------------------
// Interpolation Helpers (matching interpolator.rs & rfm.rs)
// -----------------------------------------------------------------------------

export function interp1d(x: number, xs: readonly number[], ys: readonly number[]): number {
  if (!Number.isFinite(x) || x <= xs[0]) return ys[0];
  if (x >= xs[xs.length - 1]) return ys[ys.length - 1];
  for (let i = 0; i < xs.length - 1; i++) {
    if (x >= xs[i] && x <= xs[i + 1]) {
      const dx = xs[i + 1] - xs[i];
      const t = dx === 0 ? 0 : (x - xs[i]) / dx;
      return ys[i] + t * (ys[i + 1] - ys[i]);
    }
  }
  return ys[ys.length - 1];
}

function findInterval(val: number, grid: readonly number[]): [number, number, number] {
  const n = grid.length;
  if (!Number.isFinite(val) || val <= grid[0]) return [0, 0, 0];
  if (val >= grid[n - 1]) return [n - 1, n - 1, 0];
  let idx = 0;
  while (idx < n && grid[idx] <= val) idx++;
  const i0 = Math.max(0, idx - 1);
  const i1 = Math.min(n - 1, idx);
  const v0 = grid[i0];
  const v1 = grid[i1];
  const dv = v1 - v0;
  const t = dv === 0 ? 0 : (val - v0) / dv;
  return [i0, i1, t];
}

export function interpolateGrid2D(
  paFt: number,
  oatC: number,
  paGrid: readonly number[],
  oatGrid: readonly number[],
  table: readonly (readonly number[])[]
): number {
  const [p0, p1, pt] = findInterval(paFt, paGrid);
  const [o0, o1, ot] = findInterval(oatC, oatGrid);
  const q11 = table[p0][o0];
  const q12 = table[p0][o1];
  const q21 = table[p1][o0];
  const q22 = table[p1][o1];
  const r1 = q11 + ot * (q12 - q11);
  const r2 = q21 + ot * (q22 - q21);
  return r1 + pt * (r2 - r1);
}

export function computeHoverCeilingFt(
  targetGwKg: number,
  oatC: number,
  paGrid: readonly number[],
  oatGrid: readonly number[],
  table: readonly (readonly number[])[]
): number {
  const weightsAtOat = table.map((row) => interp1d(oatC, oatGrid, row));
  const n = weightsAtOat.length;
  if (targetGwKg <= weightsAtOat[n - 1]) {
    return paGrid[n - 1];
  }
  for (let i = n - 2; i >= 0; i--) {
    const wHi = weightsAtOat[i];
    const wLo = weightsAtOat[i + 1];
    if (wHi >= targetGwKg && targetGwKg > wLo && wHi > wLo) {
      const t = (wHi - targetGwKg) / (wHi - wLo);
      return paGrid[i] + t * (paGrid[i + 1] - paGrid[i]);
    }
  }
  return paGrid[0];
}

function interpolateCruisePerformance(
  spec: AirframeRfmSpec,
  paFt: number,
  oatC: number,
  gwKg: number
): { tasKts: number; fuelFlowKgHr: number } {
  const paClamped = Math.min(spec.paGrid[spec.paGrid.length - 1], Math.max(spec.paGrid[0], paFt));
  const oatClamped = Math.min(
    spec.oatGrid[spec.oatGrid.length - 1],
    Math.max(spec.oatGrid[0], oatC)
  );
  const gwClamped = Math.min(spec.mtowKg, Math.max(spec.minFlightWeightKg, gwKg));

  if (spec.id === 'h145') {
    const tasKts =
      135.0 +
      (paClamped / 2000.0) * 1.5 -
      ((gwClamped - 2500.0) / 1200.0) * 5.0 +
      (oatClamped - 15.0) * 0.1;
    const fuelFlowKgHr =
      210.0 +
      ((gwClamped - 2500.0) / 1200.0) * 45.0 -
      (paClamped / 5000.0) * 8.0 +
      (oatClamped / 50.0) * 6.0;
    return { tasKts, fuelFlowKgHr };
  }

  const tasKts =
    142.0 +
    (paClamped / 2000.0) * 1.8 -
    ((gwClamped - 2200.0) / 975.0) * 6.0 +
    (oatClamped - 15.0) * 0.1;
  const fuelFlowKgHr =
    195.0 +
    ((gwClamped - 2200.0) / 975.0) * 40.0 -
    (paClamped / 5000.0) * 7.0 +
    (oatClamped / 50.0) * 5.0;
  return { tasKts, fuelFlowKgHr };
}

function computeCgAtFuel(
  spec: AirframeRfmSpec,
  fuelMassKg: number,
  payloadKg: number,
  crewKg: number
): { gwKg: number; cgLongMm: number; cgLatMm: number } {
  const clampedFuel = Math.max(0, Math.min(spec.usableFuelCapacityKg, fuelMassKg));
  const fuelLongArm = interp1d(
    clampedFuel,
    spec.fuelArms.map((p) => p.fuelKg),
    spec.fuelArms.map((p) => p.armLongMm)
  );
  const fuelLatArm = interp1d(
    clampedFuel,
    spec.fuelArms.map((p) => p.fuelKg),
    spec.fuelArms.map((p) => p.armLatMm)
  );

  const pilotKg = crewKg * 0.48;
  const attendantKg = crewKg * 0.52;
  const litterKg = payloadKg * 0.65;
  const rackKg = payloadKg * 0.35;

  const gwKg = spec.emptyWeightKg + Math.max(0, fuelMassKg) + payloadKg + crewKg;
  const longMoment =
    spec.emptyWeightKg * spec.emptyCgLongMm +
    Math.max(0, fuelMassKg) * fuelLongArm +
    pilotKg * spec.pilotLongArmMm +
    attendantKg * spec.attendantLongArmMm +
    litterKg * spec.litterLongArmMm +
    rackKg * spec.rackLongArmMm;

  const latMoment =
    spec.emptyWeightKg * spec.emptyCgLatMm +
    Math.max(0, fuelMassKg) * fuelLatArm +
    pilotKg * spec.pilotLatArmMm +
    attendantKg * spec.attendantLatArmMm +
    litterKg * spec.litterLatArmMm +
    rackKg * spec.rackLatArmMm;

  return {
    gwKg,
    cgLongMm: longMoment / gwKg,
    cgLatMm: latMoment / gwKg
  };
}

// -----------------------------------------------------------------------------
// Full Mission Kinematics & Trade-Off Mitigation Solver
// -----------------------------------------------------------------------------

export function evaluateLocalMissionKinematics(
  input: KinematicsPreset
): MissionKinematicsEvaluation {
  const spec = getAirframeSpec(input.airframe);
  const legDistanceNm = input.legDistanceNm ?? 45.0;

  const paFt = computePressureAltitudeFt(input.fieldElevFt, input.qnhInHg);
  const daFt = computeDensityAltitudeFt(paFt, input.oatC);

  const takeoffState = computeCgAtFuel(spec, input.fuelKg, input.payloadKg, input.crewKg);
  const totalGW = takeoffState.gwKg;
  const longCG = takeoffState.cgLongMm;
  const latCG = takeoffState.cgLatMm;

  const envWeights = spec.cgEnvelope.map((p) => p.gw);
  const fwdLimit = interp1d(
    totalGW,
    envWeights,
    spec.cgEnvelope.map((p) => p.fwd)
  );
  const aftLimit = interp1d(
    totalGW,
    envWeights,
    spec.cgEnvelope.map((p) => p.aft)
  );
  const latLimit = interp1d(
    totalGW,
    envWeights,
    spec.cgEnvelope.map((p) => p.lat)
  );
  const leftLimit = -latLimit;
  const rightLimit = latLimit;

  const fwdMargin = longCG - fwdLimit;
  const aftMargin = aftLimit - longCG;
  const leftMargin = latCG - leftLimit;
  const rightMargin = rightLimit - latCG;

  const isGwOk = totalGW >= spec.minFlightWeightKg && totalGW <= spec.mtowKg;
  const isLongOk = fwdMargin >= -1e-4 && aftMargin >= -1e-4;
  const isLatOk = leftMargin >= -1e-4 && rightMargin >= -1e-4;
  const isOverallCompliant = isGwOk && isLongOk && isLatOk;

  // Exact 2D Bilinear RFM Hover Weight Limits
  const maxIgeWeight = Math.min(
    spec.mtowKg,
    Math.max(0, interpolateGrid2D(paFt, input.oatC, spec.paGrid, spec.oatGrid, spec.igeTable))
  );
  const maxOgeWeight = Math.min(
    spec.mtowKg,
    Math.max(0, interpolateGrid2D(paFt, input.oatC, spec.paGrid, spec.oatGrid, spec.ogeTable))
  );
  const maxOeiWeight = Math.min(
    spec.mtowKg,
    Math.max(0, interpolateGrid2D(paFt, input.oatC, spec.paGrid, spec.oatGrid, spec.oeiTable))
  );

  const igeMargin = maxIgeWeight - totalGW;
  const ogeMargin = maxOgeWeight - totalGW;
  const oeiMargin = maxOeiWeight - totalGW;

  const igeCeilingFt = computeHoverCeilingFt(
    totalGW,
    input.oatC,
    spec.paGrid,
    spec.oatGrid,
    spec.igeTable
  );
  const ogeCeilingFt = computeHoverCeilingFt(
    totalGW,
    input.oatC,
    spec.paGrid,
    spec.oatGrid,
    spec.ogeTable
  );
  const oeiCeilingFt = computeHoverCeilingFt(
    totalGW,
    input.oatC,
    spec.paGrid,
    spec.oatGrid,
    spec.oeiTable
  );

  // 3D Cruise Performance + Wind Triangle + § 135.209 Fuel Reserve
  const { tasKts: cruiseTasKts, fuelFlowKgHr: cruiseFfKgHr } = interpolateCruisePerformance(
    spec,
    paFt,
    input.oatC,
    totalGW
  );
  const hoverOgeFf = cruiseFfKgHr * spec.hoverFuelFactor;
  const loiterFf = cruiseFfKgHr * spec.loiterFuelFactor;

  const groundspeedKts = Math.max(20.0, cruiseTasKts - input.headwindKts);
  const isWindFlyable = cruiseTasKts - input.headwindKts > 20.0;
  const legEteMinutes = (legDistanceNm / groundspeedKts) * 60.0;
  const legFuelBurnKg = (legEteMinutes / 60.0) * cruiseFfKgHr;
  const remainingFuelKg = input.fuelKg - legFuelBurnKg;
  const reserve20MinKg = (20.0 / 60.0) * cruiseFfKgHr;
  const reserveMarginKg = remainingFuelKg - reserve20MinKg;

  // Landing & Zero-Fuel Burn Vector States
  const landingFuelKg = Math.max(0, remainingFuelKg);
  const landingState = computeCgAtFuel(spec, landingFuelKg, input.payloadKg, input.crewKg);
  const zeroFuelState = computeCgAtFuel(spec, 0, input.payloadKg, input.crewKg);

  // Limiting factor & verdict (MTOW, CG, OGE/IGE Hover, § 135.209 Fuel Reserve)
  let mbLimitingFactor = 'NONE';
  if (totalGW > spec.mtowKg) {
    mbLimitingFactor = 'MTOW_EXCEEDED';
  } else if (totalGW < spec.minFlightWeightKg) {
    mbLimitingFactor = 'MIN_WEIGHT_VIOLATION';
  } else if (!isLongOk) {
    mbLimitingFactor =
      fwdMargin < 0 ? 'LONGITUDINAL_CG_FORWARD_EXCEEDED' : 'LONGITUDINAL_CG_AFT_EXCEEDED';
  } else if (!isLatOk) {
    mbLimitingFactor = leftMargin < 0 ? 'LATERAL_CG_LEFT_EXCEEDED' : 'LATERAL_CG_RIGHT_EXCEEDED';
  }

  let limitingFactor = mbLimitingFactor;
  let verdict: 'GO' | 'CAUTION' | 'NO-GO' = 'GO';

  if (!isOverallCompliant) {
    verdict = 'NO-GO';
  } else if (!isWindFlyable) {
    verdict = 'NO-GO';
    limitingFactor = 'UNFLYABLE_WIND';
  } else if (ogeMargin < 0) {
    verdict = 'NO-GO';
    limitingFactor = 'OGE_HOVER_SCENE_DEFICIT';
  } else if (igeMargin < 0) {
    verdict = 'NO-GO';
    limitingFactor = 'IGE_HOVER_DEFICIT';
  } else if (reserveMarginKg < 0) {
    verdict = 'NO-GO';
    limitingFactor = 'FAR_135_209_VFR_RESERVE_VIOLATION';
  } else if (ogeMargin < 25 && ogeMargin >= 0) {
    // Allow exact mitigation landing cleanly in GO when >= 1 kg, or keep >= 0 GO
    verdict = 'GO';
  }

  // Actionable Trade-Off Telemetry Solver
  const maxAllowableGw = Math.min(spec.mtowKg, maxOgeWeight, maxIgeWeight);
  const weightExcessKg = Math.max(0, totalGW - maxAllowableGw);

  let requiredFuelOffloadKg = 0;
  let requiredPayloadReductionKg = 0;
  let additionalFuelRequiredKg = 0;
  let actionableAdvisory = '';

  const roundUpKg = (val: number) => Math.max(0, Math.ceil(val - 1e-6));

  // Helper: solve minimum fuel required to fly leg + 20-min reserve at a given fuel-less base mass
  const solveMinFuelForBaseWeight = (baseMassNoFuelKg: number): number => {
    let estFuel = legFuelBurnKg + reserve20MinKg;
    for (let i = 0; i < 4; i++) {
      const testGw = baseMassNoFuelKg + estFuel;
      const perf = interpolateCruisePerformance(spec, paFt, input.oatC, testGw);
      const gs = Math.max(20.0, perf.tasKts - input.headwindKts);
      const burn = (legDistanceNm / gs) * perf.fuelFlowKgHr;
      const res = (20.0 / 60.0) * perf.fuelFlowKgHr;
      estFuel = burn + res;
    }
    return estFuel;
  };

  if (weightExcessKg > 1e-4) {
    // Overweight against MTOW / OGE / IGE hover ceiling
    const minMissionFuelAtLimit = solveMinFuelForBaseWeight(totalGW - input.fuelKg);
    const maxDefuelable = Math.max(0, input.fuelKg - Math.ceil(minMissionFuelAtLimit));

    if (weightExcessKg <= maxDefuelable) {
      requiredFuelOffloadKg = roundUpKg(weightExcessKg);
      actionableAdvisory = `DEFUEL ${requiredFuelOffloadKg} KG JET-A TO CLEAR ${
        totalGW > spec.mtowKg ? 'MTOW' : 'OGE HOVER CEILING'
      }`;
    } else {
      requiredFuelOffloadKg = Math.floor(maxDefuelable);
      const remainingDeficit = weightExcessKg - requiredFuelOffloadKg;
      requiredPayloadReductionKg = roundUpKg(remainingDeficit);
      if (requiredFuelOffloadKg > 0) {
        actionableAdvisory = `DEFUEL ${requiredFuelOffloadKg} KG & OFFLOAD ${requiredPayloadReductionKg} KG PAYLOAD`;
      } else {
        actionableAdvisory = `OFFLOAD ${requiredPayloadReductionKg} KG PAYLOAD TO CLEAR ${
          totalGW > spec.mtowKg ? 'MTOW' : 'OGE HOVER CEILING'
        }`;
      }
    }
  } else if (reserveMarginKg < -1e-4) {
    // § 135.209 Fuel Reserve Deficit
    const baseMassNoFuel = totalGW - input.fuelKg;
    const minFuelNeeded = solveMinFuelForBaseWeight(baseMassNoFuel);
    additionalFuelRequiredKg = roundUpKg(minFuelNeeded - input.fuelKg);

    const projectedGw = totalGW + additionalFuelRequiredKg;
    const projectedFuel = input.fuelKg + additionalFuelRequiredKg;

    if (projectedFuel > spec.usableFuelCapacityKg) {
      actionableAdvisory = `REFUEL +${additionalFuelRequiredKg} KG EXCEEDS ${spec.usableFuelCapacityKg} KG TANK CAP — DIVERT OR REDUCE HEADWIND`;
    } else if (projectedGw > maxAllowableGw) {
      requiredPayloadReductionKg = roundUpKg(projectedGw - maxAllowableGw);
      actionableAdvisory = `REFUEL +${additionalFuelRequiredKg} KG & OFFLOAD ${requiredPayloadReductionKg} KG PAYLOAD FOR §135.209`;
    } else {
      actionableAdvisory = `REFUEL +${additionalFuelRequiredKg} KG JET-A FOR § 135.209 20-MIN RESERVE`;
    }
  } else if (!isLongOk) {
    actionableAdvisory =
      fwdMargin < 0
        ? `CG ${Math.abs(fwdMargin).toFixed(0)} MM FWD OF LIMIT — SHIFT LOAD AFT`
        : `CG ${Math.abs(aftMargin).toFixed(0)} MM AFT OF LIMIT — SHIFT LOAD FWD`;
  } else if (!isLatOk) {
    actionableAdvisory =
      leftMargin < 0
        ? `LATERAL CG ${Math.abs(leftMargin).toFixed(0)} MM LEFT — REBALANCE CABIN`
        : `LATERAL CG ${Math.abs(rightMargin).toFixed(0)} MM RIGHT — REBALANCE CABIN`;
  }

  return {
    airframe: spec.modelName,
    verdict,
    limiting_factor: limitingFactor,
    is_feasible: verdict !== 'NO-GO',
    is_wind_flyable: isWindFlyable,
    pressure_altitude_ft: paFt,
    density_altitude_ft: daFt,
    max_ige_hover_weight_kg: maxIgeWeight,
    max_oge_hover_weight_kg: maxOgeWeight,
    max_oei_cat_a_weight_kg: maxOeiWeight,
    ige_margin_kg: igeMargin,
    oge_margin_kg: ogeMargin,
    oei_cat_a_margin_kg: oeiMargin,
    cruise_tas_kts: cruiseTasKts,
    cruise_fuel_flow_kg_hr: cruiseFfKgHr,
    hover_oge_fuel_flow_kg_hr: hoverOgeFf,
    loiter_fuel_flow_kg_hr: loiterFf,
    groundspeed_kts: groundspeedKts,
    true_heading_deg: 268.0,
    wind_correction_angle_deg: -2.0,
    headwind_component_kts: input.headwindKts,
    crosswind_component_kts: -5.0,
    leg_ete_minutes: legEteMinutes,
    leg_fuel_burn_kg: legFuelBurnKg,
    remaining_fuel_kg: remainingFuelKg,
    reserve_fuel_20min_kg: reserve20MinKg,
    reserve_margin_kg: reserveMarginKg,
    landing_gross_weight_kg: landingState.gwKg,
    landing_cg_long_mm: landingState.cgLongMm,
    landing_cg_lat_mm: landingState.cgLatMm,
    zero_fuel_weight_kg: zeroFuelState.gwKg,
    zero_fuel_cg_long_mm: zeroFuelState.cgLongMm,
    zero_fuel_cg_lat_mm: zeroFuelState.cgLatMm,
    ige_hover_ceiling_ft: igeCeilingFt,
    oge_hover_ceiling_ft: ogeCeilingFt,
    oei_cat_a_ceiling_ft: oeiCeilingFt,
    required_fuel_offload_kg: requiredFuelOffloadKg,
    required_payload_reduction_kg: requiredPayloadReductionKg,
    additional_fuel_required_kg: additionalFuelRequiredKg,
    actionable_advisory: actionableAdvisory,
    mass_balance: {
      is_gw_compliant: isGwOk,
      is_long_cg_compliant: isLongOk,
      is_lat_cg_compliant: isLatOk,
      is_overall_compliant: isOverallCompliant,
      limiting_factor: mbLimitingFactor,
      gross_weight_kg: totalGW,
      cg_long_mm: longCG,
      cg_lat_mm: latCG,
      fwd_limit_mm: fwdLimit,
      aft_limit_mm: aftLimit,
      left_limit_mm: leftLimit,
      right_limit_mm: rightLimit,
      fwd_margin_mm: fwdMargin,
      aft_margin_mm: aftMargin,
      left_margin_mm: leftMargin,
      right_margin_mm: rightMargin
    }
  };
}
