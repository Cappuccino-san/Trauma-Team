//! Strict Rust Structs for Rotorcraft Flight Manual (RFM) Performance Tables.

use crate::interpolator::{CruisePerformanceGrid3D, PerformanceGrid2D};
use crate::units::{Celsius, Feet, KgPerHour, Kilograms, Knots, Millimeters};
use std::collections::HashMap;

/// Dispatch verdict for HEMS missions under FAA Part 135.
#[derive(Clone, Copy, Debug, PartialEq, Eq, Hash)]
#[cfg_attr(feature = "serde", derive(serde::Serialize, serde::Deserialize))]
pub enum DispatchVerdict {
    #[cfg_attr(feature = "serde", serde(rename = "GO"))]
    Go,
    #[cfg_attr(feature = "serde", serde(rename = "CAUTION"))]
    Caution,
    #[cfg_attr(feature = "serde", serde(rename = "NO-GO"))]
    NoGo,
}

impl DispatchVerdict {
    pub const fn as_str(&self) -> &'static str {
        match self {
            Self::Go => "GO",
            Self::Caution => "CAUTION",
            Self::NoGo => "NO-GO",
        }
    }
}

impl std::fmt::Display for DispatchVerdict {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "{}", self.as_str())
    }
}

/// Limiting factors for dispatch feasibility and operational envelopes.
#[derive(Clone, Copy, Debug, PartialEq, Eq, Hash)]
#[cfg_attr(feature = "serde", derive(serde::Serialize, serde::Deserialize))]
#[cfg_attr(feature = "serde", serde(rename_all = "SCREAMING_SNAKE_CASE"))]
pub enum LimitingFactor {
    None,
    MtowExceeded,
    MinWeightViolation,
    LongitudinalCgForwardExceeded,
    LongitudinalCgAftExceeded,
    LateralCgLeftExceeded,
    LateralCgRightExceeded,
    OgeHoverSceneDeficit,
    IgeHoverDeficit,
    OeiCatAHelipadDeficit,
    Far135_615CeilingBelowMinimums,
    Far135_615VisibilityBelowMinimums,
    Far135_209VfrReserveViolation,
    IfrReserveViolation,
    SevereWeatherHazard,
    TerrainObstacleClearanceViolation,
    UnflyableWind,
}

impl LimitingFactor {
    pub const fn as_str(&self) -> &'static str {
        match self {
            Self::None => "NONE",
            Self::MtowExceeded => "MTOW_EXCEEDED",
            Self::MinWeightViolation => "MIN_WEIGHT_VIOLATION",
            Self::LongitudinalCgForwardExceeded => "LONGITUDINAL_CG_FORWARD_EXCEEDED",
            Self::LongitudinalCgAftExceeded => "LONGITUDINAL_CG_AFT_EXCEEDED",
            Self::LateralCgLeftExceeded => "LATERAL_CG_LEFT_EXCEEDED",
            Self::LateralCgRightExceeded => "LATERAL_CG_RIGHT_EXCEEDED",
            Self::OgeHoverSceneDeficit => "OGE_HOVER_SCENE_DEFICIT",
            Self::IgeHoverDeficit => "IGE_HOVER_DEFICIT",
            Self::OeiCatAHelipadDeficit => "OEI_CAT_A_HELIPAD_DEFICIT",
            Self::Far135_615CeilingBelowMinimums => "FAR_135_615_CEILING_BELOW_MINIMUMS",
            Self::Far135_615VisibilityBelowMinimums => "FAR_135_615_VISIBILITY_BELOW_MINIMUMS",
            Self::Far135_209VfrReserveViolation => "FAR_135_209_VFR_RESERVE_VIOLATION",
            Self::IfrReserveViolation => "IFR_RESERVE_VIOLATION",
            Self::SevereWeatherHazard => "SEVERE_WEATHER_HAZARD",
            Self::TerrainObstacleClearanceViolation => "TERRAIN_OBSTACLE_CLEARANCE_VIOLATION",
            Self::UnflyableWind => "UNFLYABLE_WIND",
        }
    }
}

impl std::fmt::Display for LimitingFactor {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "{}", self.as_str())
    }
}

/// Payload item assigned to a specific airframe station.
#[derive(Clone, Debug, PartialEq)]
#[cfg_attr(feature = "serde", derive(serde::Serialize))]
pub struct PayloadItem {
    pub id: &'static str,
    pub name: &'static str,
    pub station_id: StationId,
    pub weight: Kilograms,
}

