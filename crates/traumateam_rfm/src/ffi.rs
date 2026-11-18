//! C-Compatible Foreign Function Interface (FFI) for the TraumaTeam Rust Physics & RFM Core.
//!
//! Exposes `#[repr(C)]` data structures and `#[no_mangle] pub extern "C"` functions so that
//! the Go `weather_ingest` backend (via `cgo` or native dynamic library loading) can invoke
//! the certified Rotorcraft Flight Manual (RFM) interpolators, mass-and-balance solver, and
//! atmospheric/wind-triangle physics natively with zero heap allocations on the hot path.

use crate::airframes::{get_bell429_rfm_spec, get_h145_rfm_spec};
use crate::mass_balance::compute_mass_and_balance;
use crate::rfm::{DispatchVerdict, LimitingFactor, PayloadItem, RFMAirframeSpec, StationId};
use crate::units::{
    Celsius, Feet, Kilograms, FEET_TO_METERS, METERS_TO_FEET, STD_SEA_LEVEL_PRESSURE_HPA,
    STD_SEA_LEVEL_PRESSURE_INHG,
};
use std::sync::OnceLock;

// Cached singleton airframe specifications to guarantee zero heap allocation per FFI call.
static H145_SPEC: OnceLock<RFMAirframeSpec> = OnceLock::new();
static BELL429_SPEC: OnceLock<RFMAirframeSpec> = OnceLock::new();

#[inline]
fn airframe_spec(airframe_id: u32) -> Option<&'static RFMAirframeSpec> {
    match airframe_id {
        0 => Some(H145_SPEC.get_or_init(get_h145_rfm_spec)),
        1 => Some(BELL429_SPEC.get_or_init(get_bell429_rfm_spec)),
        _ => None,
    }
}

/// FFI Airframe selector discriminant.
#[repr(u32)]
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum FfiAirframeId {
    AirbusH145 = 0,
    Bell429 = 1,
}

/// FFI Station identifier discriminant matching `StationId`.
#[repr(u32)]
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum FfiStationId {
    PilotRh = 0,
    CopilotLh = 1,
    CrewAttendantAftLh = 2,
    CrewAttendantAftRh = 3,
    CrewAttendantLh = 4,
    CrewAttendantRh = 5,
    PrimaryPatientLitter = 6,
    SecondaryPatientLitter = 7,
    MedicalCabinetRack = 8,
    MedicalEquipmentRack = 9,
    OxygenSystem = 10,
    AftAvionicsCargo = 11,
    AftBaggage = 12,
    RescueHoistRh = 13,
}

impl FfiStationId {
    pub const fn from_u32(code: u32) -> Option<StationId> {
        match code {
            0 => Some(StationId::PilotRh),
            1 => Some(StationId::CopilotLh),
            2 => Some(StationId::CrewAttendantAftLh),
            3 => Some(StationId::CrewAttendantAftRh),
            4 => Some(StationId::CrewAttendantLh),
            5 => Some(StationId::CrewAttendantRh),
            6 => Some(StationId::PrimaryPatientLitter),
            7 => Some(StationId::SecondaryPatientLitter),
            8 => Some(StationId::MedicalCabinetRack),
            9 => Some(StationId::MedicalEquipmentRack),
            10 => Some(StationId::OxygenSystem),
            11 => Some(StationId::AftAvionicsCargo),
            12 => Some(StationId::AftBaggage),
            13 => Some(StationId::RescueHoistRh),
            _ => None,
        }
    }
}

/// FFI LimitingFactor numeric code matching `LimitingFactor`.
#[repr(u32)]
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum FfiLimitingFactorCode {
    None = 0,
    MtowExceeded = 1,
    MinWeightViolation = 2,
    LongitudinalCgForwardExceeded = 3,
    LongitudinalCgAftExceeded = 4,
    LateralCgLeftExceeded = 5,
    LateralCgRightExceeded = 6,
    OgeHoverSceneDeficit = 7,
    IgeHoverDeficit = 8,
    OeiCatAHelipadDeficit = 9,
    Far135_615CeilingBelowMinimums = 10,
    Far135_615VisibilityBelowMinimums = 11,
    Far135_209VfrReserveViolation = 12,
    IfrReserveViolation = 13,
    SevereWeatherHazard = 14,
    TerrainObstacleClearanceViolation = 15,
    UnflyableWind = 16,
    InvalidInputOrStation = 99,
}

