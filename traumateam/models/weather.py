"""Aviation Weather, Atmospheric Observations, and Corridor Hazards Models."""

from __future__ import annotations
from enum import Enum
from typing import List, Optional
from pydantic import BaseModel, Field

class FlightRuleCategory(str, Enum):
    VFR = "VFR"
    MVFR = "MVFR"
    IFR = "IFR"
    LIFR = "LIFR"

class HazardSeverity(str, Enum):
    LIGHT = "LIGHT"
    MODERATE = "MODERATE"
    SEVERE = "SEVERE"
    EXTREME = "EXTREME"

class HazardType(str, Enum):
    CONVECTIVE_SIGMET = "CONVECTIVE_SIGMET"
    SEVERE_TURBULENCE = "SEVERE_TURBULENCE"
    SEVERE_ICING = "SEVERE_ICING"
    IFR_CLOUD_COVER = "IFR_CLOUD_COVER"
    MOUNTAIN_OBSCURATION = "MOUNTAIN_OBSCURATION"

class METARObservation(BaseModel):
    station_id: str
    observation_time_utc: str
    wind_dir_deg: float
    wind_speed_kts: float
    wind_gust_kts: float = 0.0
    visibility_sm: float
    ceiling_agl_ft: float
    temperature_c: float
    dewpoint_c: float
    altimeter_inhg: float
    flight_category: FlightRuleCategory = FlightRuleCategory.VFR
    raw_text: str = ""

class WindsAloftLayer(BaseModel):
    altitude_msl_ft: float
    direction_deg: float
    speed_kts: float
    temperature_c: float

class WeatherHazardAdvisory(BaseModel):
    advisory_id: str
    hazard_type: HazardType
    severity: HazardSeverity
    summary: str
    min_altitude_ft: float = 0.0
    max_altitude_ft: float = 60000.0
    active: bool = True

class CorridorWeather(BaseModel):
    departure_metar: METARObservation
    destination_metar: Optional[METARObservation] = None
    intermediate_metars: List[METARObservation] = Field(default_factory=list)
    winds_aloft_profile: List[WindsAloftLayer] = Field(default_factory=list)
    active_hazards: List[WeatherHazardAdvisory] = Field(default_factory=list)
