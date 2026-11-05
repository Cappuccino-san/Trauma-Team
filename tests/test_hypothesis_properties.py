"""Property-Based Testing Harness with Hypothesis.

Proves mathematical invariants and aerodynamic boundary conditions across continuous state spaces:
1. Wind triangle speed bounds and flyability invariants.
2. Thermodynamic monotonicity of Density Altitude with temperature.
3. Aerodynamic monotonicity of RFM hover ceilings with pressure altitude.
4. Physical bounding of Center of Gravity within station coordinate convex hulls.
"""

import pytest
import math
from hypothesis import given, strategies as st, settings, HealthCheck
from traumateam.core.kinematics import solve_wind_triangle, WindVector
from traumateam.core.atmosphere import density_altitude_ft
from traumateam.performance.airframes.h145 import get_h145_rfm_spec
from traumateam.performance.rfm_engine import RFMPerformanceEngine
from traumateam.performance.mass_balance import compute_mass_and_balance
from traumateam.models.mission import PayloadItem

@given(
    tc=st.floats(min_value=0.0, max_value=359.999),
    tas=st.floats(min_value=50.0, max_value=160.0),
    wind_dir=st.floats(min_value=0.0, max_value=359.999),
    wind_spd=st.floats(min_value=0.0, max_value=45.0), # Wind speed strictly below TAS
)
@settings(max_examples=50, suppress_health_check=[HealthCheck.too_slow])
def test_wind_triangle_physical_bounds(tc: float, tas: float, wind_dir: float, wind_spd: float):
    """Property: Groundspeed must always be mathematically bounded by [TAS - WS, TAS + WS]."""
    wind = WindVector(direction_deg=wind_dir, speed_kts=wind_spd)
    res = solve_wind_triangle(true_course_deg=tc, tas_kts=tas, wind=wind)

    assert res.is_flyable is True
    # Mathematical bounds of groundspeed triangle:
    lower_bound = tas - wind_spd - 1e-4
    upper_bound = tas + wind_spd + 1e-4

    assert res.groundspeed_kts >= lower_bound
    assert res.groundspeed_kts <= upper_bound
    assert 0.0 <= res.true_heading_deg < 360.0

@given(
    pa=st.floats(min_value=-1000.0, max_value=12000.0),
    t1=st.floats(min_value=-20.0, max_value=25.0),
    delta_t=st.floats(min_value=0.1, max_value=25.0),
)
@settings(max_examples=50, suppress_health_check=[HealthCheck.too_slow])
def test_density_altitude_monotonicity_with_temperature(pa: float, t1: float, delta_t: float):
    """Property: At any fixed Pressure Altitude, Density Altitude strictly increases with OAT."""
    t2 = t1 + delta_t
    da1 = density_altitude_ft(pa, t1)
    da2 = density_altitude_ft(pa, t2)

    assert da2 > da1

@given(
    oat=st.floats(min_value=-10.0, max_value=40.0),
    pa1=st.floats(min_value=0.0, max_value=10000.0),
    delta_pa=st.floats(min_value=500.0, max_value=4000.0),
)
@settings(max_examples=40, suppress_health_check=[HealthCheck.too_slow])
def test_rfm_hover_ceiling_monotonicity(oat: float, pa1: float, delta_pa: float):
    """Property: At any fixed temperature, RFM Max OGE Hover Weight monotonically decreases or stays constant with increasing PA."""
    pa2 = min(14000.0, pa1 + delta_pa)
    h145 = get_h145_rfm_spec()
    engine = RFMPerformanceEngine(h145)

    w1 = engine.get_max_oge_hover_weight(pa1, oat)
    w2 = engine.get_max_oge_hover_weight(pa2, oat)

    assert w1 >= w2 - 1e-6

@given(
    pilot_w=st.floats(min_value=60.0, max_value=120.0),
    copilot_w=st.floats(min_value=60.0, max_value=120.0),
    nurse_w=st.floats(min_value=50.0, max_value=110.0),
    medic_w=st.floats(min_value=60.0, max_value=120.0),
    fuel=st.floats(min_value=100.0, max_value=694.0),
)
@settings(max_examples=40, suppress_health_check=[HealthCheck.too_slow])
def test_cg_strictly_bounded_within_geometry(pilot_w, copilot_w, nurse_w, medic_w, fuel):
    """Property: Aircraft CG must always remain strictly within the physical coordinate bounds of the airframe."""
    h145 = get_h145_rfm_spec()
    payloads = [
        PayloadItem(item_id="p1", name="Pilot", station_id="PILOT_RH", weight_kg=pilot_w),
        PayloadItem(item_id="p2", name="Copilot", station_id="COPILOT_LH", weight_kg=copilot_w),
        PayloadItem(item_id="p3", name="Nurse", station_id="CREW_ATTENDANT_AFT_LH", weight_kg=nurse_w),
        PayloadItem(item_id="p4", name="Medic", station_id="CREW_ATTENDANT_AFT_RH", weight_kg=medic_w),
    ]
    mb = compute_mass_and_balance(h145, fuel_kg=fuel, active_payloads=payloads)

    # Station longitudinal arms range between 2420 mm (cockpit) and 4950 mm (aft cargo)
    assert 2400.0 <= mb.cg_long_mm <= 5000.0
    # Lateral arms range between -420 mm (LH) and +420 mm (RH)
    assert -450.0 <= mb.cg_lat_mm <= 450.0