impl From<LimitingFactor> for FfiLimitingFactorCode {
    fn from(lf: LimitingFactor) -> Self {
        match lf {
            LimitingFactor::None => Self::None,
            LimitingFactor::MtowExceeded => Self::MtowExceeded,
            LimitingFactor::MinWeightViolation => Self::MinWeightViolation,
            LimitingFactor::LongitudinalCgForwardExceeded => Self::LongitudinalCgForwardExceeded,
            LimitingFactor::LongitudinalCgAftExceeded => Self::LongitudinalCgAftExceeded,
            LimitingFactor::LateralCgLeftExceeded => Self::LateralCgLeftExceeded,
            LimitingFactor::LateralCgRightExceeded => Self::LateralCgRightExceeded,
            LimitingFactor::OgeHoverSceneDeficit => Self::OgeHoverSceneDeficit,
            LimitingFactor::IgeHoverDeficit => Self::IgeHoverDeficit,
            LimitingFactor::OeiCatAHelipadDeficit => Self::OeiCatAHelipadDeficit,
            LimitingFactor::Far135_615CeilingBelowMinimums => Self::Far135_615CeilingBelowMinimums,
            LimitingFactor::Far135_615VisibilityBelowMinimums => {
                Self::Far135_615VisibilityBelowMinimums
            }
            LimitingFactor::Far135_209VfrReserveViolation => Self::Far135_209VfrReserveViolation,
            LimitingFactor::IfrReserveViolation => Self::IfrReserveViolation,
            LimitingFactor::SevereWeatherHazard => Self::SevereWeatherHazard,
            LimitingFactor::TerrainObstacleClearanceViolation => {
                Self::TerrainObstacleClearanceViolation
            }
            LimitingFactor::UnflyableWind => Self::UnflyableWind,
        }
    }
}

/// FFI DispatchVerdict numeric code matching `DispatchVerdict`.
#[repr(u32)]
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum FfiVerdictCode {
    Go = 0,
    Caution = 1,
    NoGo = 2,
}

impl From<DispatchVerdict> for FfiVerdictCode {
    fn from(v: DispatchVerdict) -> Self {
        match v {
            DispatchVerdict::Go => Self::Go,
            DispatchVerdict::Caution => Self::Caution,
            DispatchVerdict::NoGo => Self::NoGo,
        }
    }
}

/// C-compatible payload item passed across FFI boundary.
#[repr(C)]
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct FfiPayloadItem {
    pub station_id: u32,
    pub _padding: u32,
    pub weight_kg: f64,
}

/// C-compatible Weather & Mission Environment input structure marshaled from Go.
#[repr(C)]
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct FfiWeatherMissionInput {
    pub airframe_id: u32,
    pub _padding: u32,
    pub station_elevation_ft: f64,
    pub altimeter_inhg: f64,
    pub oat_celsius: f64,
    pub wind_dir_deg: f64,
    pub wind_speed_kts: f64,
    pub true_course_deg: f64,
    pub leg_distance_nm: f64,
    pub fuel_weight_kg: f64,
}

/// C-compatible Mass & Balance output structure returned to Go.
#[repr(C)]
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct FfiMassBalanceResult {
    pub status_code: i32,
    pub limiting_factor_code: u32,
    pub is_gw_compliant: u8,
    pub is_long_cg_compliant: u8,
    pub is_lat_cg_compliant: u8,
    pub is_overall_compliant: u8,
    pub _padding: u32,
    pub gross_weight_kg: f64,
    pub cg_long_mm: f64,
    pub cg_lat_mm: f64,
    pub fwd_limit_mm: f64,
    pub aft_limit_mm: f64,
    pub left_limit_mm: f64,
    pub right_limit_mm: f64,
    pub fwd_margin_mm: f64,
    pub aft_margin_mm: f64,
    pub left_margin_mm: f64,
    pub right_margin_mm: f64,
}

impl Default for FfiMassBalanceResult {
    fn default() -> Self {
        Self {
            status_code: -1,
            limiting_factor_code: FfiLimitingFactorCode::InvalidInputOrStation as u32,
            is_gw_compliant: 0,
            is_long_cg_compliant: 0,
            is_lat_cg_compliant: 0,
            is_overall_compliant: 0,
            _padding: 0,
            gross_weight_kg: 0.0,
            cg_long_mm: 0.0,
            cg_lat_mm: 0.0,
            fwd_limit_mm: 0.0,
            aft_limit_mm: 0.0,
            left_limit_mm: 0.0,
            right_limit_mm: 0.0,
            fwd_margin_mm: 0.0,
            aft_margin_mm: 0.0,
            left_margin_mm: 0.0,
            right_margin_mm: 0.0,
        }
    }
}

/// C-compatible RFM Performance & Kinematic Evaluation output structure returned to Go.
#[repr(C)]
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct FfiPerformanceKinematicResult {
    pub status_code: i32,
    pub verdict_code: u32,
    pub limiting_factor_code: u32,
    pub is_feasible: u8,
    pub is_wind_flyable: u8,
    pub _padding: [u8; 2],
    pub pressure_altitude_ft: f64,
    pub density_altitude_ft: f64,
    pub max_ige_hover_weight_kg: f64,
    pub max_oge_hover_weight_kg: f64,
    pub max_oei_cat_a_weight_kg: f64,
    pub ige_margin_kg: f64,
    pub oge_margin_kg: f64,
    pub oei_cat_a_margin_kg: f64,
    pub cruise_tas_kts: f64,
    pub cruise_fuel_flow_kg_hr: f64,
    pub hover_oge_fuel_flow_kg_hr: f64,
    pub loiter_fuel_flow_kg_hr: f64,
    pub groundspeed_kts: f64,
    pub true_heading_deg: f64,
    pub wind_correction_angle_deg: f64,
    pub headwind_component_kts: f64,
    pub crosswind_component_kts: f64,
    pub leg_ete_minutes: f64,
    pub leg_fuel_burn_kg: f64,
    pub remaining_fuel_kg: f64,
    pub reserve_fuel_20min_kg: f64,
    pub reserve_margin_kg: f64,
    pub mass_balance: FfiMassBalanceResult,
}