impl PayloadItem {
    pub const fn new(
        id: &'static str,
        name: &'static str,
        station_id: StationId,
        weight: Kilograms,
    ) -> Self {
        Self {
            id,
            name,
            station_id,
            weight,
        }
    }
}

/// Unique station identifier for mass and balance placement.
#[derive(Clone, Copy, Debug, PartialEq, Eq, Hash)]
#[cfg_attr(feature = "serde", derive(serde::Serialize))]
#[cfg_attr(feature = "serde", serde(rename_all = "SCREAMING_SNAKE_CASE"))]
pub enum StationId {
    PilotRh,
    CopilotLh,
    CrewAttendantAftLh,
    CrewAttendantAftRh,
    CrewAttendantLh,
    CrewAttendantRh,
    PrimaryPatientLitter,
    SecondaryPatientLitter,
    MedicalCabinetRack,
    MedicalEquipmentRack,
    OxygenSystem,
    AftAvionicsCargo,
    AftBaggage,
    RescueHoistRh,
    Custom(&'static str),
}

impl StationId {
    pub const fn as_str(&self) -> &'static str {
        match self {
            Self::PilotRh => "PILOT_RH",
            Self::CopilotLh => "COPILOT_LH",
            Self::CrewAttendantAftLh => "CREW_ATTENDANT_AFT_LH",
            Self::CrewAttendantAftRh => "CREW_ATTENDANT_AFT_RH",
            Self::CrewAttendantLh => "CREW_ATTENDANT_LH",
            Self::CrewAttendantRh => "CREW_ATTENDANT_RH",
            Self::PrimaryPatientLitter => "PRIMARY_PATIENT_LITTER",
            Self::SecondaryPatientLitter => "SECONDARY_PATIENT_LITTER",
            Self::MedicalCabinetRack => "MEDICAL_CABINET_RACK",
            Self::MedicalEquipmentRack => "MEDICAL_EQUIPMENT_RACK",
            Self::OxygenSystem => "OXYGEN_SYSTEM",
            Self::AftAvionicsCargo => "AFT_AVIONICS_CARGO",
            Self::AftBaggage => "AFT_BAGGAGE",
            Self::RescueHoistRh => "RESCUE_HOIST_RH",
            Self::Custom(s) => s,
        }
    }

    pub fn from_known_str(s: &str) -> Option<Self> {
        match s {
            "PILOT_RH" => Some(Self::PilotRh),
            "COPILOT_LH" => Some(Self::CopilotLh),
            "CREW_ATTENDANT_AFT_LH" => Some(Self::CrewAttendantAftLh),
            "CREW_ATTENDANT_AFT_RH" => Some(Self::CrewAttendantAftRh),
            "CREW_ATTENDANT_LH" => Some(Self::CrewAttendantLh),
            "CREW_ATTENDANT_RH" => Some(Self::CrewAttendantRh),
            "PRIMARY_PATIENT_LITTER" => Some(Self::PrimaryPatientLitter),
            "SECONDARY_PATIENT_LITTER" => Some(Self::SecondaryPatientLitter),
            "MEDICAL_CABINET_RACK" => Some(Self::MedicalCabinetRack),
            "MEDICAL_EQUIPMENT_RACK" => Some(Self::MedicalEquipmentRack),
            "OXYGEN_SYSTEM" => Some(Self::OxygenSystem),
            "AFT_AVIONICS_CARGO" => Some(Self::AftAvionicsCargo),
            "AFT_BAGGAGE" => Some(Self::AftBaggage),
            "RESCUE_HOIST_RH" => Some(Self::RescueHoistRh),
            _ => None,
        }
    }
}

impl std::fmt::Display for StationId {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "{}", self.as_str())
    }
}

/// Fixed station arm coordinates on the airframe.
#[derive(Clone, Debug, PartialEq)]
#[cfg_attr(feature = "serde", derive(serde::Serialize))]
pub struct StationArm {
    pub id: StationId,
    pub name: &'static str,
    pub arm_long: Millimeters,
    pub arm_lat: Millimeters,
    pub max_capacity: Kilograms,
    pub is_fuel: bool,
}

