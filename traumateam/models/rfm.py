"""Pydantic & Dataclass Schemas for Rotorcraft Flight Manual (RFM) Performance Tables."""

from __future__ import annotations
from typing import List, Dict, Optional, Tuple
from pydantic import BaseModel, Field

class StationArm(BaseModel):
    station_id: str
    name: str
    arm_long_mm: float
    arm_lat_mm: float
    max_capacity_kg: float = 500.0
    is_fuel: bool = False
    description: Optional[str] = None

class VariableFuelArmPoint(BaseModel):
    fuel_kg: float
    arm_long_mm: float
    arm_lat_mm: float

class FuelSystemSpec(BaseModel):
    usable_fuel_capacity_kg: float
    unusable_fuel_kg: float
    tank_arms: List[VariableFuelArmPoint] = Field(
        default_factory=list,
        description="Non-linear fuel arm schedule if center of gravity shifts with fuel level"
    )
    default_arm_long_mm: float
    default_arm_lat_mm: float

class CGEnvelopePoint(BaseModel):
    gross_weight_kg: float
    cg_long_fwd_mm: float
    cg_long_aft_mm: float
    cg_lat_left_mm: float  # Negative convention for left
    cg_lat_right_mm: float # Positive convention for right

class RFMPerformanceGrid(BaseModel):
    name: str
    pa_grid_ft: List[float]       # e.g., [-2000, 0, 2000, 4000, 6000, 8000, 10000, 12000, 14000]
    oat_grid_c: List[float]       # e.g., [-20, -10, 0, 10, 20, 30, 40, 50]
    weight_table_kg: List[List[float]] # 2D array [len(pa), len(oat)]

class RFMCruiseGrid(BaseModel):
    pa_grid_ft: List[float]
    oat_grid_c: List[float]
    gw_grid_kg: List[float]
    tas_table_kts: List[List[List[float]]]       # 3D array [len(pa), len(oat), len(gw)]
    fuel_flow_kg_hr: List[List[List[float]]]     # 3D array [len(pa), len(oat), len(gw)]

class RFMAirframeSpec(BaseModel):
    model_name: str
    oem: str
    certification_basis: str = "FAR Part 29 / Category A"
    empty_weight_kg: float
    empty_cg_long_mm: float
    empty_cg_lat_mm: float
    mtow_kg: float
    min_flight_weight_kg: float
    fuel_system: FuelSystemSpec
    stations: Dict[str, StationArm]
    cg_envelope: List[CGEnvelopePoint]
    ige_hover_grid: RFMPerformanceGrid
    oge_hover_grid: RFMPerformanceGrid
    oei_cat_a_grid: RFMPerformanceGrid
    cruise_performance: RFMCruiseGrid
    startup_fuel_kg: float = 15.0
    hover_fuel_flow_factor: float = 1.18  # Hover fuel flow multiplier relative to mid cruise
    loiter_fuel_flow_factor: float = 0.88 # Loiter / hold fuel flow multiplier