impl Default for FfiPerformanceKinematicResult {
    fn default() -> Self {
        Self {
            status_code: -1,
            verdict_code: FfiVerdictCode::NoGo as u32,
            limiting_factor_code: FfiLimitingFactorCode::InvalidInputOrStation as u32,
            is_feasible: 0,
            is_wind_flyable: 0,
            _padding: [0; 2],
            pressure_altitude_ft: 0.0,
            density_altitude_ft: 0.0,
            max_ige_hover_weight_kg: 0.0,
            max_oge_hover_weight_kg: 0.0,
            max_oei_cat_a_weight_kg: 0.0,
            ige_margin_kg: 0.0,
            oge_margin_kg: 0.0,
            oei_cat_a_margin_kg: 0.0,
            cruise_tas_kts: 0.0,
            cruise_fuel_flow_kg_hr: 0.0,
            hover_oge_fuel_flow_kg_hr: 0.0,
            loiter_fuel_flow_kg_hr: 0.0,
            groundspeed_kts: 0.0,
            true_heading_deg: 0.0,
            wind_correction_angle_deg: 0.0,
            headwind_component_kts: 0.0,
            crosswind_component_kts: 0.0,
            leg_ete_minutes: 0.0,
            leg_fuel_burn_kg: 0.0,
            remaining_fuel_kg: 0.0,
            reserve_fuel_20min_kg: 0.0,
            reserve_margin_kg: 0.0,
            mass_balance: FfiMassBalanceResult::default(),
        }
    }
}

// -----------------------------------------------------------------------------
// Atmospheric & Wind-Triangle Physics Helpers (ICAO Doc 7488/3)
// -----------------------------------------------------------------------------

const T0_K: f64 = 288.15;
const P0_PA: f64 = 101_325.0;
const R_AIR: f64 = 287.05287;
const G0: f64 = 9.80665;
const LAPSE_RATE_K_PER_M: f64 = 0.0065;
const LAPSE_RATE_K_PER_FT: f64 = LAPSE_RATE_K_PER_M * FEET_TO_METERS;
const RHO0_KG_PER_M3: f64 = 1.2250;
const INHG_TO_HPA: f64 = STD_SEA_LEVEL_PRESSURE_HPA / STD_SEA_LEVEL_PRESSURE_INHG;

/// Computes exact ICAO barometric pressure altitude in feet.
#[inline]
pub fn compute_pressure_altitude_ft(elevation_ft: f64, altimeter_inhg: f64) -> f64 {
    let qnh_hpa = if altimeter_inhg.is_finite() && altimeter_inhg > 10.0 {
        altimeter_inhg * INHG_TO_HPA
    } else {
        STD_SEA_LEVEL_PRESSURE_HPA
    };
    let exponent = (R_AIR * LAPSE_RATE_K_PER_M) / G0;
    let t0_over_l_ft = (T0_K / LAPSE_RATE_K_PER_M) * METERS_TO_FEET;
    let pa_correction =
        t0_over_l_ft * (1.0 - (qnh_hpa / STD_SEA_LEVEL_PRESSURE_HPA).powf(exponent));
    elevation_ft + pa_correction
}

/// Computes exact thermodynamic density altitude in feet.
#[inline]
pub fn compute_density_altitude_ft(pressure_altitude_ft: f64, oat_c: f64) -> f64 {
    let t0_ft = T0_K / LAPSE_RATE_K_PER_FT;
    let term = (1.0 - (pressure_altitude_ft / t0_ft)).max(0.01);
    let power = G0 / (R_AIR * LAPSE_RATE_K_PER_M);
    let pressure_pa = term.powf(power) * P0_PA;

    let tk = (oat_c + 273.15).max(100.0);
    let rho = pressure_pa / (R_AIR * tk);
    let density_ratio = (rho / RHO0_KG_PER_M3).max(0.001);
    let inv_exp = 1.0 / (power - 1.0);
    t0_ft * (1.0 - density_ratio.powf(inv_exp))
}

/// Solves the aeronautical wind triangle:
/// Returns `(groundspeed_kts, true_heading_deg, wca_deg, headwind_kts, crosswind_kts, is_flyable)`.
#[inline]
pub fn solve_wind_triangle_native(
    true_course_deg: f64,
    tas_kts: f64,
    wind_dir_deg: f64,
    wind_speed_kts: f64,
) -> (f64, f64, f64, f64, f64, bool) {
    if !tas_kts.is_finite()
        || tas_kts <= 0.0
        || !true_course_deg.is_finite()
        || !wind_dir_deg.is_finite()
        || !wind_speed_kts.is_finite()
        || wind_speed_kts < 0.0
    {
        let safe_tc = if true_course_deg.is_finite() {
            true_course_deg
        } else {
            0.0
        };
        return (0.0, safe_tc, 0.0, 0.0, 0.0, false);
    }
    let tc = true_course_deg.rem_euclid(360.0);
    let wd = wind_dir_deg.rem_euclid(360.0);
    let ws = wind_speed_kts;

    let alpha_rad = (wd - tc).to_radians();
    let headwind = ws * alpha_rad.cos();
    let crosswind = ws * alpha_rad.sin();

    let sin_wca = crosswind / tas_kts;
    if sin_wca.abs() >= 1.0 {
        return (0.0, tc, 0.0, headwind, crosswind, false);
    }

    let wca_rad = sin_wca.asin();
    let wca_deg = wca_rad.to_degrees();
    let true_heading = (tc + wca_deg).rem_euclid(360.0);
    let groundspeed = (tas_kts * wca_rad.cos()) - headwind;

    if groundspeed <= 0.0 {
        return (0.0, true_heading, wca_deg, headwind, crosswind, false);
    }

    (
        groundspeed,
        true_heading,
        wca_deg,
        headwind,
        crosswind,
        true,
    )
}

