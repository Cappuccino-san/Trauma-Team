"""Bell 429 GlobalRanger RFM Performance Specification & Envelopes.

Data derived from Bell 429 Rotorcraft Flight Manual (BHT-429-FM-1).
Twin Pratt & Whitney Canada PW207D1 engines.
"""

from __future__ import annotations
from traumateam.models.rfm import (
    RFMAirframeSpec,
    StationArm,
    FuelSystemSpec,
    VariableFuelArmPoint,
    CGEnvelopePoint,
    RFMPerformanceGrid,
    RFMCruiseGrid,
)

def get_bell429_rfm_spec() -> RFMAirframeSpec:
    """Builds Bell 429 RFM engineering dataset."""
    stations = {
        "PILOT_RH": StationArm(
            station_id="PILOT_RH",
            name="Flight Crew RH Pilot Seat",
            arm_long_mm=2180.0,
            arm_lat_mm=330.0,
            max_capacity_kg=130.0,
        ),
        "COPILOT_LH": StationArm(
            station_id="COPILOT_LH",
            name="Flight Crew LH Co-Pilot Seat",
            arm_long_mm=2180.0,
            arm_lat_mm=-330.0,
            max_capacity_kg=130.0,
        ),
        "CREW_ATTENDANT_LH": StationArm(
            station_id="CREW_ATTENDANT_LH",
            name="Medical Attendant LH Seat",
            arm_long_mm=3100.0,
            arm_lat_mm=-400.0,
            max_capacity_kg=130.0,
        ),
        "CREW_ATTENDANT_RH": StationArm(
            station_id="CREW_ATTENDANT_RH",
            name="Medical Attendant RH Seat",
            arm_long_mm=3100.0,
            arm_lat_mm=400.0,
            max_capacity_kg=130.0,
        ),
        "PRIMARY_PATIENT_LITTER": StationArm(
            station_id="PRIMARY_PATIENT_LITTER",
            name="Patient Litter Deck",
            arm_long_mm=3550.0,
            arm_lat_mm=-250.0,
            max_capacity_kg=200.0,
        ),
        "MEDICAL_EQUIPMENT_RACK": StationArm(
            station_id="MEDICAL_EQUIPMENT_RACK",
            name="Intensive Care Unit Equipment Rack",
            arm_long_mm=3900.0,
            arm_lat_mm=350.0,
            max_capacity_kg=100.0,
        ),
        "AFT_BAGGAGE": StationArm(
            station_id="AFT_BAGGAGE",
            name="Aft Baggage Compartment",
            arm_long_mm=4650.0,
            arm_lat_mm=0.0,
            max_capacity_kg=180.0,
        ),
    }

    fuel_arms = [
        VariableFuelArmPoint(fuel_kg=0.0, arm_long_mm=3320.0, arm_lat_mm=0.0),
        VariableFuelArmPoint(fuel_kg=200.0, arm_long_mm=3340.0, arm_lat_mm=0.0),
        VariableFuelArmPoint(fuel_kg=400.0, arm_long_mm=3360.0, arm_lat_mm=0.0),
        VariableFuelArmPoint(fuel_kg=590.0, arm_long_mm=3380.0, arm_lat_mm=0.0),
    ]
    fuel_system = FuelSystemSpec(
        usable_fuel_capacity_kg=590.0,
        unusable_fuel_kg=6.0,
        tank_arms=fuel_arms,
        default_arm_long_mm=3350.0,
        default_arm_lat_mm=0.0,
    )

    cg_envelope = [
        CGEnvelopePoint(
            gross_weight_kg=1900.0,
            cg_long_fwd_mm=3120.0,
            cg_long_aft_mm=3480.0,
            cg_lat_left_mm=-70.0,
            cg_lat_right_mm=70.0,
        ),
        CGEnvelopePoint(
            gross_weight_kg=2600.0,
            cg_long_fwd_mm=3140.0,
            cg_long_aft_mm=3470.0,
            cg_lat_left_mm=-68.0,
            cg_lat_right_mm=68.0,
        ),
        CGEnvelopePoint(
            gross_weight_kg=3175.0,  # Standard MTOW 7000 lbs
            cg_long_fwd_mm=3170.0,
            cg_long_aft_mm=3450.0,
            cg_lat_left_mm=-65.0,
            cg_lat_right_mm=65.0,
        ),
    ]

    pa_grid = [-2000.0, 0.0, 2000.0, 4000.0, 6000.0, 8000.0, 10000.0, 12000.0]
    oat_grid = [-20.0, -10.0, 0.0, 10.0, 20.0, 30.0, 40.0, 50.0]

    ige_weights = [
        [3175, 3175, 3175, 3175, 3175, 3175, 3175, 3175],
        [3175, 3175, 3175, 3175, 3175, 3175, 3175, 3120],
        [3175, 3175, 3175, 3175, 3175, 3175, 3110, 3010],
        [3175, 3175, 3175, 3175, 3175, 3100, 3000, 2880],
        [3175, 3175, 3175, 3150, 3090, 2980, 2860, 2740],
        [3175, 3175, 3140, 3060, 2950, 2830, 2700, 2580],
        [3110, 3050, 2970, 2880, 2770, 2640, 2510, 2390],
        [2930, 2860, 2780, 2680, 2570, 2440, 2320, 2200],
    ]

    oge_weights = [
        [3175, 3175, 3175, 3175, 3175, 3175, 3175, 3080],
        [3175, 3175, 3175, 3175, 3175, 3175, 3070, 2950],
        [3175, 3175, 3175, 3175, 3120, 3020, 2900, 2780],
        [3175, 3175, 3175, 3080, 2970, 2850, 2730, 2610],
        [3175, 3110, 3030, 2920, 2800, 2680, 2560, 2440],
        [3050, 2960, 2860, 2750, 2630, 2500, 2380, 2260],
        [2880, 2790, 2680, 2570, 2450, 2320, 2200, 2080],
        [2700, 2600, 2500, 2380, 2260, 2140, 2010, 1900],
    ]

    oei_weights = [
        [3175, 3175, 3175, 3175, 3175, 3175, 3050, 2910],
        [3175, 3175, 3175, 3175, 3090, 2970, 2840, 2710],
        [3175, 3175, 3080, 2960, 2830, 2710, 2580, 2460],
        [3070, 2970, 2850, 2730, 2600, 2480, 2360, 2230],
        [2890, 2780, 2660, 2530, 2410, 2290, 2160, 2040],
        [2690, 2580, 2460, 2330, 2210, 2080, 1960, 1840],
        [2490, 2380, 2250, 2120, 2000, 1870, 1750, 1630],
        [2280, 2160, 2040, 1910, 1790, 1660, 1540, 1420],
    ]

    gw_grid = [2200.0, 2700.0, 3175.0]
    tas_3d = []
    ff_3d = []
    for pa in pa_grid:
        tas_oat_layer = []
        ff_oat_layer = []
        for oat in oat_grid:
            tas_gw_vec = []
            ff_gw_vec = []
            for gw in gw_grid:
                base_tas = 142.0 + (pa / 2000.0) * 1.8 - ((gw - 2200.0) / 975.0) * 6.0 + (oat - 15.0) * 0.1
                base_ff = 195.0 + ((gw - 2200.0) / 975.0) * 40.0 - (pa / 5000.0) * 7.0 + (oat / 50.0) * 5.0
                tas_gw_vec.append(round(base_tas, 1))
                ff_gw_vec.append(round(base_ff, 1))
            tas_oat_layer.append(tas_gw_vec)
            ff_oat_layer.append(ff_gw_vec)
        tas_3d.append(tas_oat_layer)
        ff_3d.append(ff_oat_layer)

    return RFMAirframeSpec(
        model_name="Bell 429 GlobalRanger",
        oem="Bell Helicopter",
        certification_basis="14 CFR Part 29 Category A",
        empty_weight_kg=2025.0,
        empty_cg_long_mm=3260.0,
        empty_cg_lat_mm=-3.0,
        mtow_kg=3175.0,
        min_flight_weight_kg=1800.0,
        fuel_system=fuel_system,
        stations=stations,
        cg_envelope=cg_envelope,
        ige_hover_grid=RFMPerformanceGrid(
            name="Bell 429 IGE Hover Ceiling",
            pa_grid_ft=pa_grid,
            oat_grid_c=oat_grid,
            weight_table_kg=ige_weights,
        ),
        oge_hover_grid=RFMPerformanceGrid(
            name="Bell 429 OGE Hover Ceiling",
            pa_grid_ft=pa_grid,
            oat_grid_c=oat_grid,
            weight_table_kg=oge_weights,
        ),
        oei_cat_a_grid=RFMPerformanceGrid(
            name="Bell 429 OEI Category A Helipad Performance",
            pa_grid_ft=pa_grid,
            oat_grid_c=oat_grid,
            weight_table_kg=oei_weights,
        ),
        cruise_performance=RFMCruiseGrid(
            pa_grid_ft=pa_grid,
            oat_grid_c=oat_grid,
            gw_grid_kg=gw_grid,
            tas_table_kts=tas_3d,
            fuel_flow_kg_hr=ff_3d,
        ),
        startup_fuel_kg=12.0,
        hover_fuel_flow_factor=1.18,
        loiter_fuel_flow_factor=0.86,
    )
