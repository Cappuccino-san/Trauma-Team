"""NOAA Aviation Weather Center (AWC) Ingestion & Corridor Weather Evaluation Engine."""

from __future__ import annotations
from dataclasses import dataclass
from typing import List, Optional, Tuple
import numpy as np

from traumateam.models.weather import (
    CorridorWeather,
    METARObservation,
    WindsAloftLayer,
    WeatherHazardAdvisory,
    HazardType,
    HazardSeverity,
)
from traumateam.core.kinematics import WindVector
from traumateam.models.verdict import LimitingFactor, DispatchVerdict
from traumateam.regulations.far135_615 import evaluate_far_135_615_compliance, WeatherComplianceReport

@dataclass(frozen=True)
class CorridorWeatherVerdict:
    is_flyable: bool
    verdict: DispatchVerdict
    limiting_factor: LimitingFactor
    compliance_report: WeatherComplianceReport
    worst_ceiling_ft: float
    worst_vis_sm: float
    active_hazard_summary: Optional[str]
    audit_notes: List[str]

def interpolate_winds_aloft(
    altitude_msl_ft: float,
    winds_profile: List[WindsAloftLayer]
) -> WindVector:
    """Interpolates wind direction and speed at specified altitude from winds aloft profile."""
    if not winds_profile:
        return WindVector(direction_deg=0.0, speed_kts=0.0)

    sorted_layers = sorted(winds_profile, key=lambda l: l.altitude_msl_ft)
    if len(sorted_layers) == 1:
        return WindVector(direction_deg=sorted_layers[0].direction_deg, speed_kts=sorted_layers[0].speed_kts)

    alts = np.array([l.altitude_msl_ft for l in sorted_layers])
    speeds = np.array([l.speed_kts for l in sorted_layers])
    
    # Decompose wind directions into u and v components to prevent 359 -> 010 discontinuity error!
    rads = np.radians([l.direction_deg for l in sorted_layers])
    u_comp = -speeds * np.sin(rads)
    v_comp = -speeds * np.cos(rads)

    # Linear interpolation
    target_u = float(np.interp(altitude_msl_ft, alts, u_comp))
    target_v = float(np.interp(altitude_msl_ft, alts, v_comp))

    interp_speed = float(np.hypot(target_u, target_v))
    interp_dir = (np.degrees(np.arctan2(-target_u, -target_v)) + 360.0) % 360.0

    return WindVector(direction_deg=float(interp_dir), speed_kts=float(interp_speed))

def evaluate_corridor_weather(
    corridor: CorridorWeather,
    is_day: bool,
    is_mountainous: bool,
    is_local: bool,
    nvg_or_htaws_equipped: bool
) -> CorridorWeatherVerdict:
    """Evaluates weather across all corridor reporting stations against FAR 135.615 and severe hazards."""
    audit_notes = []

    # 1. Collect all reporting stations along the flight corridor
    stations = [corridor.departure_metar]
    if corridor.destination_metar:
        stations.append(corridor.destination_metar)
    stations.extend(corridor.intermediate_metars)

    # Find worst (lowest) ceiling and worst (lowest) visibility
    worst_ceil = min(st.ceiling_agl_ft for st in stations)
    worst_vis = min(st.visibility_sm for st in stations)

    for st in stations:
        audit_notes.append(
            f"Station {st.station_id}: Ceiling {st.ceiling_agl_ft:.0f} ft AGL, Vis {st.visibility_sm:.1f} SM, "
            f"Wind {st.wind_dir_deg:03.0f}@{st.wind_speed_kts:.0f}kt, Temp {st.temperature_c:.1f}C, QNH {st.altimeter_inhg:.2f} inHg"
        )

    # 2. FAR 135.615 Minimums Check
    compliance = evaluate_far_135_615_compliance(
        observed_ceiling_ft=worst_ceil,
        observed_vis_sm=worst_vis,
        is_day=is_day,
        is_mountainous=is_mountainous,
        is_local=is_local,
        nvg_or_htaws_equipped=nvg_or_htaws_equipped
    )
    audit_notes.append(compliance.explanation)

    # 3. Severe Weather Hazard check (Convective SIGMETs, Severe Icing, Mountain Obscuration)
    hazard_triggered = False
    hazard_msg = None
    for hazard in corridor.active_hazards:
        if hazard.active and hazard.severity in (HazardSeverity.SEVERE, HazardSeverity.EXTREME):
            hazard_triggered = True
            hazard_msg = f"{hazard.hazard_type.value}: {hazard.summary}"
            audit_notes.append(f"CRITICAL WEATHER HAZARD: {hazard_msg}")
            break

    if hazard_triggered:
        return CorridorWeatherVerdict(
            is_flyable=False,
            verdict=DispatchVerdict.NO_GO,
            limiting_factor=LimitingFactor.SEVERE_WEATHER_HAZARD,
            compliance_report=compliance,
            worst_ceiling_ft=worst_ceil,
            worst_vis_sm=worst_vis,
            active_hazard_summary=hazard_msg,
            audit_notes=audit_notes,
        )

    if not compliance.is_compliant:
        return CorridorWeatherVerdict(
            is_flyable=False,
            verdict=DispatchVerdict.NO_GO,
            limiting_factor=compliance.limiting_factor,
            compliance_report=compliance,
            worst_ceiling_ft=worst_ceil,
            worst_vis_sm=worst_vis,
            active_hazard_summary=None,
            audit_notes=audit_notes,
        )

    return CorridorWeatherVerdict(
        is_flyable=True,
        verdict=DispatchVerdict.GO,
        limiting_factor=LimitingFactor.NONE,
        compliance_report=compliance,
        worst_ceiling_ft=worst_ceil,
        worst_vis_sm=worst_vis,
        active_hazard_summary=None,
        audit_notes=audit_notes,
    )
