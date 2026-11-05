"""TraumaTeam Demonstration CLI & Benchmark Scenario Runner."""

import json
from traumateam.dispatch.decision_engine import TraumaTeamDispatchEngine
from traumateam.models.mission import Waypoint
from traumateam.models.weather import CorridorWeather, METARObservation, WindsAloftLayer, FlightRuleCategory

def run_demonstration():
    engine = TraumaTeamDispatchEngine()
    print("=" * 80)
    print(" TRAUMATEAM HEMS MISSION FEASIBILITY & DISPATCH ENGINE (14 CFR PART 135 SUBPART L)")
    print("=" * 80)

    # Scenario 1: Nominal Lowland Flight
    print("\n[SCENARIO 1] Standard Denver Metro Lowland Flight (KAPA -> Scene -> Swedish Hospital -> KAPA)")
    base = Waypoint(identifier="KAPA", name="Centennial Airport", lat=39.5701, lon=-104.8493, elevation_ft=5885.0)
    scene = Waypoint(identifier="SCENE", name="Highway 85 Trauma Scene", lat=39.7500, lon=-104.9900, elevation_ft=5280.0)
    hosp = Waypoint(identifier="SWEDISH", name="Swedish Medical Center (Helipad)", lat=39.6536, lon=-104.9781, elevation_ft=5350.0, is_elevated_helipad=True)

    wx1 = CorridorWeather(
        departure_metar=METARObservation(
            station_id="KAPA", observation_time_utc="1400Z", wind_dir_deg=180.0, wind_speed_kts=10.0,
            visibility_sm=10.0, ceiling_agl_ft=5000.0, temperature_c=18.0, dewpoint_c=4.0, altimeter_inhg=29.95,
            flight_category=FlightRuleCategory.VFR
        ),
        winds_aloft_profile=[WindsAloftLayer(altitude_msl_ft=6000.0, direction_deg=180.0, speed_kts=12.0, temperature_c=15.0)]
    )

    mission1 = engine.create_standard_hems_mission(
        mission_id="HEMS_SCENARIO_01",
        tail_number="N145TT",
        base_waypoint=base,
        scene_waypoint=scene,
        hospital_waypoint=hosp,
        initial_fuel_kg=420.0,
        flight_crew_weights={"PILOT_RH": 85.0, "COPILOT_LH": 80.0, "CREW_ATTENDANT_AFT_LH": 75.0, "CREW_ATTENDANT_AFT_RH": 82.0},
        patient_weight_kg=80.0,
        medical_gear_weight_kg=60.0,
        cruise_alt_msl_ft=6500.0,
        scene_hover_oge_min=3.0,
        is_day=True,
        is_mountainous=False,
        is_local_flying_area=True,
        nvg_or_htaws_equipped=True,
    )

    eval1 = engine.evaluate_mission("H145", mission1, wx1, altimeter_inhg=29.95, ambient_temp_c=18.0)
    print(f"VERDICT: {eval1.verdict.value} | LIMITING FACTOR: {eval1.limiting_factor.value}")
    print(f"Summary: {eval1.limiting_description}")
    print(f"Total Distance: {eval1.total_distance_nm:.1f} NM | Flight Time: {eval1.total_flight_time_min:.1f} min | Total Burn: {eval1.total_fuel_burn_kg:.1f} kg")
    print(f"Landing Reserve: {eval1.landing_reserve_fuel_kg:.1f} kg ({eval1.reserve_margin_minutes:.1f} min vs 20-min FAR 135.209 req)")

    # Scenario 2: High Density Altitude Mountain Rescue Edge Case
    print("\n" + "-" * 80)
    print("[SCENARIO 2] High-Density-Altitude Mountain Rescue Edge Case (Leadville / Turquoise Lake, 9800 ft)")
    base2 = Waypoint(identifier="KLXV", name="Leadville Airport", lat=39.2244, lon=-106.3169, elevation_ft=9934.0)
    scene2 = Waypoint(identifier="PASS", name="Turquoise Lake Pass", lat=39.2600, lon=-106.4000, elevation_ft=9800.0)
    hosp2 = Waypoint(identifier="SUMMIT", name="St. Anthony Summit Helipad", lat=39.5772, lon=-106.0714, elevation_ft=9100.0, is_elevated_helipad=True)

    wx2 = CorridorWeather(
        departure_metar=METARObservation(
            station_id="KLXV", observation_time_utc="1900Z", wind_dir_deg=260.0, wind_speed_kts=12.0,
            visibility_sm=10.0, ceiling_agl_ft=8000.0, temperature_c=28.0, dewpoint_c=2.0, altimeter_inhg=29.70,
            flight_category=FlightRuleCategory.VFR
        ),
        winds_aloft_profile=[WindsAloftLayer(altitude_msl_ft=12000.0, direction_deg=270.0, speed_kts=20.0, temperature_c=18.0)]
    )

    mission2 = engine.create_standard_hems_mission(
        mission_id="HEMS_SCENARIO_02_HIGH_DA",
        tail_number="N145TT",
        base_waypoint=base2,
        scene_waypoint=scene2,
        hospital_waypoint=hosp2,
        initial_fuel_kg=550.0,
        flight_crew_weights={"PILOT_RH": 90.0, "COPILOT_LH": 85.0, "CREW_ATTENDANT_AFT_LH": 80.0, "CREW_ATTENDANT_AFT_RH": 85.0},
        patient_weight_kg=90.0,
        medical_gear_weight_kg=85.0,
        cruise_alt_msl_ft=11500.0,
        scene_hover_oge_min=6.0,
        is_day=True,
        is_mountainous=True,
        is_local_flying_area=True,
        nvg_or_htaws_equipped=True,
    )

    eval2 = engine.evaluate_mission("H145", mission2, wx2, altimeter_inhg=29.70, ambient_temp_c=28.0)
    print(f"VERDICT: {eval2.verdict.value} | LIMITING FACTOR: {eval2.limiting_factor.value}")
    print(f"Summary: {eval2.limiting_description}")
    print(f"Actionable Mitigation: {eval2.mitigation_actions[0] if eval2.mitigation_actions else 'None'}")

    print("\n" + "=" * 80)

if __name__ == "__main__":
    run_demonstration()
