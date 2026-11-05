"""TraumaTeam Core Package."""

from traumateam.core.units import (
    celsius_to_fahrenheit,
    fahrenheit_to_celsius,
    celsius_to_kelvin,
    kelvin_to_celsius,
    liters_to_kg,
    gallons_to_kg,
    kg_to_gallons,
    lbs_to_kg_val,
    kg_to_lbs_val,
)
from traumateam.core.atmosphere import (
    isa_temperature_c,
    pressure_altitude_ft,
    density_altitude_ft,
)
from traumateam.core.kinematics import (
    GeoCoord,
    WindVector,
    great_circle_distance_nm,
    initial_bearing_deg,
    solve_wind_triangle,
)

__all__ = [
    "celsius_to_fahrenheit",
    "fahrenheit_to_celsius",
    "celsius_to_kelvin",
    "kelvin_to_celsius",
    "liters_to_kg",
    "gallons_to_kg",
    "kg_to_gallons",
    "lbs_to_kg_val",
    "kg_to_lbs_val",
    "isa_temperature_c",
    "pressure_altitude_ft",
    "density_altitude_ft",
    "GeoCoord",
    "WindVector",
    "great_circle_distance_nm",
    "initial_bearing_deg",
    "solve_wind_triangle",
]
