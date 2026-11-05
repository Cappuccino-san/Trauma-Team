//! Mass and Balance Calculation & Certified Envelope Compliance Engine in Strict Rust.

use crate::rfm::{LimitingFactor, PayloadItem, RFMAirframeSpec, StationId};
use crate::units::{KilogramMillimeters, Kilograms, Millimeters};
use std::fmt;

/// Errors encountered during mass and balance calculations.
#[derive(Debug, Clone, PartialEq, Eq)]
#[cfg_attr(feature = "serde", derive(serde::Serialize))]
pub enum MassBalanceError {
    StationNotFound(StationId),
    ZeroGrossWeight,
}

impl fmt::Display for MassBalanceError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::StationNotFound(id) => write!(f, "Station ID '{:?}' not found on airframe", id),
            Self::ZeroGrossWeight => write!(f, "Computed gross weight is zero or negative"),
        }
    }
}

impl std::error::Error for MassBalanceError {}

/// Immutable Mass and Balance snapshot evaluated against certified RFM envelope limits.
#[derive(Clone, Debug, PartialEq)]
#[cfg_attr(feature = "serde", derive(serde::Serialize, serde::Deserialize))]
pub struct MassBalanceSnapshot {
    pub gross_weight: Kilograms,
    pub cg_long: Millimeters,
    pub cg_lat: Millimeters,
    pub fwd_limit: Millimeters,
    pub aft_limit: Millimeters,
    pub left_limit: Millimeters,
    pub right_limit: Millimeters,
    pub fwd_margin: Millimeters,
    pub aft_margin: Millimeters,
    pub left_margin: Millimeters,
    pub right_margin: Millimeters,
    pub is_gw_compliant: bool,
    pub is_long_cg_compliant: bool,
    pub is_lat_cg_compliant: bool,
    pub is_overall_compliant: bool,
    pub limiting_factor: LimitingFactor,
    pub detail_message: String,
}

/// Computes the exact Gross Weight, Longitudinal CG, and Lateral CG,
/// checking against certified RFM flight envelope boundaries.
pub fn compute_mass_and_balance(
    airframe: &RFMAirframeSpec,
    fuel: Kilograms,
    active_payloads: &[PayloadItem],
) -> Result<MassBalanceSnapshot, MassBalanceError> {
    // 1. Base empty helicopter mass and moments
    let mut total_gw = airframe.empty_weight;
    let mut m_long: KilogramMillimeters = airframe.empty_weight * airframe.empty_cg_long;
    let mut m_lat: KilogramMillimeters = airframe.empty_weight * airframe.empty_cg_lat;

    // 2. Add active payloads
    for item in active_payloads {
        let station = airframe
            .stations
            .get(&item.station_id)
            .ok_or(MassBalanceError::StationNotFound(item.station_id))?;

        total_gw += item.weight;
        m_long += item.weight * station.arm_long;
        m_lat += item.weight * station.arm_lat;
    }

    // 3. Add fuel
    let (fuel_long_arm, fuel_lat_arm) = airframe.fuel_system.fuel_arm(fuel);
    total_gw += fuel;
    m_long += fuel * fuel_long_arm;
    m_lat += fuel * fuel_lat_arm;

    if total_gw.get().is_nan() || total_gw.get() <= 0.0 {
        return Err(MassBalanceError::ZeroGrossWeight);
    }

    // 4. Center of gravity coordinates
    let cg_long = m_long / total_gw;
    let cg_lat = m_lat / total_gw;

    // 5. Envelope limits and margins
    let limits = airframe.get_cg_limits_at_weight(total_gw);

    let fwd_margin = cg_long - limits.fwd_limit;
    let aft_margin = limits.aft_limit - cg_long;
    let left_margin = cg_lat - limits.left_limit;
    let right_margin = limits.right_limit - cg_lat;

    // 6. Compliance verdicts
    let gw_ok = total_gw >= airframe.min_flight_weight && total_gw <= airframe.mtow;
    let long_ok = fwd_margin.get() >= -1e-4 && aft_margin.get() >= -1e-4;
    let lat_ok = left_margin.get() >= -1e-4 && right_margin.get() >= -1e-4;

    let mut limiting_factor = LimitingFactor::None;
    let mut msg_parts = Vec::new();

    if total_gw > airframe.mtow {
        limiting_factor = LimitingFactor::MtowExceeded;
        let excess = total_gw - airframe.mtow;
        msg_parts.push(format!(
            "Gross weight ({:.1} kg) exceeds MTOW ({:.1} kg) by {:.1} kg.",
            total_gw.get(),
            airframe.mtow.get(),
            excess.get()
        ));
    } else if total_gw < airframe.min_flight_weight {
        limiting_factor = LimitingFactor::MinWeightViolation;
        msg_parts.push(format!(
            "Gross weight ({:.1} kg) below certified minimum ({:.1} kg).",
            total_gw.get(),
            airframe.min_flight_weight.get()
        ));
    }

    if !long_ok {
        if fwd_margin.get() < 0.0 {
            limiting_factor = LimitingFactor::LongitudinalCgForwardExceeded;
            msg_parts.push(format!(
                "Longitudinal CG ({:.1} mm) is {:.1} mm ahead of forward limit ({:.1} mm).",
                cg_long.get(),
                fwd_margin.get().abs(),
                limits.fwd_limit.get()
            ));
        } else {
            limiting_factor = LimitingFactor::LongitudinalCgAftExceeded;
            msg_parts.push(format!(
                "Longitudinal CG ({:.1} mm) is {:.1} mm behind aft limit ({:.1} mm).",
                cg_long.get(),
                aft_margin.get().abs(),
                limits.aft_limit.get()
            ));
        }
    }

    if !lat_ok {
        if left_margin.get() < 0.0 {
            limiting_factor = LimitingFactor::LateralCgLeftExceeded;
            msg_parts.push(format!(
                "Lateral CG ({:.1} mm) is {:.1} mm left of boundary ({:.1} mm).",
                cg_lat.get(),
                left_margin.get().abs(),
                limits.left_limit.get()
            ));
        } else {
            limiting_factor = LimitingFactor::LateralCgRightExceeded;
            msg_parts.push(format!(
                "Lateral CG ({:.1} mm) is {:.1} mm right of boundary ({:.1} mm).",
                cg_lat.get(),
                right_margin.get().abs(),
                limits.right_limit.get()
            ));
        }
    }

    let detail_message = if msg_parts.is_empty() {
        format!(
            "Weight and Balance fully compliant: GW={:.1} kg, CG_long={:.1} mm, CG_lat={:.1} mm.",
            total_gw.get(),
            cg_long.get(),
            cg_lat.get()
        )
    } else {
        msg_parts.join(" | ")
    };

    let is_overall_compliant = gw_ok && long_ok && lat_ok;

    Ok(MassBalanceSnapshot {
        gross_weight: total_gw,
        cg_long,
        cg_lat,
        fwd_limit: limits.fwd_limit,
        aft_limit: limits.aft_limit,
        left_limit: limits.left_limit,
        right_limit: limits.right_limit,
        fwd_margin,
        aft_margin,
        left_margin,
        right_margin,
        is_gw_compliant: gw_ok,
        is_long_cg_compliant: long_ok,
        is_lat_cg_compliant: lat_ok,
        is_overall_compliant,
        limiting_factor,
        detail_message,
    })
}
