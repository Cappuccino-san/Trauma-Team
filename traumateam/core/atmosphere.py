"""Standard Atmosphere & Altimetry Calculations for Rotorcraft Performance.

Implements ICAO Standard Atmosphere (Doc 7488/3) and barometric pressure/density altitude
determinations required for RFM chart entry.
"""

from __future__ import annotations
import math
from .units import (
    FEET_TO_METERS,
    METERS_TO_FEET,
    celsius_to_kelvin,
    STD_SEA_LEVEL_PRESSURE_HPA,
    STD_SEA_LEVEL_PRESSURE_INHG,
    INHG_TO_HPA,
)

# Standard Atmosphere Constants (Troposphere up to 11,000 m / 36,089 ft)
T0_K = 288.15        # Sea-level standard temperature (15 C in K)
P0_PA = 101325.0     # Sea-level standard pressure in Pa
R_AIR = 287.05287    # Specific gas constant for dry air (J/(kg*K))
G0 = 9.80665         # Standard gravitational acceleration (m/s^2)
LAPSE_RATE_K_PER_M = 0.0065  # Troposphere temperature lapse rate (K/m)
LAPSE_RATE_K_PER_FT = LAPSE_RATE_K_PER_M * FEET_TO_METERS  # ~0.0019812 K/ft
RHO0_KG_PER_M3 = 1.2250      # Sea-level standard density (kg/m^3)
GAMMA_AIR = 1.40             # Ratio of specific heats

def isa_temperature_c(altitude_ft: float) -> float:
    """Calculates standard temperature in Celsius at a given geopotential altitude in feet."""
    return 15.0 - (LAPSE_RATE_K_PER_FT * altitude_ft)

def isa_temperature_k(altitude_ft: float) -> float:
    """Calculates standard temperature in Kelvin at a given geopotential altitude in feet."""
    return celsius_to_kelvin(isa_temperature_c(altitude_ft))

def pressure_altitude_ft(
    geometric_altitude_ft: float,
    altimeter_inhg: float | None = None,
    qnh_hpa: float | None = None,
    exact_barometric: bool = True
) -> float:
    """Computes Pressure Altitude (PA) in feet from geometric elevation and barometric setting.
    
    If neither altimeter_inhg nor qnh_hpa is provided, assumes standard 29.92 inHg / 1013.25 hPa.
    """
    if not math.isfinite(geometric_altitude_ft):
        raise ValueError(f"Invalid geometric altitude: {geometric_altitude_ft}")

    if qnh_hpa is None:
        if altimeter_inhg is not None:
            qnh_hpa = altimeter_inhg * INHG_TO_HPA
        else:
            qnh_hpa = STD_SEA_LEVEL_PRESSURE_HPA

    if not math.isfinite(qnh_hpa) or qnh_hpa <= 0.0:
        raise ValueError(f"Invalid barometric pressure: {qnh_hpa}")

    if exact_barometric:
        # ICAO Standard Geopotential Pressure Formula:
        # P = P0 * (1 - L*h/T0)^(g/(R*L))
        # Given local station pressure P_st = QNH * (1 - L*h_geom/T0)^(g/(R*L))
        # Pressure altitude is the altitude where P(PA) = P_st in standard atmosphere
        # PA = (T0 / L) * [1 - (P_st / P0)^(R*L/g)]
        # When entering with QNH and station elevation:
        # delta_h = (T0 / L) * [1 - (QNH / P0)^(R*L / g)]
        # PA = h_geom + (T0 / L) * [1 - (QNH / P0)^(R*L / g)]
        exponent = (R_AIR * LAPSE_RATE_K_PER_M) / G0  # ~0.190263
        t0_over_l_m = T0_K / LAPSE_RATE_K_PER_M        # ~44330.77 m
        t0_over_l_ft = t0_over_l_m * METERS_TO_FEET    # ~145442 ft
        pa_correction = t0_over_l_ft * (1.0 - math.pow(qnh_hpa / STD_SEA_LEVEL_PRESSURE_HPA, exponent))
        return geometric_altitude_ft + pa_correction
    else:
        # Standard aviation rule of thumb: (29.92 - Altimeter) * 1000
        inhg = qnh_hpa / INHG_TO_HPA
        return geometric_altitude_ft + (STD_SEA_LEVEL_PRESSURE_INHG - inhg) * 1000.0

def density_altitude_ft(
    pressure_altitude_ft: float,
    oat_c: float,
    exact_thermodynamic: bool = True
) -> float:
    """Computes Density Altitude (DA) in feet from Pressure Altitude and Outside Air Temp (OAT).
    
    Density altitude is the single most critical aerodynamic governor for helicopter hover performance,
    directly defining rotor blade stall margins and turboshaft power available.
    """
    if not math.isfinite(pressure_altitude_ft) or not math.isfinite(oat_c):
        raise ValueError(f"Invalid non-finite atmosphere inputs: PA={pressure_altitude_ft}, OAT={oat_c}")
    tk = celsius_to_kelvin(oat_c)
    if tk <= 0.0:
        raise ValueError(f"OAT below absolute zero: {oat_c} C")

    isa_c = isa_temperature_c(pressure_altitude_ft)
    delta_isa = oat_c - isa_c

    if not exact_thermodynamic:
        # FAA standard approximation: DA = PA + 118.8 * delta_ISA
        return pressure_altitude_ft + (118.8 * delta_isa)

    # Exact thermodynamic air density derivation:
    # 1. Standard pressure ratio at PA: delta = (1 - L * PA / T0)^(g/(R*L))
    t0_ft = T0_K / LAPSE_RATE_K_PER_FT
    # Clamp to avoid non-physical zero or negative values at extreme heights
    term = max(0.01, 1.0 - (pressure_altitude_ft / t0_ft))
    power = G0 / (R_AIR * LAPSE_RATE_K_PER_M)  # ~5.25588
    delta_press = math.pow(term, power)
    pressure_pa = delta_press * P0_PA

    # 2. Local air density: rho = P / (R * T_K)
    rho = pressure_pa / (R_AIR * tk)

    # 3. Altitude in ISA having density rho:
    # rho / rho0 = (1 - L * DA / T0)^( (g/(R*L)) - 1 )
    density_exponent = (G0 / (R_AIR * LAPSE_RATE_K_PER_M)) - 1.0  # ~4.25588
    density_ratio = rho / RHO0_KG_PER_M3
    if density_ratio <= 0.0:
        return 50000.0  # Stratospheric limit safeguard
    
    da_ft = t0_ft * (1.0 - math.pow(density_ratio, 1.0 / density_exponent))
    return da_ft

def speed_of_sound_knots(oat_c: float) -> float:
    """Speed of sound in knots as a function of temperature."""
    if not math.isfinite(oat_c):
        raise ValueError(f"Invalid non-finite OAT: {oat_c}")
    tk = celsius_to_kelvin(oat_c)
    if tk <= 0.0:
        raise ValueError(f"OAT below absolute zero: {oat_c} C")
    a_mps = math.sqrt(GAMMA_AIR * R_AIR * tk)
    return a_mps * (3600.0 / 1852.0)
