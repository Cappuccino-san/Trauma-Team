"""Dispatch Verdict & Limiting Factors for HEMS Operations under FAA Part 135."""

from __future__ import annotations
from enum import Enum
from pydantic import BaseModel, Field
from typing import List, Optional, Dict, Any

class DispatchVerdict(str, Enum):
    GO = "GO"
    CAUTION = "CAUTION"
    NO_GO = "NO-GO"

class LimitingFactor(str, Enum):
    NONE = "NONE"
    MTOW_EXCEEDED = "MTOW_EXCEEDED"
    MIN_WEIGHT_VIOLATION = "MIN_WEIGHT_VIOLATION"
    LONGITUDINAL_CG_FORWARD_EXCEEDED = "LONGITUDINAL_CG_FORWARD_EXCEEDED"
    LONGITUDINAL_CG_AFT_EXCEEDED = "LONGITUDINAL_CG_AFT_EXCEEDED"
    LATERAL_CG_LEFT_EXCEEDED = "LATERAL_CG_LEFT_EXCEEDED"
    LATERAL_CG_RIGHT_EXCEEDED = "LATERAL_CG_RIGHT_EXCEEDED"
    OGE_HOVER_SCENE_DEFICIT = "OGE_HOVER_SCENE_DEFICIT"
    IGE_HOVER_DEFICIT = "IGE_HOVER_DEFICIT"
    OEI_CAT_A_HELIPAD_DEFICIT = "OEI_CAT_A_HELIPAD_DEFICIT"
    FAR_135_615_CEILING_BELOW_MINIMUMS = "FAR_135_615_CEILING_BELOW_MINIMUMS"
    FAR_135_615_VISIBILITY_BELOW_MINIMUMS = "FAR_135_615_VISIBILITY_BELOW_MINIMUMS"
    FAR_135_209_VFR_RESERVE_VIOLATION = "FAR_135_209_VFR_RESERVE_VIOLATION"
    IFR_RESERVE_VIOLATION = "IFR_RESERVE_VIOLATION"
    SEVERE_WEATHER_HAZARD = "SEVERE_WEATHER_HAZARD"
    TERRAIN_OBSTACLE_CLEARANCE_VIOLATION = "TERRAIN_OBSTACLE_CLEARANCE_VIOLATION"
    UNFLYABLE_WIND = "UNFLYABLE_WIND"

class ComplianceMetric(BaseModel):
    name: str
    actual_value: float
    limit_value: float
    margin: float
    unit: str
    is_compliant: bool
    notes: Optional[str] = None

class MissionLegReport(BaseModel):
    leg_index: int
    name: str
    duration_min: float
    distance_nm: float
    start_gw_kg: float
    end_gw_kg: float
    fuel_burn_kg: float
    remaining_fuel_kg: float
    cg_long_mm: float
    cg_lat_mm: float
    tas_kts: float
    gs_kts: float
    headwind_kts: float
    crosswind_kts: float
    pa_ft: float
    oat_c: float
    da_ft: float
    max_allowable_gw_kg: float
    hover_margin_kg: float
    metrics: List[ComplianceMetric] = Field(default_factory=list)

class DispatchEvaluation(BaseModel):
    verdict: DispatchVerdict
    limiting_factor: LimitingFactor
    limiting_description: str
    timestamp_utc: str
    tail_number: str
    airframe_type: str
    mission_id: str
    total_distance_nm: float
    total_flight_time_min: float
    total_fuel_burn_kg: float
    landing_reserve_fuel_kg: float
    required_reserve_fuel_kg: float
    reserve_margin_kg: float
    reserve_margin_minutes: float
    legs: List[MissionLegReport] = Field(default_factory=list)
    mitigation_actions: List[str] = Field(default_factory=list)
    audit_trace: List[str] = Field(default_factory=list)
