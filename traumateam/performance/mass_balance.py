"""Mass and Balance Calculation & Flight Envelope Verification Engine.

Calculates continuous longitudinal and lateral Center of Gravity (CG) across variable
fuel depletion curves, modular medical gear, and crew/patient reconfigurations.
"""

from __future__ import annotations
from dataclasses import dataclass
from typing import Dict, List, Optional, Tuple
import numpy as np

from traumateam.models.rfm import (
    RFMAirframeSpec,
    StationArm,
    FuelSystemSpec,
    CGEnvelopePoint,
)
from traumateam.models.mission import PayloadItem
from traumateam.models.verdict import LimitingFactor

@dataclass(frozen=True)
class MassBalanceSnapshot:
    gross_weight_kg: float
    cg_long_mm: float
    cg_lat_mm: float
    fwd_limit_mm: float
    aft_limit_mm: float
    left_limit_mm: float
    right_limit_mm: float
    fwd_margin_mm: float
    aft_margin_mm: float
    left_margin_mm: float
    right_margin_mm: float
    is_gw_compliant: bool
    is_long_cg_compliant: bool
    is_lat_cg_compliant: bool
    is_overall_compliant: bool
    limiting_factor: LimitingFactor
    detail_message: str

def get_fuel_arm(fuel_kg: float, fuel_system: FuelSystemSpec) -> Tuple[float, float]:
    """Calculates longitudinal and lateral arms for fuel based on current quantity.
    
    If the airframe specifies a non-linear schedule (e.g. forward/aft tank depletion sequence),
    interpolates along the schedule. Otherwise returns default arm.
    """
    if not fuel_system.tank_arms:
        return fuel_system.default_arm_long_mm, fuel_system.default_arm_lat_mm

    # Sort tank schedule by fuel quantity
    arms = sorted(fuel_system.tank_arms, key=lambda p: p.fuel_kg)
    fuels = [p.fuel_kg for p in arms]
    long_arms = [p.arm_long_mm for p in arms]
    lat_arms = [p.arm_lat_mm for p in arms]

    # Interpolate piecewise
    clamped_fuel = max(0.0, min(fuel_kg, fuel_system.usable_fuel_capacity_kg))
    arm_long = float(np.interp(clamped_fuel, fuels, long_arms))
    arm_lat = float(np.interp(clamped_fuel, fuels, lat_arms))
    return arm_long, arm_lat

def get_cg_limits_at_weight(
    weight_kg: float,
    envelope: List[CGEnvelopePoint]
) -> Tuple[float, float, float, float]:
    """Interpolates CG envelope boundary limits at a specific gross weight.
    
    Returns (fwd_limit_mm, aft_limit_mm, left_limit_mm, right_limit_mm).
    """
    sorted_env = sorted(envelope, key=lambda p: p.gross_weight_kg)
    weights = [p.gross_weight_kg for p in sorted_env]
    fwd_limits = [p.cg_long_fwd_mm for p in sorted_env]
    aft_limits = [p.cg_long_aft_mm for p in sorted_env]
    left_limits = [p.cg_lat_left_mm for p in sorted_env]
    right_limits = [p.cg_lat_right_mm for p in sorted_env]

    # Linear interpolation along envelope edges
    fwd = float(np.interp(weight_kg, weights, fwd_limits))
    aft = float(np.interp(weight_kg, weights, aft_limits))
    left = float(np.interp(weight_kg, weights, left_limits))
    right = float(np.interp(weight_kg, weights, right_limits))
    return fwd, aft, left, right