impl StationArm {
    pub const fn new(
        id: StationId,
        name: &'static str,
        arm_long: Millimeters,
        arm_lat: Millimeters,
        max_capacity: Kilograms,
        is_fuel: bool,
    ) -> Self {
        Self {
            id,
            name,
            arm_long,
            arm_lat,
            max_capacity,
            is_fuel,
        }
    }
}

/// Point on the non-linear fuel tank depletion schedule.
#[derive(Clone, Copy, Debug, PartialEq)]
#[cfg_attr(feature = "serde", derive(serde::Serialize, serde::Deserialize))]
pub struct VariableFuelArmPoint {
    pub fuel_mass: Kilograms,
    pub arm_long: Millimeters,
    pub arm_lat: Millimeters,
}

/// Aircraft fuel system specification.
#[derive(Clone, Debug, PartialEq)]
#[cfg_attr(feature = "serde", derive(serde::Serialize, serde::Deserialize))]
pub struct FuelSystemSpec {
    pub usable_fuel_capacity: Kilograms,
    pub unusable_fuel: Kilograms,
    pub tank_arms: Vec<VariableFuelArmPoint>,
    pub default_arm_long: Millimeters,
    pub default_arm_lat: Millimeters,
}

impl FuelSystemSpec {
    /// Computes longitudinal and lateral arms for the current fuel quantity.
    /// Interpolates along the fuel depletion schedule with zero heap allocations.
    pub fn fuel_arm(&self, fuel: Kilograms) -> (Millimeters, Millimeters) {
        if self.tank_arms.is_empty() {
            return (self.default_arm_long, self.default_arm_lat);
        }
        let n = self.tank_arms.len();
        if n == 1 {
            return (self.tank_arms[0].arm_long, self.tank_arms[0].arm_lat);
        }

        let clamped = fuel.get().clamp(0.0, self.usable_fuel_capacity.get());

        if clamped.is_nan() || clamped <= self.tank_arms[0].fuel_mass.get() {
            return (self.tank_arms[0].arm_long, self.tank_arms[0].arm_lat);
        }
        if clamped >= self.tank_arms[n - 1].fuel_mass.get() {
            return (
                self.tank_arms[n - 1].arm_long,
                self.tank_arms[n - 1].arm_lat,
            );
        }

        let idx = self
            .tank_arms
            .partition_point(|p| p.fuel_mass.get() <= clamped);
        let i0 = idx.saturating_sub(1);
        let i1 = idx.min(n - 1);
        let p0 = &self.tank_arms[i0];
        let p1 = &self.tank_arms[i1];

        let denom = p1.fuel_mass.get() - p0.fuel_mass.get();
        let t = if denom <= 0.0 {
            0.0
        } else {
            (clamped - p0.fuel_mass.get()) / denom
        };

        let long_arm = p0.arm_long.get() + t * (p1.arm_long.get() - p0.arm_long.get());
        let lat_arm = p0.arm_lat.get() + t * (p1.arm_lat.get() - p0.arm_lat.get());

        (Millimeters::new(long_arm), Millimeters::new(lat_arm))
    }
}

/// CG Envelope boundary definition at a specific gross weight.
#[derive(Clone, Copy, Debug, PartialEq)]
#[cfg_attr(feature = "serde", derive(serde::Serialize, serde::Deserialize))]
pub struct CGEnvelopePoint {
    pub gross_weight: Kilograms,
    pub cg_long_fwd: Millimeters,
    pub cg_long_aft: Millimeters,
    pub cg_lat_left: Millimeters,
    pub cg_lat_right: Millimeters,
}

/// CG limits interpolated at a specific gross weight.
#[derive(Clone, Copy, Debug, PartialEq)]
#[cfg_attr(feature = "serde", derive(serde::Serialize, serde::Deserialize))]
pub struct CGLimits {
    pub fwd_limit: Millimeters,
    pub aft_limit: Millimeters,
    pub left_limit: Millimeters,
    pub right_limit: Millimeters,
}

/// Complete certified Rotorcraft Flight Manual specification.
#[derive(Clone, Debug, PartialEq)]
#[cfg_attr(feature = "serde", derive(serde::Serialize))]
pub struct RFMAirframeSpec {
    pub model_name: &'static str,
    pub oem: &'static str,
    pub certification_basis: &'static str,
    pub empty_weight: Kilograms,
    pub empty_cg_long: Millimeters,
    pub empty_cg_lat: Millimeters,
    pub mtow: Kilograms,
    pub min_flight_weight: Kilograms,
    pub fuel_system: FuelSystemSpec,
    pub stations: HashMap<StationId, StationArm>,
    pub cg_envelope: Vec<CGEnvelopePoint>,
    pub ige_hover_grid: PerformanceGrid2D,
    pub oge_hover_grid: PerformanceGrid2D,
    pub oei_cat_a_grid: PerformanceGrid2D,
    pub cruise_performance: CruisePerformanceGrid3D,
    pub startup_fuel: Kilograms,
    pub hover_fuel_flow_factor: f64,
    pub loiter_fuel_flow_factor: f64,
}

