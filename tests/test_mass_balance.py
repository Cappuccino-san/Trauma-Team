"""Tests for Mass, Balance, and Certified Envelope Limits."""

import pytest
from traumateam.performance.airframes.h145 import get_h145_rfm_spec
from traumateam.performance.mass_balance import compute_mass_and_balance
from traumateam.models.mission import PayloadItem
from traumateam.models.verdict import LimitingFactor

def test_mass_balance_hand_calculation_accuracy():
    """Validates that computed CG matches analytical static moment summation to within 0.01%."""
    h145 = get_h145_rfm_spec()

    payloads = [
        PayloadItem(item_id="p1", name="Pilot", station_id="PILOT_RH", weight_kg=85.0),
        PayloadItem(item_id="p2", name="Copilot", station_id="COPILOT_LH", weight_kg=80.0),
        PayloadItem(item_id="p3", name="Nurse", station_id="CREW_ATTENDANT_AFT_LH", weight_kg=70.0),
        PayloadItem(item_id="p4", name="Medic", station_id="CREW_ATTENDANT_AFT_RH", weight_kg=85.0),
        PayloadItem(item_id="p5", name="Med Rack", station_id="MEDICAL_CABINET_RACK", weight_kg=60.0),
    ]
    fuel_kg = 500.0

    # Hand calculate exact moments:
    # Empty: 2140 kg * 3910 mm = 8367400 kg*mm (long), 2140 kg * -5 mm = -10700 kg*mm (lat)
    # Pilot: 85 * 2420 = 205700, 85 * 360 = 30600
    # Copilot: 80 * 2420 = 193600, 80 * -360 = -28800
    # Nurse: 70 * 3350 = 234500, 70 * -420 = -29400
    # Medic: 85 * 3350 = 284750, 85 * 420 = 35700
    # Rack: 60 * 4280 = 256800, 60 * 380 = 22800
    # Fuel arm at 500 kg: interpolated between 350kg (3975mm) and 550kg (4005mm):
    # frac = (500 - 350) / (550 - 350) = 150/200 = 0.75
    # arm_long = 3975 + 0.75 * 30 = 3997.5 mm; lat = 0.0 mm
    # Fuel: 500 * 3997.5 = 1998750
    # Total W = 2140 + 85 + 80 + 70 + 85 + 60 + 500 = 3020.0 kg
    # Total M_long = 8367400 + 205700 + 193600 + 234500 + 284750 + 256800 + 1998750 = 11541500 kg*mm
    # Expected CG_long = 11541500 / 3020.0 = 3821.6887 mm
    # Total M_lat = -10700 + 30600 - 28800 - 29400 + 35700 + 22800 + 0 = 20200 kg*mm
    # Expected CG_lat = 20200 / 3020.0 = 6.6887 mm

    expected_gw = 3020.0
    expected_cg_long = 11541500.0 / 3020.0
    expected_cg_lat = 20200.0 / 3020.0

    mb = compute_mass_and_balance(h145, fuel_kg, payloads)

    assert pytest.approx(mb.gross_weight_kg, rel=1e-5) == expected_gw
    assert pytest.approx(mb.cg_long_mm, rel=1e-4) == expected_cg_long
    assert pytest.approx(mb.cg_lat_mm, rel=1e-4) == expected_cg_lat
    assert mb.is_overall_compliant is True

def test_lateral_cg_exceedance_detection():
    """Validates that asymmetric heavy lateral loading correctly flags LATERAL_CG_RIGHT_EXCEEDED."""
    h145 = get_h145_rfm_spec()

    # Extreme right load: Pilot RH, Attendant RH, Med Rack RH, and heavy hoist load on RH boom (1050 mm arm)
    payloads = [
        PayloadItem(item_id="p1", name="Heavy Pilot", station_id="PILOT_RH", weight_kg=110.0),
        PayloadItem(item_id="p2", name="Heavy Attendant", station_id="CREW_ATTENDANT_AFT_RH", weight_kg=110.0),
        PayloadItem(item_id="p3", name="Heavy Med Rack", station_id="MEDICAL_CABINET_RACK", weight_kg=100.0),
        PayloadItem(item_id="p4", name="Rescue Hoist Load", station_id="RESCUE_HOIST_RH", weight_kg=220.0),
    ]
    # Low fuel so cabin moments dominate
    mb = compute_mass_and_balance(h145, fuel_kg=100.0, active_payloads=payloads)

    assert mb.is_lat_cg_compliant is False
    assert mb.limiting_factor == LimitingFactor.LATERAL_CG_RIGHT_EXCEEDED
    assert mb.cg_lat_mm > mb.right_limit_mm

def test_mtow_exceedance_detection():
    """Validates that loading above MTOW (3700 kg for H145) triggers MTOW_EXCEEDED."""
    h145 = get_h145_rfm_spec()

    payloads = [
        PayloadItem(item_id="p1", name="Pilot", station_id="PILOT_RH", weight_kg=100.0),
        PayloadItem(item_id="p2", name="Copilot", station_id="COPILOT_LH", weight_kg=100.0),
        PayloadItem(item_id="p3", name="Crew LH", station_id="CREW_ATTENDANT_AFT_LH", weight_kg=100.0),
        PayloadItem(item_id="p4", name="Crew RH", station_id="CREW_ATTENDANT_AFT_RH", weight_kg=100.0),
        PayloadItem(item_id="p5", name="Patient", station_id="PRIMARY_PATIENT_LITTER", weight_kg=180.0),
        PayloadItem(item_id="p6", name="Patient 2", station_id="SECONDARY_PATIENT_LITTER", weight_kg=150.0),
        PayloadItem(item_id="p7", name="Cargo", station_id="AFT_AVIONICS_CARGO", weight_kg=150.0),
        PayloadItem(item_id="p8", name="Rack", station_id="MEDICAL_CABINET_RACK", weight_kg=120.0),
        PayloadItem(item_id="p9", name="Oxygen", station_id="OXYGEN_SYSTEM", weight_kg=50.0),
    ]
    # Full fuel 694 kg -> Gross weight = 2140 + 950 + 694 = 3784 kg > 3700 kg
    mb = compute_mass_and_balance(h145, fuel_kg=694.0, active_payloads=payloads)

    assert mb.is_gw_compliant is False
    assert mb.limiting_factor == LimitingFactor.MTOW_EXCEEDED
    assert mb.gross_weight_kg > 3700.0
