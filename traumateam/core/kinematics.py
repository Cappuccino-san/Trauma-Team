"""Kinematics & Wind Vector Navigation Engine.

Implements spherical geodesics (Haversine/Great Circle) and vector wind triangles
for wind-corrected groundspeed (GS), wind correction angle (WCA), and leg time en route (ETE).
"""

from __future__ import annotations
import math
from dataclasses import dataclass
from .units import NM_TO_METERS, METERS_TO_FEET

EARTH_RADIUS_NM = 3440.065  # ICAO mean Earth radius in nautical miles

@dataclass(frozen=True)
class GeoCoord:
    lat: float  # Degrees north (+), south (-)
    lon: float  # Degrees east (+), west (-)
    elev_ft: float = 0.0

@dataclass(frozen=True)
class WindVector:
    direction_deg: float  # True direction from which wind blows [0, 360)
    speed_kts: float      # Wind velocity in knots

@dataclass(frozen=True)
class WindTriangleResult:
    true_course_deg: float
    true_heading_deg: float
    wind_correction_angle_deg: float
    tas_kts: float
    groundspeed_kts: float
    headwind_component_kts: float
    crosswind_component_kts: float
    is_flyable: bool
    unflyable_reason: str | None = None

def great_circle_distance_nm(c1: GeoCoord, c2: GeoCoord) -> float:
    """Computes great circle distance between two points in nautical miles using Haversine."""
    lat1_rad = math.radians(c1.lat)
    lon1_rad = math.radians(c1.lon)
    lat2_rad = math.radians(c2.lat)
    lon2_rad = math.radians(c2.lon)

    dlat = lat2_rad - lat1_rad
    dlon = lon2_rad - lon1_rad

    a = (math.sin(dlat / 2.0) ** 2 +
         math.cos(lat1_rad) * math.cos(lat2_rad) * math.sin(dlon / 2.0) ** 2)
    c = 2.0 * math.atan2(math.sqrt(a), math.sqrt(max(0.0, 1.0 - a)))
    return EARTH_RADIUS_NM * c

def initial_bearing_deg(c1: GeoCoord, c2: GeoCoord) -> float:
    """Computes the initial true bearing (track) from point 1 to point 2 in degrees [0, 360)."""
    lat1_rad = math.radians(c1.lat)
    lon1_rad = math.radians(c1.lon)
    lat2_rad = math.radians(c2.lat)
    lon2_rad = math.radians(c2.lon)

    dlon = lon2_rad - lon1_rad
    y = math.sin(dlon) * math.cos(lat2_rad)
    x = (math.cos(lat1_rad) * math.sin(lat2_rad) -
         math.sin(lat1_rad) * math.cos(lat2_rad) * math.cos(dlon))

    bearing = math.degrees(math.atan2(y, x))
    return (bearing + 360.0) % 360.0

def solve_wind_triangle(
    true_course_deg: float,
    tas_kts: float,
    wind: WindVector
) -> WindTriangleResult:
    """Solves the standard aeronautical wind triangle.
    
    Given:
      TC = True Course to steer along ground track
      TAS = True Airspeed
      wind = Direction and speed from which wind blows
      
    Returns:
      WindTriangleResult with True Heading, Groundspeed, WCA, and wind components.
    """
    if tas_kts <= 0:
        return WindTriangleResult(
            true_course_deg=true_course_deg,
            true_heading_deg=true_course_deg,
            wind_correction_angle_deg=0.0,
            tas_kts=tas_kts,
            groundspeed_kts=0.0,
            headwind_component_kts=0.0,
            crosswind_component_kts=0.0,
            is_flyable=False,
            unflyable_reason="TAS is zero or negative."
        )

    # Relative angle between wind source direction and aircraft course
    # wind.direction_deg is where the wind comes FROM.
    # Relative wind angle alpha = wind.direction - true_course
    alpha_deg = (wind.direction_deg - true_course_deg) % 360.0
    alpha_rad = math.radians(alpha_deg)

    # Crosswind component (positive means crosswind from right pushing left, requiring right heading correction)
    crosswind = wind.speed_kts * math.sin(alpha_rad)
    # Headwind component (positive means blowing in aircraft face, decreasing GS)
    headwind = wind.speed_kts * math.cos(alpha_rad)

    # Check if crosswind exceeds TAS (physically impossible to hold track)
    sin_wca = crosswind / tas_kts
    if abs(sin_wca) > 1.0:
        return WindTriangleResult(
            true_course_deg=true_course_deg,
            true_heading_deg=true_course_deg,
            wind_correction_angle_deg=0.0,
            tas_kts=tas_kts,
            groundspeed_kts=0.0,
            headwind_component_kts=headwind,
            crosswind_component_kts=crosswind,
            is_flyable=False,
            unflyable_reason=f"Crosswind ({abs(crosswind):.1f} kt) exceeds True Airspeed ({tas_kts:.1f} kt)."
        )

    wca_rad = math.asin(sin_wca)
    wca_deg = math.degrees(wca_rad)
    true_heading_deg = (true_course_deg + wca_deg) % 360.0
    if true_heading_deg >= 360.0 or true_heading_deg < 0.0:
        true_heading_deg = 0.0

    # Groundspeed calculation: GS = TAS * cos(WCA) - Headwind
    gs = (tas_kts * math.cos(wca_rad)) - headwind

    if gs <= 0:
        return WindTriangleResult(
            true_course_deg=true_course_deg,
            true_heading_deg=true_heading_deg,
            wind_correction_angle_deg=wca_deg,
            tas_kts=tas_kts,
            groundspeed_kts=0.0,
            headwind_component_kts=headwind,
            crosswind_component_kts=crosswind,
            is_flyable=False,
            unflyable_reason=f"Direct headwind ({headwind:.1f} kt) exceeds or matches forward airspeed component."
        )

    return WindTriangleResult(
        true_course_deg=true_course_deg,
        true_heading_deg=true_heading_deg,
        wind_correction_angle_deg=wca_deg,
        tas_kts=tas_kts,
        groundspeed_kts=gs,
        headwind_component_kts=headwind,
        crosswind_component_kts=crosswind,
        is_flyable=True,
        unflyable_reason=None
    )
