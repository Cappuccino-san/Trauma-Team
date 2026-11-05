//! TraumaTeam RFM & Mass-Balance Core Engine in Strict Rust.
//!
//! Provides zero-cost abstractions, compile-time dimensional correctness, and
//! certified Rotorcraft Flight Manual (RFM) performance and weight-and-balance
//! evaluations conforming to FAA Part 135 Subpart L.

pub mod airframes;
pub mod ffi;
pub mod interpolator;
pub mod mass_balance;
pub mod rfm;
pub mod units;

pub use airframes::{get_bell429_rfm_spec, get_h145_rfm_spec};
pub use ffi::{
    traumateam_compute_atmosphere_ffi, traumateam_eval_mass_balance_ffi,
    traumateam_eval_mission_kinematics_ffi, traumateam_ffi_abi_version, FfiAirframeId,
    FfiLimitingFactorCode, FfiMassBalanceResult, FfiPayloadItem, FfiPerformanceKinematicResult,
    FfiStationId, FfiVerdictCode, FfiWeatherMissionInput,
};
pub use interpolator::{CruisePerformanceGrid3D, PerformanceGrid2D};
pub use mass_balance::{compute_mass_and_balance, MassBalanceError, MassBalanceSnapshot};
pub use rfm::{
    CGEnvelopePoint, CGLimits, DispatchVerdict, FuelSystemSpec, LimitingFactor, PayloadItem,
    RFMAirframeSpec, StationArm, StationId, VariableFuelArmPoint,
};
pub use units::{
    gallons_to_jet_a_kg, jet_a_density_kg_per_l, jet_a_kg_to_gallons, liters_to_jet_a_kg, Celsius,
    Feet, KgPerHour, KilogramMillimeters, Kilograms, Knots, Millimeters, FEET_TO_METERS,
    INCHES_TO_MM, JET_A_STD_DENSITY_KG_PER_L, KG_TO_LBS, LBS_TO_KG, METERS_TO_FEET, METERS_TO_NM,
    MM_TO_INCHES, NM_TO_FEET, NM_TO_METERS, SM_TO_FEET, SM_TO_NM, STD_SEA_LEVEL_PRESSURE_HPA,
    STD_SEA_LEVEL_PRESSURE_INHG, US_GAL_TO_LITERS,
};