// -----------------------------------------------------------------------------
// Exported `extern "C"` FFI Functions
// -----------------------------------------------------------------------------

/// Returns the ABI version of the TraumaTeam Rust FFI library (`100` = v1.0.0).
#[no_mangle]
pub extern "C" fn traumateam_ffi_abi_version() -> u32 {
    std::panic::catch_unwind(|| 100).unwrap_or(0)
}

/// Computes exact ICAO Pressure Altitude (ft) and Density Altitude (ft) via C ABI.
///
/// # Safety
/// `out_pa_ft` and `out_da_ft` must be valid, non-null mutable pointers to `f64`.
#[no_mangle]
pub unsafe extern "C" fn traumateam_compute_atmosphere_ffi(
    elevation_ft: f64,
    altimeter_inhg: f64,
    oat_celsius: f64,
    out_pa_ft: *mut f64,
    out_da_ft: *mut f64,
) -> i32 {
    if out_pa_ft.is_null() || out_da_ft.is_null() {
        return -1;
    }
    match std::panic::catch_unwind(std::panic::AssertUnwindSafe(|| {
        if !elevation_ft.is_finite()
            || !altimeter_inhg.is_finite()
            || !oat_celsius.is_finite()
            || altimeter_inhg <= 0.0
        {
            return -2;
        }
        let pa = compute_pressure_altitude_ft(elevation_ft, altimeter_inhg);
        let da = compute_density_altitude_ft(pa, oat_celsius);
        if !pa.is_finite() || !da.is_finite() {
            return -2;
        }
        *out_pa_ft = pa;
        *out_da_ft = da;
        0
    })) {
        Ok(rc) => rc,
        Err(_) => -99,
    }
}

/// Evaluates aircraft Mass & Balance and CG envelope margins via C ABI.
///
/// # Safety
/// - `payloads_ptr` may be null only if `payloads_len == 0`; otherwise it must point to
///   `payloads_len` contiguous `FfiPayloadItem` elements.
/// - `out_result` must be a valid, non-null mutable pointer to `FfiMassBalanceResult`.
#[no_mangle]
pub unsafe extern "C" fn traumateam_eval_mass_balance_ffi(
    airframe_id: u32,
    fuel_weight_kg: f64,
    payloads_ptr: *const FfiPayloadItem,
    payloads_len: usize,
    out_result: *mut FfiMassBalanceResult,
) -> i32 {
    if out_result.is_null() {
        return -1;
    }
    *out_result = FfiMassBalanceResult::default();

    match std::panic::catch_unwind(std::panic::AssertUnwindSafe(|| {
        eval_mass_balance_inner(
            airframe_id,
            fuel_weight_kg,
            payloads_ptr,
            payloads_len,
            out_result,
        )
    })) {
        Ok(rc) => rc,
        Err(_) => {
            *out_result = FfiMassBalanceResult::default();
            (*out_result).status_code = -99;
            -99
        }
    }
}

