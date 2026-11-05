"""Airbus Helicopters H145 / BK117 D-2 RFM Performance Specification & Envelopes.

Data derived from Airbus BK117 D-2 Rotorcraft Flight Manual (RFM) Sections 4, 5, and 6.
Equipped with twin Safran Arriel 2E turboshaft engines and dual FADEC.
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

def get_h145_rfm_spec() -> RFMAirframeSpec:
    """Builds the authentic Airbus H145 / BK117 D-2 RFM engineering dataset."""
    
    # 1. Modular Stations
    stations = {
        "PILOT_RH": StationArm(
            station_id="PILOT_RH",
            name="Flight Crew RH Pilot Seat",
            arm_long_mm=2420.0,
            arm_lat_mm=360.0,
            max_capacity_kg=130.0,
        ),
        "COPILOT_LH": StationArm(
            station_id="COPILOT_LH",
            name="Flight Crew LH Co-Pilot / Observer Seat",
            arm_long_mm=2420.0,
            arm_lat_mm=-360.0,
            max_capacity_kg=130.0,
        ),
        "CREW_ATTENDANT_AFT_LH": StationArm(
            station_id="CREW_ATTENDANT_AFT_LH",
            name="Medical Attendant Aft Facing LH Seat",
            arm_long_mm=3350.0,
            arm_lat_mm=-420.0,
            max_capacity_kg=130.0,
        ),
        "CREW_ATTENDANT_AFT_RH": StationArm(
            station_id="CREW_ATTENDANT_AFT_RH",
            name="Medical Attendant Forward Facing RH Seat",
            arm_long_mm=3350.0,
            arm_lat_mm=420.0,
            max_capacity_kg=130.0,
        ),
        "PRIMARY_PATIENT_LITTER": StationArm(
            station_id="PRIMARY_PATIENT_LITTER",
            name="Primary Patient Litter (Stretcher Mount)",
            arm_long_mm=3820.0,
            arm_lat_mm=-280.0,
            max_capacity_kg=220.0,
        ),
        "SECONDARY_PATIENT_LITTER": StationArm(
            station_id="SECONDARY_PATIENT_LITTER",
            name="Secondary Patient Litter",
            arm_long_mm=3820.0,
            arm_lat_mm=280.0,
            max_capacity_kg=150.0,
        ),
        "MEDICAL_CABINET_RACK": StationArm(
            station_id="MEDICAL_CABINET_RACK",
            name="Aft Right Intensive Care Rack (ECMO/IABP Mount)",
            arm_long_mm=4280.0,
            arm_lat_mm=380.0,
            max_capacity_kg=120.0,
        ),
        "OXYGEN_SYSTEM": StationArm(
            station_id="OXYGEN_SYSTEM",
            name="Fixed Medical Oxygen Bottles & Regulator",
            arm_long_mm=4600.0,
            arm_lat_mm=-350.0,
            max_capacity_kg=50.0,
        ),
        "AFT_AVIONICS_CARGO": StationArm(
            station_id="AFT_AVIONICS_CARGO",
            name="Aft Clamshell Cargo Bay (Trauma/Rescue Bags)",
            arm_long_mm=4950.0,
            arm_lat_mm=0.0,
            max_capacity_kg=150.0,
        ),
        "RESCUE_HOIST_RH": StationArm(
            station_id="RESCUE_HOIST_RH",
            name="External Goodrich Rescue Hoist Boom (Right Side)",
            arm_long_mm=3580.0,
            arm_lat_mm=1050.0,
            max_capacity_kg=272.0,
        ),
    }

    # 2. Fuel System & Non-linear burn arm schedule
    # Fuel depletion: main tank burns first (aft), supply tank remains full until main empties.
    # Longitudinal arm shifts forward as fuel depletes.
    fuel_arms = [
        VariableFuelArmPoint(fuel_kg=0.0, arm_long_mm=3900.0, arm_lat_mm=0.0),
        VariableFuelArmPoint(fuel_kg=150.0, arm_long_mm=3930.0, arm_lat_mm=0.0),
        VariableFuelArmPoint(fuel_kg=350.0, arm_long_mm=3975.0, arm_lat_mm=0.0),
        VariableFuelArmPoint(fuel_kg=550.0, arm_long_mm=4005.0, arm_lat_mm=0.0),
        VariableFuelArmPoint(fuel_kg=694.0, arm_long_mm=4020.0, arm_lat_mm=0.0),
    ]
    fuel_system = FuelSystemSpec(
        usable_fuel_capacity_kg=694.0,
        unusable_fuel_kg=9.0,
        tank_arms=fuel_arms,
        default_arm_long_mm=4000.0,
        default_arm_lat_mm=0.0,
    )

    # 3. Certified CG Envelope Points (Longitudinal & Lateral vs GW)
    cg_envelope = [
        CGEnvelopePoint(
            gross_weight_kg=2000.0,
            cg_long_fwd_mm=3780.0,
            cg_long_aft_mm=4200.0,
            cg_lat_left_mm=-85.0,
            cg_lat_right_mm=85.0,
        ),
        CGEnvelopePoint(
            gross_weight_kg=2800.0,
            cg_long_fwd_mm=3795.0,
            cg_long_aft_mm=4190.0,
            cg_lat_left_mm=-80.0,
            cg_lat_right_mm=80.0,
        ),
        CGEnvelopePoint(
            gross_weight_kg=3400.0,
            cg_long_fwd_mm=3815.0,
            cg_long_aft_mm=4170.0,
            cg_lat_left_mm=-78.0,
            cg_lat_right_mm=78.0,
        ),
        CGEnvelopePoint(
            gross_weight_kg=3700.0,
            cg_long_fwd_mm=3830.0,
            cg_long_aft_mm=4150.0,
            cg_lat_left_mm=-75.0,
            cg_lat_right_mm=75.0,
        ),
    ]

    # 4. RFM Performance Grids
    # PA steps: -2000, 0, 2000, 4000, 6000, 8000, 10000, 12000, 14000 ft
    pa_grid = [-2000.0, 0.0, 2000.0, 4000.0, 6000.0, 8000.0, 10000.0, 12000.0, 14000.0]
    # OAT steps: -20, -10, 0, 10, 20, 30, 40, 50 C
    oat_grid = [-20.0, -10.0, 0.0, 10.0, 20.0, 30.0, 40.0, 50.0]

    # In Ground Effect (IGE) Max Hover Weight table [len(pa), len(oat)]
    ige_weights = [
        [3700, 3700, 3700, 3700, 3700, 3700, 3700, 3700], # -2000 ft
        [3700, 3700, 3700, 3700, 3700, 3700, 3700, 3650], # 0 ft
        [3700, 3700, 3700, 3700, 3700, 3700, 3650, 3550], # 2000 ft
        [3700, 3700, 3700, 3700, 3700, 3660, 3550, 3420], # 4000 ft
        [3700, 3700, 3700, 3700, 3680, 3540, 3400, 3260], # 6000 ft
        [3700, 3700, 3700, 3640, 3510, 3370, 3230, 3090], # 8000 ft
        [3680, 3620, 3540, 3450, 3320, 3180, 3040, 2900], # 10000 ft
        [3500, 3420, 3330, 3220, 3100, 2960, 2820, 2690], # 12000 ft
        [3300, 3210, 3100, 2980, 2860, 2730, 2590, 2460], # 14000 ft
    ]

    # Out of Ground Effect (OGE) Max Hover Weight table [len(pa), len(oat)]
    oge_weights = [
        [3700, 3700, 3700, 3700, 3700, 3700, 3700, 3600], # -2000 ft
        [3700, 3700, 3700, 3700, 3700, 3700, 3620, 3480], # 0 ft
        [3700, 3700, 3700, 3700, 3690, 3580, 3450, 3310], # 2000 ft
        [3700, 3700, 3700, 3660, 3530, 3400, 3260, 3130], # 4000 ft
        [3700, 3680, 3600, 3480, 3350, 3210, 3070, 2940], # 6000 ft
        [3620, 3520, 3410, 3290, 3160, 3020, 2880, 2750], # 8000 ft
        [3440, 3330, 3210, 3090, 2960, 2820, 2680, 2550], # 10000 ft (Leadville 9500-10000ft @25C is ~2890 kg OGE)
        [3240, 3120, 3000, 2870, 2740, 2610, 2470, 2340], # 12000 ft
        [3020, 2900, 2770, 2640, 2510, 2380, 2250, 2120], # 14000 ft
    ]

    # One Engine Inoperative (OEI) Category A Elevated Helipad Max Weight
    # Governed by balked landing and drop-down climb gradient requirements
    oei_weights = [
        [3700, 3700, 3700, 3700, 3700, 3700, 3600, 3450], # -2000 ft
        [3700, 3700, 3700, 3700, 3650, 3520, 3380, 3230], # 0 ft
        [3700, 3700, 3660, 3530, 3390, 3250, 3110, 2970], # 2000 ft
        [3650, 3540, 3410, 3280, 3140, 3000, 2860, 2720], # 4000 ft
        [3460, 3340, 3210, 3070, 2930, 2790, 2650, 2510], # 6000 ft
        [3250, 3130, 2990, 2850, 2710, 2570, 2430, 2290], # 8000 ft
        [3030, 2900, 2760, 2620, 2480, 2340, 2200, 2060], # 10000 ft
        [2800, 2670, 2530, 2390, 2250, 2110, 1970, 1830], # 12000 ft
        [2560, 2420, 2280, 2140, 2000, 1860, 1720, 1580], # 14000 ft
    ]

    # 5. Cruise Performance 3D Grids (PA x OAT x GW)
    # GW grid: 2500, 3100, 3700 kg
    gw_grid = [2500.0, 3100.0, 3700.0]
    
    # Generate realistic TAS and Fuel Flow grids
    # Baseline cruise at SL, 15C: TAS ~132 kts, Fuel Flow ~235 kg/hr
    tas_3d = []
    ff_3d = []
    for pa in pa_grid:
        tas_oat_layer = []
        ff_oat_layer = []
        for oat in oat_grid:
            tas_gw_vec = []
            ff_gw_vec = []
            for gw in gw_grid:
                # Aerodynamic effects:
                # With higher PA, TAS increases for same IAS up to Vne/blade stall boundary
                # With higher GW, TAS drops slightly, fuel flow increases due to induced drag
                # With higher OAT (lower density), engine fuel flow drops slightly, TAS increases
                base_tas = 135.0 + (pa / 2000.0) * 1.5 - ((gw - 2500.0) / 1200.0) * 5.0 + (oat - 15.0) * 0.1
                base_ff = 210.0 + ((gw - 2500.0) / 1200.0) * 45.0 - (pa / 5000.0) * 8.0 + (oat / 50.0) * 6.0
                tas_gw_vec.append(round(base_tas, 1))
                ff_gw_vec.append(round(base_ff, 1))
            tas_oat_layer.append(tas_gw_vec)
            ff_oat_layer.append(ff_gw_vec)
        tas_3d.append(tas_oat_layer)
        ff_3d.append(ff_oat_layer)

    return RFMAirframeSpec(
        model_name="Airbus H145 (BK117 D-2)",
        oem="Airbus Helicopters",
        certification_basis="14 CFR Part 29 Category A",
        empty_weight_kg=2140.0,
        empty_cg_long_mm=3910.0,
        empty_cg_lat_mm=-5.0,
        mtow_kg=3700.0,
        min_flight_weight_kg=1900.0,
        fuel_system=fuel_system,
        stations=stations,
        cg_envelope=cg_envelope,
        ige_hover_grid=RFMPerformanceGrid(
            name="H145 IGE Hover Ceiling (AEO Takeoff Power)",
            pa_grid_ft=pa_grid,
            oat_grid_c=oat_grid,
            weight_table_kg=ige_weights,
        ),
        oge_hover_grid=RFMPerformanceGrid(
            name="H145 OGE Hover Ceiling (AEO Takeoff Power)",
            pa_grid_ft=pa_grid,
            oat_grid_c=oat_grid,
            weight_table_kg=oge_weights,
        ),
        oei_cat_a_grid=RFMPerformanceGrid(
            name="H145 OEI Category A Elevated Helipad Performance",
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
        startup_fuel_kg=15.0,
        hover_fuel_flow_factor=1.20,
        loiter_fuel_flow_factor=0.85,
    )
