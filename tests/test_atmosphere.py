"""Tests for Atmospheric Altimetry, Pressure Altitude, and Density Altitude."""

import pytest
import math
from traumateam.core.atmosphere import (
    isa_temperature_c,
    pressure_altitude_ft,
    density_altitude_ft,
    speed_of_sound_knots,
)
from traumateam.core.units import (
    celsius_to_fahrenheit,
    fahrenheit_to_celsius,
    liters_to_kg,
    gallons_to_kg,
)

def test_isa_temperature_standard_altitudes():
    # Sea level ISA is exactly 15 C
    assert pytest.approx(isa_temperature_c(0.0), abs=1e-2) == 15.0
    # At 10,000 ft: 15 - 1.9812 * 10 = -4.81 C
    assert pytest.approx(isa_temperature_c(10000.0), abs=0.1) == -4.81
    # At 5,000 ft: 15 - 1.9812 * 5 = 5.09 C
    assert pytest.approx(isa_temperature_c(5000.0), abs=0.1) == 5.09

def test_pressure_altitude_standard_and_non_standard():
    # At standard 29.92 inHg, PA equals geometric altitude
    assert pytest.approx(pressure_altitude_ft(5000.0, altimeter_inhg=29.92126), abs=1.0) == 5000.0
    
    # Low pressure (29.42 inHg): exact barometric ICAO formula gives 5466.8 ft
    pa_low = pressure_altitude_ft(5000.0, altimeter_inhg=29.42, exact_barometric=True)
    assert pa_low > 5000.0
    assert pytest.approx(pa_low, abs=5.0) == 5466.8

    # Simplified aviation rule-of-thumb: (29.92 - 29.42) * 1000 + 5000 = 5501.3 ft
    pa_low_approx = pressure_altitude_ft(5000.0, altimeter_inhg=29.42, exact_barometric=False)
    assert pytest.approx(pa_low_approx, abs=5.0) == 5501.3

    # High pressure (30.42 inHg): PA decreases
    pa_high = pressure_altitude_ft(5000.0, altimeter_inhg=30.42)
    assert pa_high < 5000.0

def test_density_altitude_high_elevation_and_heat():
    # At 9,500 ft PA and 25 C (hot mountain day, ISA is -3.8 C, delta ISA is +28.8 C):
    # Rule of thumb: DA ~ 9500 + 118.8 * 28.8 = 12,921 ft
    da = density_altitude_ft(9500.0, oat_c=25.0)
    assert 12500.0 < da < 13200.0

    # At sea level on standard day (0 ft PA, 15 C): DA is 0 ft
    da_sl = density_altitude_ft(0.0, oat_c=15.0)
    assert pytest.approx(da_sl, abs=2.0) == 0.0

def test_unit_conversions():
    assert pytest.approx(celsius_to_fahrenheit(0.0), abs=1e-4) == 32.0
    assert pytest.approx(celsius_to_fahrenheit(100.0), abs=1e-4) == 212.0
    assert pytest.approx(fahrenheit_to_celsius(32.0), abs=1e-4) == 0.0
    
    # Jet-A fuel volume to weight
    kg_100_gal = gallons_to_kg(100.0, oat_c=15.0)
    # 100 gal = 378.54 L * 0.804 kg/L = 304.3 kg
    assert pytest.approx(kg_100_gal, abs=1.0) == 304.3