unsafe fn eval_mass_balance_inner(
    airframe_id: u32,
    fuel_weight_kg: f64,
    payloads_ptr: *const FfiPayloadItem,
    payloads_len: usize,
    out_result: *mut FfiMassBalanceResult,
) -> i32 {
    let Some(spec) = airframe_spec(airframe_id) else {
        (*out_result).status_code = -2;
        return -2;
    };

    if !fuel_weight_kg.is_finite() || fuel_weight_kg < 0.0 {
        (*out_result).status_code = -3;
        return -3;
    }

    if payloads_len > 0 && payloads_ptr.is_null() {
        (*out_result).status_code = -4;
        return -4;
    }

    let raw_slice = if payloads_len == 0 {
        &[][..]
    } else {
        std::slice::from_raw_parts(payloads_ptr, payloads_len)
    };

    let mut items = Vec::with_capacity(raw_slice.len());
    for p in raw_slice {
        if !p.weight_kg.is_finite() || p.weight_kg < 0.0 {
            (*out_result).status_code = -5;
            return -5;
        }
        let Some(st_id) = FfiStationId::from_u32(p.station_id) else {
            (*out_result).status_code = -6;
            return -6;
        };
        items.push(PayloadItem::new(
            "FFI_ITEM",
            "FFI Payload",
            st_id,
            Kilograms::new(p.weight_kg),
        ));
    }

    match compute_mass_and_balance(spec, Kilograms::new(fuel_weight_kg), &items) {
        Ok(snap) => {
            *out_result = FfiMassBalanceResult {
                status_code: 0,
                limiting_factor_code: FfiLimitingFactorCode::from(snap.limiting_factor) as u32,
                is_gw_compliant: u8::from(snap.is_gw_compliant),
                is_long_cg_compliant: u8::from(snap.is_long_cg_compliant),
                is_lat_cg_compliant: u8::from(snap.is_lat_cg_compliant),
                is_overall_compliant: u8::from(snap.is_overall_compliant),
                _padding: 0,
                gross_weight_kg: snap.gross_weight.get(),
                cg_long_mm: snap.cg_long.get(),
                cg_lat_mm: snap.cg_lat.get(),
                fwd_limit_mm: snap.fwd_limit.get(),
                aft_limit_mm: snap.aft_limit.get(),
                left_limit_mm: snap.left_limit.get(),
                right_limit_mm: snap.right_limit.get(),
                fwd_margin_mm: snap.fwd_margin.get(),
                aft_margin_mm: snap.aft_margin.get(),
                left_margin_mm: snap.left_margin.get(),
                right_margin_mm: snap.right_margin.get(),
            };
            0
        }
        Err(_) => {
            (*out_result).status_code = -7;
            -7
        }
    }
}

/// Evaluates full Mission Kinematics, RFM Hover/Cruise/OEI Performance, Wind Triangle,
/// 14 CFR § 135.209 20-minute VFR fuel reserve, and Mass-Balance in a single native call.
///
/// # Safety
/// - `input_ptr` must be a valid, non-null pointer to `FfiWeatherMissionInput`.
/// - `payloads_ptr` may be null only if `payloads_len == 0`.
/// - `out_result` must be a valid, non-null mutable pointer to `FfiPerformanceKinematicResult`.
#[no_mangle]
pub unsafe extern "C" fn traumateam_eval_mission_kinematics_ffi(
    input_ptr: *const FfiWeatherMissionInput,
    payloads_ptr: *const FfiPayloadItem,
    payloads_len: usize,
    out_result: *mut FfiPerformanceKinematicResult,
) -> i32 {
    if out_result.is_null() {
        return -1;
    }
    *out_result = FfiPerformanceKinematicResult::default();
    if input_ptr.is_null() {
        (*out_result).status_code = -1;
        return -1;
    }

    match std::panic::catch_unwind(std::panic::AssertUnwindSafe(|| {
        eval_mission_kinematics_inner(input_ptr, payloads_ptr, payloads_len, out_result)
    })) {
        Ok(rc) => rc,
        Err(_) => {
            *out_result = FfiPerformanceKinematicResult::default();
            (*out_result).status_code = -99;
            -99
        }
    }
}