impl RFMAirframeSpec {
    /// Evaluates the CG envelope boundaries at a target gross weight.
    pub fn get_cg_limits_at_weight(&self, weight: Kilograms) -> CGLimits {
        let n = self.cg_envelope.len();
        if n == 0 {
            return CGLimits {
                fwd_limit: Millimeters::new(0.0),
                aft_limit: Millimeters::new(0.0),
                left_limit: Millimeters::new(0.0),
                right_limit: Millimeters::new(0.0),
            };
        }
        if n == 1 {
            let p = &self.cg_envelope[0];
            return CGLimits {
                fwd_limit: p.cg_long_fwd,
                aft_limit: p.cg_long_aft,
                left_limit: p.cg_lat_left,
                right_limit: p.cg_lat_right,
            };
        }

        let w_val = weight.get();
        if w_val.is_nan() || w_val <= self.cg_envelope[0].gross_weight.get() {
            let p = &self.cg_envelope[0];
            return CGLimits {
                fwd_limit: p.cg_long_fwd,
                aft_limit: p.cg_long_aft,
                left_limit: p.cg_lat_left,
                right_limit: p.cg_lat_right,
            };
        }
        if w_val >= self.cg_envelope[n - 1].gross_weight.get() {
            let p = &self.cg_envelope[n - 1];
            return CGLimits {
                fwd_limit: p.cg_long_fwd,
                aft_limit: p.cg_long_aft,
                left_limit: p.cg_lat_left,
                right_limit: p.cg_lat_right,
            };
        }

        let idx = self
            .cg_envelope
            .partition_point(|p| p.gross_weight.get() <= w_val);
        let i0 = idx.saturating_sub(1);
        let i1 = idx.min(n - 1);
        let p0 = &self.cg_envelope[i0];
        let p1 = &self.cg_envelope[i1];

        let denom = p1.gross_weight.get() - p0.gross_weight.get();
        let t = if denom <= 0.0 {
            0.0
        } else {
            (w_val - p0.gross_weight.get()) / denom
        };

        CGLimits {
            fwd_limit: Millimeters::new(
                p0.cg_long_fwd.get() + t * (p1.cg_long_fwd.get() - p0.cg_long_fwd.get()),
            ),
            aft_limit: Millimeters::new(
                p0.cg_long_aft.get() + t * (p1.cg_long_aft.get() - p0.cg_long_aft.get()),
            ),
            left_limit: Millimeters::new(
                p0.cg_lat_left.get() + t * (p1.cg_lat_left.get() - p0.cg_lat_left.get()),
            ),
            right_limit: Millimeters::new(
                p0.cg_lat_right.get() + t * (p1.cg_lat_right.get() - p0.cg_lat_right.get()),
            ),
        }
    }

    /// Evaluates Max In-Ground-Effect (IGE) hover weight.
    #[inline]
    pub fn get_max_ige_hover_weight(&self, pa: Feet, oat: Celsius) -> Kilograms {
        let val = self.ige_hover_grid.interpolate(pa, oat);
        Kilograms::new(val.get().clamp(0.0, self.mtow.get()))
    }

    /// Evaluates Max Out-of-Ground-Effect (OGE) hover weight.
    #[inline]
    pub fn get_max_oge_hover_weight(&self, pa: Feet, oat: Celsius) -> Kilograms {
        let val = self.oge_hover_grid.interpolate(pa, oat);
        Kilograms::new(val.get().clamp(0.0, self.mtow.get()))
    }

    /// Evaluates One-Engine-Inoperative (OEI) Category A helipad max allowable gross weight.
    #[inline]
    pub fn get_max_oei_cat_a_weight(&self, pa: Feet, oat: Celsius) -> Kilograms {
        let val = self.oei_cat_a_grid.interpolate(pa, oat);
        Kilograms::new(val.get().clamp(0.0, self.mtow.get()))
    }

