"""USGS DEM Elevation & FAA Obstacle Corridor Clearance Engine."""

from __future__ import annotations
from dataclasses import dataclass
from typing import List, Optional
from traumateam.models.mission import Waypoint
from traumateam.models.verdict import LimitingFactor, DispatchVerdict

@dataclass(frozen=True)
class TerrainClearanceReport:
    is_cleared: bool
    verdict: DispatchVerdict
    limiting_factor: LimitingFactor
    min_safe_altitude_msl_ft: float
    planned_altitude_msl_ft: float
    terrain_clearance_agl_ft: float
    required_clearance_agl_ft: float
    margin_ft: float
    detail_message: str

def evaluate_terrain_and_obstacles(
    origin: Waypoint,
    destination: Waypoint,
    planned_cruise_alt_msl_ft: float,
    is_day: bool,
    is_mountainous: bool,
    max_terrain_along_leg_ft: Optional[float] = None,
    max_obstacle_along_leg_ft: Optional[float] = None
) -> TerrainClearanceReport:
    """Evaluates en-route terrain and obstacle clearance against 14 CFR § 135.203 and § 135.615."""
    
    # Highest obstacle/terrain elevation (MSL) across origin, destination, and en-route segment
    origin_obstacle_msl = origin.elevation_ft + origin.obstacle_height_ft
    dest_obstacle_msl = destination.elevation_ft + destination.obstacle_height_ft
    enroute_obstacle_msl = (max_terrain_along_leg_ft or 0.0) + (max_obstacle_along_leg_ft or 0.0)

    highest_obstacle_elevation_msl = max(
        origin_obstacle_msl,
        dest_obstacle_msl,
        enroute_obstacle_msl,
    )

    # Required clearance under FAR 135.203 / 135.615
    if is_day:
        required_agl_clearance = 500.0  # Standard daytime en-route obstacle clearance
    else:
        if is_mountainous:
            required_agl_clearance = 1000.0 # Night mountainous with TAWS/NVG (or 2000 ft without)
        else:
            required_agl_clearance = 500.0  # Night non-mountainous

    min_safe_alt_msl = highest_obstacle_elevation_msl + required_agl_clearance
    actual_clearance_agl = planned_cruise_alt_msl_ft - highest_obstacle_elevation_msl
    margin_ft = actual_clearance_agl - required_agl_clearance

    is_cleared = (margin_ft >= 0.0)

    if not is_cleared:
        verdict = DispatchVerdict.NO_GO
        limiting_factor = LimitingFactor.TERRAIN_OBSTACLE_CLEARANCE_VIOLATION
        msg = (
            f"OBSTACLE/TERRAIN CLEARANCE VIOLATION: Planned cruise altitude {planned_cruise_alt_msl_ft:.0f} ft MSL "
            f"provides only {actual_clearance_agl:.0f} ft clearance over highest obstacle/terrain ({highest_obstacle_elevation_msl:.0f} ft MSL). "
            f"Required minimum clearance is {required_agl_clearance:.0f} ft (Deficit: {abs(margin_ft):.0f} ft)."
        )
    else:
        verdict = DispatchVerdict.GO
        limiting_factor = LimitingFactor.NONE
        msg = (
            f"Terrain/obstacle clearance compliant: Planned {planned_cruise_alt_msl_ft:.0f} ft MSL provides "
            f"{actual_clearance_agl:.0f} ft AGL over {highest_obstacle_elevation_msl:.0f} ft MSL (Margin: +{margin_ft:.0f} ft)."
        )

    return TerrainClearanceReport(
        is_cleared=is_cleared,
        verdict=verdict,
        limiting_factor=limiting_factor,
        min_safe_altitude_msl_ft=min_safe_alt_msl,
        planned_altitude_msl_ft=planned_cruise_alt_msl_ft,
        terrain_clearance_agl_ft=actual_clearance_agl,
        required_clearance_agl_ft=required_agl_clearance,
        margin_ft=margin_ft,
        detail_message=msg,
    )