unsafe fn eval_mission_kinematics_inner(
    input_ptr: *const FfiWeatherMissionInput,
    payloads_ptr: *const FfiPayloadItem,
    payloads_len: usize,
    out_result: *mut FfiPerformanceKinematicResult,
) -> i32 {
    let input = &*input_ptr;

    let Some(spec) = airframe_spec(input.airframe_id) else {
        (*out_result).status_code = -2;
        return -2;
    };

    // Fail closed on any NaN, Inf, or physically invalid mission parameters
    if !input.station_elevation_ft.is_finite()
        || !input.altimeter_inhg.is_finite()
        || input.altimeter_inhg <= 0.0
        || !input.oat_celsius.is_finite()
        || !input.wind_dir_deg.is_finite()
        || !input.wind_speed_kts.is_finite()
        || input.wind_speed_kts < 0.0
        || !input.true_course_deg.is_finite()
        || !input.leg_distance_nm.is_finite()
        || input.leg_distance_nm < 0.0
    {
        (*out_result).status_code = -8;
        return -8;
    }

    // 1. Evaluate Mass & Balance first
    let mut mb_res = FfiMassBalanceResult::default();
    let mb_rc = eval_mass_balance_inner(
        input.airframe_id,
        input.fuel_weight_kg,
        payloads_ptr,
        payloads_len,
        &mut mb_res,
    );
    if mb_rc != 0 {
        (*out_result).status_code = mb_rc;
        return mb_rc;
    }

    // 2. Compute Barometric Pressure Altitude & Thermodynamic Density Altitude
    let pa_ft = compute_pressure_altitude_ft(input.station_elevation_ft, input.altimeter_inhg);
    let da_ft = compute_density_altitude_ft(pa_ft, input.oat_celsius);

    let pa = Feet::new(pa_ft);
    let oat = Celsius::new(input.oat_celsius);
    let gw = Kilograms::new(mb_res.gross_weight_kg);

    // 3. Interpolate Certified RFM Hover, OEI Cat A, and Cruise Grids
    let max_ige = spec.get_max_ige_hover_weight(pa, oat).get();
    let max_oge = spec.get_max_oge_hover_weight(pa, oat).get();
    let max_oei = spec.get_max_oei_cat_a_weight(pa, oat).get();

    let ige_margin = max_ige - gw.get();
    let oge_margin = max_oge - gw.get();
    let oei_margin = max_oei - gw.get();

    let cruise_tas = spec.get_cruise_tas(pa, oat, gw).get();
    let cruise_ff = spec.get_cruise_fuel_flow(pa, oat, gw).get();
    let hover_ff = spec.get_hover_oge_fuel_flow(pa, oat, gw).get();
    let loiter_ff = spec.get_loiter_fuel_flow(pa, oat, gw).get();

    // 4. Solve Wind Triangle & Leg Fuel Integration
    let (gs_kts, th_deg, wca_deg, hw_kts, xw_kts, wind_flyable) = solve_wind_triangle_native(
        input.true_course_deg,
        cruise_tas,
        input.wind_dir_deg,
        input.wind_speed_kts,
    );

    let dist_nm = input.leg_distance_nm.max(0.0);
    let ete_hours = if wind_flyable && gs_kts > 0.0 {
        dist_nm / gs_kts
    } else {
        0.0
    };
    let ete_minutes = ete_hours * 60.0;
    let leg_fuel_burn = ete_hours * cruise_ff;
    let remaining_fuel = input.fuel_weight_kg - leg_fuel_burn;

    // 14 CFR § 135.209 VFR Day/Night minimum reserve: 20 minutes at cruise fuel flow
    let reserve_20min_kg = (20.0 / 60.0) * cruise_ff;
    let reserve_margin_kg = remaining_fuel - reserve_20min_kg;

    // 5. Determine overall LimitingFactor and DispatchVerdict
    let mut limiting_factor = mb_res.limiting_factor_code;
    let mut verdict = FfiVerdictCode::Go;

    if mb_res.is_overall_compliant == 0 {
        verdict = FfiVerdictCode::NoGo;
    } else if !wind_flyable {
        verdict = FfiVerdictCode::NoGo;
        limiting_factor = FfiLimitingFactorCode::UnflyableWind as u32;
    } else if oge_margin < 0.0 {
        verdict = FfiVerdictCode::NoGo;
        limiting_factor = FfiLimitingFactorCode::OgeHoverSceneDeficit as u32;
    } else if ige_margin < 0.0 {
        verdict = FfiVerdictCode::NoGo;
        limiting_factor = FfiLimitingFactorCode::IgeHoverDeficit as u32;
    } else if oei_margin < 0.0 {
        verdict = FfiVerdictCode::NoGo;
        limiting_factor = FfiLimitingFactorCode::OeiCatAHelipadDeficit as u32;
    } else if reserve_margin_kg < 0.0 {
        verdict = FfiVerdictCode::NoGo;
        limiting_factor = FfiLimitingFactorCode::Far135_209VfrReserveViolation as u32;
    } else if oge_margin < 50.0 || reserve_margin_kg < 25.0 {
        verdict = FfiVerdictCode::Caution;
    }

    let is_feasible = u8::from(verdict != FfiVerdictCode::NoGo);

    *out_result = FfiPerformanceKinematicResult {
        status_code: 0,
        verdict_code: verdict as u32,
        limiting_factor_code: limiting_factor,
        is_feasible,
        is_wind_flyable: u8::from(wind_flyable),
        _padding: [0; 2],
        pressure_altitude_ft: pa_ft,
        density_altitude_ft: da_ft,
        max_ige_hover_weight_kg: max_ige,
        max_oge_hover_weight_kg: max_oge,
        max_oei_cat_a_weight_kg: max_oei,
        ige_margin_kg: ige_margin,
        oge_margin_kg: oge_margin,
        oei_cat_a_margin_kg: oei_margin,
        cruise_tas_kts: cruise_tas,
        cruise_fuel_flow_kg_hr: cruise_ff,
        hover_oge_fuel_flow_kg_hr: hover_ff,
        loiter_fuel_flow_kg_hr: loiter_ff,
        groundspeed_kts: gs_kts,
        true_heading_deg: th_deg,
        wind_correction_angle_deg: wca_deg,
        headwind_component_kts: hw_kts,
        crosswind_component_kts: xw_kts,
        leg_ete_minutes: ete_minutes,
        leg_fuel_burn_kg: leg_fuel_burn,
        remaining_fuel_kg: remaining_fuel,
        reserve_fuel_20min_kg: reserve_20min_kg,
        reserve_margin_kg,
        mass_balance: mb_res,
    };

    0
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_ffi_abi_version() {
        assert_eq!(traumateam_ffi_abi_version(), 100);
    }

    #[test]
    fn test_ffi_atmosphere_leadville() {
        let mut pa = 0.0;
        let mut da = 0.0;
        let rc =
            unsafe { traumateam_compute_atmosphere_ffi(9800.0, 29.70, 28.0, &mut pa, &mut da) };
        assert_eq!(rc, 0);
        assert!(pa > 9950.0 && pa < 10100.0);
        assert!(da > 13400.0 && da < 13800.0);
    }

    #[test]
    fn test_ffi_mass_balance_h145_nominal_and_mtow_exceeded() {
        let payloads = [
            FfiPayloadItem {
                station_id: FfiStationId::PilotRh as u32,
                _padding: 0,
                weight_kg: 90.0,
            },
            FfiPayloadItem {
                station_id: FfiStationId::CrewAttendantAftLh as u32,
                _padding: 0,
                weight_kg: 85.0,
            },
            FfiPayloadItem {
                station_id: FfiStationId::PrimaryPatientLitter as u32,
                _padding: 0,
                weight_kg: 95.0,
            },
        ];

        let mut res = FfiMassBalanceResult::default();
        let rc = unsafe {
            traumateam_eval_mass_balance_ffi(
                FfiAirframeId::AirbusH145 as u32,
                450.0,
                payloads.as_ptr(),
                payloads.len(),
                &mut res,
            )
        };
        assert_eq!(rc, 0);
        assert_eq!(res.is_overall_compliant, 1);
        assert_eq!(res.limiting_factor_code, FfiLimitingFactorCode::None as u32);
        assert!((res.gross_weight_kg - (2140.0 + 90.0 + 85.0 + 95.0 + 450.0)).abs() < 1e-6);

        // Exceed MTOW (3700 kg) with balanced cabin & cockpit loading
        let heavy_payloads = [
            FfiPayloadItem {
                station_id: FfiStationId::PilotRh as u32,
                _padding: 0,
                weight_kg: 100.0,
            },
            FfiPayloadItem {
                station_id: FfiStationId::CopilotLh as u32,
                _padding: 0,
                weight_kg: 100.0,
            },
            FfiPayloadItem {
                station_id: FfiStationId::CrewAttendantAftLh as u32,
                _padding: 0,
                weight_kg: 100.0,
            },
            FfiPayloadItem {
                station_id: FfiStationId::CrewAttendantAftRh as u32,
                _padding: 0,
                weight_kg: 100.0,
            },
            FfiPayloadItem {
                station_id: FfiStationId::PrimaryPatientLitter as u32,
                _padding: 0,
                weight_kg: 180.0,
            },
            FfiPayloadItem {
                station_id: FfiStationId::SecondaryPatientLitter as u32,
                _padding: 0,
                weight_kg: 150.0,
            },
            FfiPayloadItem {
                station_id: FfiStationId::AftAvionicsCargo as u32,
                _padding: 0,
                weight_kg: 150.0,
            },
            FfiPayloadItem {
                station_id: FfiStationId::MedicalCabinetRack as u32,
                _padding: 0,
                weight_kg: 120.0,
            },
            FfiPayloadItem {
                station_id: FfiStationId::OxygenSystem as u32,
                _padding: 0,
                weight_kg: 50.0,
            },
        ];
        let mut heavy_res = FfiMassBalanceResult::default();
        let rc2 = unsafe {
            traumateam_eval_mass_balance_ffi(
                FfiAirframeId::AirbusH145 as u32,
                694.0,
                heavy_payloads.as_ptr(),
                heavy_payloads.len(),
                &mut heavy_res,
            )
        };
        assert_eq!(rc2, 0);
        assert_eq!(heavy_res.is_overall_compliant, 0);
        assert_eq!(
            heavy_res.limiting_factor_code,
            FfiLimitingFactorCode::MtowExceeded as u32
        );
    }

    #[test]
    fn test_ffi_mission_kinematics_end_to_end() {
        let payloads = [
            FfiPayloadItem {
                station_id: FfiStationId::PilotRh as u32,
                _padding: 0,
                weight_kg: 88.0,
            },
            FfiPayloadItem {
                station_id: FfiStationId::CrewAttendantAftLh as u32,
                _padding: 0,
                weight_kg: 82.0,
            },
            FfiPayloadItem {
                station_id: FfiStationId::PrimaryPatientLitter as u32,
                _padding: 0,
                weight_kg: 90.0,
            },
        ];

        // 1. Nominal KAPA low-altitude mission -> GO
        let nominal_input = FfiWeatherMissionInput {
            airframe_id: FfiAirframeId::AirbusH145 as u32,
            _padding: 0,
            station_elevation_ft: 5885.0,
            altimeter_inhg: 29.95,
            oat_celsius: 15.0,
            wind_dir_deg: 180.0,
            wind_speed_kts: 10.0,
            true_course_deg: 270.0,
            leg_distance_nm: 45.0,
            fuel_weight_kg: 450.0,
        };

        let mut out_go = FfiPerformanceKinematicResult::default();
        let rc = unsafe {
            traumateam_eval_mission_kinematics_ffi(
                &nominal_input,
                payloads.as_ptr(),
                payloads.len(),
                &mut out_go,
            )
        };
        assert_eq!(rc, 0);
        assert_eq!(out_go.is_feasible, 1);
        assert_eq!(out_go.verdict_code, FfiVerdictCode::Go as u32);
        assert!(out_go.oge_margin_kg > 50.0);
        assert!(out_go.reserve_margin_kg > 50.0);

        // 2. High-DA Leadville (9,800 ft, 28C) heavy load -> OGE_HOVER_SCENE_DEFICIT NO-GO
        let high_da_input = FfiWeatherMissionInput {
            airframe_id: FfiAirframeId::AirbusH145 as u32,
            _padding: 0,
            station_elevation_ft: 9800.0,
            altimeter_inhg: 29.70,
            oat_celsius: 28.0,
            wind_dir_deg: 260.0,
            wind_speed_kts: 15.0,
            true_course_deg: 260.0,
            leg_distance_nm: 60.0,
            fuel_weight_kg: 600.0,
        };

        let mut out_nogo = FfiPerformanceKinematicResult::default();
        let rc2 = unsafe {
            traumateam_eval_mission_kinematics_ffi(
                &high_da_input,
                payloads.as_ptr(),
                payloads.len(),
                &mut out_nogo,
            )
        };
        assert_eq!(rc2, 0);
        assert_eq!(out_nogo.is_feasible, 0);
        assert_eq!(out_nogo.verdict_code, FfiVerdictCode::NoGo as u32);
        assert_eq!(
            out_nogo.limiting_factor_code,
            FfiLimitingFactorCode::OgeHoverSceneDeficit as u32
        );
        assert!(out_nogo.oge_margin_kg < 0.0);
    }

    #[test]
    fn test_ffi_c_abi_struct_layout_and_alignment() {
        assert_eq!(std::mem::size_of::<FfiPayloadItem>(), 16);
        assert_eq!(std::mem::align_of::<FfiPayloadItem>(), 8);
        assert_eq!(std::mem::size_of::<FfiWeatherMissionInput>(), 72);
        assert_eq!(std::mem::align_of::<FfiWeatherMissionInput>(), 8);
        assert_eq!(std::mem::size_of::<FfiMassBalanceResult>(), 104);
        assert_eq!(std::mem::align_of::<FfiMassBalanceResult>(), 8);
        assert_eq!(std::mem::size_of::<FfiPerformanceKinematicResult>(), 296);
        assert_eq!(std::mem::align_of::<FfiPerformanceKinematicResult>(), 8);
    }

    #[test]
    fn test_ffi_nan_inf_and_null_pointer_fail_closed() {
        // 1. Null pointer guards
        let mut pa = 0.0;
        assert_eq!(
            unsafe {
                traumateam_compute_atmosphere_ffi(0.0, 29.92, 15.0, &mut pa, std::ptr::null_mut())
            },
            -1
        );
        assert_eq!(
            unsafe {
                traumateam_eval_mass_balance_ffi(0, 300.0, std::ptr::null(), 0, std::ptr::null_mut())
            },
            -1
        );
        let mut out_kin = FfiPerformanceKinematicResult::default();
        assert_eq!(
            unsafe {
                traumateam_eval_mission_kinematics_ffi(
                    std::ptr::null(),
                    std::ptr::null(),
                    0,
                    &mut out_kin,
                )
            },
            -1
        );

        // 2. Wind triangle NaN/Inf/negative wind guards
        for &(tc, tas, wd, ws) in &[
            (f64::NAN, 130.0, 180.0, 10.0),
            (180.0, f64::NAN, 180.0, 10.0),
            (180.0, 130.0, f64::INFINITY, 10.0),
            (180.0, 130.0, 180.0, f64::NEG_INFINITY),
            (180.0, 130.0, 180.0, -5.0),
        ] {
            let (gs, _th, _wca, _hw, _xw, flyable) = solve_wind_triangle_native(tc, tas, wd, ws);
            assert!(!flyable);
            assert_eq!(gs, 0.0);
        }

        // 3. Every float field of FfiWeatherMissionInput rejects NaN/Inf
        let base = FfiWeatherMissionInput {
            airframe_id: 0,
            _padding: 0,
            station_elevation_ft: 5000.0,
            altimeter_inhg: 29.92,
            oat_celsius: 15.0,
            wind_dir_deg: 180.0,
            wind_speed_kts: 10.0,
            true_course_deg: 270.0,
            leg_distance_nm: 30.0,
            fuel_weight_kg: 400.0,
        };

        let mut cases = Vec::new();
        for bad in [f64::NAN, f64::INFINITY, f64::NEG_INFINITY] {
            let mut c1 = base;
            c1.station_elevation_ft = bad;
            cases.push(c1);
            let mut c2 = base;
            c2.altimeter_inhg = bad;
            cases.push(c2);
            let mut c3 = base;
            c3.oat_celsius = bad;
            cases.push(c3);
            let mut c4 = base;
            c4.wind_dir_deg = bad;
            cases.push(c4);
            let mut c5 = base;
            c5.wind_speed_kts = bad;
            cases.push(c5);
            let mut c6 = base;
            c6.true_course_deg = bad;
            cases.push(c6);
            let mut c7 = base;
            c7.leg_distance_nm = bad;
            cases.push(c7);
            let mut c8 = base;
            c8.fuel_weight_kg = bad;
            cases.push(c8);
        }

        for bad_input in cases {
            let mut res = FfiPerformanceKinematicResult::default();
            let rc = unsafe {
                traumateam_eval_mission_kinematics_ffi(
                    &bad_input,
                    std::ptr::null(),
                    0,
                    &mut res,
                )
            };
            assert!(rc < 0, "expected error status for bad input {bad_input:?}, got {rc}");
            assert_eq!(res.is_feasible, 0);
            assert_eq!(res.verdict_code, FfiVerdictCode::NoGo as u32);
        }
    }
}