    /// Evaluates cruise True Airspeed in knots.
    #[inline]
    pub fn get_cruise_tas(&self, pa: Feet, oat: Celsius, gw: Kilograms) -> Knots {
        let gw_raw = gw.get();
        let gw_clamped = if gw_raw.is_nan() || gw_raw < self.min_flight_weight.get() {
            self.min_flight_weight.get()
        } else if gw_raw > self.mtow.get() {
            self.mtow.get()
        } else {
            gw_raw
        };
        self.cruise_performance
            .interpolate_tas(pa, oat, Kilograms::new(gw_clamped))
    }

    /// Evaluates cruise fuel flow rate in kg/hr.
    #[inline]
    pub fn get_cruise_fuel_flow(&self, pa: Feet, oat: Celsius, gw: Kilograms) -> KgPerHour {
        let gw_raw = gw.get();
        let gw_clamped = if gw_raw.is_nan() || gw_raw < self.min_flight_weight.get() {
            self.min_flight_weight.get()
        } else if gw_raw > self.mtow.get() {
            self.mtow.get()
        } else {
            gw_raw
        };
        self.cruise_performance
            .interpolate_fuel_flow(pa, oat, Kilograms::new(gw_clamped))
    }

    /// Evaluates OGE hover fuel flow rate in kg/hr.
    #[inline]
    pub fn get_hover_oge_fuel_flow(&self, pa: Feet, oat: Celsius, gw: Kilograms) -> KgPerHour {
        self.get_cruise_fuel_flow(pa, oat, gw) * self.hover_fuel_flow_factor
    }