def compute_mass_and_balance(
    airframe: RFMAirframeSpec,
    fuel_kg: float,
    active_payloads: List[PayloadItem]
) -> MassBalanceSnapshot:
    """Computes exact Gross Weight, Longitudinal CG, and Lateral CG,
    verifying compliance against certified RFM envelope limits.
    """
    # 1. Base empty helicopter mass and moments
    w_empty = airframe.empty_weight_kg
    m_long = w_empty * airframe.empty_cg_long_mm
    m_lat = w_empty * airframe.empty_cg_lat_mm
    total_gw = w_empty

    # 2. Add active payloads
    for item in active_payloads:
        station = airframe.stations.get(item.station_id)
        if station is None:
            raise KeyError(f"Station ID '{item.station_id}' for payload '{item.name}' not found on airframe {airframe.model_name}")
        
        weight = item.weight_kg
        total_gw += weight
        m_long += weight * station.arm_long_mm
        m_lat += weight * station.arm_lat_mm

    # 3. Add fuel
    fuel_long_arm, fuel_lat_arm = get_fuel_arm(fuel_kg, airframe.fuel_system)
    total_gw += fuel_kg
    m_long += fuel_kg * fuel_long_arm
    m_lat += fuel_kg * fuel_lat_arm

    # 4. Center of Gravity coordinates
    cg_long = m_long / total_gw
    cg_lat = m_lat / total_gw

    # 5. Envelope limits and margin check
    fwd_lim, aft_lim, left_lim, right_lim = get_cg_limits_at_weight(total_gw, airframe.cg_envelope)

    fwd_margin = cg_long - fwd_lim      # positive means aft of forward limit (compliant)
    aft_margin = aft_lim - cg_long      # positive means forward of aft limit (compliant)
    left_margin = cg_lat - left_lim     # positive means right of left limit (compliant, since left is negative)
    right_margin = right_lim - cg_lat   # positive means left of right limit (compliant)

    # 6. Compliance verdicts
    gw_ok = (airframe.min_flight_weight_kg <= total_gw <= airframe.mtow_kg)
    long_ok = (fwd_margin >= -1e-4 and aft_margin >= -1e-4)
    lat_ok = (left_margin >= -1e-4 and right_margin >= -1e-4)

    limiting_factor = LimitingFactor.NONE
    msg_parts = []

    if total_gw > airframe.mtow_kg:
        limiting_factor = LimitingFactor.MTOW_EXCEEDED
        excess = total_gw - airframe.mtow_kg
        msg_parts.append(f"Gross weight ({total_gw:.1f} kg) exceeds MTOW ({airframe.mtow_kg:.1f} kg) by {excess:.1f} kg.")
    elif total_gw < airframe.min_flight_weight_kg:
        limiting_factor = LimitingFactor.MIN_WEIGHT_VIOLATION
        msg_parts.append(f"Gross weight ({total_gw:.1f} kg) below certified minimum ({airframe.min_flight_weight_kg:.1f} kg).")
    
    if not long_ok:
        if fwd_margin < 0:
            limiting_factor = LimitingFactor.LONGITUDINAL_CG_FORWARD_EXCEEDED
            msg_parts.append(f"Longitudinal CG ({cg_long:.1f} mm) is {abs(fwd_margin):.1f} mm ahead of forward limit ({fwd_lim:.1f} mm).")
        else:
            limiting_factor = LimitingFactor.LONGITUDINAL_CG_AFT_EXCEEDED
            msg_parts.append(f"Longitudinal CG ({cg_long:.1f} mm) is {abs(aft_margin):.1f} mm behind aft limit ({aft_lim:.1f} mm).")

    if not lat_ok:
        if left_margin < 0:
            limiting_factor = LimitingFactor.LATERAL_CG_LEFT_EXCEEDED
            msg_parts.append(f"Lateral CG ({cg_lat:.1f} mm) is {abs(left_margin):.1f} mm left of boundary ({left_lim:.1f} mm).")
        else:
            limiting_factor = LimitingFactor.LATERAL_CG_RIGHT_EXCEEDED
            msg_parts.append(f"Lateral CG ({cg_lat:.1f} mm) is {abs(right_margin):.1f} mm right of boundary ({right_lim:.1f} mm).")

    if not msg_parts:
        detail_msg = f"Weight and Balance fully compliant: GW={total_gw:.1f} kg, CG_long={cg_long:.1f} mm, CG_lat={cg_lat:.1f} mm."
    else:
        detail_msg = " | ".join(msg_parts)

    is_overall_ok = (gw_ok and long_ok and lat_ok)

    return MassBalanceSnapshot(
        gross_weight_kg=total_gw,
        cg_long_mm=cg_long,
        cg_lat_mm=cg_lat,
        fwd_limit_mm=fwd_lim,
        aft_limit_mm=aft_lim,
        left_limit_mm=left_lim,
        right_limit_mm=right_lim,
        fwd_margin_mm=fwd_margin,
        aft_margin_mm=aft_margin,
        left_margin_mm=left_margin,
        right_margin_mm=right_margin,
        is_gw_compliant=gw_ok,
        is_long_cg_compliant=long_ok,
        is_lat_cg_compliant=lat_ok,
        is_overall_compliant=is_overall_ok,
        limiting_factor=limiting_factor,
        detail_message=detail_msg,
    )
