"""Mission Definition, Payload Schedule, and Leg Architecture."""

from __future__ import annotations
from enum import Enum
from typing import List, Optional
from pydantic import BaseModel, Field
from traumateam.core.kinematics import GeoCoord

class LegType(str, Enum):
    STARTUP = "STARTUP"
    TRANSIT = "TRANSIT"
    SCENE_OPERATIONS = "SCENE_OPERATIONS"
    HOSPITAL_OPERATIONS = "HOSPITAL_OPERATIONS"
    BASE_SHUTDOWN = "BASE_SHUTDOWN"

class PayloadItem(BaseModel):
    item_id: str
    name: str
    station_id: str
    weight_kg: float
    loaded_at_leg: int = 0
    unloaded_at_leg: Optional[int] = None
    description: Optional[str] = None

class Waypoint(BaseModel):
    identifier: str
    name: str
    lat: float
    lon: float
    elevation_ft: float
    is_elevated_helipad: bool = False
    obstacle_height_ft: float = 0.0

    @property
    def geo_coord(self) -> GeoCoord:
        return GeoCoord(lat=self.lat, lon=self.lon, elev_ft=self.elevation_ft)

class LegSpec(BaseModel):
    leg_index: int
    name: str
    leg_type: LegType
    origin: Waypoint
    destination: Waypoint
    planned_cruise_altitude_msl_ft: float
    hover_oge_min: float = 0.0
    hover_ige_min: float = 0.0
    ground_idle_min: float = 0.0
    target_tas_kts: Optional[float] = None

class MissionProfile(BaseModel):
    mission_id: str
    tail_number: str
    initial_fuel_kg: float
    payloads: List[PayloadItem]
    legs: List[LegSpec]
    is_vfr: bool = True
    is_day: bool = True
    is_mountainous: bool = False
    is_local_flying_area: bool = False
    nvg_or_htaws_equipped: bool = True