    /// Evaluates loiter / holding fuel flow rate in kg/hr.
    #[inline]
    pub fn get_loiter_fuel_flow(&self, pa: Feet, oat: Celsius, gw: Kilograms) -> KgPerHour {
        self.get_cruise_fuel_flow(pa, oat, gw) * self.loiter_fuel_flow_factor
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_station_id_conversions() {
        assert_eq!(StationId::PilotRh.as_str(), "PILOT_RH");
        assert_eq!(StationId::CopilotLh.as_str(), "COPILOT_LH");
        assert_eq!(StationId::RescueHoistRh.as_str(), "RESCUE_HOIST_RH");
        assert_eq!(
            StationId::from_known_str("PILOT_RH"),
            Some(StationId::PilotRh)
        );
        assert_eq!(
            StationId::from_known_str("COPILOT_LH"),
            Some(StationId::CopilotLh)
        );
        assert_eq!(StationId::from_known_str("UNKNOWN_ID"), None);
        assert_eq!(format!("{}", StationId::PilotRh), "PILOT_RH");
    }

    #[test]
    fn test_limiting_factor_display() {
        assert_eq!(LimitingFactor::None.as_str(), "NONE");
        assert_eq!(LimitingFactor::MtowExceeded.as_str(), "MTOW_EXCEEDED");
        assert_eq!(format!("{}", LimitingFactor::MtowExceeded), "MTOW_EXCEEDED");
        assert_eq!(
            LimitingFactor::Far135_615CeilingBelowMinimums.as_str(),
            "FAR_135_615_CEILING_BELOW_MINIMUMS"
        );
    }

    #[test]
    fn test_dispatch_verdict() {
        assert_eq!(DispatchVerdict::Go.as_str(), "GO");
        assert_eq!(DispatchVerdict::Caution.as_str(), "CAUTION");
        assert_eq!(DispatchVerdict::NoGo.as_str(), "NO-GO");
        assert_eq!(format!("{}", DispatchVerdict::Go), "GO");
    }

    #[test]
    fn test_fuel_arm_piecewise_interpolation() {
        let fuel_arms = vec![
            VariableFuelArmPoint {
                fuel_mass: Kilograms::new(0.0),
                arm_long: Millimeters::new(3900.0),
                arm_lat: Millimeters::new(0.0),
            },
            VariableFuelArmPoint {
                fuel_mass: Kilograms::new(200.0),
                arm_long: Millimeters::new(3940.0),
                arm_lat: Millimeters::new(10.0),
            },
        ];

        let fuel_spec = FuelSystemSpec {
            usable_fuel_capacity: Kilograms::new(500.0),
            unusable_fuel: Kilograms::new(10.0),
            tank_arms: fuel_arms,
            default_arm_long: Millimeters::new(3920.0),
            default_arm_lat: Millimeters::new(0.0),
        };

        // At 0 kg:
        let (arm_l, arm_w) = fuel_spec.fuel_arm(Kilograms::new(0.0));
        assert_eq!(arm_l.get(), 3900.0);
        assert_eq!(arm_w.get(), 0.0);

        // At 100 kg (midpoint):
        let (arm_l, arm_w) = fuel_spec.fuel_arm(Kilograms::new(100.0));
        assert_eq!(arm_l.get(), 3920.0);
        assert_eq!(arm_w.get(), 5.0);

        // Above max table (clamped to last table point at 200 kg):
        let (arm_l, arm_w) = fuel_spec.fuel_arm(Kilograms::new(300.0));
        assert_eq!(arm_l.get(), 3940.0);
        assert_eq!(arm_w.get(), 10.0);

        // Negative fuel clamped to 0:
        let (arm_l, arm_w) = fuel_spec.fuel_arm(Kilograms::new(-50.0));
        assert_eq!(arm_l.get(), 3900.0);
        assert_eq!(arm_w.get(), 0.0);
    }

    #[test]
    fn test_fuel_arm_nan_and_single_point_safety() {
        let fuel_arms = vec![
            VariableFuelArmPoint {
                fuel_mass: Kilograms::new(0.0),
                arm_long: Millimeters::new(3900.0),
                arm_lat: Millimeters::new(0.0),
            },
            VariableFuelArmPoint {
                fuel_mass: Kilograms::new(200.0),
                arm_long: Millimeters::new(3940.0),
                arm_lat: Millimeters::new(10.0),
            },
        ];

        let fuel_spec = FuelSystemSpec {
            usable_fuel_capacity: Kilograms::new(500.0),
            unusable_fuel: Kilograms::new(10.0),
            tank_arms: fuel_arms,
            default_arm_long: Millimeters::new(3920.0),
            default_arm_lat: Millimeters::new(0.0),
        };

        // NaN fuel should not panic and must return lower bound
        let (arm_l, arm_w) = fuel_spec.fuel_arm(Kilograms::new(f64::NAN));
        assert_eq!(arm_l.get(), 3900.0);
        assert_eq!(arm_w.get(), 0.0);

        // Single-point fuel system
        let single_spec = FuelSystemSpec {
            usable_fuel_capacity: Kilograms::new(500.0),
            unusable_fuel: Kilograms::new(10.0),
            tank_arms: vec![VariableFuelArmPoint {
                fuel_mass: Kilograms::new(100.0),
                arm_long: Millimeters::new(3910.0),
                arm_lat: Millimeters::new(2.0),
            }],
            default_arm_long: Millimeters::new(3920.0),
            default_arm_lat: Millimeters::new(0.0),
        };
        let (s_long, s_lat) = single_spec.fuel_arm(Kilograms::new(f64::NAN));
        assert_eq!(s_long.get(), 3910.0);
        assert_eq!(s_lat.get(), 2.0);
    }

    #[test]
    fn test_cg_limits_nan_safety() {
        let h145 = crate::airframes::get_h145_rfm_spec();

        // NaN weight should not panic and must return lower envelope point
        let limits = h145.get_cg_limits_at_weight(Kilograms::new(f64::NAN));
        let expected = &h145.cg_envelope[0];
        assert_eq!(limits.fwd_limit, expected.cg_long_fwd);
        assert_eq!(limits.aft_limit, expected.cg_long_aft);
        assert_eq!(limits.left_limit, expected.cg_lat_left);
        assert_eq!(limits.right_limit, expected.cg_lat_right);
    }

    #[test]
    fn test_cruise_nan_gw_safety() {
        let h145 = crate::airframes::get_h145_rfm_spec();

        let tas = h145.get_cruise_tas(Feet::new(0.0), Celsius::new(15.0), Kilograms::new(f64::NAN));
        let tas_min =
            h145.get_cruise_tas(Feet::new(0.0), Celsius::new(15.0), h145.min_flight_weight);
        assert_eq!(tas, tas_min);

        let ff =
            h145.get_cruise_fuel_flow(Feet::new(0.0), Celsius::new(15.0), Kilograms::new(f64::NAN));
        let ff_min =
            h145.get_cruise_fuel_flow(Feet::new(0.0), Celsius::new(15.0), h145.min_flight_weight);
        assert_eq!(ff, ff_min);
    }
}
