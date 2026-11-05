"""TraumaTeam Core Aviation Units & Coordinate Transforms.

Provides deterministic conversions conforming to ICAO Annex 5 and FAA Part 135 specs.
"""

from __future__ import annotations
import math

# Length / Distance
NM_TO_METERS = 1852.0
METERS_TO_FEET = 3.280839895013123
FEET_TO_METERS = 0.3048
NM_TO_FEET = NM_TO_METERS * METERS_TO_FEET
SM_TO_FEET = 5280.0
SM_TO_NM = SM_TO_FEET / NM_TO_FEET
INCHES_TO_MM = 25.4
MM_TO_INCHES = 1.0 / INCHES_TO_MM

# Mass
LBS_TO_KG = 0.45359237
KG_TO_LBS = 1.0 / LBS_TO_KG

# Pressure
INHG_TO_HPA = 33.863886666667
HPA_TO_INHG = 1.0 / INHG_TO_HPA
STD_SEA_LEVEL_PRESSURE_HPA = 1013.25
STD_SEA_LEVEL_PRESSURE_INHG = 29.92126

# Jet-A1 standard fuel density at 15 deg C
# Standard Jet-A is 0.804 kg/L = 6.709 lb/US gal (varies 0.775 to 0.840 kg/L with temperature)
JET_A_STD_DENSITY_KG_PER_L = 0.804
JET_A_STD_DENSITY_LB_PER_GAL = 6.709
US_GAL_TO_LITERS = 3.785411784

def celsius_to_fahrenheit(c: float) -> float:
    return c * 9.0 / 5.0 + 32.0

def fahrenheit_to_celsius(f: float) -> float:
    return (f - 32.0) * 5.0 / 9.0

def celsius_to_kelvin(c: float) -> float:
    return c + 273.15

def kelvin_to_celsius(k: float) -> float:
    return k - 273.15

def jet_a_density_kg_per_l(oat_c: float) -> float:
    """Computes temperature-corrected Jet-A1 density in kg/L.
    Standard volumetric expansion coefficient beta approx 0.001 / K.
    rho(T) = rho_15 * [1 - beta * (T - 15)]
    """
    beta = 0.00095
    return JET_A_STD_DENSITY_KG_PER_L * (1.0 - beta * (oat_c - 15.0))

def liters_to_kg(liters: float, oat_c: float = 15.0) -> float:
    return liters * jet_a_density_kg_per_l(oat_c)

def gallons_to_kg(gallons: float, oat_c: float = 15.0) -> float:
    return liters_to_kg(gallons * US_GAL_TO_LITERS, oat_c)

def kg_to_gallons(mass_kg: float, oat_c: float = 15.0) -> float:
    liters = mass_kg / jet_a_density_kg_per_l(oat_c)
    return liters / US_GAL_TO_LITERS

def lbs_to_kg_val(lbs: float) -> float:
    return lbs * LBS_TO_KG

def kg_to_lbs_val(kg: float) -> float:
    return kg * KG_TO_LBS

def in_lb_to_kg_mm(moment_in_lb: float) -> float:
    return moment_in_lb * LBS_TO_KG * INCHES_TO_MM

def kg_mm_to_in_lb(moment_kg_mm: float) -> float:
    return moment_kg_mm * KG_TO_LBS * MM_TO_INCHES
